package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/obalunenko/Fallout-Terminal/internal/buildtool"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "build: resolve repository root: %v\n", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := buildtool.Run(ctx, root, os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "build: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: go run ./cmd/build <dev|build|package|run|prepare> [application arguments]")
}
