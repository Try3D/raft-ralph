package raft

import (
	"sync"
	"testing"
)

// TestStartElectionIncrementsTermAndVotesForSelf tests that StartElection
// increments the term and votes for self
func TestStartElectionIncrementsTermAndVotesForSelf(t *testing.T) {
	node := NewNode(1)

	initialTerm := node.CurrentTerm

	node.StartElection()

	if node.CurrentTerm != initialTerm+1 {
		t.Errorf("Expected term to increment from %d to %d, got %d",
			initialTerm, initialTerm+1, node.CurrentTerm)
	}

	if node.VotedFor != node.ID {
		t.Errorf("Expected node to vote for self (%d), got %d",
			node.ID, node.VotedFor)
	}
}

// TestStartElectionTransitionsToCandidate tests that StartElection
// transitions the node to Candidate state
func TestStartElectionTransitionsToCandidate(t *testing.T) {
	node := NewNode(1)
	
	if node.State != Follower {
		t.Fatalf("Expected node to start as Follower, got %v", node.State)
	}
	
	node.StartElection()
	
	if node.State != Candidate {
		t.Errorf("Expected node to transition to Candidate, got %v", node.State)
	}
}

// TestStartElectionConcurrent tests that concurrent calls to StartElection
// maintain invariants using sync.Mutex to protect shared state
func TestStartElectionConcurrent(t *testing.T) {
	node := NewNode(1)
	
	const numGoroutines = 5
	var wg sync.WaitGroup
	var mu sync.Mutex // Protect access to shared state during tests
	
	// Capture initial values
	mu.Lock()
	initialTerm := node.CurrentTerm
	mu.Unlock()
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			// Lock to ensure we're checking the state properly
			mu.Lock()
			node.StartElection()
			currentTerm := node.CurrentTerm
			currentVotedFor := node.VotedFor
			currentState := node.State
			mu.Unlock()
			
			// Check invariants after each call
			if currentTerm <= initialTerm {
				t.Errorf("Goroutine %d: Expected term to increment, initial: %d, current: %d", 
					goroutineID, initialTerm, currentTerm)
			}
			
			if currentVotedFor != node.ID {
				t.Errorf("Goroutine %d: Expected node to vote for self (%d), got %d", 
					goroutineID, node.ID, currentVotedFor)
			}
			
			if currentState != Candidate {
				t.Errorf("Goroutine %d: Expected node to be Candidate, got %v", 
					goroutineID, currentState)
			}
		}(i)
	}
	
	wg.Wait()
	
	// After all goroutines complete, verify final state
	finalTerm := node.CurrentTerm
	finalVotedFor := node.VotedFor
	finalState := node.State
	
	// The term should have been incremented at least once (could be more due to race conditions)
	if finalTerm <= initialTerm {
		t.Errorf("Expected final term to be greater than initial term, initial: %d, final: %d", 
			initialTerm, finalTerm)
	}
	
	if finalVotedFor != node.ID {
		t.Errorf("Expected final votedFor to be self (%d), got %d", 
			node.ID, finalVotedFor)
	}
	
	if finalState != Candidate {
		t.Errorf("Expected final state to be Candidate, got %v", finalState)
	}
}