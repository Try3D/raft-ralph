package raft

import (
	"context"
	"testing"

	"github.com/try3d/raft-ralph/internal/storage"
)

func TestStorageInterfaceDefinition(t *testing.T) {
	// Verify that MockStorage implements the storage.Storage interface
	var _ storage.Storage = &MockStorage{}
	
	// Test that we can call the required methods
	mockStorage := &MockStorage{}
	ctx := context.Background()
	
	// Test SaveVote method
	err := mockStorage.SaveVote(ctx, 1, 2)
	if err != nil {
		t.Errorf("SaveVote returned unexpected error: %v", err)
	}
	
	// Test LoadVote method
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
	// Verify that the expected error types exist in the storage package
	_ = storage.ErrCorrupted
	_ = storage.ErrIO
}