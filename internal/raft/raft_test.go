package raft

import (
	"sync"
	"testing"
)

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

func TestStartElectionConcurrent(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	var mu sync.Mutex

	const numGoroutines = 5
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			mu.Lock()
			defer mu.Unlock()

			node.StartElection()
		}(i)
	}

	wg.Wait()

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

func TestNodeAlwaysInValidState(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	if node.State != Follower {
		t.Errorf("Expected initial state to be Follower, got %v", node.State)
	}

	if !node.IsValidState() {
		t.Error("Node should be in a valid state initially")
	}

	node.StartElection()
	if node.State != Candidate {
		t.Errorf("Expected state to be Candidate after election, got %v", node.State)
	}

	if !node.IsValidState() {
		t.Error("Node should be in a valid state after election")
	}

	node.TransitionToLeader()
	if node.State != Leader {
		t.Errorf("Expected state to be Leader after transition, got %v", node.State)
	}

	if !node.IsValidState() {
		t.Error("Node should be in a valid state after becoming leader")
	}

	node.TransitionToFollower(node.CurrentTerm + 1)
	if node.State != Follower {
		t.Errorf("Expected state to be Follower after transition, got %v", node.State)
	}

	if !node.IsValidState() {
		t.Error("Node should be in a valid state after becoming follower")
	}
}

func TestStateTransitionsAreExplicit(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	if node.State != Follower {
		t.Errorf("Expected initial state to be Follower, got %v", node.State)
	}

	node.TransitionToCandidate()
	if node.State != Candidate {
		t.Errorf("Expected state to be Candidate, got %v", node.State)
	}

	node.TransitionToLeader()
	if node.State != Leader {
		t.Errorf("Expected state to be Leader, got %v", node.State)
	}

	node.TransitionToFollower(node.CurrentTerm + 1)
	if node.State != Follower {
		t.Errorf("Expected state to be Follower, got %v", node.State)
	}
}