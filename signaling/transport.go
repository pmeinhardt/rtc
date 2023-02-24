package signaling

import (
	"bytes"
	"encoding/json"
	"os/exec"

	"github.com/pion/webrtc/v3"
)

type Transport interface {
	Receive() (*webrtc.SessionDescription, error)
	Send(desc *webrtc.SessionDescription) error
}

type CommandTransport struct {
	cmd string
}

func (t *CommandTransport) Receive() (*webrtc.SessionDescription, error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.Command(t.cmd, "recv")

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	desc := webrtc.SessionDescription{}

	if err := json.Unmarshal(stdout.Bytes(), &desc); err != nil {
		return nil, err
	}

	return &desc, nil
}

func (t *CommandTransport) Send(desc *webrtc.SessionDescription) error {
	var stdout, stderr bytes.Buffer

	cmd := exec.Command(t.cmd, "send")

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	data, err := json.Marshal(desc)
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()
		stdin.Write(data)
	}()

	return cmd.Run()
}

func NewCommandTransport(cmd string) Transport {
	return &CommandTransport{cmd}
}
