package raft

import (
	"sync"
	"testing"
)

// TestLogAppendIncrementsIndex tests that appending entries increments the log index
func TestLogAppendIncrementsIndex(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	
	// Initially, the log should be empty
	index, _ := node.GetLastLogIndexAndTerm()
	if index != -1 {
		t.Errorf("Expected empty log index to be -1, got %d", index)
	}

	// Start an election to set the current term
	node.StartElection()
	currentTerm := node.CurrentTerm

	// Append an entry
	entry1 := LogEntry{Command: "command1", Term: currentTerm}
	success := node.AppendEntry(entry1)
	if !success {
		t.Error("Expected AppendEntry to succeed when term matches")
	}

	// Check that the index is now 0
	index, term := node.GetLastLogIndexAndTerm()
	if index != 0 {
		t.Errorf("Expected log index to be 0 after first append, got %d", index)
	}
	if term != currentTerm {
		t.Errorf("Expected log term to be %d, got %d", currentTerm, term)
	}

	// Append another entry
	entry2 := LogEntry{Command: "command2", Term: currentTerm}
	success = node.AppendEntry(entry2)
	if !success {
		t.Error("Expected AppendEntry to succeed when term matches")
	}

	// Check that the index is now 1
	index, term = node.GetLastLogIndexAndTerm()
	if index != 1 {
		t.Errorf("Expected log index to be 1 after second append, got %d", index)
	}
	if term != currentTerm {
		t.Errorf("Expected log term to be %d, got %d", currentTerm, term)
	}
}

// TestLogTermNeverChanges tests that entry term is immutable after append
func TestLogTermNeverChanges(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	
	// Start an election to set the current term
	node.StartElection()
	currentTerm := node.CurrentTerm

	// Append an entry
	entry := LogEntry{Command: "command1", Term: currentTerm}
	success := node.AppendEntry(entry)
	if !success {
		t.Error("Expected AppendEntry to succeed when term matches")
	}

	// Verify the term in the log hasn't changed
	_, term := node.GetLastLogIndexAndTerm()
	if term != currentTerm {
		t.Errorf("Expected log term to remain %d, got %d", currentTerm, term)
	}

	// Try to append an entry with a different term - should fail
	differentTermEntry := LogEntry{Command: "command2", Term: currentTerm + 1}
	success = node.AppendEntry(differentTermEntry)
	if success {
		t.Error("Expected AppendEntry to fail when term doesn't match current term")
	}

	// The last entry should still have the original term
	_, term = node.GetLastLogIndexAndTerm()
	if term != currentTerm {
		t.Errorf("Expected log term to remain %d after failed append, got %d", currentTerm, term)
	}
}

// TestLogCannotSkipIndices tests that log indices are contiguous
func TestLogCannotSkipIndices(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	
	// Start an election to set the current term
	node.StartElection()
	currentTerm := node.CurrentTerm

	// Append entries one by one - indices should be 0, 1, 2, etc.
	for i := 0; i < 5; i++ {
		entry := LogEntry{Command: "command" + string(rune('0' + i)), Term: currentTerm}
		success := node.AppendEntry(entry)
		if !success {
			t.Errorf("Expected AppendEntry to succeed for entry %d", i)
		}

		// Check that the index is correct
		expectedIndex := i
		actualIndex, _ := node.GetLastLogIndexAndTerm()
		if actualIndex != expectedIndex {
			t.Errorf("Expected log index to be %d after append %d, got %d", expectedIndex, i, actualIndex)
		}
	}
}

// TestConcurrentAppends tests concurrent appending of entries
func TestConcurrentAppends(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	
	// Start an election to set the current term
	node.StartElection()
	currentTerm := node.CurrentTerm

	const numGoroutines = 10
	var wg sync.WaitGroup

	// Channel to collect results
	results := make(chan int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			entry := LogEntry{Command: "command" + string(rune('0' + goroutineID)), Term: currentTerm}
			success := node.AppendEntry(entry)
			
			// Send the result to the channel
			if success {
				results <- 1
			} else {
				results <- 0
			}
		}(i)
	}

	wg.Wait()
	close(results)

	// Count successful appends
	successfulAppends := 0
	for result := range results {
		successfulAppends += result
	}

	// All appends should succeed since they have the correct term
	if successfulAppends != numGoroutines {
		t.Errorf("Expected %d successful appends, got %d", numGoroutines, successfulAppends)
	}

	// Check that the final log length is correct
	finalIndex, _ := node.GetLastLogIndexAndTerm()
	expectedFinalIndex := numGoroutines - 1
	if finalIndex != expectedFinalIndex {
		t.Errorf("Expected final log index to be %d, got %d", expectedFinalIndex, finalIndex)
	}
}

// TestAppendEntryWithWrongTerm tests that AppendEntry fails when term doesn't match
func TestAppendEntryWithWrongTerm(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	
	// Start an election to set the current term
	node.StartElection()
	currentTerm := node.CurrentTerm

	// Try to append an entry with a wrong term
	wrongTermEntry := LogEntry{Command: "wrongCommand", Term: currentTerm + 1}
	success := node.AppendEntry(wrongTermEntry)
	if success {
		t.Error("Expected AppendEntry to fail when term doesn't match current term")
	}

	// The log should still be empty
	index, _ := node.GetLastLogIndexAndTerm()
	if index != -1 {
		t.Errorf("Expected log index to remain -1 after failed append, got %d", index)
	}
}