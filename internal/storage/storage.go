package storage

import "context"

// ErrCorrupted indicates that the stored data is corrupted
type ErrCorrupted struct {
	Reason string
}

func (e *ErrCorrupted) Error() string {
	return "storage corrupted: " + e.Reason
}

// ErrIO indicates an I/O error during storage operations
type ErrIO struct {
	Reason string
}

func (e *ErrIO) Error() string {
	return "storage I/O error: " + e.Reason
}

// Storage defines the interface for storing and retrieving Raft persistent state
type Storage interface {
	// SaveHardState saves the current hard state (term and vote) to storage
	SaveHardState(ctx context.Context, term int, votedFor int) error
	
	// LoadHardState loads the previously saved hard state from storage
	LoadHardState(ctx context.Context) (term int, votedFor int, err error)
}