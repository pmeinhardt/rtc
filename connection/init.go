package connection

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pion/webrtc/v3"
)

func Init(cmd *exec.Cmd, params Params, signals chan webrtc.SessionDescription) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: params.URLs}},
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

	connection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateFailed {
			os.Stderr.WriteString("Peer connection failed")
			os.Exit(1)
		}
	})

	// TODO: Create separate channels for stdout and stderr?
	channel, err := connection.CreateDataChannel("data", nil)
	if err != nil {
		panic(err)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	channel.OnClose(func() {
		os.Exit(0)
	})

	channel.OnError(func(err error) {
		os.Stderr.WriteString(fmt.Sprintf("%v\n", err))
	})

	channel.OnMessage(func(msg webrtc.DataChannelMessage) {
		stdin.Write(msg.Data)
	})

	channel.OnOpen(func() {
		// TODO: Start cmd and write incoming channel messages to stdin, send stdout + stderr through channel
		if err := cmd.Start(); err != nil {
			panic(err)
		}

		go func() {
			buffer := make([]byte, 4096)

			for {
				nbytes, err := stdout.Read(buffer)
				if err != nil {
					panic(err)
				}

				if nbytes > 0 {
					channel.Send(buffer[:nbytes])
				}
			}
		}()
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

	// Send the offer through the signaling channel
	signals <- *offer2

	// Wait for the offer to be received
	answer := <-signals

	// Set the remote SessionDescription
	err = connection.SetRemoteDescription(answer)
	if err != nil {
		panic(err)
	}

	// Block forever
	select {}
}
