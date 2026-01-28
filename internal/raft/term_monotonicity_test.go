package raft

import (
	"sync"
	"testing"
)

func TestTermNeverDecreases(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	initialTerm := node.CurrentTerm
	if initialTerm != 0 {
		t.Errorf("Expected initial term to be 0, got %d", initialTerm)
	}

	node.StartElection()
	termAfterElection := node.CurrentTerm

	if termAfterElection <= initialTerm {
		t.Errorf("Expected term to increase after election, got %d, was %d",
			termAfterElection, initialTerm)
	}

	higherTerm := termAfterElection + 5
	node.TransitionToFollower(higherTerm)

	if node.CurrentTerm != higherTerm {
		t.Errorf("Expected term to be updated to higher term %d, got %d",
			higherTerm, node.CurrentTerm)
	}

	lowerTerm := higherTerm - 3
	node.TransitionToFollower(lowerTerm)

	if node.CurrentTerm != higherTerm {
		t.Errorf("Expected term to remain %d (not decrease to %d)",
			higherTerm, lowerTerm)
	}
}

func TestHigherTermForcesFollower(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	if node.State != Follower {
		t.Fatalf("Expected initial state to be Follower, got %v", node.State)
	}

	node.StartElection()
	if node.State != Candidate {
		t.Fatalf("Expected state to be Candidate after election, got %v", node.State)
	}

	higherTerm := node.CurrentTerm + 1
	msg := Message{
		Type: RequestVoteMsg,
		From: 2,
		To:   node.ID,
		Term: higherTerm,
	}

	node.Step(msg)

	if node.State != Follower {
		t.Errorf("Expected state to be Follower after receiving higher-term message, got %v",
			node.State)
	}

	if node.CurrentTerm != higherTerm {
		t.Errorf("Expected term to be updated to higher term %d, got %d",
			higherTerm, node.CurrentTerm)
	}
}

func TestTermUpdateWithMessage(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	initialTerm := node.CurrentTerm
	higherTerm := initialTerm + 3

	msg := Message{
		Type: AppendEntriesMsg,
		From: 2,
		To:   node.ID,
		Term: higherTerm,
	}

	node.Step(msg)

	if node.CurrentTerm != higherTerm {
		t.Errorf("Expected term to be updated to %d after receiving higher-term message, got %d",
			higherTerm, node.CurrentTerm)
	}

	if node.State != Follower {
		t.Errorf("Expected node to be in Follower state after receiving higher-term message, got %v",
			node.State)
	}
}

func TestMultipleConcurrentMessages(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	const numGoroutines = 10
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			term := 10 + goroutineID
			msg := Message{
				Type: AppendEntriesMsg,
				From: goroutineID + 10,
				To:   node.ID,
				Term: term,
			}

			node.Step(msg)
		}(i)
	}

	wg.Wait()

	expectedHighestTerm := 10 + (numGoroutines - 1)
	if node.CurrentTerm < expectedHighestTerm {
		t.Errorf("Expected term to be at least %d after concurrent messages, got %d",
			expectedHighestTerm, node.CurrentTerm)
	}

	if !node.IsValidState() {
		t.Errorf("Node is in an invalid state after concurrent messages: %v", node.State)
	}
}

func TestLowerTermMessageDoesNotChangeTerm(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	node.StartElection()
	initialTerm := node.CurrentTerm

	lowerTermMsg := Message{
		Type: RequestVoteMsg,
		From: 2,
		To:   node.ID,
		Term: initialTerm - 1,
	}

	node.Step(lowerTermMsg)

	if node.CurrentTerm != initialTerm {
		t.Errorf("Expected term to remain %d after receiving lower-term message, got %d",
			initialTerm, node.CurrentTerm)
	}

	equalTermMsg := Message{
		Type: AppendEntriesMsg,
		From: 3,
		To:   node.ID,
		Term: initialTerm,
	}

	node.Step(equalTermMsg)

	if node.CurrentTerm != initialTerm {
		t.Errorf("Expected term to remain %d after receiving equal-term message, got %d",
			initialTerm, node.CurrentTerm)
	}
}