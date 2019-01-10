package domain

import (
	"context"
	"time"
)

// Diff represents two time for which a network graph diff will be computed
type Diff struct {
	ID            string
	PreviousStart time.Time
	PreviousStop  time.Time
	NextStart     time.Time
	NextStop      time.Time
}

// Queuer provides an interface for queuing diff jobs onto a streaming appliance
type Queuer interface {
	Queue(ctx context.Context, d Diff) error
}
