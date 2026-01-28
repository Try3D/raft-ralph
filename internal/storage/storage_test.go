package storage

import (
	"context"
	"errors"
	"testing"
)

// MockStorage implements the Storage interface for testing purposes
type MockStorage struct {
	savedTerm    int
	savedVotedFor int
	loadError    error
	saveError    error
}

func (ms *MockStorage) SaveVote(ctx context.Context, term, votedFor int) error {
	if ms.saveError != nil {
		return ms.saveError
	}
	ms.savedTerm = term
	ms.savedVotedFor = votedFor
	return nil
}

func (ms *MockStorage) LoadVote(ctx context.Context) (int, int, error) {
	if ms.loadError != nil {
		return 0, 0, ms.loadError
	}
	return ms.savedTerm, ms.savedVotedFor, nil
}

// TestStorageInterfaceImplementation tests that the Storage interface is properly defined
func TestStorageInterfaceImplementation(t *testing.T) {
	// This test verifies that the Storage interface has the required methods
	var s Storage

	// Verify that the interface can be assigned
	if s == nil {
		// This is just to use the variable to avoid "declared but not used" error
		t.Log("Storage interface is defined")
	}

	// Create a mock implementation to verify the interface
	mock := &MockStorage{}

	// Verify that MockStorage implements Storage interface
	var _ Storage = mock
}

// TestSaveVote tests the SaveVote method
func TestSaveVote(t *testing.T) {
	ctx := context.Background()
	mock := &MockStorage{}

	err := mock.SaveVote(ctx, 5, 2)
	if err != nil {
		t.Errorf("SaveVote returned unexpected error: %v", err)
	}

	if mock.savedTerm != 5 {
		t.Errorf("Expected saved term to be 5, got %d", mock.savedTerm)
	}

	if mock.savedVotedFor != 2 {
		t.Errorf("Expected saved votedFor to be 2, got %d", mock.savedVotedFor)
	}
}

// TestLoadVote tests the LoadVote method
func TestLoadVote(t *testing.T) {
	ctx := context.Background()
	mock := &MockStorage{
		savedTerm:    5,
		savedVotedFor: 2,
	}

	term, votedFor, err := mock.LoadVote(ctx)
	if err != nil {
		t.Errorf("LoadVote returned unexpected error: %v", err)
	}

	if term != 5 {
		t.Errorf("Expected loaded term to be 5, got %d", term)
	}

	if votedFor != 2 {
		t.Errorf("Expected loaded votedFor to be 2, got %d", votedFor)
	}
}

// TestStorageErrors tests error handling
func TestStorageErrors(t *testing.T) {
	ctx := context.Background()

	// Test LoadVote error
	mock := &MockStorage{loadError: ErrCorrupted}
	_, _, err := mock.LoadVote(ctx)
	if err == nil {
		t.Error("Expected error from LoadVote, got nil")
	}

	if !errors.Is(err, ErrCorrupted) {
		t.Errorf("Expected ErrCorrupted, got %v", err)
	}

	// Test SaveVote error
	mock = &MockStorage{saveError: ErrIO}
	err = mock.SaveVote(ctx, 1, 1)
	if err == nil {
		t.Error("Expected error from SaveVote, got nil")
	}

	if !errors.Is(err, ErrIO) {
		t.Errorf("Expected ErrIO, got %v", err)
	}
}