package raft

import (
	"sync"
	"testing"
)

// TestVoteGrantedIfLogUpToDate tests that a candidate receives a vote if its log is at least as up-to-date as the voter's
// and the voter hasn't voted yet in this term
func TestVoteGrantedIfLogUpToDate(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	// Manually set up the node with some log entries but don't vote yet
	node.CurrentTerm = 1 // Manually set term to 1
	currentTerm := node.CurrentTerm

	// Add some entries to the node's log
	entry1 := LogEntry{Command: "command1", Term: currentTerm}
	node.AppendEntry(entry1) // Index 0

	entry2 := LogEntry{Command: "command2", Term: currentTerm}
	node.AppendEntry(entry2) // Index 1

	// The node's last log index is 1, term is 1, and VotedFor should be -1 (no vote in this term)
	if node.VotedFor != -1 {
		t.Fatalf("Expected VotedFor to be -1 initially, got %d", node.VotedFor)
	}

	// Send a RequestVote from a candidate with a more up-to-date log (higher index)
	msg := Message{
		Type:     RequestVoteMsg,
		From:     2,
		To:       1,
		Term:     currentTerm,
		LogTerm:  currentTerm,
		LogIndex: 2, // Higher index than node's log
	}

	// Process the message
	node.Step(msg)

	// Check that the vote was granted by checking the voted-for state
	if node.VotedFor != 2 {
		t.Errorf("Expected node to vote for candidate 2, got vote for %d", node.VotedFor)
	}
}

// TestVoteDeniedIfLogOutOfDate tests that a candidate is denied a vote if its log is less up-to-date than the voter's
func TestVoteDeniedIfLogOutOfDate(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	// Manually set up the node with some log entries
	node.CurrentTerm = 1 // Manually set term to 1
	currentTerm := node.CurrentTerm

	// Add some entries to the node's log
	entry1 := LogEntry{Command: "command1", Term: currentTerm}
	node.AppendEntry(entry1) // Index 0

	entry2 := LogEntry{Command: "command2", Term: currentTerm}
	node.AppendEntry(entry2) // Index 1

	// Node's log: index 1, term 1
	// Candidate's log: index 0, term 1 (less up-to-date)

	msg := Message{
		Type:     RequestVoteMsg,
		From:     2,
		To:       1,
		Term:     currentTerm,
		LogTerm:  currentTerm, // Same term
		LogIndex: 0,           // Lower index
	}

	// Initially VotedFor should be -1
	if node.VotedFor != -1 {
		t.Fatalf("Expected VotedFor to be -1 initially, got %d", node.VotedFor)
	}

	// Process the message - vote should not be granted because the candidate's log is not up-to-date
	node.Step(msg)

	// The vote should not be granted since the candidate's log is not up-to-date
	// VotedFor should remain -1
	if node.VotedFor != -1 {
		t.Errorf("Expected node not to vote for candidate with out-of-date log, got vote for %d", node.VotedFor)
	}
}

// TestLogComparisonByTerm tests that term takes precedence over index in log comparison
func TestLogComparisonByTerm(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	// Manually set up the node with some log entries
	node.CurrentTerm = 1 // Manually set term to 1
	currentTerm := node.CurrentTerm

	// Add an entry to the node's log with a high index but low term
	entry1 := LogEntry{Command: "command1", Term: 1} // Low term
	node.AppendEntry(entry1) // Index 0

	entry2 := LogEntry{Command: "command2", Term: 1} // Low term
	node.AppendEntry(entry2) // Index 1

	// Node's last log: index 1, term 1

	// Send a RequestVote from a candidate with a lower index but higher term
	msg := Message{
		Type:     RequestVoteMsg,
		From:     2,
		To:       1,
		Term:     currentTerm,
		LogTerm:  2, // Higher term
		LogIndex: 0, // Lower index
	}

	// Initially VotedFor should be -1
	if node.VotedFor != -1 {
		t.Fatalf("Expected VotedFor to be -1 initially, got %d", node.VotedFor)
	}

	// Process the message
	node.Step(msg)

	// The vote should be granted because the candidate has a higher log term
	if node.VotedFor != 2 {
		t.Errorf("Expected node to vote for candidate with higher log term, got vote for %d", node.VotedFor)
	}
}

// TestConcurrentVoteComparison tests multiple nodes comparing logs concurrently
func TestConcurrentVoteComparison(t *testing.T) {
	const numNodes = 5
	nodes := make([]*Node, numNodes)
	
	// Create nodes
	for i := 0; i < numNodes; i++ {
		nodes[i] = NewNode(i+1, &MockStorage{})
		nodes[i].StartElection() // Set to term 1
	}
	
	// Add different amounts of entries to each node to vary their log lengths
	for i := 0; i < numNodes; i++ {
		for j := 0; j <= i; j++ { // Node i gets i+1 entries
			entry := LogEntry{Command: "command" + string(rune('0'+j)), Term: 1}
			nodes[i].AppendEntry(entry)
		}
	}
	
	const numCandidates = 3
	var wg sync.WaitGroup
	
	// Launch goroutines to send vote requests concurrently
	for i := 0; i < numCandidates; i++ {
		wg.Add(1)
		go func(candidateID int) {
			defer wg.Done()
			
			// Each candidate sends vote requests to all nodes
			for j := 0; j < numNodes; j++ {
				// Create a message with varying log properties
				msg := Message{
					Type:     RequestVoteMsg,
					From:     candidateID + 10, // Different from node IDs
					To:       j + 1,
					Term:     1,           // Same term as nodes
					LogTerm:  1,           // Same or higher term
					LogIndex: candidateID, // Varying index
				}
				
				nodes[j].Step(msg)
			}
		}(i)
	}
	
	wg.Wait()
	
	// Verify that all nodes are still in valid states
	for i, node := range nodes {
		if !node.IsValidState() {
			t.Errorf("Node %d is in invalid state after concurrent vote comparisons", i+1)
		}
	}
}