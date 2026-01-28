package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestVotePersistsAcrossRestart tests that votes persist across restarts
func TestVotePersistsAcrossRestart(t *testing.T) {
	ctx := context.Background()

	// Create a temporary file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "vote_storage.json")

	// Create storage instance
	storage := NewFileStorage(testFile)

	// Save a vote
	err := storage.SaveVote(ctx, 5, 2)
	if err != nil {
		t.Fatalf("Failed to save vote: %v", err)
	}

	// Create a new storage instance to simulate restart
	newStorage := NewFileStorage(testFile)

	// Load the vote
	term, votedFor, err := newStorage.LoadVote(ctx)
	if err != nil {
		t.Fatalf("Failed to load vote: %v", err)
	}

	if term != 5 {
		t.Errorf("Expected term 5, got %d", term)
	}

	if votedFor != 2 {
		t.Errorf("Expected votedFor 2, got %d", votedFor)
	}
}

// TestCorruptionDetection tests that corruption is detected
func TestCorruptionDetection(t *testing.T) {
	ctx := context.Background()

	// Create a temporary file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "vote_storage.json")

	// Create storage instance and save a vote
	storage := NewFileStorage(testFile)
	err := storage.SaveVote(ctx, 5, 2)
	if err != nil {
		t.Fatalf("Failed to save vote: %v", err)
	}

	// Manually corrupt the file
	corruptedContent := `{"term":5,"voted_for":2,"checksum":12345}`
	err = os.WriteFile(testFile, []byte(corruptedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write corrupted content: %v", err)
	}

	// Try to load the vote - should return ErrCorrupted
	newStorage := NewFileStorage(testFile)
	_, _, err = newStorage.LoadVote(ctx)
	if err == nil {
		t.Fatal("Expected error when loading corrupted file, got nil")
	}

	if !errors.Is(err, ErrCorrupted) {
		t.Errorf("Expected ErrCorrupted, got %v", err)
	}
}

// TestAtomicWrites tests that concurrent writes don't corrupt state
func TestAtomicWrites(t *testing.T) {
	ctx := context.Background()

	// Create a temporary file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "vote_storage.json")

	storage := NewFileStorage(testFile)

	const numGoroutines = 10
	var wg sync.WaitGroup

	// Launch multiple goroutines to save votes concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			err := storage.SaveVote(ctx, goroutineID, goroutineID*2)
			if err != nil {
				t.Errorf("Goroutine %d failed to save vote: %v", goroutineID, err)
			}
		}(i)
	}

	wg.Wait()

	// Load the final value to ensure it's valid
	term, votedFor, err := storage.LoadVote(ctx)
	if err != nil {
		t.Fatalf("Failed to load final vote: %v", err)
	}

	// Since goroutines run concurrently, we can't predict which value will be final,
	// but it should be one of the values we tried to save
	if term < 0 || term >= numGoroutines {
		t.Errorf("Loaded invalid term: %d", term)
	}

	if votedFor != term*2 {
		t.Errorf("Loaded inconsistent votedFor: %d, expected: %d", votedFor, term*2)
	}
}

// TestSaveVote tests the SaveVote method
func TestSaveVote(t *testing.T) {
	ctx := context.Background()

	// Create a temporary file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "vote_storage.json")

	storage := NewFileStorage(testFile)

	err := storage.SaveVote(ctx, 5, 2)
	if err != nil {
		t.Errorf("SaveVote returned unexpected error: %v", err)
	}

	// Load and verify
	term, votedFor, err := storage.LoadVote(ctx)
	if err != nil {
		t.Fatalf("LoadVote failed: %v", err)
	}

	if term != 5 {
		t.Errorf("Expected saved term to be 5, got %d", term)
	}

	if votedFor != 2 {
		t.Errorf("Expected saved votedFor to be 2, got %d", votedFor)
	}
}

// TestLoadVote tests the LoadVote method with default values
func TestLoadVoteDefaultValues(t *testing.T) {
	ctx := context.Background()

	// Create a temporary file for testing (doesn't exist yet)
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "nonexistent_vote_storage.json")

	storage := NewFileStorage(testFile)

	// Load from non-existent file should return defaults
	term, votedFor, err := storage.LoadVote(ctx)
	if err != nil {
		t.Errorf("LoadVote returned unexpected error for non-existent file: %v", err)
	}

	if term != 0 {
		t.Errorf("Expected default term to be 0, got %d", term)
	}

	if votedFor != -1 {
		t.Errorf("Expected default votedFor to be -1, got %d", votedFor)
	}
}

// TestStorageErrors tests error handling
func TestStorageErrors(t *testing.T) {
	ctx := context.Background()

	// Test with invalid path
	invalidStorage := NewFileStorage("/invalid/path/file.json")

	err := invalidStorage.SaveVote(ctx, 1, 1)
	if err == nil {
		t.Error("Expected error when saving to invalid path, got nil")
	}
}

// TestConcurrentAccess tests concurrent access to the storage
func TestConcurrentAccess(t *testing.T) {
	ctx := context.Background()

	// Create a temporary file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "vote_storage.json")

	storage := NewFileStorage(testFile)

	const numOperations = 20
	var wg sync.WaitGroup

	// Launch goroutines for concurrent saves and loads
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(opID int) {
			defer wg.Done()

			// Save a value
			err := storage.SaveVote(ctx, opID, opID*3)
			if err != nil {
				t.Errorf("Operation %d failed to save: %v", opID, err)
				return
			}

			// Load the value back
			term, votedFor, err := storage.LoadVote(ctx)
			if err != nil {
				t.Errorf("Operation %d failed to load: %v", opID, err)
				return
			}

			// The value might not match exactly due to concurrent operations,
			// but it should be a valid value we've written
			if term < 0 || term >= numOperations {
				t.Errorf("Operation %d loaded invalid term: %d", opID, term)
			}

			if votedFor != term*3 {
				t.Errorf("Operation %d loaded inconsistent votedFor: %d, expected: %d", opID, votedFor, term*3)
			}
		}(i)
	}

	wg.Wait()
}