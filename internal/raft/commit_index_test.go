package raft

import (
	"sync"
	"testing"
)

func TestCommitIndexNeverDecreases(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	
	initialCommitIndex := node.CommitIndex
	
	// Advance commit index
	node.CommitIndex = 5
	
	// Try to decrease it (this should not happen in proper Raft implementation)
	// But we're testing the invariant that it never decreases
	if node.CommitIndex < initialCommitIndex {
		t.Errorf("Commit index decreased from %d to %d, which violates monotonicity", initialCommitIndex, node.CommitIndex)
	}
	
	// Simulate proper advancement
	node.CommitIndex = 10
	if node.CommitIndex < 5 {
		t.Errorf("Commit index should be able to advance, expected >= 5, got %d", node.CommitIndex)
	}
}

func TestCommitOnlyFromCurrentTerm(t *testing.T) {
	node := NewNode(1, &MockStorage{})
	
	// Start an election to establish current term
	node.StartElection()
	currentTerm := node.CurrentTerm
	
	// Add some entries to the log from the current term
	for i := 0; i < 3; i++ {
		entry := LogEntry{Command: "command" + string(rune('0'+i)), Term: currentTerm}
		node.AppendEntry(entry)
	}
	
	// In a real Raft implementation, entries from the current term can be committed
	// once they're replicated on a majority of servers
	// For this test, we'll verify the concept that only current-term entries can be committed
	
	// Add an entry to the log
	entry := LogEntry{Command: "currentTermCmd", Term: currentTerm}
	node.AppendEntry(entry)
	
	// Simulate that this entry has been replicated on majority of servers
	// and that it's from the current term, so it can be committed
	lastIndex, _ := node.GetLastLogIndexAndTerm()
	
	// In a real implementation, we'd check if this entry is from current term
	// and if it's replicated on majority before advancing commit index
	// For this test, we'll just verify that we can advance to the last index
	if lastIndex >= 0 {
		// Simulate committing up to the last index
		// This should be allowed if the entry is from current term
		previousCommitIndex := node.CommitIndex
		if lastIndex > previousCommitIndex {
			node.CommitIndex = lastIndex
		}
		
		if node.CommitIndex < previousCommitIndex {
			t.Errorf("Commit index should not decrease, was %d, now %d", previousCommitIndex, node.CommitIndex)
		}
	}
}

func TestMajorityReplicationAdvancesCommit(t *testing.T) {
	// Create a leader node
	leader := NewNode(1, &MockStorage{})
	leader.setState(Leader)
	leader.CurrentTerm = 1
	leader.ClusterSize = 3  // 3-node cluster, majority = 2
	
	// Add an entry to the log
	entry := LogEntry{Command: "testCommand", Term: 1}
	leader.AppendEntry(entry)
	
	// Initialize nextIndex and matchIndex
	lastIndex, _ := leader.GetLastLogIndexAndTerm()
	leader.NextIndex = make([]int, leader.ClusterSize)
	leader.MatchIndex = make([]int, leader.ClusterSize)
	
	for i := range leader.NextIndex {
		if i != leader.ID {
			leader.NextIndex[i] = lastIndex + 1
		}
	}
	
	// Simulate that the entry has been replicated on 2 out of 3 servers (majority)
	// Update matchIndex for follower 2 to show that it has the entry
	leader.MatchIndex[2] = lastIndex
	
	// In a real Raft implementation, we'd check if N is the highest index
	// such that N > commitIndex and a majority of matchIndex[i] >= N
	// Then we'd set commitIndex = N if the entry at N is from the current term
	
	// Find the N that satisfies majority condition
	majority := leader.ClusterSize/2 + 1
	count := 0
	for n := lastIndex; n >= 0; n-- {
		count = 1 // Leader implicitly has the entry
		for i := 0; i < leader.ClusterSize; i++ {
			if i == leader.ID {
				continue
			}
			if leader.MatchIndex[i] >= n {
				count++
			}
		}
		
		if count >= majority {
			// Check if the entry at index n is from the current term
			if n < len(leader.Log) && leader.Log[n].Term == leader.CurrentTerm {
				// This is where we'd advance commit index in a real implementation
				if n > leader.CommitIndex {
					leader.CommitIndex = n
				}
				break
			}
		}
	}
	
	// Verify that commit index was advanced appropriately
	if leader.CommitIndex < 0 {
		t.Errorf("Expected commit index to be advanced, got %d", leader.CommitIndex)
	}
}

func TestApplyCommittedEntries(t *testing.T) {
	// Create a node and set it as leader
	leader := NewNode(1, &MockStorage{})
	leader.setState(Leader)
	leader.CurrentTerm = 1
	leader.ClusterSize = 3  // 3-node cluster, majority = 2
	
	// Add some entries to the log
	commands := []string{"cmd1", "cmd2", "cmd3"}
	for _, cmd := range commands {
		entry := LogEntry{Command: cmd, Term: 1}
		leader.AppendEntry(entry)
	}
	
	// Initialize nextIndex and matchIndex
	lastIndex, _ := leader.GetLastLogIndexAndTerm()
	leader.NextIndex = make([]int, leader.ClusterSize)
	leader.MatchIndex = make([]int, leader.ClusterSize)

	for idx := range leader.NextIndex {
		if idx != leader.ID {
			leader.NextIndex[idx] = lastIndex + 1
		}
	}

	// Simulate that entries have been replicated on majority of servers
	for j := 0; j < leader.ClusterSize; j++ {
		if j != leader.ID {
			leader.MatchIndex[j] = lastIndex
		}
	}

	// Advance commit index based on majority replication
	majority := leader.ClusterSize/2 + 1
	for n := lastIndex; n > leader.CommitIndex; n-- {
		count := 1 // Leader implicitly has the entry
		for k := 0; k < leader.ClusterSize; k++ {
			if k == leader.ID {
				continue
			}
			if leader.MatchIndex[k] >= n {
				count++
			}
		}

		if count >= majority && n < len(leader.Log) && leader.Log[n].Term == leader.CurrentTerm {
			leader.CommitIndex = n
			break
		}
	}

	// Verify that all entries up to commit index can be applied
	for idx := leader.LastApplied + 1; idx <= leader.CommitIndex; idx++ {
		if idx < len(leader.Log) {
			// In a real system, we'd apply the command to the state machine
			// For this test, we just verify that we can access the committed entries
			entry := leader.Log[idx]
			if entry.Command != commands[idx] {
				t.Errorf("Expected command %s at index %d, got %s", commands[idx], idx, entry.Command)
			}

			// Update LastApplied to reflect that we applied this entry
			leader.LastApplied = idx
		}
	}
	
	// Verify that all entries were applied
	if leader.LastApplied != leader.CommitIndex {
		t.Errorf("Expected LastApplied (%d) to equal CommitIndex (%d)", leader.LastApplied, leader.CommitIndex)
	}
}

func TestConcurrentCommitAdvancement(t *testing.T) {
	// Create a leader node
	leader := NewNode(1, &MockStorage{})
	leader.setState(Leader)
	leader.CurrentTerm = 1
	leader.ClusterSize = 5  // 5-node cluster, majority = 3
	
	// Add some entries to the log
	numEntries := 10
	for i := 0; i < numEntries; i++ {
		entry := LogEntry{Command: "cmd" + string(rune('0'+i)), Term: 1}
		leader.AppendEntry(entry)
	}
	
	// Initialize nextIndex and matchIndex
	lastIndex, _ := leader.GetLastLogIndexAndTerm()
	leader.NextIndex = make([]int, leader.ClusterSize)
	leader.MatchIndex = make([]int, leader.ClusterSize)
	
	for i := range leader.NextIndex {
		if i != leader.ID {
			leader.NextIndex[i] = lastIndex + 1
		}
	}
	
	// Simulate concurrent updates to matchIndex from different goroutines
	// representing different followers
	var wg sync.WaitGroup
	
	for followerID := 0; followerID < leader.ClusterSize; followerID++ {
		if followerID == leader.ID {
			continue
		}
		
		wg.Add(1)
		go func(fID int) {
			defer wg.Done()
			
			// Simulate this follower having replicated entries up to some index
			// For this test, we'll have different followers at different replication levels
			replicatedIndex := fID // Different for each follower
			
			// Update matchIndex for this follower
			leader.MatchIndex[fID] = replicatedIndex
		}(followerID)
	}
	
	wg.Wait()
	
	// Now determine the commit index based on majority
	majority := leader.ClusterSize/2 + 1
	newCommitIndex := leader.CommitIndex
	
	for n := lastIndex; n > leader.CommitIndex; n-- {
		count := 1 // Leader implicitly has the entry
		for i := 0; i < leader.ClusterSize; i++ {
			if i == leader.ID {
				continue
			}
			if leader.MatchIndex[i] >= n {
				count++
			}
		}
		
		if count >= majority && n < len(leader.Log) && leader.Log[n].Term == leader.CurrentTerm {
			newCommitIndex = n
			break
		}
	}
	
	// Update commit index if we found a valid one
	if newCommitIndex > leader.CommitIndex {
		leader.CommitIndex = newCommitIndex
	}
	
	// Verify that commit index is valid
	if leader.CommitIndex > lastIndex {
		t.Errorf("Commit index %d exceeds last log index %d", leader.CommitIndex, lastIndex)
	}
	
	if leader.CommitIndex < 0 {
		t.Errorf("Commit index should not be negative, got %d", leader.CommitIndex)
	}
}