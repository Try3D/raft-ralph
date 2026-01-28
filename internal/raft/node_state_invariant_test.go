package raft

import (
	"sync"
	"testing"
)

// TestNodeAlwaysInExactlyOneValidState tests the invariant that a Node is always in exactly one valid state
func TestNodeAlwaysInExactlyOneValidState(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	// Initially, node should be in Follower state
	if node.State != Follower {
		t.Errorf("Expected initial state to be Follower, got %v", node.State)
	}

	if !node.IsValidState() {
		t.Error("Initial state should be valid")
	}

	// Transition to Candidate
	node.TransitionToCandidate()
	if node.State != Candidate {
		t.Errorf("Expected state to be Candidate, got %v", node.State)
	}

	if !node.IsValidState() {
		t.Error("Candidate state should be valid")
	}

	// Transition to Leader
	node.TransitionToLeader()
	if node.State != Leader {
		t.Errorf("Expected state to be Leader, got %v", node.State)
	}

	if !node.IsValidState() {
		t.Error("Leader state should be valid")
	}

	// Transition back to Follower
	node.TransitionToFollower(node.CurrentTerm + 1)
	if node.State != Follower {
		t.Errorf("Expected state to be Follower, got %v", node.State)
	}

	if !node.IsValidState() {
		t.Error("Follower state should be valid")
	}
}

// TestNodeStateMutexProtection tests that state changes are properly protected by mutex
func TestNodeStateMutexProtection(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	const numGoroutines = 10
	var wg sync.WaitGroup

	// Launch goroutines to perform state transitions concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			switch goroutineID % 3 {
			case 0:
				node.TransitionToFollower(node.CurrentTerm)
			case 1:
				node.TransitionToCandidate()
			case 2:
				node.TransitionToLeader()
			}
		}(i)
	}

	wg.Wait()

	// After all goroutines complete, node should still be in a valid state
	if !node.IsValidState() {
		t.Errorf("Node should be in a valid state after concurrent transitions, got %v", node.State)
	}
}

// TestNodeStateTransitions tests that state transitions work correctly
func TestNodeStateTransitions(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	// Start as Follower
	if node.State != Follower {
		t.Errorf("Expected initial state to be Follower, got %v", node.State)
	}

	// Start election (becomes Candidate)
	node.StartElection()
	if node.State != Candidate {
		t.Errorf("Expected state to be Candidate after election, got %v", node.State)
	}

	// Become Leader
	node.TransitionToLeader()
	if node.State != Leader {
		t.Errorf("Expected state to be Leader, got %v", node.State)
	}

	// Step down to Follower
	node.TransitionToFollower(node.CurrentTerm + 1)
	if node.State != Follower {
		t.Errorf("Expected state to be Follower after stepping down, got %v", node.State)
	}
}