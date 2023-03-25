package cli

import (
	"log"
	"os"

	"github.com/pmeinhardt/rtc/session"
	"github.com/pmeinhardt/rtc/signaling"
	// "github.com/pmeinhardt/rtc/web"

	"github.com/spf13/cobra"
)

var (
	sign         = "./signaling-plugins/editor"
	port  uint16 = 8000
	quiet bool
)

var logger = log.New(os.Stderr, "", 0)

var initialize = &cobra.Command{
	Use:   "init [flags] command [args ...]",
	Short: "Set up a new peer connection",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var params = session.Params{}

		signal := signaling.NewCommandTransport[session.Description](sign)

		s := session.New(params)
		defer s.Close()
		go s.Loop()

		offer, err := s.Initiate()
		if err != nil {
			logger.Fatalln(err)
		}

		if err := signal.Send(*offer); err != nil {
			logger.Fatalln(err)
		}

		answer, err := signal.Receive()
		if err != nil {
			logger.Fatalln(err)
		}

		if err := s.Accept(*answer); err != nil {
			logger.Fatalln(err)
		}

		// TODO: Synchronize and do not write before peer signal.Send has finished/peer is attached
		// Peer needs to signal readiness

		if err := s.Run(args[0], args[1:]...); err != nil {
			logger.Fatalln(err)
		}

		if err := s.Close(); err != nil {
			logger.Fatalln(err)
		}
	},
}

var join = &cobra.Command{
	Use:   "join [flags] command [args ...]",
	Short: "Join a connection initialized by a peer",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var params = session.Params{}

		signal := signaling.NewCommandTransport[session.Description](sign)

		s := session.New(params)
		defer s.Close()
		go s.Loop()

		offer, err := signal.Receive()
		if err != nil {
			logger.Fatalln(err)
		}

		answer, err := s.Join(*offer)
		if err != nil {
			logger.Fatalln(err)
		}

		if err := signal.Send(*answer); err != nil {
			logger.Fatalln(err)
		}

		// TODO: Signal readiness and wait for peer to acknowledge

		if err := s.Run(args[0], args[1:]...); err != nil {
			logger.Fatalln(err)
		}

		if err := s.Close(); err != nil {
			logger.Fatalln(err)
		}
	},
}

var web = &cobra.Command{
	Use:   "web [flags]",
	Short: "Open a web interface",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Implement
	},
}

var cmd = &cobra.Command{
	Use:   "rtc",
	Short: "Communicate with peers - directly, in \"real-time\".",
}

func init() {
	// TODO: Default to true if we are running from a script (no tty)?
	cmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "be quiet, do not output status and progress messages")

	for _, c := range []*cobra.Command{initialize, join} {
		// TODO: Specify signaling command (path, args…) in a more suitable way
		c.Flags().StringVarP(&sign, "sign", "s", sign, "signaling transport to use")
	}

	web.Flags().Uint16VarP(&port, "port", "p", port, "port to listen on")

	cmd.AddCommand(initialize)
	cmd.AddCommand(join)
	cmd.AddCommand(web)
}

func Run() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
