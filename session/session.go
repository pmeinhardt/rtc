package session

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/pion/webrtc/v3"
)

type Params struct {
	Context       context.Context
	ICEServerURLs []string
}

type Description webrtc.SessionDescription

type Event interface{}

type Msg []byte

type Session struct {
	ctx    context.Context
	cancel context.CancelCauseFunc

	params Params

	events chan Event

	incoming chan Msg
	outgoing chan Msg

	pc *webrtc.PeerConnection
	dc *webrtc.DataChannel
}

type (
	ConnectionStateChangeEvent struct{ state webrtc.PeerConnectionState }
	ConnectionDataChannelEvent struct{ dc *webrtc.DataChannel }

	ChannelDialEvent              struct{ dc *webrtc.DataChannel }
	ChannelOpenEvent              struct{ dc *webrtc.DataChannel }
	ChannelCloseEvent             struct{ dc *webrtc.DataChannel }
	ChannelBufferedAmountLowEvent struct{ dc *webrtc.DataChannel }

	ChannelMessageEvent struct {
		dc  *webrtc.DataChannel
		msg webrtc.DataChannelMessage
	}

	ChannelErrorEvent struct {
		dc  *webrtc.DataChannel
		err error
	}

	CloseEvent struct{}
)

const DataChannelLabel = "data"

var (
	ErrNoICECandidates  = errors.New("no ICE candidates")
	ErrConnectionFailed = errors.New("connection failed")
	ErrChannelClosed    = errors.New("channel closed")
	Closed              = errors.New("closed")
	done                = errors.New("done")
)

func WithDefaults(params Params) Params {
	if params.ICEServerURLs == nil {
		params.ICEServerURLs = []string{"stun:stun.l.google.com:19302"}
	}

	if params.Context == nil {
		params.Context = context.Background()
	}

	return params
}

func New(params Params) *Session {
	params = WithDefaults(params)

	ctx, cancel := context.WithCancelCause(params.Context)

	s := &Session{
		ctx:    ctx,
		cancel: cancel,

		params: params,

		events: make(chan Event),

		incoming: make(chan Msg),
		outgoing: make(chan Msg),
	}

	return s
}

func assert(cond bool, msg string) {
	if !cond {
		panic(errors.New(msg))
	}
}

func config(params Params) webrtc.Configuration {
	return webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: params.ICEServerURLs}},
	}
}

func check(desc *webrtc.SessionDescription) error {
	if !strings.Contains(desc.SDP, "\na=candidate:") {
		return ErrNoICECandidates
	}

	return nil
}

func (s *Session) Initiate() (*Description, error) {
	// Initialize a new peer connection and data channel.

	pc, err := webrtc.NewPeerConnection(config(s.params))
	if err != nil {
		return nil, err
	}

	s.pc = pc

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		select {
		case <-s.ctx.Done():
		case s.events <- ConnectionStateChangeEvent{state}:
		}
	})

	dc, err := pc.CreateDataChannel(DataChannelLabel, nil)
	if err != nil {
		return nil, err
	}

	s.subscribe(dc)

	// Create a first offer that does not contain any local candidates.
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return nil, err
	}

	// Create channel that is blocked until ICE Gathering is complete.
	gathered := webrtc.GatheringCompletePromise(pc)

	// Sets the LocalDescription, and starts our UDP listeners.
	// Note: This will start the gathering of ICE candidates.
	if err := pc.SetLocalDescription(offer); err != nil {
		return nil, err
	}

	// Block until ICE Gathering is complete, not using trickle ICE.
	// We do this because we only want to exchange one signaling message.
	select {
	case <-s.ctx.Done():
		return nil, context.Cause(s.ctx)
	case <-gathered:
	}

	// Compute the local offer again. This second offer contains all found
	// candidates, and may be sent to the peer with no need for further
	// communication.
	offer2 := pc.LocalDescription()

	// Make sure we have a viable offer.
	if err := check(offer2); err != nil {
		return nil, err
	}

	return (*Description)(offer2), nil
}

func (s *Session) Join(desc Description) (*Description, error) {
	// Initialize a new peer connection and data channel.

	pc, err := webrtc.NewPeerConnection(config(s.params))
	if err != nil {
		return nil, err
	}

	s.pc = pc

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		select {
		case <-s.ctx.Done():
		case s.events <- ConnectionStateChangeEvent{state}:
		}
	})

	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		select {
		case <-s.ctx.Done():
		case s.events <- ConnectionDataChannelEvent{dc}:
		}
	})

	// Set the remote SessionDescription.
	if err := pc.SetRemoteDescription(webrtc.SessionDescription(desc)); err != nil {
		return nil, err
	}

	// Create an answer that does not contain any local candidates.
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}

	// Create channel that is blocked until ICE Gathering is complete.
	gathered := webrtc.GatheringCompletePromise(pc)

	// Sets the LocalDescription, and starts our UDP listeners.
	// Note: This will start the gathering of ICE candidates.
	if err := pc.SetLocalDescription(answer); err != nil {
		return nil, err
	}

	// Block until ICE Gathering is complete, not using trickle ICE.
	// We do this because we only want to exchange one signaling message.
	select {
	case <-s.ctx.Done():
		return nil, context.Cause(s.ctx)
	case <-gathered:
	}

	// Compute the local answer again. This second answer contains all found
	// candidates, and may be sent to the peer with no need for further
	// communication.
	answer2 := pc.LocalDescription()

	// Make sure we have a viable offer.
	if err := check(answer2); err != nil {
		return nil, err
	}

	return (*Description)(answer2), nil
}

func (s *Session) Accept(desc Description) error {
	select {
	case <-s.ctx.Done():
		return context.Cause(s.ctx)
	default:
		return s.pc.SetRemoteDescription(webrtc.SessionDescription(desc))
	}
}

func (s *Session) subscribe(dc *webrtc.DataChannel) {
	assert(dc.Ordered(), "channel not ordered")
	assert(dc.MaxPacketLifeTime() == nil, "channel packet lifetime limited")
	assert(dc.MaxRetransmits() == nil, "channel retransmits limited")

	s.dc = dc

	dc.OnBufferedAmountLow(func() {
		select {
		case <-s.ctx.Done():
		case s.events <- ChannelBufferedAmountLowEvent{dc}:
		}
	})
	dc.OnClose(func() {
		select {
		case <-s.ctx.Done():
		case s.events <- ChannelCloseEvent{dc}:
		}
	})
	dc.OnDial(func() {
		select {
		case <-s.ctx.Done():
		case s.events <- ChannelDialEvent{dc}:
		}
	})
	dc.OnError(func(err error) {
		select {
		case <-s.ctx.Done():
		case s.events <- ChannelErrorEvent{dc, err}:
		}
	})
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		select {
		case <-s.ctx.Done():
		case s.events <- ChannelMessageEvent{dc, msg}:
		}
	})
	dc.OnOpen(func() {
		select {
		case <-s.ctx.Done():
		case s.events <- ChannelOpenEvent{dc}:
		}
	})
}

func (s *Session) unsubscribe() {
	dc := s.dc
	if dc == nil {
		return
	}

	dc.OnBufferedAmountLow(nil)
	dc.OnClose(nil)
	dc.OnDial(nil)
	dc.OnError(nil)
	dc.OnMessage(nil)
	dc.OnOpen(nil)

	s.dc = nil
}

func (s *Session) Run(path string, args ...string) error {
	ctx, cancel := context.WithCancelCause(s.ctx)
	cmd := exec.CommandContext(ctx, path, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel(err)
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel(err)
		return err
	}

	go func() {
		defer stdout.Close()

		// TODO: Size buffer according to SCTP max message size
		buffer := make([]byte, 16384)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := stdout.Read(buffer)
				if err != nil {
					if err != io.EOF {
						cancel(err)
					} else {
						cancel(done)
					}

					continue
				}

				if n > 0 {
					if err := s.Write(buffer[:n]); err != nil {
						cancel(err)
					}
				}
			}
		}
	}()

	go func() {
		defer stdin.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				buffer, err := s.Read()
				if err != nil {
					cancel(err)
				}

				_, err = stdin.Write(buffer)
				if err != nil {
					cancel(err)
				}
			}
		}
	}()

	if err := cmd.Run(); err != nil {
		cancel(err)
	} else {
		cancel(done)
	}

	if err := context.Cause(ctx); err != done {
		return err
	}

	return nil
}

func (s *Session) Loop() error {
	err := s.loop()
	s.shutdown()
	return err
}

func (s *Session) loop() error {
	for {
		select {
		case <-s.ctx.Done():
			if err := context.Cause(s.ctx); err != Closed {
				return err
			}
			return nil
		case evt := <-s.events:
			switch evt := evt.(type) {
			case CloseEvent:
				s.cancel(Closed)
			case ConnectionStateChangeEvent:
				if evt.state == webrtc.PeerConnectionStateFailed {
					s.cancel(ErrConnectionFailed)
				}
			case ConnectionDataChannelEvent:
				if evt.dc.Label() == DataChannelLabel {
					s.subscribe(evt.dc)
				}
			case ChannelBufferedAmountLowEvent:
				continue
			case ChannelCloseEvent:
				s.cancel(ErrChannelClosed)
			case ChannelDialEvent:
				continue
			case ChannelErrorEvent:
				s.cancel(evt.err)
			case ChannelOpenEvent:
				continue
			case ChannelMessageEvent:
				s.incoming <- evt.msg.Data // this can block, be sure to read
			default:
				msg := fmt.Sprintf("unhandled event: %v", evt)
				panic(errors.New(msg))
			}
		case b := <-s.outgoing:
			if err := s.dc.Send(b); err != nil {
				s.cancel(err)
			}
		}
	}
}

func (s *Session) shutdown() {
	s.unsubscribe()

	// TODO:
	// If we don't see connection/channel here, could they be assigned later,
	// after we've checked them, and leak? Seems unlikely but possible.

	if s.dc != nil {
		s.dc.Close()
	}

	if s.pc != nil {
		s.pc.Close()
	}

	for {
		select {
		case <-s.events:
		case <-s.incoming:
		case <-s.outgoing:
		default:
			return
		}
	}
}

func (s *Session) Read() ([]byte, error) {
	select {
	case <-s.ctx.Done():
		return nil, context.Cause(s.ctx)
	case p := <-s.incoming:
		return p, nil
	}
}

func (s *Session) Write(p []byte) error {
	select {
	case <-s.ctx.Done():
		return context.Cause(s.ctx)
	case s.outgoing <- p:
		return nil
	}
}

func (s *Session) Close() error {
	for {
		select {
		case <-s.ctx.Done():
			if err := context.Cause(s.ctx); err != Closed {
				return err
			}
			return nil
		case s.events <- CloseEvent{}:
		}
	}
}
