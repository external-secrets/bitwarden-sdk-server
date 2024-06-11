package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/external-secrets/bitwarden-sdk-server/pkg/server"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "serve",
		Short: "Serve the Bitwarden API",
		RunE:  runServeCmd,
	}

	rootArgs struct {
		server server.Config
	}
)

func init() {
	flag := rootCmd.Flags()
	// Server Configs
	flag.BoolVar(&rootArgs.server.Debug, "debug", false, "--debug")
	flag.BoolVar(&rootArgs.server.Insecure, "insecure", false, "--insecure")
	flag.StringVar(&rootArgs.server.KeyFile, "key-file", "", "--key-file /home/user/.server/server.key")
	flag.StringVar(&rootArgs.server.CertFile, "cert-file", "", "--cert-file /home/user/.server/server.crt")
	flag.StringVar(&rootArgs.server.Addr, "hostname", ":9998", "--hostname :9998")
}

const timeout = 15 * time.Second

func runServeCmd(_ *cobra.Command, _ []string) error {
	svr := server.NewServer(rootArgs.server)
	go func() {
		if err := svr.Run(context.Background()); err != nil {
			slog.Error("failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	interruptChannel := make(chan os.Signal, 2)
	signal.Notify(interruptChannel, os.Interrupt, syscall.SIGTERM)

	<-interruptChannel
	done := make(chan struct{})
	// start the timer for the shutdown sequence
	go func() {
		select {
		case <-done:
			return
		case <-time.After(timeout):
			slog.Error("graceful shutdown timed out... forcing shutdown")
			os.Exit(1)
		}
	}()

	slog.Info("received shutdown signal... gracefully terminating servers...")
	if err := svr.Shutdown(context.Background()); err != nil {
		slog.Error("graceful shutdown failed... forcing shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("all done. Goodbye.")

	done <- struct{}{}

	return nil
}

// Execute runs the main serve command.
func Execute() error {
	return rootCmd.Execute()
}
