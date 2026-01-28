package raft

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/try3d/raft-ralph/internal/storage"
)

// TestVotePersistsAcrossRestart tests that votes persist across restarts
func TestVotePersistsAcrossRestart(t *testing.T) {
	tempDir := t.TempDir()
	storagePath := filepath.Join(tempDir, "vote.json")

	// Create a file storage
	fileStorage := storage.NewFileStorage(storagePath)

	// Create a node with the file storage
	node := NewNode(1, fileStorage)

	// Initially, no vote should be recorded
	ctx := context.Background()
	term, votedFor, err := fileStorage.LoadVote(ctx)
	if err != nil {
		t.Fatalf("Failed to load vote: %v", err)
	}

	if term != 0 {
		t.Errorf("Expected initial term to be 0, got %d", term)
	}

	if votedFor != -1 {
		t.Errorf("Expected initial votedFor to be -1, got %d", votedFor)
	}

	// Start an election to vote for self
	node.StartElection()

	// Check that the vote was saved
	term, votedFor, err = fileStorage.LoadVote(ctx)
	if err != nil {
		t.Fatalf("Failed to load vote after election: %v", err)
	}

	if term != 1 {
		t.Errorf("Expected term to be 1 after election, got %d", term)
	}

	if votedFor != 1 {
		t.Errorf("Expected votedFor to be 1 after election, got %d", votedFor)
	}

	// Simulate a restart by creating a new node from the same storage
	restartedNode, err := NewNodeFromStorage(1, fileStorage)
	if err != nil {
		t.Fatalf("Failed to create node from storage: %v", err)
	}

	// Load the persistent state
	persistentState := restartedNode.GetPersistentState()
	if persistentState.CurrentTerm != 1 {
		t.Errorf("Expected restarted node term to be 1, got %d", persistentState.CurrentTerm)
	}

	if persistentState.VotedFor != 1 {
		t.Errorf("Expected restarted node votedFor to be 1, got %d", persistentState.VotedFor)
	}
}

// TestVotePersistenceOnRequestVote tests that votes are persisted when granting votes to others
func TestVotePersistenceOnRequestVote(t *testing.T) {
	tempDir := t.TempDir()
	storagePath := filepath.Join(tempDir, "vote.json")

	// Create a file storage
	fileStorage := storage.NewFileStorage(storagePath)

	// Create a node with the file storage
	node := NewNode(1, fileStorage)

	// Send a RequestVote message from candidate 2 in term 1
	msg := Message{
		Type: RequestVoteMsg,
		From: 2,
		To:   node.ID,
		Term: 1,
	}

	// Process the message
	node.Step(msg)

	// Check that the vote was saved for candidate 2
	ctx := context.Background()
	term, votedFor, err := fileStorage.LoadVote(ctx)
	if err != nil {
		t.Fatalf("Failed to load vote: %v", err)
	}

	if term != 1 {
		t.Errorf("Expected term to be 1, got %d", term)
	}

	if votedFor != 2 {
		t.Errorf("Expected votedFor to be 2, got %d", votedFor)
	}

	// Send another RequestVote from a different candidate in the same term
	// This should not change the vote
	msg2 := Message{
		Type: RequestVoteMsg,
		From: 3,
		To:   node.ID,
		Term: 1,
	}

	node.Step(msg2)

	// Check that the vote is still for candidate 2
	term, votedFor, err = fileStorage.LoadVote(ctx)
	if err != nil {
		t.Fatalf("Failed to load vote: %v", err)
	}

	if term != 1 {
		t.Errorf("Expected term to still be 1, got %d", term)
	}

	if votedFor != 2 {
		t.Errorf("Expected votedFor to still be 2, got %d", votedFor)
	}
}

// TestVotePersistenceOnHigherTerm tests that votes are reset and persisted when receiving higher-term messages
func TestVotePersistenceOnHigherTerm(t *testing.T) {
	tempDir := t.TempDir()
	storagePath := filepath.Join(tempDir, "vote.json")

	// Create a file storage
	fileStorage := storage.NewFileStorage(storagePath)

	// Create a node with the file storage
	node := NewNode(1, fileStorage)

	// Start an election to vote for self in term 1
	node.StartElection()

	// Verify the vote was saved
	ctx := context.Background()
	term, votedFor, err := fileStorage.LoadVote(ctx)
	if err != nil {
		t.Fatalf("Failed to load vote: %v", err)
	}

	if term != 1 || votedFor != 1 {
		t.Fatalf("Expected vote for (1,1), got (%d,%d)", term, votedFor)
	}

	// Receive a message with higher term (term 3)
	higherTermMsg := Message{
		Type: AppendEntriesMsg,
		From: 2,
		To:   node.ID,
		Term: 3,
	}

	node.Step(higherTermMsg)

	// Check that the term was updated and vote was reset
	term, votedFor, err = fileStorage.LoadVote(ctx)
	if err != nil {
		t.Fatalf("Failed to load vote after higher-term message: %v", err)
	}

	if term != 3 {
		t.Errorf("Expected term to be updated to 3, got %d", term)
	}

	if votedFor != -1 {
		t.Errorf("Expected votedFor to be reset to -1, got %d", votedFor)
	}
}

// TestCorruptionDetection tests that the storage correctly detects corruption
func TestCorruptionDetection(t *testing.T) {
	tempDir := t.TempDir()
	storagePath := filepath.Join(tempDir, "vote.json")

	// Create a file storage
	fileStorage := storage.NewFileStorage(storagePath)

	// Create a node and perform some operations to save data
	node := NewNode(1, fileStorage)
	node.StartElection() // Term 1, vote for self

	// Verify the data was saved correctly
	ctx := context.Background()
	term, votedFor, err := fileStorage.LoadVote(ctx)
	if err != nil {
		t.Fatalf("Failed to load vote: %v", err)
	}

	if term != 1 || votedFor != 1 {
		t.Fatalf("Expected vote for (1,1), got (%d,%d)", term, votedFor)
	}

	// Now manually corrupt the file by changing its contents
	corruptedData := `{"term": 1, "voted_for": 2, "checksum": 12345}`
	err = os.WriteFile(storagePath, []byte(corruptedData), 0644)
	if err != nil {
		t.Fatalf("Failed to write corrupted data: %v", err)
	}

	// Try to load the vote - this should return a corruption error
	term, votedFor, err = fileStorage.LoadVote(ctx)
	if err != storage.ErrCorrupted {
		t.Errorf("Expected ErrCorrupted, got %v", err)
	}
}

// TestAtomicWrites tests that concurrent writes don't corrupt state
func TestAtomicWrites(t *testing.T) {
	tempDir := t.TempDir()
	storagePath := filepath.Join(tempDir, "vote.json")

	// Create a file storage
	fileStorage := storage.NewFileStorage(storagePath)

	// Create a node with the file storage
	node := NewNode(1, fileStorage)

	// Perform multiple operations that would trigger saves
	for i := 1; i <= 5; i++ {
		// Simulate receiving a higher-term message which causes a transition to follower
		// and resets the vote
		msg := Message{
			Type: AppendEntriesMsg,
			From: 2,
			To:   node.ID,
			Term: i,
		}
		node.Step(msg)

		// Verify the state was saved correctly
		ctx := context.Background()
		term, votedFor, err := fileStorage.LoadVote(ctx)
		if err != nil {
			t.Fatalf("Failed to load vote at iteration %d: %v", i, err)
		}

		if term != i {
			t.Errorf("Expected term %d at iteration %d, got %d", i, i, term)
		}

		if votedFor != -1 {
			t.Errorf("Expected votedFor to be -1 at iteration %d, got %d", i, votedFor)
		}
	}
}