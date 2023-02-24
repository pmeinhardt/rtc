package connection

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pion/webrtc/v3"
)

func Join(cmd *exec.Cmd, params Params, signals chan webrtc.SessionDescription) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: params.URLs}},
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

	connection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateFailed {
			os.Stderr.WriteString("Peer connection failed")
			os.Exit(1)
		}
	})

	connection.OnDataChannel(func(channel *webrtc.DataChannel) {
		// TODO: Check channel is data

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
	})

	// Wait for the offer to be received
	offer := <-signals

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

	signals <- *answer2

	// Block forever
	select {}
}
