package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"me-monitor/internal/config"
)

// ConsumerState represents the current state of a monitored consumer.
type ConsumerState struct {
	Ref      config.ConsumerRef
	Info     *nats.ConsumerInfo
	Snapshot Snapshot
	Changed  bool // True if state changed from previous poll
	Error    error
}

// Poller periodically fetches consumer info from NATS JetStream.
type Poller struct {
	js        nats.JetStreamContext
	consumers []config.ConsumerRef
	interval  time.Duration

	mu        sync.RWMutex
	snapshots map[string]Snapshot // keyed by "stream/consumer"
}

// NewPoller creates a new consumer poller.
func NewPoller(js nats.JetStreamContext, consumers []config.ConsumerRef, interval time.Duration) *Poller {
	return &Poller{
		js:        js,
		consumers: consumers,
		interval:  interval,
		snapshots: make(map[string]Snapshot),
	}
}

// Run starts the polling loop and sends state updates to the channel.
// It blocks until the context is cancelled.
func (p *Poller) Run(ctx context.Context, updates chan<- []ConsumerState) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	// Initial poll
	p.poll(updates)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.poll(updates)
		}
	}
}

func (p *Poller) poll(updates chan<- []ConsumerState) {
	states := make([]ConsumerState, len(p.consumers))

	for i, c := range p.consumers {
		key := c.Stream + "/" + c.Consumer
		state := ConsumerState{Ref: c}

		ci, err := p.js.ConsumerInfo(c.Stream, c.Consumer)
		if err != nil {
			state.Error = err
			states[i] = state
			continue
		}

		state.Info = ci
		state.Snapshot = FromConsumerInfo(ci)

		p.mu.RLock()
		prev, hasPrev := p.snapshots[key]
		p.mu.RUnlock()

		// Only mark as changed if we have a previous snapshot AND it differs
		state.Changed = hasPrev && !state.Snapshot.Equal(prev)

		p.mu.Lock()
		p.snapshots[key] = state.Snapshot
		p.mu.Unlock()

		states[i] = state
	}

	updates <- states
}
