package raft

import (
	"sync"
	"testing"
)

func TestLogAppendIncrementsIndex(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	index, _ := node.GetLastLogIndexAndTerm()
	if index != -1 {
		t.Errorf("Expected empty log index to be -1, got %d", index)
	}

	node.StartElection()
	currentTerm := node.CurrentTerm

	entry1 := LogEntry{Command: "command1", Term: currentTerm}
	success := node.AppendEntry(entry1)
	if !success {
		t.Error("Expected AppendEntry to succeed when term matches")
	}

	index, term := node.GetLastLogIndexAndTerm()
	if index != 0 {
		t.Errorf("Expected log index to be 0 after first append, got %d", index)
	}
	if term != currentTerm {
		t.Errorf("Expected log term to be %d, got %d", currentTerm, term)
	}

	entry2 := LogEntry{Command: "command2", Term: currentTerm}
	success = node.AppendEntry(entry2)
	if !success {
		t.Error("Expected AppendEntry to succeed when term matches")
	}

	index, term = node.GetLastLogIndexAndTerm()
	if index != 1 {
		t.Errorf("Expected log index to be 1 after second append, got %d", index)
	}
	if term != currentTerm {
		t.Errorf("Expected log term to be %d, got %d", currentTerm, term)
	}
}

func TestLogTermNeverChanges(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	node.StartElection()
	currentTerm := node.CurrentTerm

	entry := LogEntry{Command: "command1", Term: currentTerm}
	success := node.AppendEntry(entry)
	if !success {
		t.Error("Expected AppendEntry to succeed when term matches")
	}

	_, term := node.GetLastLogIndexAndTerm()
	if term != currentTerm {
		t.Errorf("Expected log term to remain %d, got %d", currentTerm, term)
	}

	differentTermEntry := LogEntry{Command: "command2", Term: currentTerm + 1}
	success = node.AppendEntry(differentTermEntry)
	if success {
		t.Error("Expected AppendEntry to fail when term doesn't match current term")
	}

	_, term = node.GetLastLogIndexAndTerm()
	if term != currentTerm {
		t.Errorf("Expected log term to remain %d after failed append, got %d", currentTerm, term)
	}
}

func TestLogCannotSkipIndices(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	node.StartElection()
	currentTerm := node.CurrentTerm

	for i := 0; i < 5; i++ {
		entry := LogEntry{Command: "command" + string(rune('0' + i)), Term: currentTerm}
		success := node.AppendEntry(entry)
		if !success {
			t.Errorf("Expected AppendEntry to succeed for entry %d", i)
		}

		expectedIndex := i
		actualIndex, _ := node.GetLastLogIndexAndTerm()
		if actualIndex != expectedIndex {
			t.Errorf("Expected log index to be %d after append %d, got %d", expectedIndex, i, actualIndex)
		}
	}
}

func TestConcurrentAppends(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	node.StartElection()
	currentTerm := node.CurrentTerm

	const numGoroutines = 10
	var wg sync.WaitGroup

	results := make(chan int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			entry := LogEntry{Command: "command" + string(rune('0' + goroutineID)), Term: currentTerm}
			success := node.AppendEntry(entry)

			if success {
				results <- 1
			} else {
				results <- 0
			}
		}(i)
	}

	wg.Wait()
	close(results)

	successfulAppends := 0
	for result := range results {
		successfulAppends += result
	}

	if successfulAppends != numGoroutines {
		t.Errorf("Expected %d successful appends, got %d", numGoroutines, successfulAppends)
	}

	finalIndex, _ := node.GetLastLogIndexAndTerm()
	expectedFinalIndex := numGoroutines - 1
	if finalIndex != expectedFinalIndex {
		t.Errorf("Expected final log index to be %d, got %d", expectedFinalIndex, finalIndex)
	}
}

func TestAppendEntryWithWrongTerm(t *testing.T) {
	node := NewNode(1, &MockStorage{})

	node.StartElection()
	currentTerm := node.CurrentTerm

	wrongTermEntry := LogEntry{Command: "wrongCommand", Term: currentTerm + 1}
	success := node.AppendEntry(wrongTermEntry)
	if success {
		t.Error("Expected AppendEntry to fail when term doesn't match current term")
	}

	index, _ := node.GetLastLogIndexAndTerm()
	if index != -1 {
		t.Errorf("Expected log index to remain -1 after failed append, got %d", index)
	}
}