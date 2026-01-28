package raft

import (
	"sync"
	"testing"
)

func TestVoteGrantedIfLogUpToDate(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	node.CurrentTerm = 1
	currentTerm := node.CurrentTerm

	entry1 := LogEntry{Command: "command1", Term: currentTerm}
	node.AppendEntry(entry1)

	entry2 := LogEntry{Command: "command2", Term: currentTerm}
	node.AppendEntry(entry2)

	if node.VotedFor != -1 {
		t.Fatalf("Expected VotedFor to be -1 initially, got %d", node.VotedFor)
	}

	msg := Message{
		Type:     RequestVoteMsg,
		From:     2,
		To:       1,
		Term:     currentTerm,
		LogTerm:  currentTerm,
		LogIndex: 2,
	}

	node.Step(msg)

	if node.VotedFor != 2 {
		t.Errorf("Expected node to vote for candidate 2, got vote for %d", node.VotedFor)
	}
}

func TestVoteDeniedIfLogOutOfDate(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	node.CurrentTerm = 1
	currentTerm := node.CurrentTerm

	entry1 := LogEntry{Command: "command1", Term: currentTerm}
	node.AppendEntry(entry1)

	entry2 := LogEntry{Command: "command2", Term: currentTerm}
	node.AppendEntry(entry2)

	msg := Message{
		Type:     RequestVoteMsg,
		From:     2,
		To:       1,
		Term:     currentTerm,
		LogTerm:  currentTerm,
		LogIndex: 0,
	}

	if node.VotedFor != -1 {
		t.Fatalf("Expected VotedFor to be -1 initially, got %d", node.VotedFor)
	}

	node.Step(msg)

	if node.VotedFor != -1 {
		t.Errorf("Expected node not to vote for candidate with out-of-date log, got vote for %d", node.VotedFor)
	}
}

func TestLogComparisonByTerm(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	node.CurrentTerm = 1
	currentTerm := node.CurrentTerm

	entry1 := LogEntry{Command: "command1", Term: 1}
	node.AppendEntry(entry1)

	entry2 := LogEntry{Command: "command2", Term: 1}
	node.AppendEntry(entry2)

	msg := Message{
		Type:     RequestVoteMsg,
		From:     2,
		To:       1,
		Term:     currentTerm,
		LogTerm:  2,
		LogIndex: 0,
	}

	if node.VotedFor != -1 {
		t.Fatalf("Expected VotedFor to be -1 initially, got %d", node.VotedFor)
	}

	node.Step(msg)

	if node.VotedFor != 2 {
		t.Errorf("Expected node to vote for candidate with higher log term, got vote for %d", node.VotedFor)
	}
}

func TestConcurrentVoteComparison(t *testing.T) {
	const numNodes = 5
	nodes := make([]*Node, numNodes)

	for i := 0; i < numNodes; i++ {
		nodes[i] = NewNode(i+1, &MockStorage{})
		nodes[i].StartElection()
	}

	for i := 0; i < numNodes; i++ {
		for j := 0; j <= i; j++ {
			entry := LogEntry{Command: "command" + string(rune('0'+j)), Term: 1}
			nodes[i].AppendEntry(entry)
		}
	}

	const numCandidates = 3
	var wg sync.WaitGroup

	for i := 0; i < numCandidates; i++ {
		wg.Add(1)
		go func(candidateID int) {
			defer wg.Done()

			for j := 0; j < numNodes; j++ {
				msg := Message{
					Type:     RequestVoteMsg,
					From:     candidateID + 10,
					To:       j + 1,
					Term:     1,
					LogTerm:  1,
					LogIndex: candidateID,
				}

				nodes[j].Step(msg)
			}
		}(i)
	}

	wg.Wait()

	for i, node := range nodes {
		if !node.IsValidState() {
			t.Errorf("Node %d is in invalid state after concurrent vote comparisons", i+1)
		}
	}
}