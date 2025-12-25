package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"webhookd/internal/buildinfo"
	"webhookd/internal/transport/runtime"
)

type RootOptions struct {
	Host    string
	Port    int
	Config  string
	Debug   bool
	Verbose bool
}

func NewRoot() *cobra.Command {
	opts := &RootOptions{
		Host:   "0.0.0.0",
		Port:   1337,
		Config: ".webhookdrc.json",
	}

	cmd := &cobra.Command{
		Use:     "webhookd",
		Short:   "Self-hosted webhook service",
		Long:    "webhookd is a small daemon that lets you generate and serve simple webhooks.",
		Version: buildinfo.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default behavior: serve
			return runServe(cmd.Context(), opts)
		},
	}

	cmd.SetVersionTemplate("{{.Version}}\n")

	cmd.PersistentFlags().StringVar(&opts.Host, "host", opts.Host, "Host to bind to")
	cmd.PersistentFlags().IntVar(&opts.Port, "port", opts.Port, "Port to listen on")
	cmd.PersistentFlags().StringVar(&opts.Config, "config", opts.Config, "Path to config file")
	cmd.PersistentFlags().BoolVar(&opts.Debug, "debug", false, "Enable debug output")
	cmd.PersistentFlags().BoolVar(&opts.Verbose, "verbose", false, "Enable verbose logging")

	cmd.AddCommand(newServeCmd(opts))

	return cmd
}

func newServeCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(cmd.Context(), opts)
		},
	}
}

func runServe(ctx context.Context, opts *RootOptions) error {
	addr := fmt.Sprintf("%s:%d", opts.Host, opts.Port)
	return runtime.Run(ctx, runtime.Options{
		Addr:       addr,
		ConfigPath: opts.Config,
		Debug:      opts.Debug,
		Verbose:    opts.Verbose,
		Version:    buildinfo.Version,
	})
}
