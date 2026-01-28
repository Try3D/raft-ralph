package raft

import (
	"sync"
	"testing"
)

func TestLogMatchingProperty(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	node.CurrentTerm = 1

	// Add an entry to the log
	entry1 := LogEntry{Command: "command1", Term: 1}
	node.AppendEntry(entry1)

	// Send an AppendEntries message with matching previous log info
	msg := Message{
		Type:     AppendEntriesMsg,
		From:     2,
		To:       1,
		Term:     1,
		LogTerm:  1,      // Term of the last entry in the leader's log
		LogIndex: 0,      // Index of the last entry in the leader's log
		Entries:  []LogEntry{{Command: "command2", Term: 1}},
	}

	node.Step(msg)

	// The entry should be appended successfully
	if len(node.Log) != 2 {
		t.Errorf("Expected log length to be 2, got %d", len(node.Log))
	}

	if node.Log[1].Command != "command2" {
		t.Errorf("Expected second entry to be 'command2', got %v", node.Log[1].Command)
	}
}

func TestConflictingEntriesRejected(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	node.CurrentTerm = 1

	// Add an entry to the log
	entry1 := LogEntry{Command: "command1", Term: 1}
	node.AppendEntry(entry1)

	// Send an AppendEntries message with mismatching previous log info
	msg := Message{
		Type:     AppendEntriesMsg,
		From:     2,
		To:       1,
		Term:     1,
		LogTerm:  2,      // Different term than what's in the follower's log
		LogIndex: 0,      // Index of the last entry in the leader's log
		Entries:  []LogEntry{{Command: "command2", Term: 1}},
	}

	node.Step(msg)

	// The entry should be rejected because of the mismatch
	// The response would indicate failure, but we're checking that the log didn't change
	if len(node.Log) != 1 {
		t.Errorf("Expected log length to remain 1, got %d", len(node.Log))
	}

	if node.Log[0].Command != "command1" {
		t.Errorf("Expected first entry to remain 'command1', got %v", node.Log[0].Command)
	}
}

func TestEmptyAppendEntriesAsHeartbeat(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	node.CurrentTerm = 1

	// Add an entry to the log
	entry1 := LogEntry{Command: "command1", Term: 1}
	node.AppendEntry(entry1)

	// Send an empty AppendEntries message (heartbeat)
	msg := Message{
		Type:     AppendEntriesMsg,
		From:     2,
		To:       1,
		Term:     1,
		LogTerm:  1,
		LogIndex: 0,
		Entries:  []LogEntry{}, // Empty entries - heartbeat
	}

	node.Step(msg)

	// The log should remain unchanged
	if len(node.Log) != 1 {
		t.Errorf("Expected log length to remain 1, got %d", len(node.Log))
	}

	if node.Log[0].Command != "command1" {
		t.Errorf("Expected first entry to remain 'command1', got %v", node.Log[0].Command)
	}
}

func TestConcurrentLogReplication(t *testing.T) {
	// Create a leader node
	leader := NewNode(1, &MockStorage{})
	leader.setState(Leader)
	leader.CurrentTerm = 1

	// Create 3 follower nodes
	follower1 := NewNode(2, &MockStorage{})
	follower1.CurrentTerm = 1
	
	follower2 := NewNode(3, &MockStorage{})
	follower2.CurrentTerm = 1
	
	follower3 := NewNode(4, &MockStorage{})
	follower3.CurrentTerm = 1

	// Add some initial entries to the leader
	leaderEntry1 := LogEntry{Command: "leader-command1", Term: 1}
	leader.AppendEntry(leaderEntry1)
	
	leaderEntry2 := LogEntry{Command: "leader-command2", Term: 1}
	leader.AppendEntry(leaderEntry2)

	// Simulate concurrent log replication to multiple followers
	const numGoroutines = 3
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(followerID int) {
			defer wg.Done()

			var follower *Node
			switch followerID {
			case 2:
				follower = follower1
			case 3:
				follower = follower2
			case 4:
				follower = follower3
			default:
				return
			}

			// Send AppendEntries to the follower
			msg := Message{
				Type:        AppendEntriesMsg,
				From:        1, // Leader ID
				To:          followerID,
				Term:        1,
				LogTerm:     1,
				LogIndex:    1, // Index of last entry in leader's log before new entries
				Entries:     []LogEntry{{Command: "replicated-command", Term: 1}},
				CommitIndex: 1,
			}

			follower.Step(msg)
		}(i + 2) // Start from follower ID 2
	}

	wg.Wait()

	// Verify that all followers have received the replicated entry
	followers := []*Node{follower1, follower2, follower3}
	for i, follower := range followers {
		if len(follower.Log) != 2 { // Original entry + replicated entry
			t.Errorf("Follower %d: Expected log length to be 2, got %d", i+1, len(follower.Log))
		}
		
		if follower.Log[1].Command != "replicated-command" {
			t.Errorf("Follower %d: Expected second entry to be 'replicated-command', got %v", 
				i+1, follower.Log[1].Command)
		}
	}
}

func TestAppendEntriesWithHigherTerm(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	node.CurrentTerm = 1

	// Add an entry to the log
	entry1 := LogEntry{Command: "command1", Term: 1}
	node.AppendEntry(entry1)

	// Send an AppendEntries message with a higher term
	msg := Message{
		Type:     AppendEntriesMsg,
		From:     2,
		To:       1,
		Term:     2,      // Higher term
		LogTerm:  1,
		LogIndex: 0,
		Entries:  []LogEntry{{Command: "command2", Term: 2}},
	}

	node.Step(msg)

	// The node should step down to follower and update its term
	if node.State != Follower {
		t.Errorf("Expected node to be Follower after receiving higher term, got %v", node.State)
	}

	if node.CurrentTerm != 2 {
		t.Errorf("Expected current term to be 2, got %d", node.CurrentTerm)
	}

	// The entry should be appended
	if len(node.Log) != 2 {
		t.Errorf("Expected log length to be 2, got %d", len(node.Log))
	}

	if node.Log[1].Command != "command2" {
		t.Errorf("Expected second entry to be 'command2', got %v", node.Log[1].Command)
	}
}

func TestAppendEntriesWithLowerTerm(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	node.CurrentTerm = 3

	// Add an entry to the log
	entry1 := LogEntry{Command: "command1", Term: 3}
	node.AppendEntry(entry1)

	// Send an AppendEntries message with a lower term
	msg := Message{
		Type:     AppendEntriesMsg,
		From:     2,
		To:       1,
		Term:     1,      // Lower term
		LogTerm:  1,
		LogIndex: 0,
		Entries:  []LogEntry{{Command: "command2", Term: 1}},
	}

	node.Step(msg)

	// The append should be rejected because of the lower term
	// The response would indicate failure, but we're checking that the log didn't change
	if len(node.Log) != 1 {
		t.Errorf("Expected log length to remain 1, got %d", len(node.Log))
	}

	if node.Log[0].Command != "command1" {
		t.Errorf("Expected first entry to remain 'command1', got %v", node.Log[0].Command)
	}
}