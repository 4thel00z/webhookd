package runtime

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"webhookd/internal/application/webhooks"
	"webhookd/internal/infrastructure/configfile"
	"webhookd/internal/infrastructure/repository/memory"
	"webhookd/internal/transport/httpapi"
)

type Options struct {
	Addr       string
	ConfigPath string
	Debug      bool
	Verbose    bool
	Version    string
}

func Run(ctx context.Context, opts Options) error {
	_ = godotenv.Load()

	cfgPath := opts.ConfigPath
	cfg, err := configfile.ParseFile(cfgPath)
	if err != nil {
		switch {
		case os.IsNotExist(err) && cfgPath == "webhookd.json":
			legacyPath := ".webhookdrc.json"
			if _, statErr := os.Stat(legacyPath); statErr == nil {
				log.Printf("config file %q not found; using legacy %q (deprecated)", cfgPath, legacyPath)
				cfg, err = configfile.ParseFile(legacyPath)
				if err != nil {
					return fmt.Errorf("parse config: %w", err)
				}
			} else if os.IsNotExist(statErr) {
				log.Printf("config file %q not found; starting with defaults", cfgPath)
				cfg = configfile.Config{}
			} else {
				return fmt.Errorf("stat legacy config %q: %w", legacyPath, statErr)
			}
		case os.IsNotExist(err) && cfgPath == ".webhookdrc.json":
			log.Printf("config file %q not found; starting with defaults", cfgPath)
			cfg = configfile.Config{}
		default:
			return fmt.Errorf("parse config: %w", err)
		}
	}

	repo := memory.NewWebhooksRepo()
	svc := webhooks.NewService(repo)

	app, err := httpapi.NewApp(httpapi.Deps{
		Version:  opts.Version,
		Config:   cfg,
		Webhooks: svc,
	})
	if err != nil {
		return err
	}

	// Fiber wants an addr string, but we validate it to fail early with a good error.
	if _, _, err := net.SplitHostPort(opts.Addr); err != nil {
		return fmt.Errorf("invalid addr %q: %w", opts.Addr, err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Listen(opts.Addr)
	}()

	log.Printf("listening on %s (version=%s)", opts.Addr, opts.Version)

	// Shutdown on signals or parent context cancellation.
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigs)

	select {
	case <-ctx.Done():
	case <-sigs:
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return app.ShutdownWithContext(shutdownCtx)
}
