package domain

import (
	"context"
	"fmt"
	"io"
)

// ErrNotFound represents a resource lookup that failed due to a missing record.
type ErrNotFound struct {
	ID string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("digest %s was not found", e.ID)
}

// Storage is an interface for accessing created diffs. It is the caller's responsibility to call Close on the Reader when done.
type Storage interface {
	// Get returns the diff for the given key.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Exists returns true if the diff exists, but does not download the digest body.
	Exists(ctx context.Context, key string) (bool, error)

	// Store stores the diff
	Store(ctx context.Context, key string, data io.ReadCloser) error
}
