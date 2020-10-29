package main

import (
	"context"
	"flag"
	"github.com/joho/godotenv"
	"github.com/logrusorgru/aurora"
	"github.com/monzo/typhon"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"webhookd/pkg/libwebhook"
	"webhookd/pkg/libwebhook/filters"
	"webhookd/pkg/libwebhook/modules/debug"
	"webhookd/pkg/libwebhook/modules/webhook"
)

const (
	banner = `
              ___.   .__                   __       .___
__  _  __ ____\_ |__ |  |__   ____   ____ |  | __ __| _/
\ \/ \/ // __ \| __ \|  |  \ /  _ \ /  _ \|  |/ // __ |
 \     /\  ___/| \_\ \   Y  (  <_> |  <_> )    </ /_/ |
  \/\_/  \___  >___  /___|  /\____/ \____/|__|_ \____ |
             \/    \/     \/                   \/    \/`
	version = "0.0.1"
)

var (
	port       = flag.Int("port", 1337, "<port> [defaults to 1337]")
	host       = flag.String("host", "0.0.0.0", "<host> [defaults to 0.0.0.0]")
	configPath = flag.String("config", ".webhookdrc.json", "path to config. [defaults to .webhookdrc.json]")
	debugFlag  = flag.Bool("debug", false, "enable replacement of <auth> with real token [defaults to false]")
	verbose    = flag.Bool("verbose", false, "enable verbose logging [defaults to false]")
)

func main() {
	flag.Parse()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file. Create an empty .env file if you don't intend to use them")
	}

	log.Println("\n", aurora.Magenta(banner), "\n")
	log.Println("👩	Version:", version)

	config, err := libwebhook.ParseConfig(*configPath)
	if err != nil {
		log.Fatalf("could not parse the configuration because of: %s", err.Error())
	}
	addr := *host + ":" + strconv.Itoa(*port)
	app := libwebhook.NewApp(addr, config, *verbose, *debugFlag, debug.Module,webhook.Module)

	svc := app.Router.Serve().
		Filter(typhon.ErrorFilter).
		Filter(typhon.H2cFilter).
		Filter(filters.Validation(app)).
		Filter(filters.Auth(app)).
		Filter(webhook.Validation(app))

	srv, err := typhon.Listen(svc, addr)
	if err != nil {
		panic(err)
	}

	log.Printf("🏁	Listening on %v", srv.Listener().Addr())
	if app.Debug {
		app.PrintConfig()
	}
	app.PrintRoutes(srv.Listener().Addr().String())
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
	log.Printf("☠️  Shutting down in max 10 sec..")
	c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Stop(c)
}
