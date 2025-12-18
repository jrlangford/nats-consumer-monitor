package monitor

import (
	"sync"
	"time"
)

// ThroughputMeasurement tracks message throughput for a consumer.
type ThroughputMeasurement struct {
	StartTime        time.Time
	EndTime          time.Time // Zero if still measuring
	StartDelivered   uint64
	StartAcked       uint64
	CurrentDelivered uint64
	CurrentAcked     uint64
}

// DeliveredCount returns the number of messages delivered during measurement.
func (t ThroughputMeasurement) DeliveredCount() uint64 {
	return t.CurrentDelivered - t.StartDelivered
}

// AckedCount returns the number of messages acknowledged during measurement.
func (t ThroughputMeasurement) AckedCount() uint64 {
	return t.CurrentAcked - t.StartAcked
}

// Duration returns how long the measurement ran (or is running).
func (t ThroughputMeasurement) Duration() time.Duration {
	if t.EndTime.IsZero() {
		return time.Since(t.StartTime)
	}
	return t.EndTime.Sub(t.StartTime)
}

// DeliveredRate returns messages delivered per second.
func (t ThroughputMeasurement) DeliveredRate() float64 {
	dur := t.Duration().Seconds()
	if dur == 0 {
		return 0
	}
	return float64(t.DeliveredCount()) / dur
}

// AckedRate returns messages acknowledged per second.
func (t ThroughputMeasurement) AckedRate() float64 {
	dur := t.Duration().Seconds()
	if dur == 0 {
		return 0
	}
	return float64(t.AckedCount()) / dur
}

// ThroughputTracker manages throughput measurements for all consumers.
type ThroughputTracker struct {
	mu           sync.RWMutex
	measuring    bool
	measurements map[string]*ThroughputMeasurement // keyed by "stream/consumer"
}

// NewThroughputTracker creates a new throughput tracker.
func NewThroughputTracker() *ThroughputTracker {
	return &ThroughputTracker{
		measurements: make(map[string]*ThroughputMeasurement),
	}
}

// IsMeasuring returns true if measurement is active.
func (t *ThroughputTracker) IsMeasuring() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.measuring
}

// Toggle starts or stops measurement. Returns true if now measuring.
func (t *ThroughputTracker) Toggle(states []ConsumerState) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.measuring {
		// Stop measuring - set end time on all measurements
		t.measuring = false
		endTime := time.Now()
		for _, m := range t.measurements {
			m.EndTime = endTime
		}
		return false
	}

	// Start measuring
	t.measuring = true
	t.measurements = make(map[string]*ThroughputMeasurement)
	now := time.Now()

	for _, state := range states {
		if state.Error != nil {
			continue
		}
		key := state.Ref.Stream + "/" + state.Ref.Consumer
		t.measurements[key] = &ThroughputMeasurement{
			StartTime:        now,
			StartDelivered:   state.Snapshot.DeliveredConsumer,
			StartAcked:       state.Snapshot.AckConsumer,
			CurrentDelivered: state.Snapshot.DeliveredConsumer,
			CurrentAcked:     state.Snapshot.AckConsumer,
		}
	}

	return true
}

// Update updates current values for all tracked consumers.
func (t *ThroughputTracker) Update(states []ConsumerState) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.measuring {
		return
	}

	for _, state := range states {
		if state.Error != nil {
			continue
		}
		key := state.Ref.Stream + "/" + state.Ref.Consumer
		if m, ok := t.measurements[key]; ok {
			m.CurrentDelivered = state.Snapshot.DeliveredConsumer
			m.CurrentAcked = state.Snapshot.AckConsumer
		}
	}
}

// Get returns the measurement for a consumer, if any.
func (t *ThroughputTracker) Get(stream, consumer string) *ThroughputMeasurement {
	t.mu.RLock()
	defer t.mu.RUnlock()

	key := stream + "/" + consumer
	if m, ok := t.measurements[key]; ok {
		// Return a copy to avoid race conditions
		copy := *m
		return &copy
	}
	return nil
}

// Clear removes all measurements.
func (t *ThroughputTracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.measuring = false
	t.measurements = make(map[string]*ThroughputMeasurement)
}
