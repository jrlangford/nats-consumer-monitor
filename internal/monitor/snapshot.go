package monitor

import (
	"github.com/nats-io/nats.go"
)

// Snapshot captures the state of a consumer at a point in time.
// Only includes fields that indicate actual state changes worth highlighting.
type Snapshot struct {
	DeliveredConsumer uint64
	AckConsumer       uint64
	AckStream         uint64
	NumAckPending     int
	NumRedelivered    int
	NumPending        uint64
	NumWaiting        int
}

// FromConsumerInfo creates a Snapshot from NATS consumer info.
func FromConsumerInfo(ci *nats.ConsumerInfo) Snapshot {
	return Snapshot{
		DeliveredConsumer: ci.Delivered.Consumer,
		AckConsumer:       ci.AckFloor.Consumer,
		AckStream:         ci.AckFloor.Stream,
		NumAckPending:     int(ci.NumAckPending),
		NumRedelivered:    int(ci.NumRedelivered),
		NumPending:        ci.NumPending,
		NumWaiting:        ci.NumWaiting,
	}
}

// Equal returns true if two snapshots represent the same state.
func (s Snapshot) Equal(other Snapshot) bool {
	return s.DeliveredConsumer == other.DeliveredConsumer &&
		s.AckConsumer == other.AckConsumer &&
		s.AckStream == other.AckStream &&
		s.NumAckPending == other.NumAckPending &&
		s.NumRedelivered == other.NumRedelivered &&
		s.NumPending == other.NumPending &&
		s.NumWaiting == other.NumWaiting
}

// IsZero returns true if this is a zero-value snapshot.
func (s Snapshot) IsZero() bool {
	return s == Snapshot{}
}
