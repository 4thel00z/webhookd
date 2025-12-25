package main

import (
	"context"
	"log"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/logrusorgru/aurora"

	"webhookd/internal/transport/cli"
)

const (
	banner = `
              ___.   .__                   __       .___
__  _  __ ____\_ |__ |  |__   ____   ____ |  | __ __| _/
\ \/ \/ // __ \| __ \|  |  \ /  _ \ /  _ \|  |/ // __ |
 \     /\  ___/| \_\ \   Y  (  <_> |  <_> )    </ /_/ |
  \/\_/  \___  >___  /___|  /\____/ \____/|__|_ \____ |
             \/    \/     \/                   \/    \/`
)

func main() {
	log.Println("\n", aurora.Magenta(banner))
	root := cli.NewRoot()
	if err := fang.Execute(context.Background(), root); err != nil {
		os.Exit(1)
	}
}
