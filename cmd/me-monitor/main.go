package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

	"me-monitor/internal/config"
	"me-monitor/internal/monitor"
	"me-monitor/internal/ui"
)

const pollInterval = 2 * time.Second

func main() {
	// Load configuration
	configPath := os.Getenv("CONSUMERS_CONFIG")
	if configPath == "" {
		configPath = "consumers.json"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatal(err)
	}

	// Connect to NATS
	natsURL, natsOpts, err := config.LoadNATSFromContext()
	if err != nil {
		log.Fatal(err)
	}

	nc, err := nats.Connect(natsURL, natsOpts...)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		log.Fatal(err)
	}

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Create update channel
	updates := make(chan []monitor.ConsumerState)

	// Start poller
	poller := monitor.NewPoller(js, cfg.Consumers, pollInterval)
	go poller.Run(ctx, updates)

	// Run UI
	app := ui.NewApp(len(cfg.Consumers))
	if err := app.Run(ctx, updates); err != nil {
		log.Fatal(err)
	}
}
