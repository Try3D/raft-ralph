package raft

import (
	"sync"
	"testing"
)

func TestCandidateCountsVotes(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	node.StartElection() // Becomes candidate

	if node.State != Candidate {
		t.Fatalf("Expected node to be Candidate, got %v", node.State)
	}

	// Simulate receiving a vote from another node
	voteResponse := Message{
		Type:        RequestVoteResponseMsg,
		From:        2, // Another node
		Term:        node.CurrentTerm,
		VoteGranted: true,
	}

	node.Step(voteResponse)

	// Check if the vote was counted
	if len(node.votesReceived) != 2 { // Self vote + vote from node 2
		t.Errorf("Expected 2 votes (self + node 2), got %d", len(node.votesReceived))
	}

	if !node.votesReceived[2] {
		t.Errorf("Expected vote from node 2 to be recorded")
	}
}

func TestBecomeLeaderOnMajority(t *testing.T) {
	// Create a node with a 3-node cluster
	node := NewNode(1, &MockStorage{})
	node.ClusterSize = 3
	node.StartElection() // Becomes candidate

	if node.State != Candidate {
		t.Fatalf("Expected node to be Candidate, got %v", node.State)
	}

	// Simulate receiving a vote from another node (majority = 2 in a 3-node cluster)
	voteResponse := Message{
		Type:        RequestVoteResponseMsg,
		From:        2, // Another node
		Term:        node.CurrentTerm,
		VoteGranted: true,
	}

	node.Step(voteResponse)

	// Node should now have 2 votes (self + node 2), which is majority in a 3-node cluster
	if node.State != Leader {
		t.Errorf("Expected node to become Leader with majority votes, got %v", node.State)
	}
}

func TestCannotBecomeLeaderWithoutMajority(t *testing.T) {
	// Create a node with a 5-node cluster
	node := NewNode(1, &MockStorage{})
	node.ClusterSize = 5
	node.StartElection() // Becomes candidate

	if node.State != Candidate {
		t.Fatalf("Expected node to be Candidate, got %v", node.State)
	}

	// Simulate receiving a vote from one node (need 3 for majority in 5-node cluster)
	voteResponse := Message{
		Type:        RequestVoteResponseMsg,
		From:        2, // Another node
		Term:        node.CurrentTerm,
		VoteGranted: true,
	}

	node.Step(voteResponse)

	// Node should still be candidate (has 2 votes, needs 3 for majority)
	if node.State != Candidate {
		t.Errorf("Expected node to remain Candidate without majority, got %v", node.State)
	}

	// Simulate receiving another vote (now has 3 total, which is majority)
	voteResponse2 := Message{
		Type:        RequestVoteResponseMsg,
		From:        3, // Another node
		Term:        node.CurrentTerm,
		VoteGranted: true,
	}

	node.Step(voteResponse2)

	// Node should now be leader (has 3 votes, which is majority)
	if node.State != Leader {
		t.Errorf("Expected node to become Leader with majority votes, got %v", node.State)
	}
}

func TestConcurrentVoteCollection(t *testing.T) {
	// Create a node with a 5-node cluster
	node := NewNode(1, &MockStorage{})
	node.ClusterSize = 5
	node.StartElection() // Becomes candidate

	if node.State != Candidate {
		t.Fatalf("Expected node to be Candidate, got %v", node.State)
	}

	// Simulate concurrent vote responses from multiple nodes
	const numGoroutines = 4 // Will vote from nodes 2-5
	var wg sync.WaitGroup

	for i := 2; i < numGoroutines+2; i++ {
		wg.Add(1)
		go func(nodeID int) {
			defer wg.Done()

			voteResponse := Message{
				Type:        RequestVoteResponseMsg,
				From:        nodeID,
				Term:        node.CurrentTerm,
				VoteGranted: true,
			}

			node.Step(voteResponse)
		}(i)
	}

	wg.Wait()

	// Check if the node became leader (with 5 total votes, it should have majority)
	if node.State != Leader {
		t.Errorf("Expected node to become Leader with majority votes, got %v", node.State)
	}

	// Verify all votes were recorded
	if len(node.votesReceived) != 5 { // Self vote + 4 others
		t.Errorf("Expected 5 votes, got %d", len(node.votesReceived))
	}
}

func TestVoteCountingOnlyInCandidateState(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	node.StartElection() // Becomes candidate

	if node.State != Candidate {
		t.Fatalf("Expected node to be Candidate, got %v", node.State)
	}

	// Simulate receiving a vote from another node
	voteResponse := Message{
		Type:        RequestVoteResponseMsg,
		From:        2, // Another node
		Term:        node.CurrentTerm,
		VoteGranted: true,
	}

	node.Step(voteResponse)

	// Check if the vote was counted
	if len(node.votesReceived) != 2 { // Self vote + vote from node 2
		t.Errorf("Expected 2 votes (self + node 2), got %d", len(node.votesReceived))
	}

	// Now simulate a higher-term message that forces the node to become follower
	higherTermMsg := Message{
		Type: RequestVoteMsg,
		From: 3,
		Term: node.CurrentTerm + 1,
	}

	node.Step(higherTermMsg)

	if node.State != Follower {
		t.Errorf("Expected node to become Follower after higher term message, got %v", node.State)
	}

	// Try to receive another vote response - it should not count since node is no longer candidate
	voteResponse2 := Message{
		Type:        RequestVoteResponseMsg,
		From:        4,
		Term:        node.CurrentTerm,
		VoteGranted: true,
	}

	node.Step(voteResponse2)

	// The vote count should remain the same since node is no longer candidate
	if len(node.votesReceived) != 2 { // Should still be 2, not 3
		t.Errorf("Expected vote count to remain 2 (since not candidate anymore), got %d", len(node.votesReceived))
	}
}

func TestVoteCountingIgnoresOldTerms(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	node.StartElection() // Becomes candidate

	if node.State != Candidate {
		t.Fatalf("Expected node to be Candidate, got %v", node.State)
	}

	originalTerm := node.CurrentTerm

	// Simulate receiving a vote from another node
	voteResponse := Message{
		Type:        RequestVoteResponseMsg,
		From:        2, // Another node
		Term:        originalTerm,
		VoteGranted: true,
	}

	node.Step(voteResponse)

	// Check if the vote was counted
	if len(node.votesReceived) != 2 { // Self vote + vote from node 2
		t.Errorf("Expected 2 votes (self + node 2), got %d", len(node.votesReceived))
	}

	// Simulate receiving a vote response from an old term (should be ignored)
	oldTermVoteResponse := Message{
		Type:        RequestVoteResponseMsg,
		From:        3, // Another node
		Term:        originalTerm - 1, // Old term
		VoteGranted: true,
	}

	node.Step(oldTermVoteResponse)

	// The vote count should remain the same since the vote was from an old term
	if len(node.votesReceived) != 2 { // Should still be 2, not 3
		t.Errorf("Expected vote count to remain 2 (old term vote ignored), got %d", len(node.votesReceived))
	}
}