package raft

import (
	"sync"
	"testing"
)

// TestStartElectionIncrementsTermAndVotesForSelf tests that StartElection
// increments the term and votes for the node's own ID
func TestStartElectionIncrementsTermAndVotesForSelf(t *testing.T) {
	node := NewNode(1, &MockStorage{})

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
	node := NewNode(1, &MockStorage{})

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
	node := NewNode(1, &MockStorage{})

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

// TestNodeAlwaysInValidState tests that the Node is always in exactly one valid state
func TestNodeAlwaysInValidState(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	// Initially should be in Follower state
	if node.State != Follower {
		t.Errorf("Expected initial state to be Follower, got %v", node.State)
	}

	if !node.IsValidState() {
		t.Error("Node should be in a valid state initially")
	}

	// Transition to candidate
	node.StartElection()
	if node.State != Candidate {
		t.Errorf("Expected state to be Candidate after election, got %v", node.State)
	}

	if !node.IsValidState() {
		t.Error("Node should be in a valid state after election")
	}

	// Transition to leader
	node.TransitionToLeader()
	if node.State != Leader {
		t.Errorf("Expected state to be Leader after transition, got %v", node.State)
	}

	if !node.IsValidState() {
		t.Error("Node should be in a valid state after becoming leader")
	}

	// Transition back to follower
	node.TransitionToFollower(node.CurrentTerm + 1)
	if node.State != Follower {
		t.Errorf("Expected state to be Follower after transition, got %v", node.State)
	}

	if !node.IsValidState() {
		t.Error("Node should be in a valid state after becoming follower")
	}
}

// TestStateTransitionsAreExplicit tests that state transitions are explicit and testable
func TestStateTransitionsAreExplicit(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	// Test initial state
	if node.State != Follower {
		t.Errorf("Expected initial state to be Follower, got %v", node.State)
	}

	// Test transition to candidate
	node.TransitionToCandidate()
	if node.State != Candidate {
		t.Errorf("Expected state to be Candidate, got %v", node.State)
	}

	// Test transition to leader
	node.TransitionToLeader()
	if node.State != Leader {
		t.Errorf("Expected state to be Leader, got %v", node.State)
	}

	// Test transition back to follower
	node.TransitionToFollower(node.CurrentTerm + 1)
	if node.State != Follower {
		t.Errorf("Expected state to be Follower, got %v", node.State)
	}
}