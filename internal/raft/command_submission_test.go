package raft

import (
	"sync"
	"testing"
	"time"
)

func TestLeaderAcceptsCommands(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	node.setState(Leader)
	node.CurrentTerm = 1

	command := "set x 42"
	result, err := node.SubmitCommand(command)
	if err != nil {
		t.Errorf("Leader should accept commands, got error: %v", err)
	}

	if !result {
		t.Error("Expected command submission to return true for leader")
	}
}

func TestFollowerRejectsCommands(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	node.setState(Follower)
	node.CurrentTerm = 1

	command := "set x 42"
	result, err := node.SubmitCommand(command)
	if err == nil {
		t.Error("Follower should reject commands with an error")
	}

	if result {
		t.Error("Expected command submission to return false for follower")
	}
}

func TestCommandEventuallyCommitted(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	node.setState(Leader)
	node.CurrentTerm = 1
	node.ClusterSize = 3

	command := "set x 42"
	submitted, err := node.SubmitCommand(command)
	if err != nil || !submitted {
		t.Fatalf("Failed to submit command: %v", err)
	}

	// Simulate that the command gets committed by advancing the commit index
	lastIndex, _ := node.GetLastLogIndexAndTerm()
	if lastIndex >= 0 {
		node.CommitIndex = lastIndex
	}

	// Wait a bit to allow for processing
	time.Sleep(10 * time.Millisecond)

	// Verify the command is in the log and committed
	if len(node.Log) == 0 {
		t.Error("Expected command to be in the log")
	} else if node.Log[0].Command != command {
		t.Errorf("Expected command '%s', got '%v'", command, node.Log[0].Command)
	}
}

func TestConcurrentClientSubmissions(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	node.setState(Leader)
	node.CurrentTerm = 1
	node.ClusterSize = 3

	const numClients = 20
	var wg sync.WaitGroup
	results := make(chan bool, numClients)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			command := "set key" + string(rune('0'+clientID%10)) + " " + string(rune('0'+clientID))
			result, err := node.SubmitCommand(command)
			
			if err != nil {
				results <- false
			} else {
				results <- result
			}
		}(i)
	}

	wg.Wait()
	close(results)

	successfulSubmissions := 0
	for result := range results {
		if result {
			successfulSubmissions++
		}
	}

	if successfulSubmissions != numClients {
		t.Errorf("Expected %d successful submissions, got %d", numClients, successfulSubmissions)
	}

	// Verify that all commands were added to the log
	lastIndex, _ := node.GetLastLogIndexAndTerm()
	if lastIndex+1 < numClients {
		t.Errorf("Expected at least %d entries in log, got %d", numClients, lastIndex+1)
	}
}

func TestNonLeaderRejectsCommands(t *testing.T) {
	// Test candidate state
	candidateNode := NewNode(1, &MockStorage{})
	candidateNode.setState(Candidate)
	candidateNode.CurrentTerm = 1

	command := "set x 42"
	result, err := candidateNode.SubmitCommand(command)
	if err == nil {
		t.Error("Candidate should reject commands with an error")
	}

	if result {
		t.Error("Expected command submission to return false for candidate")
	}
}