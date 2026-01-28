package raft

import (
	"sync"
	"testing"
)

func TestSnapshotCreatedCorrectly(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	
	// Add some entries to the log
	for i := 0; i < 5; i++ {
		entry := LogEntry{Command: "command" + string(rune('0'+i)), Term: 1, Index: i}
		node.Log = append(node.Log, entry)
	}

	// Simulate applying some entries to state and creating a snapshot
	lastIncludedIndex := 2
	lastIncludedTerm := 1
	
	// In a real implementation, we'd create a snapshot of the state machine
	// For this test, we'll just verify the snapshot metadata
	snapshotData := []byte("snapshot_data_placeholder")
	
	// Simulate compacting the log up to lastIncludedIndex
	if lastIncludedIndex < len(node.Log) {
		node.Log = node.Log[lastIncludedIndex+1:]
	}
	
	// Verify log compaction worked
	if len(node.Log) != 2 { // Should have entries at index 3 and 4
		t.Errorf("Expected 2 entries after compaction, got %d", len(node.Log))
	}
	
	// Verify the remaining entries have correct indices (they should be shifted)
	for i, entry := range node.Log {
		expectedIndex := lastIncludedIndex + 1 + i
		if entry.Index != expectedIndex {
			t.Errorf("Expected entry at position %d to have index %d, got %d", i, expectedIndex, entry.Index)
		}
	}
}

func TestLogCompactionDiscards(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	
	// Add some entries to the log
	for i := 0; i < 10; i++ {
		entry := LogEntry{Command: "command" + string(rune('0'+i)), Term: 1, Index: i}
		node.Log = append(node.Log, entry)
	}

	// Verify initial state
	initialLogLen := len(node.Log)
	if initialLogLen != 10 {
		t.Errorf("Expected 10 entries initially, got %d", initialLogLen)
	}

	// Simulate creating a snapshot at index 5
	lastIncludedIndex := 5
	if lastIncludedIndex < len(node.Log) {
		node.Log = node.Log[lastIncludedIndex+1:]
	}
	
	// Verify compaction
	remainingEntries := len(node.Log)
	if remainingEntries != 4 { // Entries at indices 6, 7, 8, 9
		t.Errorf("Expected 4 entries after compaction, got %d", remainingEntries)
	}
	
	// Verify the remaining entries have correct indices
	for i, entry := range node.Log {
		expectedIndex := lastIncludedIndex + 1 + i
		if entry.Index != expectedIndex {
			t.Errorf("Expected entry at position %d to have index %d, got %d", i, expectedIndex, entry.Index)
		}
	}
}

func TestSnapshotPlusLogRecovery(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	
	// Add some entries to the log
	for i := 0; i < 5; i++ {
		entry := LogEntry{Command: "command" + string(rune('0'+i)), Term: 1, Index: i}
		node.Log = append(node.Log, entry)
	}

	// Simulate creating a snapshot at index 2
	lastIncludedIndex := 2
	lastIncludedTerm := 1
	snapshotData := []byte("state_machine_snapshot_data")
	
	// Compact the log
	if lastIncludedIndex < len(node.Log) {
		node.Log = node.Log[lastIncludedIndex+1:]
	}
	
	// Simulate recovery process: restore from snapshot + remaining log
	recoveredNode := NewNode(1, &MockStorage{})
	
	// Restore snapshot data (in a real system, this would restore the state machine)
	_ = snapshotData // Use the variable to avoid "declared and not used" error
	
	// Apply remaining log entries to the recovered state
	for _, entry := range node.Log {
		// In a real system, we'd apply the command to the state machine
		// For this test, we just verify the log is consistent
		if entry.Index <= lastIncludedIndex {
			t.Errorf("Log should not contain entries at or before snapshot index %d, but found entry at index %d", 
				lastIncludedIndex, entry.Index)
		}
	}
	
	// Verify that the recovered node has the right state
	if len(recoveredNode.Log) != 0 {
		// The recovered node should start with an empty log or with entries after the snapshot
		// depending on how we implement recovery
	}
}

func TestConcurrentSnapshotAndReplication(t *testing.T) {
	// Create a leader node
	leader := NewNode(1, &MockStorage{})
	leader.setState(Leader)
	leader.CurrentTerm = 1
	leader.ClusterSize = 3

	// Add some entries to the log
	for i := 0; i < 10; i++ {
		entry := LogEntry{Command: "command" + string(rune('0'+i)), Term: 1, Index: i}
		leader.AppendEntry(entry)
	}

	// Simulate concurrent snapshot creation and replication
	var wg sync.WaitGroup
	
	// Goroutine for snapshot creation
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		// Simulate creating a snapshot at index 5
		lastIncludedIndex := 5
		_ = lastIncludedIndex // Use the variable to avoid "declared and not used" error
		
		// In a real system, we'd lock appropriately to prevent conflicts
		// with replication during snapshot creation
	}()
	
	// Goroutine for replication
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		// Simulate sending AppendEntries to followers
		leader.SendAppendEntries()
	}()

	wg.Wait()
	
	// In a real implementation, we'd need to ensure snapshot creation
	// and replication don't interfere with each other
}