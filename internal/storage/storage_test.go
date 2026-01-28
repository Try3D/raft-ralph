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

func (ms *MockStorage) SaveHardState(ctx context.Context, term int, votedFor int) error {
	if ms.saveError != nil {
		return ms.saveError
	}
	ms.savedTerm = term
	ms.savedVotedFor = votedFor
	return nil
}

func (ms *MockStorage) LoadHardState(ctx context.Context) (int, int, error) {
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

// TestSaveHardState tests the SaveHardState method
func TestSaveHardState(t *testing.T) {
	ctx := context.Background()
	mock := &MockStorage{}
	
	err := mock.SaveHardState(ctx, 5, 2)
	if err != nil {
		t.Errorf("SaveHardState returned unexpected error: %v", err)
	}
	
	if mock.savedTerm != 5 {
		t.Errorf("Expected saved term to be 5, got %d", mock.savedTerm)
	}
	
	if mock.savedVotedFor != 2 {
		t.Errorf("Expected saved votedFor to be 2, got %d", mock.savedVotedFor)
	}
}

// TestLoadHardState tests the LoadHardState method
func TestLoadHardState(t *testing.T) {
	ctx := context.Background()
	mock := &MockStorage{
		savedTerm:    5,
		savedVotedFor: 2,
	}
	
	term, votedFor, err := mock.LoadHardState(ctx)
	if err != nil {
		t.Errorf("LoadHardState returned unexpected error: %v", err)
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
	
	// Test LoadHardState error
	mock := &MockStorage{loadError: &ErrCorrupted{Reason: "test corruption"}}
	_, _, err := mock.LoadHardState(ctx)
	if err == nil {
		t.Error("Expected error from LoadHardState, got nil")
	}
	
	var corruptedErr *ErrCorrupted
	if !errors.As(err, &corruptedErr) {
		t.Errorf("Expected ErrCorrupted, got %T", err)
	}
	
	// Test SaveHardState error
	mock = &MockStorage{saveError: &ErrIO{Reason: "test io error"}}
	err = mock.SaveHardState(ctx, 1, 1)
	if err == nil {
		t.Error("Expected error from SaveHardState, got nil")
	}
	
	var ioErr *ErrIO
	if !errors.As(err, &ioErr) {
		t.Errorf("Expected ErrIO, got %T", err)
	}
}