package domain

import (
	"context"
	"io"
)

// Differ provides an interface for generating a Diff of two network graphs.
// The network graphs will be retrieved based on the time ranges specified by
// the provided Diff type.
type Differ interface {
	Diff(ctx context.Context, d Diff) (io.ReadCloser, error)
}
