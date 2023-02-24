package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/pmeinhardt/rtc/connection"
	"github.com/pmeinhardt/rtc/signaling"
	// "github.com/pmeinhardt/rtc/web"

	"github.com/pion/webrtc/v3"
	"github.com/spf13/cobra"
)

var (
	quiet bool
	port  uint16
)

var params = connection.Params{
	URLs: []string{"stun:stun.l.google.com:19302"},
}

var initialize = &cobra.Command{
	Use:   "init [flags] [command [args ...]]",
	Short: "Set up a new peer connection",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		var ccmd *exec.Cmd

		if len(args) > 0 {
			ccmd = exec.Command(args[0])
		} else {
			ccmd = exec.Command("cat")
		}

		sig := signaling.NewCommandTransport("./builtin-signaling-helpers/editor")
		signals := make(chan webrtc.SessionDescription)

		go connection.Init(ccmd, params, signals)

		offer := <-signals

		err = sig.Send(&offer)
		if err != nil {
			return err
		}

		answer, err := sig.Receive()
		if err != nil {
			return err
		}

		signals <- *answer

		select {} // block

		// return nil
	},
}

var join = &cobra.Command{
	Use:   "join [flags] [command [args ...]]",
	Short: "Join a connection initialized by a peer",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement

		var err error
		var ccmd *exec.Cmd

		if len(args) > 0 {
			ccmd = exec.Command(args[0])
		} else {
			ccmd = exec.Command("cat")
		}

		sig := signaling.NewCommandTransport("./builtin-signaling-helpers/editor")
		signals := make(chan webrtc.SessionDescription)

		go connection.Join(ccmd, params, signals)

		answer, err := sig.Receive()
		if err != nil {
			return err
		}

		signals <- *answer

		offer := <-signals

		err = sig.Send(&offer)
		if err != nil {
			return err
		}

		// TODO: Synchronize?

		select {} // block

		// return nil
	},
}

var web = &cobra.Command{
	Use:   "web [flags]",
	Short: "Open a web interface",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement
		return nil
	},
}

var cmd = &cobra.Command{
	Use:   "rtc",
	Short: "Communicate with peers - directly, in \"real-time\".",
}

func init() {
	// TODO: Default to true if we are running from a script (no tty)?
	cmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "be quiet, do not output status and progress messages")

	web.Flags().Uint16VarP(&port, "port", "p", 8000, "port to listen on")

	cmd.AddCommand(initialize)
	cmd.AddCommand(join)
	cmd.AddCommand(web)
}

func Run() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "An error occurred: %v\n", err)
		os.Exit(1)
	}
}
