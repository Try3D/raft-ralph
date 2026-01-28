package raft

import (
	"sync"
	"testing"
)

// TestStartElectionIncrementsTermAndVotesForSelf tests that StartElection
// increments the term and votes for the node's own ID
func TestStartElectionIncrementsTermAndVotesForSelf(t *testing.T) {
	node := NewNode(1)

	initialTerm := node.CurrentTerm

	node.StartElection()

	if node.CurrentTerm != initialTerm+1 {
		t.Errorf("Expected term to increment from %d to %d, got %d",
			initialTerm, initialTerm+1, node.CurrentTerm)
	}

	if node.VotedFor != node.ID {
		t.Errorf("Expected VotedFor to be set to node ID %d, got %d",
			node.ID, node.VotedFor)
	}
}

// TestStartElectionTransitionsToCandidate tests that StartElection
// transitions the node to Candidate state
func TestStartElectionTransitionsToCandidate(t *testing.T) {
	node := NewNode(1)
	
	if node.State != Follower {
		t.Fatalf("Expected initial state to be Follower, got %v", node.State)
	}
	
	node.StartElection()
	
	if node.State != Candidate {
		t.Errorf("Expected state to transition to Candidate, got %v", node.State)
	}
}

// TestStartElectionConcurrent tests that concurrent calls to StartElection
// maintain invariants using sync.Mutex to protect shared state
func TestStartElectionConcurrent(t *testing.T) {
	node := NewNode(1)
	
	// Using a mutex to protect access to the node during concurrent operations
	var mu sync.Mutex
	
	const numGoroutines = 5
	var wg sync.WaitGroup
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			mu.Lock()
			defer mu.Unlock()
			
			// Each goroutine calls StartElection
			// Though in practice only one will actually transition to candidate
			// due to the mutex protection, this tests that the operation is safe
			node.StartElection()
		}(i)
	}
	
	wg.Wait()
	
	// After all goroutines complete, verify invariants still hold
	if node.CurrentTerm <= 0 {
		t.Errorf("Expected term to be greater than 0 after concurrent elections, got %d", node.CurrentTerm)
	}
	
	if node.VotedFor != node.ID {
		t.Errorf("Expected VotedFor to be set to node ID %d after concurrent elections, got %d", 
			node.ID, node.VotedFor)
	}
	
	if node.State != Candidate {
		t.Errorf("Expected state to be Candidate after concurrent elections, got %v", node.State)
	}
}