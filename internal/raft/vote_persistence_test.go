package raft

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/try3d/raft-ralph/internal/storage"
)

func TestVotePersistsAcrossRestart(t *testing.T) {
	tempDir := t.TempDir()
	storagePath := filepath.Join(tempDir, "vote.json")

	fileStorage := storage.NewFileStorage(storagePath)

	node := NewNode(1, fileStorage)

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

	node.StartElection()

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

	restartedNode, err := NewNodeFromStorage(1, fileStorage)
	if err != nil {
		t.Fatalf("Failed to create node from storage: %v", err)
	}

	persistentState := restartedNode.GetPersistentState()
	if persistentState.CurrentTerm != 1 {
		t.Errorf("Expected restarted node term to be 1, got %d", persistentState.CurrentTerm)
	}

	if persistentState.VotedFor != 1 {
		t.Errorf("Expected restarted node votedFor to be 1, got %d", persistentState.VotedFor)
	}
}

func TestVotePersistenceOnRequestVote(t *testing.T) {
	tempDir := t.TempDir()
	storagePath := filepath.Join(tempDir, "vote.json")

	fileStorage := storage.NewFileStorage(storagePath)

	node := NewNode(1, fileStorage)

	msg := Message{
		Type: RequestVoteMsg,
		From: 2,
		To:   node.ID,
		Term: 1,
	}

	node.Step(msg)

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

	msg2 := Message{
		Type: RequestVoteMsg,
		From: 3,
		To:   node.ID,
		Term: 1,
	}

	node.Step(msg2)

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

func TestVotePersistenceOnHigherTerm(t *testing.T) {
	tempDir := t.TempDir()
	storagePath := filepath.Join(tempDir, "vote.json")

	fileStorage := storage.NewFileStorage(storagePath)

	node := NewNode(1, fileStorage)

	node.StartElection()

	ctx := context.Background()
	term, votedFor, err := fileStorage.LoadVote(ctx)
	if err != nil {
		t.Fatalf("Failed to load vote: %v", err)
	}

	if term != 1 || votedFor != 1 {
		t.Fatalf("Expected vote for (1,1), got (%d,%d)", term, votedFor)
	}

	higherTermMsg := Message{
		Type: AppendEntriesMsg,
		From: 2,
		To:   node.ID,
		Term: 3,
	}

	node.Step(higherTermMsg)

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

func TestCorruptionDetection(t *testing.T) {
	tempDir := t.TempDir()
	storagePath := filepath.Join(tempDir, "vote.json")

	fileStorage := storage.NewFileStorage(storagePath)

	node := NewNode(1, fileStorage)
	node.StartElection()

	ctx := context.Background()
	term, votedFor, err := fileStorage.LoadVote(ctx)
	if err != nil {
		t.Fatalf("Failed to load vote: %v", err)
	}

	if term != 1 || votedFor != 1 {
		t.Fatalf("Expected vote for (1,1), got (%d,%d)", term, votedFor)
	}

	corruptedData := `{"term": 1, "voted_for": 2, "checksum": 12345}`
	err = os.WriteFile(storagePath, []byte(corruptedData), 0644)
	if err != nil {
		t.Fatalf("Failed to write corrupted data: %v", err)
	}

	term, votedFor, err = fileStorage.LoadVote(ctx)
	if err != storage.ErrCorrupted {
		t.Errorf("Expected ErrCorrupted, got %v", err)
	}
}

func TestAtomicWrites(t *testing.T) {
	tempDir := t.TempDir()
	storagePath := filepath.Join(tempDir, "vote.json")

	fileStorage := storage.NewFileStorage(storagePath)

	node := NewNode(1, fileStorage)

	for i := 1; i <= 5; i++ {
		msg := Message{
			Type: AppendEntriesMsg,
			From: 2,
			To:   node.ID,
			Term: i,
		}
		node.Step(msg)

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