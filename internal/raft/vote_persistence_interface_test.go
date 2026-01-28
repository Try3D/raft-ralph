package raft

import (
	"context"
	"testing"

	"github.com/try3d/raft-ralph/internal/storage"
)

func TestStorageInterfaceDefinition(t *testing.T) {
	var _ storage.Storage = &MockStorage{}

	mockStorage := &MockStorage{}
	ctx := context.Background()

	err := mockStorage.SaveVote(ctx, 1, 2)
	if err != nil {
		t.Errorf("SaveVote returned unexpected error: %v", err)
	}

	term, votedFor, err := mockStorage.LoadVote(ctx)
	if err != nil {
		t.Errorf("LoadVote returned unexpected error: %v", err)
	}

	if term != 0 {
		t.Errorf("Expected term 0, got %d", term)
	}

	if votedFor != -1 {
		t.Errorf("Expected votedFor -1, got %d", votedFor)
	}
}

func TestStorageErrors(t *testing.T) {
	_ = storage.ErrCorrupted
	_ = storage.ErrIO
}