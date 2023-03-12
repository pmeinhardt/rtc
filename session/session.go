package session

import (
	"context"
	"errors"
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

type Msg []byte

type Event interface{}

type Session struct {
	ctx    context.Context
	cancel context.CancelCauseFunc

	params Params

	incoming chan Msg
	outgoing chan Msg

	events chan Event
	errors chan error

	pc *webrtc.PeerConnection
	dc *webrtc.DataChannel
}

type (
	ConnectionStateChangeEvent struct{ state webrtc.PeerConnectionState }

	ChannelDialEvent              struct{}
	ChannelOpenEvent              struct{}
	ChannelCloseEvent             struct{}
	ChannelErrorEvent             struct{ err error }
	ChannelBufferedAmountLowEvent struct{}
	ChannelMessageEvent           struct{ msg webrtc.DataChannelMessage }

	CloseEvent struct{}
)

const DataChannelLabel = "data"

var (
	ErrNoICECandidate   = errors.New("no ICE candidate")
	ErrConnectionFailed = errors.New("connection failed")
	ErrChannelClosed    = errors.New("channel closed")
	ErrClosed           = errors.New("closed")
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

func NewSession(params Params) *Session {
	params = WithDefaults(params)

	ctx, cancel := context.WithCancelCause(params.Context)

	s := &Session{
		ctx:    ctx,
		cancel: cancel,

		params: params,

		incoming: make(chan Msg),
		outgoing: make(chan Msg),

		events: make(chan Event),
		errors: make(chan error),
	}

	return s
}

func config(params Params) webrtc.Configuration {
	return webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: params.ICEServerURLs}},
	}
}

func check(desc *webrtc.SessionDescription) error {
	if !strings.Contains(desc.SDP, "\na=candidate:") {
		return ErrNoICECandidate
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
	<-gathered

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
		if dc.Label() == DataChannelLabel {
			s.subscribe(dc)
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
	<-gathered

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
	return s.pc.SetRemoteDescription(webrtc.SessionDescription(desc))
}

func (s *Session) subscribe(dc *webrtc.DataChannel) {
	s.dc = dc

	dc.OnBufferedAmountLow(func() {
		select {
		case <-s.ctx.Done():
		case s.events <- ChannelBufferedAmountLowEvent{}:
		}
	})
	dc.OnClose(func() {
		select {
		case <-s.ctx.Done():
		case s.events <- ChannelCloseEvent{}:
		}
	})
	dc.OnDial(func() {
		select {
		case <-s.ctx.Done():
		case s.events <- ChannelDialEvent{}:
		}
	})
	dc.OnError(func(err error) {
		select {
		case <-s.ctx.Done():
		case s.events <- ChannelErrorEvent{err}:
		}
	})
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		select {
		case <-s.ctx.Done():
		case s.events <- ChannelMessageEvent{msg}:
		}

		select {
		case <-s.ctx.Done():
		case s.incoming <- msg.Data:
		}
	})
	dc.OnOpen(func() {
		select {
		case <-s.ctx.Done():
		case s.events <- ChannelOpenEvent{}:
		}
	})
}

func (s *Session) Attach(path string, args ...string) error {
	cmd := exec.CommandContext(s.ctx, path, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		defer stdout.Close()

		// TODO: Size buffer according to SCTP max message size
		buffer := make([]byte, 16384)

		for {
			select {
			case <-s.ctx.Done():
				if err := stdout.Close(); err != nil {
					s.errors <- err
				}

				return
			default:
				n, err := stdout.Read(buffer)
				if err != nil {
					if err == io.EOF {
						s.errors <- ErrClosed
					} else {
						s.errors <- err
					}
					return
				}

				if n > 0 {
					s.Send(buffer[:n])
				}
			}
		}
	}()

	go func() {
		defer stdin.Close()

		for {
			select {
			case <-s.ctx.Done():
				if err := stdin.Close(); err != nil {
					s.errors <- err
				}

				return
			default:
				buffer := s.Read()
				_, err := stdin.Write(buffer)
				if err != nil {
					s.errors <- err
					return
				}
			}
		}
	}()

	go func() {
		if err := cmd.Wait(); err != nil {
			s.errors <- err
		}
	}()

	return nil
}

func (s *Session) Loop() error {
	err := s.loop()

	if errors.Is(err, ErrClosed) {
		err = nil
	}

	s.shutdown()

	return err
}

func (s *Session) loop() error {
	for {
		select {
		case <-s.ctx.Done():
			err := context.Cause(s.ctx)
			return err
		case err := <-s.errors:
			s.cancel(err)
		case evt := <-s.events:
			switch evt := evt.(type) {
			case CloseEvent:
				s.errors <- ErrClosed
			case ConnectionStateChangeEvent:
				if evt.state == webrtc.PeerConnectionStateFailed {
					s.errors <- ErrConnectionFailed
				}
			case ChannelCloseEvent:
				s.errors <- ErrChannelClosed
			case ChannelErrorEvent:
				s.errors <- evt.err
			default:
				// ?
			}
		case b := <-s.outgoing:
			// TODO: Split message or, alternatively, error on message larger than max

			// caps := s.pc.SCTP().GetCapabilities()
			// max := caps.MaxMessageSize

			if err := s.dc.Send(b); err != nil {
				s.errors <- err
			}
		}
	}
}

func (s *Session) shutdown() {
	// s.dc.Close()
	// s.pc.Close()

	for {
		select {
		case <-s.events:
		case <-s.errors:
		case <-s.incoming:
		case <-s.outgoing:
		default:
			return
		}
	}
}

func (s *Session) Read() []byte {
	select {
	case <-s.ctx.Done():
		return []byte{}
	case p := <-s.incoming:
		return p
	}
}

func (s *Session) Send(p []byte) {
	select {
	case <-s.ctx.Done():
	case s.outgoing <- p:
	}
}

func (s *Session) Close() {
	select {
	case <-s.ctx.Done():
	case s.events <- CloseEvent{}:
	}
}
