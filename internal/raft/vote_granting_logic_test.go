package raft

import (
	"sync"
	"testing"
)

func TestSingleVotePerTerm(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	if node.VotedFor != -1 {
		t.Errorf("Expected initial VotedFor to be -1, got %d", node.VotedFor)
	}

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

	msgB := Message{
		Type: RequestVoteMsg,
		From: 3,
		To:   node.ID,
		Term: 1,
	}

	node.Step(msgB)

	if node.VotedFor != 2 {
		t.Errorf("Expected vote to remain with candidate 2, got %d", node.VotedFor)
	}
}

func TestVoteResetOnTermIncrease(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	if node.VotedFor != -1 {
		t.Errorf("Expected initial VotedFor to be -1, got %d", node.VotedFor)
	}

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

	msgB := Message{
		Type: RequestVoteMsg,
		From: 3,
		To:   node.ID,
		Term: 2,
	}

	node.Step(msgB)

	if node.VotedFor != 3 {
		t.Errorf("Expected vote to be granted to candidate 3, got %d", node.VotedFor)
	}

	if node.CurrentTerm != 2 {
		t.Errorf("Expected term to be updated to 2, got %d", node.CurrentTerm)
	}
}

func TestVotingDifferentCandidates(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	if node.VotedFor != -1 {
		t.Errorf("Expected initial VotedFor to be -1, got %d", node.VotedFor)
	}

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

	msgB := Message{
		Type: RequestVoteMsg,
		From: 3,
		To:   node.ID,
		Term: 1,
	}

	node.Step(msgB)

	if node.VotedFor != 2 {
		t.Errorf("Expected vote to remain with candidate 2, got %d", node.VotedFor)
	}
}

func TestConcurrentVoteRequests(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	node.StartElection()

	const numGoroutines = 5
	var wg sync.WaitGroup

	voteResults := make(chan int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(candidateID int) {
			defer wg.Done()

			msg := Message{
				Type: RequestVoteMsg,
				From: candidateID + 10,
				To:   node.ID,
				Term: 1,
			}

			node.Step(msg)

			node.mutex.RLock()
			currentVote := node.VotedFor
			node.mutex.RUnlock()

			voteResults <- currentVote
		}(i)
	}

	wg.Wait()
	close(voteResults)

	finalVote := -1
	node.mutex.RLock()
	finalVote = node.VotedFor
	node.mutex.RUnlock()

	if finalVote == -1 {
		t.Error("Expected a candidate to receive the vote, but got -1 (no vote)")
	}

	count := 0
	for vote := range voteResults {
		if vote == finalVote {
			count++
		}
	}

	if count == 0 {
		t.Errorf("Final vote %d not found in results", finalVote)
	}
}

func TestVoteGrantingRules(t *testing.T) {
	node := NewNode(1, &MockStorage{})

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

	msg3 := Message{
		Type: RequestVoteMsg,
		From: 4,
		To:   node.ID,
		Term: 1,
	}

	node.Step(msg3)

	if node.VotedFor != 4 {
		t.Errorf("Expected vote to be granted to candidate 4 in term 1, got %d", node.VotedFor)
	}

	if node.CurrentTerm != 1 {
		t.Errorf("Expected term to be updated to 1, got %d", node.CurrentTerm)
	}
}