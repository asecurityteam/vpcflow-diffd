package domain

import (
	"context"
	"io"
	"time"
)

// Grapher provides an interface for fetching a generated network graph in a given time range
type Grapher interface {
	Graph(ctx context.Context, start, stop time.Time) (io.ReadCloser, error)
}
