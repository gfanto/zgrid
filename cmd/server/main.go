package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
	"zgrid/api"
	"zgrid/business"
	"zgrid/foundation"
)

const (
	shutdownTimeout = 30 * time.Second
	bufferSize      = 4096
)

var (
	version = "--- set from makefile ---"

	help        = flag.Bool("help", false, "show help message")
	showVersion = flag.Bool("version", false, "show command version")
	addr        = flag.String("addr", ":8000", "HTTP network address")
)

func init() {
	flag.Parse()
}

func main() {
	if *help {
		flag.Usage()
		return
	}

	if *showVersion {
		fmt.Println(version)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := run(ctx, logger); err != nil {
		logger.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	wg := sync.WaitGroup{}

	// ----------------------------------------------------------------------------
	// Initialization

	// Buffer events so measurement updates can queue while a graph recomputation
	// is in progress, matching docs/golang_exercise.md.
	events := make(chan business.Event, bufferSize)
	defer close(events)

	grid := business.NewGrid()
	wg.Go(func() {
		grid.Loop(ctx, events)
	})

	// ----------------------------------------------------------------------------
	// Server Setup

	handler := foundation.WrapMiddleware(api.All(),
		foundation.WithRequestID,
		foundation.WithLogger(logger),
		foundation.Recover(logger),
		foundation.AccessLog(logger),
		api.GridEventsMiddleware(events),
	)

	server := &http.Server{
		Addr:    *addr,
		Handler: handler,
	}

	serverErrs := make(chan error, 1)
	wg.Go(func() {
		defer close(serverErrs)

		logger.Info("starting http server", "addr", *addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrs <- fmt.Errorf("server error: %w", err)
		}
	})

	// ----------------------------------------------------------------------------
	// Shutdown

	select {
	case err := <-serverErrs:
		return fmt.Errorf("received server error: %w", err)
	case <-ctx.Done():
		logger.Info("shutting down application")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			server.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	logger.Info("waiting for background tasks to complete")
	wg.Wait()
	return nil
}
