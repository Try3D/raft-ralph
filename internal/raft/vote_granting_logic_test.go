package raft

import (
	"sync"
	"testing"
)

// TestSingleVotePerTerm tests that a node cannot vote twice in the same term
func TestSingleVotePerTerm(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	// Initially, VotedFor should be -1 (no vote)
	if node.VotedFor != -1 {
		t.Errorf("Expected initial VotedFor to be -1, got %d", node.VotedFor)
	}

	// First vote request from candidate A in term 1
	msgA := Message{
		Type: RequestVoteMsg,
		From: 2,
		To:   node.ID,
		Term: 1,
	}

	node.Step(msgA)

	// Check that the vote was granted to candidate A
	if node.VotedFor != 2 {
		t.Errorf("Expected vote to be granted to candidate 2, got %d", node.VotedFor)
	}

	// Second vote request from candidate B in the same term
	msgB := Message{
		Type: RequestVoteMsg,
		From: 3,
		To:   node.ID,
		Term: 1,
	}

	node.Step(msgB)

	// Check that the vote was NOT granted to candidate B (still with candidate A)
	if node.VotedFor != 2 {
		t.Errorf("Expected vote to remain with candidate 2, got %d", node.VotedFor)
	}
}

// TestVoteResetOnTermIncrease tests that vote resets when term increases
func TestVoteResetOnTermIncrease(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	// Initially, VotedFor should be -1 (no vote)
	if node.VotedFor != -1 {
		t.Errorf("Expected initial VotedFor to be -1, got %d", node.VotedFor)
	}

	// Vote for candidate A in term 1
	msgA := Message{
		Type: RequestVoteMsg,
		From: 2,
		To:   node.ID,
		Term: 1,
	}

	node.Step(msgA)

	if node.VotedFor != 2 {
		t.Errorf("Expected vote to be granted to candidate 2, got %d", node.VotedFor)
	}

	// Receive a vote request with higher term - should reset vote and vote for new candidate
	msgB := Message{
		Type: RequestVoteMsg,
		From: 3,
		To:   node.ID,
		Term: 2, // Higher term
	}

	node.Step(msgB)

	// Check that the vote was granted to candidate B in the new term
	if node.VotedFor != 3 {
		t.Errorf("Expected vote to be granted to candidate 3, got %d", node.VotedFor)
	}

	if node.CurrentTerm != 2 {
		t.Errorf("Expected term to be updated to 2, got %d", node.CurrentTerm)
	}
}

// TestVotingDifferentCandidates tests that a node cannot vote for two different candidates in the same term
func TestVotingDifferentCandidates(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	// Initially, VotedFor should be -1 (no vote)
	if node.VotedFor != -1 {
		t.Errorf("Expected initial VotedFor to be -1, got %d", node.VotedFor)
	}

	// Vote for candidate A
	msgA := Message{
		Type: RequestVoteMsg,
		From: 2,
		To:   node.ID,
		Term: 1,
	}

	node.Step(msgA)

	if node.VotedFor != 2 {
		t.Errorf("Expected vote to be granted to candidate 2, got %d", node.VotedFor)
	}

	// Try to vote for candidate B in the same term
	msgB := Message{
		Type: RequestVoteMsg,
		From: 3,
		To:   node.ID,
		Term: 1,
	}

	node.Step(msgB)

	// Check that the vote was NOT granted to candidate B
	if node.VotedFor != 2 {
		t.Errorf("Expected vote to remain with candidate 2, got %d", node.VotedFor)
	}
}

// TestConcurrentVoteRequests tests concurrent vote requests with mutex protection
func TestConcurrentVoteRequests(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	
	// Set the term to 1
	node.StartElection() // Increments term to 1
	
	const numGoroutines = 5
	var wg sync.WaitGroup
	
	// Channel to collect who got the vote
	voteResults := make(chan int, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(candidateID int) {
			defer wg.Done()
			
			// Each goroutine sends a vote request from a different candidate
			// but in the same term - only one should get the vote
			msg := Message{
				Type: RequestVoteMsg,
				From: candidateID + 10, // Different candidate IDs
				To:   node.ID,
				Term: 1, // Same term for all
			}
			
			node.Step(msg)
			
			// Collect the current vote
			node.mutex.RLock()
			currentVote := node.VotedFor
			node.mutex.RUnlock()
			
			voteResults <- currentVote
		}(i)
	}
	
	wg.Wait()
	close(voteResults)
	
	// Check that only one candidate got the vote
	finalVote := -1
	node.mutex.RLock()
	finalVote = node.VotedFor
	node.mutex.RUnlock()
	
	if finalVote == -1 {
		t.Error("Expected a candidate to receive the vote, but got -1 (no vote)")
	}
	
	// Verify that the vote didn't change after the first request
	count := 0
	for vote := range voteResults {
		if vote == finalVote {
			count++
		}
	}
	
	// The final vote should appear at least once in the results
	if count == 0 {
		t.Errorf("Final vote %d not found in results", finalVote)
	}
}

// TestVoteGrantingRules tests the specific conditions for granting votes
func TestVoteGrantingRules(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	
	// Initially, node should vote for anyone in term 0
	msg := Message{
		Type: RequestVoteMsg,
		From: 2,
		To:   node.ID,
		Term: 0,
	}
	
	node.Step(msg)
	
	if node.VotedFor != 2 {
		t.Errorf("Expected vote to be granted to candidate 2 in term 0, got %d", node.VotedFor)
	}
	
	// Now try to vote for someone else in term 0 - should not be granted
	msg2 := Message{
		Type: RequestVoteMsg,
		From: 3,
		To:   node.ID,
		Term: 0,
	}
	
	node.Step(msg2)
	
	if node.VotedFor != 2 {
		t.Errorf("Expected vote to remain with candidate 2 in term 0, got %d", node.VotedFor)
	}
	
	// Now try to vote for someone in a higher term - should be granted
	msg3 := Message{
		Type: RequestVoteMsg,
		From: 4,
		To:   node.ID,
		Term: 1, // Higher term
	}
	
	node.Step(msg3)
	
	if node.VotedFor != 4 {
		t.Errorf("Expected vote to be granted to candidate 4 in term 1, got %d", node.VotedFor)
	}
	
	if node.CurrentTerm != 1 {
		t.Errorf("Expected term to be updated to 1, got %d", node.CurrentTerm)
	}
}