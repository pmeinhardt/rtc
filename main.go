package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pion/webrtc/v3"
)

func encode(obj interface{}) string {
	b, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	return string(b)
}

func decode(in string, obj interface{}) {
	b := []byte(in)

	err := json.Unmarshal(b, obj)
	if err != nil {
		panic(err)
	}
}

func send(helper, value string) error {
	cmd := exec.Command(helper, "send")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}

	go func() {
		defer stdin.Close()
		stdin.Write([]byte(value))
	}()

	return cmd.Run()
}

func recv(helper string) (string, error) {
	var str strings.Builder

	cmd := exec.Command(helper, "recv")
	cmd.Stdout = &str

	err := cmd.Run()
	stdout := str.String()

	return stdout, err
}

func call(params Params, input chan []byte) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: params.urls}},
	}

	connection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := connection.Close(); err != nil {
			fmt.Printf("Cannot close connection: %v\n", err)
		}
	}()

	stdin := os.Stdin
	stdout := os.Stdout
	stderr := os.Stderr

	connection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateFailed {
			stderr.WriteString("Peer connection failed")
			os.Exit(1)
		}
	})

	channel, err := connection.CreateDataChannel("data", nil)
	if err != nil {
		panic(err)
	}

	channel.OnOpen(func() {
		buffer := make([]byte, 4096)

		for {
			nbytes, err := stdin.Read(buffer)

			if nbytes > 0 {
				channel.Send(buffer[:nbytes-1])
			}

			if err != nil {
				// continue receiving?
			}
		}
	})

	channel.OnClose(func() {
		os.Exit(0)
	})

	channel.OnError(func(err error) {
		stderr.WriteString(fmt.Sprintf("%v", err))
	})

	channel.OnMessage(func(msg webrtc.DataChannelMessage) {
		stdout.Write(msg.Data)
	})

	// Create a first offer that does not contain any local candidates
	offer, err := connection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	// Create channel that is blocked until ICE Gathering is complete
	gathered := webrtc.GatheringCompletePromise(connection)

	// Sets the LocalDescription, and starts our UDP listeners
	// Note: this will start the gathering of ICE candidates
	if err = connection.SetLocalDescription(offer); err != nil {
		panic(err)
	}

	// Block until ICE Gathering is complete, not using trickle ICE.
	// We do this because we only want to exchange one signaling message.
	<-gathered

	// Compute the local offer again. This second offer contains all found
	// candidates, and may be sent to the peer with no need for further
	// communication. We simply check that it contains at least one candidate.
	offer2 := connection.LocalDescription()
	if !strings.Contains(offer2.SDP, "\na=candidate:") {
		panic("No SDP offer candidate")
	}

	err = send(params.helper, encode(offer2))
	if err != nil {
		panic(err)
	}

	sdp, err := recv(params.helper)
	if err != nil {
		panic(err)
	}

	// Wait for the offer to be pasted
	answer := webrtc.SessionDescription{}
	decode(sdp, &answer)

	// Set the remote SessionDescription
	err = connection.SetRemoteDescription(answer)
	if err != nil {
		panic(err)
	}

	// Block forever
	select {}
}

func join(params Params, input chan []byte) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: params.urls}},
	}

	connection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := connection.Close(); err != nil {
			fmt.Printf("cannot close connection: %v\n", err)
		}
	}()

	stdin := os.Stdin
	stdout := os.Stdout
	stderr := os.Stderr

	connection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateFailed {
			stderr.WriteString("Peer connection failed")
			os.Exit(1)
		}
	})

	connection.OnDataChannel(func(channel *webrtc.DataChannel) {
		channel.OnOpen(func() {
			buffer := make([]byte, 4096)

			for {
				nbytes, err := stdin.Read(buffer)

				if nbytes > 0 {
					channel.Send(buffer[:nbytes-1])
				}

				if err != nil {
					// continue receiving?
				}
			}
		})

		channel.OnClose(func() {
			os.Exit(0)
		})

		channel.OnError(func(err error) {
			stderr.WriteString(fmt.Sprintf("%v", err))
		})

		channel.OnMessage(func(msg webrtc.DataChannelMessage) {
			stdout.Write(msg.Data)
		})
	})

	sdp, err := recv(params.helper)
	if err != nil {
		panic(err)
	}

	// Wait for the offer to be pasted
	offer := webrtc.SessionDescription{}
	decode(sdp, &offer)

	// Set the remote SessionDescription
	err = connection.SetRemoteDescription(offer)
	if err != nil {
		panic(err)
	}

	// Create an answer that does not contain any local candidates
	answer, err := connection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	// Create channel that is blocked until ICE Gathering is complete
	gathered := webrtc.GatheringCompletePromise(connection)

	// Sets the LocalDescription, and starts our UDP listeners
	// Note: this will start the gathering of ICE candidates
	if err = connection.SetLocalDescription(answer); err != nil {
		panic(err)
	}

	// Block until ICE Gathering is complete, disabling trickle ICE.
	// We do this because we only can exchange one signaling message.
	<-gathered

	// Compute the local answer again. This second answer contains all
	// candidates, and may be sent to the peer with no need for further
	// communication. We simply check that it contains at least one candidate.
	answer2 := connection.LocalDescription()
	if !strings.Contains(answer2.SDP, "\na=candidate:") {
		panic("No candidate")
	}

	err = send(params.helper, encode(answer2))
	if err != nil {
		panic(err)
	}

	// Block forever
	select {}
}

type Params struct {
	helper string
	urls   []string
}

func main() {
	helper := flag.String("helper", "./signal/editor", "signaling helper")
	dojoin := flag.Bool("join", false, "join an existing session")
	flag.Parse()

	params := Params{
		helper: *helper,
		urls:   []string{"stun:stun.l.google.com:19302"},
	}

	input := make(chan []byte)

	if *dojoin {
		join(params, input)
	} else {
		call(params, input)
	}
}
