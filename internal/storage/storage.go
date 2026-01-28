package storage

import (
	"context"
	"fmt"
)

var ErrCorrupted = fmt.Errorf("storage corrupted")

var ErrIO = fmt.Errorf("I/O error")

type Storage interface {
	SaveVote(ctx context.Context, term, votedFor int) error
	LoadVote(ctx context.Context) (term, votedFor int, err error)
}