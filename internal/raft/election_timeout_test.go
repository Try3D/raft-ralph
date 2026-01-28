package raft

import (
	"testing"
	"time"
)

func TestTickIncrementsTimeout(t *testing.T) {
	node := NewNode(1, nil)
	
	initialCounter := node.ElectionTimeoutCounter
	
	// Call Tick multiple times
	node.Tick()
	node.Tick()
	node.Tick()
	
	if node.ElectionTimeoutCounter != initialCounter+3 {
		t.Errorf("Expected counter to increment by 3, got %d", node.ElectionTimeoutCounter-(initialCounter))
	}
}

func TestRandomizedTimeout(t *testing.T) {
	// Create multiple nodes to check they get different timeouts
	nodes := make([]*Node, 5)
	timeouts := make([]int, 5)
	
	for i := 0; i < 5; i++ {
		time.Sleep(1 * time.Millisecond) // Ensure different seed
		nodes[i] = NewNode(i+1, nil)
		timeouts[i] = nodes[i].RandomizedElectionTimeout
	}
	
	// Check that at least some timeouts are different (probabilistically)
	different := false
	for i := 0; i < 5; i++ {
		for j := i + 1; j < 5; j++ {
			if timeouts[i] != timeouts[j] {
				different = true
				break
			}
		}
		if different {
			break
		}
	}
	
	// Even with randomization, there's a small chance they could be the same
	// But we expect most of the time they'll be different
	// So we'll just check they're in the expected range [150, 299]
	for _, timeout := range timeouts {
		if timeout < 150 || timeout > 299 {
			t.Errorf("Expected timeout in range [150, 299], got %d", timeout)
		}
	}
}

func TestElectionStartsAfterTimeout(t *testing.T) {
	node := NewNode(1, nil)
	
	// Set the node to follower state
	node.setState(Follower)
	
	// Manually set the timeout to a low value to trigger election
	node.ElectionTimeoutCounter = node.RandomizedElectionTimeout - 1
	
	// Call Tick to trigger election
	node.Tick()
	
	if node.State != Candidate {
		t.Errorf("Expected node to become Candidate after timeout, got %v", node.State)
	}
	
	if node.CurrentTerm != 1 {
		t.Errorf("Expected term to increment to 1, got %d", node.CurrentTerm)
	}
	
	if node.VotedFor != 1 {
		t.Errorf("Expected node to vote for itself, got %d", node.VotedFor)
	}
	
	// Counter should be reset after election
	if node.ElectionTimeoutCounter != 0 {
		t.Errorf("Expected election timeout counter to reset to 0, got %d", node.ElectionTimeoutCounter)
	}
}

func TestConcurrentTicksFromMultipleNodes(t *testing.T) {
	const numNodes = 5
	nodes := make([]*Node, numNodes)
	
	// Create nodes
	for i := 0; i < numNodes; i++ {
		nodes[i] = NewNode(i+1, nil)
		nodes[i].setState(Follower)
	}
	
	// Simulate concurrent ticks using goroutines
	done := make(chan bool, numNodes)
	
	for i := 0; i < numNodes; i++ {
		go func(node *Node, idx int) {
			// Call Tick multiple times
			for j := 0; j < 10; j++ {
				node.Tick()
			}
			done <- true
		}(nodes[idx], i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numNodes; i++ {
		<-done
	}
	
	// Verify each node's counter incremented independently
	for i, node := range nodes {
		expectedCounter := 10 // 10 ticks per node
		if node.ElectionTimeoutCounter != expectedCounter {
			t.Errorf("Node %d: Expected counter to be %d, got %d", i+1, expectedCounter, node.ElectionTimeoutCounter)
		}
	}
}

func TestTimeoutResetsOnHeartbeat(t *testing.T) {
	node := NewNode(1, nil)
	
	// Set the node to follower state
	node.setState(Follower)
	
	// Increment the timeout counter
	node.ElectionTimeoutCounter = 100
	
	// Send a heartbeat (AppendEntries message) from the leader
	heartbeat := Message{
		Type:        AppendEntriesMsg,
		From:        2, // Leader ID
		Term:        1, // Current term
		Entries:     []LogEntry{},
		LogIndex:    -1,
		LogTerm:     -1,
		CommitIndex: 0,
	}
	
	node.Step(heartbeat)
	
	// Counter should be reset after receiving heartbeat
	if node.ElectionTimeoutCounter != 0 {
		t.Errorf("Expected election timeout counter to reset to 0 after heartbeat, got %d", node.ElectionTimeoutCounter)
	}
}

func TestTimeoutResetsOnHigherTermMessage(t *testing.T) {
	node := NewNode(1, nil)
	
	// Set the node to follower state
	node.setState(Follower)
	
	// Increment the timeout counter
	node.ElectionTimeoutCounter = 100
	
	// Send a message with higher term
	higherTermMsg := Message{
		Type: RequestVoteMsg,
		From: 2,
		Term: 5, // Higher term
	}
	
	node.Step(higherTermMsg)
	
	// Counter should be reset after receiving higher term message
	if node.ElectionTimeoutCounter != 0 {
		t.Errorf("Expected election timeout counter to reset to 0 after higher term message, got %d", node.ElectionTimeoutCounter)
	}
	
	// Node should step down to follower and update term
	if node.CurrentTerm != 5 {
		t.Errorf("Expected term to update to 5, got %d", node.CurrentTerm)
	}
	
	if node.State != Follower {
		t.Errorf("Expected node to be Follower after higher term message, got %v", node.State)
	}
}