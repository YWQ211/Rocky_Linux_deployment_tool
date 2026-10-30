package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"rockdev/internal/cli"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	app := &cli.App{}
	if err := app.Run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, app.Text("错误:"), err)
		os.Exit(1)
	}
}
