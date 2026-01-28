package raft

import (
	"sync"
	"testing"
)

// TestTermNeverDecreases tests that currentTerm never decreases
func TestTermNeverDecreases(t *testing.T) {
	node := NewNode(1)
	
	// Start with term 0
	initialTerm := node.CurrentTerm
	if initialTerm != 0 {
		t.Errorf("Expected initial term to be 0, got %d", initialTerm)
	}
	
	// Increment term by starting an election
	node.StartElection()
	termAfterElection := node.CurrentTerm
	
	if termAfterElection <= initialTerm {
		t.Errorf("Expected term to increase after election, got %d, was %d", 
			termAfterElection, initialTerm)
	}
	
	// Manually set a higher term to simulate receiving a higher-term message
	higherTerm := termAfterElection + 5
	node.TransitionToFollower(higherTerm)
	
	if node.CurrentTerm != higherTerm {
		t.Errorf("Expected term to be updated to higher term %d, got %d", 
			higherTerm, node.CurrentTerm)
	}
	
	// Now try to transition to a lower term - this should not decrease the term
	lowerTerm := higherTerm - 3
	node.TransitionToFollower(lowerTerm)
	
	if node.CurrentTerm != higherTerm {
		t.Errorf("Expected term to remain %d (not decrease to %d)", 
			higherTerm, lowerTerm)
	}
}

// TestHigherTermForcesFollower tests that any higher term message forces follower state
func TestHigherTermForcesFollower(t *testing.T) {
	node := NewNode(1)
	
	// Start as follower
	if node.State != Follower {
		t.Fatalf("Expected initial state to be Follower, got %v", node.State)
	}
	
	// Transition to candidate
	node.StartElection()
	if node.State != Candidate {
		t.Fatalf("Expected state to be Candidate after election, got %v", node.State)
	}
	
	// Receive a message with higher term - should force follower state
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

// TestTermUpdateWithMessage tests that receiving higher-term message updates local term
func TestTermUpdateWithMessage(t *testing.T) {
	node := NewNode(1)
	
	initialTerm := node.CurrentTerm
	higherTerm := initialTerm + 3
	
	// Send a message with higher term
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
	
	// Verify that the node is now a follower
	if node.State != Follower {
		t.Errorf("Expected node to be in Follower state after receiving higher-term message, got %v", 
			node.State)
	}
}

// TestMultipleConcurrentMessages tests concurrent messages to verify no data races and term monotonicity
func TestMultipleConcurrentMessages(t *testing.T) {
	node := NewNode(1)
	
	const numGoroutines = 10
	var wg sync.WaitGroup
	
	// Launch multiple goroutines that send messages with different terms
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			// Send a message with a unique term
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
	
	// After all goroutines complete, verify that the term is the highest one sent
	expectedHighestTerm := 10 + (numGoroutines - 1) // 10 + 9 = 19
	if node.CurrentTerm < expectedHighestTerm {
		t.Errorf("Expected term to be at least %d after concurrent messages, got %d", 
			expectedHighestTerm, node.CurrentTerm)
	}
	
	// Verify the node is still in a valid state
	if !node.IsValidState() {
		t.Errorf("Node is in an invalid state after concurrent messages: %v", node.State)
	}
}

// TestLowerTermMessageDoesNotChangeTerm tests that receiving lower/equal term messages does not change term
func TestLowerTermMessageDoesNotChangeTerm(t *testing.T) {
	node := NewNode(1)
	
	// Start an election to increase the term
	node.StartElection()
	initialTerm := node.CurrentTerm
	
	// Send a message with lower term
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
	
	// Send a message with equal term
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