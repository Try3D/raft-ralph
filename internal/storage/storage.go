package storage

import (
	"context"
	"fmt"
)

// ErrCorrupted indicates that the storage contains corrupted data
var ErrCorrupted = fmt.Errorf("storage corrupted")

// ErrIO indicates an I/O error occurred during storage operations
var ErrIO = fmt.Errorf("I/O error")

// Storage defines the interface for persistent storage operations in Raft
type Storage interface {
	// SaveVote persists the current term and the ID of the candidate voted for
	// This should be called before responding to RequestVote RPCs
	SaveVote(ctx context.Context, term, votedFor int) error
	
	// LoadVote retrieves the previously saved term and votedFor values
	// Returns ErrCorrupted if the data is corrupted
	// Returns ErrIO if an I/O error occurs
	LoadVote(ctx context.Context) (term, votedFor int, err error)
}