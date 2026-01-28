package raft

import (
	"sync"
	"testing"
)

func TestStepDownOnHigherTermMessage(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	// Start as a candidate
	node.StartElection()
	if node.State != Candidate {
		t.Fatalf("Expected node to be Candidate, got %v", node.State)
	}

	// Receive a message with higher term
	higherTerm := node.CurrentTerm + 1
	msg := Message{
		Type: AppendEntriesMsg,
		From: 2,
		To:   node.ID,
		Term: higherTerm,
	}

	node.Step(msg)

	// Verify the node stepped down to follower and updated its term
	if node.State != Follower {
		t.Errorf("Expected node to be Follower after receiving higher-term message, got %v", node.State)
	}

	if node.CurrentTerm != higherTerm {
		t.Errorf("Expected term to be updated to %d, got %d", higherTerm, node.CurrentTerm)
	}
}

func TestStepDownOnHigherTermRequestVote(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	// Start as a leader
	node.StartElection()
	node.TransitionToLeader()
	if node.State != Leader {
		t.Fatalf("Expected node to be Leader, got %v", node.State)
	}

	// Receive a RequestVote message with higher term
	higherTerm := node.CurrentTerm + 1
	msg := Message{
		Type: RequestVoteMsg,
		From: 2,
		To:   node.ID,
		Term: higherTerm,
	}

	node.Step(msg)

	// Verify the node stepped down to follower and updated its term
	if node.State != Follower {
		t.Errorf("Expected node to be Follower after receiving higher-term message, got %v", node.State)
	}

	if node.CurrentTerm != higherTerm {
		t.Errorf("Expected term to be updated to %d, got %d", higherTerm, node.CurrentTerm)
	}
}

func TestStepDownOnHigherTermAppendEntries(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	// Start as a leader
	node.StartElection()
	node.TransitionToLeader()
	if node.State != Leader {
		t.Fatalf("Expected node to be Leader, got %v", node.State)
	}

	// Receive an AppendEntries message with higher term
	higherTerm := node.CurrentTerm + 1
	msg := Message{
		Type: AppendEntriesMsg,
		From: 2,
		To:   node.ID,
		Term: higherTerm,
	}

	node.Step(msg)

	// Verify the node stepped down to follower and updated its term
	if node.State != Follower {
		t.Errorf("Expected node to be Follower after receiving higher-term message, got %v", node.State)
	}

	if node.CurrentTerm != higherTerm {
		t.Errorf("Expected term to be updated to %d, got %d", higherTerm, node.CurrentTerm)
	}
}

func TestNoStepDownOnEqualOrLowerTerm(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	// Start as a candidate
	node.StartElection()
	initialState := node.State
	initialTerm := node.CurrentTerm

	if initialState != Candidate {
		t.Fatalf("Expected node to be Candidate, got %v", initialState)
	}

	// Receive a message with equal term
	equalTermMsg := Message{
		Type: AppendEntriesMsg,
		From: 2,
		To:   node.ID,
		Term: initialTerm,
	}

	node.Step(equalTermMsg)

	// Verify the node did not step down and term remains the same
	if node.State != initialState {
		t.Errorf("Expected node to remain %v after receiving equal-term message, got %v", 
			initialState, node.State)
	}

	if node.CurrentTerm != initialTerm {
		t.Errorf("Expected term to remain %d after receiving equal-term message, got %d",
			initialTerm, node.CurrentTerm)
	}

	// Receive a message with lower term
	lowerTermMsg := Message{
		Type: RequestVoteMsg,
		From: 3,
		To:   node.ID,
		Term: initialTerm - 1,
	}

	node.Step(lowerTermMsg)

	// Verify the node did not step down and term remains the same
	if node.State != initialState {
		t.Errorf("Expected node to remain %v after receiving lower-term message, got %v", 
			initialState, node.State)
	}

	if node.CurrentTerm != initialTerm {
		t.Errorf("Expected term to remain %d after receiving lower-term message, got %d",
			initialTerm, node.CurrentTerm)
	}
}

func TestConcurrentHigherTermMessages(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	// Start as a leader
	node.StartElection()
	node.TransitionToLeader()
	if node.State != Leader {
		t.Fatalf("Expected node to be Leader, got %v", node.State)
	}

	initialTerm := node.CurrentTerm
	const numGoroutines = 5
	var wg sync.WaitGroup

	// Send multiple messages with different higher terms concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			higherTerm := initialTerm + 10 + goroutineID
			msg := Message{
				Type: AppendEntriesMsg,
				From: goroutineID + 10,
				To:   node.ID,
				Term: higherTerm,
			}

			node.Step(msg)
		}(i)
	}

	wg.Wait()

	// Verify the node stepped down to follower and term is at least as high as the highest sent
	if node.State != Follower {
		t.Errorf("Expected node to be Follower after concurrent higher-term messages, got %v", node.State)
	}

	if node.CurrentTerm < initialTerm+10 {
		t.Errorf("Expected term to be at least %d after concurrent higher-term messages, got %d",
			initialTerm+10, node.CurrentTerm)
	}
}

func TestStepDownPreservesOtherState(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	// Add some log entries
	node.AppendEntry(LogEntry{Command: "cmd1", Term: 1, Index: 0})
	node.AppendEntry(LogEntry{Command: "cmd2", Term: 1, Index: 1})

	// Start as a leader
	node.StartElection()
	node.TransitionToLeader()
	
	// Set some state values
	node.CommitIndex = 1
	node.LastApplied = 0

	logLenBefore := len(node.Log)
	commitIndexBefore := node.CommitIndex
	lastAppliedBefore := node.LastApplied

	// Receive a message with higher term
	higherTerm := node.CurrentTerm + 1
	msg := Message{
		Type: AppendEntriesMsg,
		From: 2,
		To:   node.ID,
		Term: higherTerm,
	}

	node.Step(msg)

	// Verify the node stepped down to follower
	if node.State != Follower {
		t.Errorf("Expected node to be Follower after receiving higher-term message, got %v", node.State)
	}

	// Verify other state is preserved
	if len(node.Log) != logLenBefore {
		t.Errorf("Expected log length to be preserved (%d), got %d", logLenBefore, len(node.Log))
	}

	if node.CommitIndex != commitIndexBefore {
		t.Errorf("Expected commit index to be preserved (%d), got %d", commitIndexBefore, node.CommitIndex)
	}

	if node.LastApplied != lastAppliedBefore {
		t.Errorf("Expected last applied to be preserved (%d), got %d", lastAppliedBefore, node.LastApplied)
	}

	if node.CurrentTerm != higherTerm {
		t.Errorf("Expected term to be updated to %d, got %d", higherTerm, node.CurrentTerm)
	}
}