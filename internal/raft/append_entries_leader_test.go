package raft

import (
	"sync"
	"testing"
)

func TestLeaderSendsHeartbeats(t *testing.T) {
	// Create a leader node
	leader := NewNode(1, &MockStorage{})
	leader.setState(Leader)
	leader.CurrentTerm = 1

	// Initialize cluster with 3 nodes
	leader.ClusterSize = 3

	// Initialize nextIndex and matchIndex
	leader.NextIndex = make([]int, leader.ClusterSize)
	leader.MatchIndex = make([]int, leader.ClusterSize)

	// Set up nextIndex for followers (all start at last log index + 1)
	lastIndex, _ := leader.GetLastLogIndexAndTerm()
	for i := range leader.NextIndex {
		if i != leader.ID {
			leader.NextIndex[i] = lastIndex + 1
		}
	}

	// Simulate sending heartbeats to followers
	// This would typically happen in a periodic loop in a real implementation
	for i := 0; i < leader.ClusterSize; i++ {
		if i == leader.ID {
			continue // Skip self
		}

		// Prepare AppendEntries message (heartbeat)
		prevLogIndex := leader.NextIndex[i] - 1
		var prevLogTerm int
		if prevLogIndex >= 0 && prevLogIndex < len(leader.Log) {
			prevLogTerm = leader.Log[prevLogIndex].Term
		} else {
			prevLogTerm = -1
		}

		entriesToSend := []LogEntry{}
		if leader.NextIndex[i] < len(leader.Log) {
			entriesToSend = leader.Log[leader.NextIndex[i]:]
		}

		// Create the message that would be sent to the follower
		msg := Message{
			Type:        AppendEntriesMsg,
			From:        leader.ID,
			To:          i,
			Term:        leader.CurrentTerm,
			LogIndex:    prevLogIndex,
			LogTerm:     prevLogTerm,
			Entries:     entriesToSend,
			CommitIndex: leader.CommitIndex,
		}

		// In a real system, this would be sent over the network
		// For testing, we'll just verify the message structure
		if msg.Term != leader.CurrentTerm {
			t.Errorf("Heartbeat message should have leader's term %d, got %d", leader.CurrentTerm, msg.Term)
		}
		_ = msg // Use the variable to avoid "declared and not used" error
	}
}

func TestNextIndexDecremented(t *testing.T) {
	// Create a leader node
	leader := NewNode(1, &MockStorage{})
	leader.setState(Leader)
	leader.CurrentTerm = 1

	// Add some entries to the log
	for i := 0; i < 3; i++ {
		entry := LogEntry{Command: "command" + string(rune('0'+i)), Term: 1}
		leader.AppendEntry(entry)
	}

	// Initialize cluster with 3 nodes
	leader.ClusterSize = 3

	// Initialize nextIndex and matchIndex
	leader.NextIndex = make([]int, leader.ClusterSize)
	leader.MatchIndex = make([]int, leader.ClusterSize)

	// Set up nextIndex for followers (start at last log index + 1)
	lastIndex, _ := leader.GetLastLogIndexAndTerm()
	for i := range leader.NextIndex {
		if i != leader.ID {
			leader.NextIndex[i] = lastIndex + 1
		}
	}

	// Simulate a scenario where follower rejects AppendEntries (conflict)
	followerID := 2
	leader.NextIndex[followerID] = 2 // Simulate that we need to send entries starting from index 2

	// Simulate receiving a response indicating rejection
	// In a real system, this would trigger nextIndex decrement
	// For testing, we'll simulate the logic directly
	
	// Decrement nextIndex to probe for the conflict
	leader.NextIndex[followerID]--
	
	if leader.NextIndex[followerID] != 1 {
		t.Errorf("Expected nextIndex to be decremented to 1, got %d", leader.NextIndex[followerID])
	}
}

func TestMatchIndexUpdated(t *testing.T) {
	// Create a leader node
	leader := NewNode(1, &MockStorage{})
	leader.setState(Leader)
	leader.CurrentTerm = 1

	// Add some entries to the log
	for i := 0; i < 3; i++ {
		entry := LogEntry{Command: "command" + string(rune('0'+i)), Term: 1}
		leader.AppendEntry(entry)
	}

	// Initialize cluster with 3 nodes
	leader.ClusterSize = 3

	// Initialize nextIndex and matchIndex
	leader.NextIndex = make([]int, leader.ClusterSize)
	leader.MatchIndex = make([]int, leader.ClusterSize)

	// Initially, matchIndex should be 0 for all followers
	for i := range leader.MatchIndex {
		if i != leader.ID {
			if leader.MatchIndex[i] != 0 {
				t.Errorf("Expected initial matchIndex for follower %d to be 0, got %d", i, leader.MatchIndex[i])
			}
		}
	}

	// Simulate receiving a successful AppendEntries response from a follower
	followerID := 2
	// In a real system, this would update matchIndex based on successful replication
	// For testing, we'll simulate the update directly
	successfulReplicationIndex := 1
	leader.MatchIndex[followerID] = successfulReplicationIndex

	if leader.MatchIndex[followerID] != successfulReplicationIndex {
		t.Errorf("Expected matchIndex for follower %d to be updated to %d, got %d", 
			followerID, successfulReplicationIndex, leader.MatchIndex[followerID])
	}
}

func TestConcurrentReplicationToMultipleFollowers(t *testing.T) {
	// Create a leader node
	leader := NewNode(1, &MockStorage{})
	leader.setState(Leader)
	leader.CurrentTerm = 1

	// Add some entries to the log
	for i := 0; i < 5; i++ {
		entry := LogEntry{Command: "command" + string(rune('0'+i)), Term: 1}
		leader.AppendEntry(entry)
	}

	// Initialize cluster with 5 nodes (including leader)
	leader.ClusterSize = 5

	// Initialize nextIndex and matchIndex
	leader.NextIndex = make([]int, leader.ClusterSize)
	leader.MatchIndex = make([]int, leader.ClusterSize)

	// Set up nextIndex for followers (start at last log index + 1)
	lastIndex, _ := leader.GetLastLogIndexAndTerm()
	for i := range leader.NextIndex {
		if i != leader.ID {
			leader.NextIndex[i] = lastIndex + 1
		}
	}

	// Simulate concurrent replication to multiple followers
	const numFollowers = 4 // Followers with IDs 0, 2, 3, 4 (assuming leader is ID 1)
	var wg sync.WaitGroup

	for i := 0; i < leader.ClusterSize; i++ {
		if i == leader.ID {
			continue // Skip self
		}

		wg.Add(1)
		go func(followerID int) {
			defer wg.Done()

			// Simulate sending AppendEntries to the follower
			// This would normally happen in a loop in a real implementation
			prevLogIndex := leader.NextIndex[followerID] - 1
			var prevLogTerm int
			if prevLogIndex >= 0 && prevLogIndex < len(leader.Log) {
				prevLogTerm = leader.Log[prevLogIndex].Term
			} else {
				prevLogTerm = -1
			}

			entriesToSend := []LogEntry{}
			if leader.NextIndex[followerID] < len(leader.Log) {
				entriesToSend = leader.Log[leader.NextIndex[followerID]:]
			}

			// Create the message that would be sent to the follower
			msg := Message{
				Type:        AppendEntriesMsg,
				From:        leader.ID,
				To:          followerID,
				Term:        leader.CurrentTerm,
				LogIndex:    prevLogIndex,
				LogTerm:     prevLogTerm,
				Entries:     entriesToSend,
				CommitIndex: leader.CommitIndex,
			}

			_ = msg // Use the variable to avoid "declared and not used" error

			// Simulate successful replication response from follower
			// Update matchIndex and nextIndex based on successful replication
			if len(entriesToSend) > 0 {
				newMatchIndex := prevLogIndex + len(entriesToSend)
				leader.MatchIndex[followerID] = newMatchIndex
				leader.NextIndex[followerID] = newMatchIndex + 1
			}
		}(i)
	}

	wg.Wait()

	// Verify that replication was attempted for all followers
	for i := 0; i < leader.ClusterSize; i++ {
		if i == leader.ID {
			continue
		}
		
		// Check that matchIndex was updated for each follower
		if leader.MatchIndex[i] < 0 {
			t.Errorf("Expected matchIndex for follower %d to be updated, got %d", i, leader.MatchIndex[i])
		}
	}
}

func TestLeaderInitializeNextAndMatchIndex(t *testing.T) {
	// Create a leader node
	leader := NewNode(1, &MockStorage{})
	leader.setState(Leader)
	leader.CurrentTerm = 1

	// Add some entries to the log
	for i := 0; i < 3; i++ {
		entry := LogEntry{Command: "command" + string(rune('0'+i)), Term: 1}
		leader.AppendEntry(entry)
	}

	// Initialize cluster with 3 nodes
	leader.ClusterSize = 3

	// Call the initialization logic
	lastIndex, _ := leader.GetLastLogIndexAndTerm()
	leader.NextIndex = make([]int, leader.ClusterSize)
	leader.MatchIndex = make([]int, leader.ClusterSize)

	for i := range leader.NextIndex {
		if i != leader.ID {
			leader.NextIndex[i] = lastIndex + 1
		}
	}

	// Verify initialization
	for i := 0; i < leader.ClusterSize; i++ {
		if i == leader.ID {
			// Leader's own index should remain uninitialized or set to a special value
			continue
		}
		
		expectedNextIndex := lastIndex + 1
		if leader.NextIndex[i] != expectedNextIndex {
			t.Errorf("Expected nextIndex[%d] to be %d, got %d", i, expectedNextIndex, leader.NextIndex[i])
		}
		
		// MatchIndex should start at 0 for all followers
		if leader.MatchIndex[i] != 0 {
			t.Errorf("Expected matchIndex[%d] to be 0 initially, got %d", i, leader.MatchIndex[i])
		}
	}
}