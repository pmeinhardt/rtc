package signaling

import (
	"bytes"
	"os/exec"
)

type Transport[Signal any] interface {
	Receive() (*Signal, error)
	Send(signal Signal) error
}

type CommandTransport[Signal any] struct {
	Path string
}

func NewCommandTransport[Signal any](path string) Transport[Signal] {
	return &CommandTransport[Signal]{path}
}

func (t *CommandTransport[Signal]) Receive() (*Signal, error) {
	var stdout bytes.Buffer

	cmd := exec.Command(t.Path, "recv")

	cmd.Stdout = &stdout
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var signal Signal
	data := stdout.Bytes()

	if err := Unmarshal(data, &signal); err != nil {
		return nil, err
	}

	return &signal, nil
}

func (t *CommandTransport[Signal]) Send(signal Signal) error {
	data, err := Marshal(signal)
	if err != nil {
		return err
	}

	cmd := exec.Command(t.Path, "send")

	cmd.Stdin = bytes.NewBuffer(data)
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run()
}
