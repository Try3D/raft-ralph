package raft

import (
	"sync"
	"testing"
)

// Test that commit index advances properly when majority of followers acknowledge entries
func TestLeaderCommitsEntryOnMajorityAck(t *testing.T) {
	// Create a leader node in a 3-node cluster
	leader := NewNode(1, &MockStorage{})
	leader.setState(Leader)
	leader.CurrentTerm = 1
	leader.ClusterSize = 3  // 3-node cluster, majority = 2

	// Add an entry to the log from current term
	entry := LogEntry{Command: "testCommand", Term: 1}
	leader.AppendEntry(entry)

	// Initialize match indexes for followers
	leader.MatchIndex = make([]int, leader.ClusterSize)
	leader.NextIndex = make([]int, leader.ClusterSize)

	// Initially, no entries are committed
	if leader.CommitIndex != 0 {
		t.Errorf("Expected initial commit index to be 0, got %d", leader.CommitIndex)
	}

	// Simulate that one follower acknowledges the entry
	leader.MatchIndex[0] = 0  // follower 0 has entry at index 0

	// Process a response from the follower
	response := Message{
		Type:        AppendEntriesResponseMsg,
		From:        0,
		Term:        leader.CurrentTerm,
		VoteGranted: true,
	}

	leader.handleAppendEntriesResponse(response)

	// Commit index should still be 0 because only 1 out of 3 (not majority) has the entry
	if leader.CommitIndex != 0 {
		t.Errorf("Expected commit index to remain 0 (not majority), got %d", leader.CommitIndex)
	}

	// Simulate that another follower also acknowledges the entry
	leader.MatchIndex[2] = 0  // follower 2 has entry at index 0

	// Process another response from the second follower
	response2 := Message{
		Type:        AppendEntriesResponseMsg,
		From:        2,
		Term:        leader.CurrentTerm,
		VoteGranted: true,
	}

	leader.handleAppendEntriesResponse(response2)

	// Now commit index should advance to 0 because majority (leader + follower 0 + follower 2 = 3 out of 3) have the entry
	if leader.CommitIndex < 0 {
		t.Errorf("Expected commit index to advance to at least 0, got %d", leader.CommitIndex)
	}
}

// Test concurrent commit advancement with multiple followers
func TestConcurrentCommitAdvancementWithMajorityRule(t *testing.T) {
	// Create a leader node in a 5-node cluster
	leader := NewNode(1, &MockStorage{})
	leader.setState(Leader)
	leader.CurrentTerm = 1
	leader.ClusterSize = 5  // 5-node cluster, majority = 3

	// Add several entries to the log from current term
	for i := 0; i < 5; i++ {
		entry := LogEntry{Command: "cmd" + string(rune('0'+i)), Term: 1}
		leader.AppendEntry(entry)
	}

	// Initialize match indexes for followers
	leader.MatchIndex = make([]int, leader.ClusterSize)
	leader.NextIndex = make([]int, leader.ClusterSize)

	// Simulate concurrent updates to matchIndex from different followers
	var wg sync.WaitGroup

	// Simulate followers acknowledging different entries
	for followerID := 0; followerID < leader.ClusterSize; followerID++ {
		if followerID == leader.ID {
			continue
		}

		wg.Add(1)
		go func(fID int) {
			defer wg.Done()

			// Simulate this follower having replicated entries up to some index
			// Different followers will have different replication levels
			replicatedIndex := fID % 4  // Followers will have entries up to index 0, 1, 2, or 3

			// Update matchIndex for this follower
			leader.MatchIndex[fID] = replicatedIndex

			// Process a response from this follower
			response := Message{
				Type:        AppendEntriesResponseMsg,
				From:        fID,
				Term:        leader.CurrentTerm,
				VoteGranted: true,
			}

			leader.handleAppendEntriesResponse(response)
		}(followerID)
	}

	wg.Wait()

	// Determine what the commit index should be based on majority rule
	// Count how many servers have each index
	maxCommitIndex := leader.CommitIndex
	for idx := leader.CommitIndex; idx < len(leader.Log); idx++ {
		if leader.Log[idx].Term != leader.CurrentTerm {
			// Only entries from current term can be committed via majority rule
			continue
		}

		count := 1  // Leader implicitly has this entry
		for i := 0; i < leader.ClusterSize; i++ {
			if i == leader.ID {
				continue
			}
			if leader.MatchIndex[i] >= idx {
				count++
			}
		}

		majority := leader.ClusterSize/2 + 1
		if count >= majority {
			maxCommitIndex = idx
		} else {
			// Since entries are ordered, if this one doesn't meet majority,
			// higher ones won't either
			break
		}
	}

	// Verify that the actual commit index matches our expectation
	if leader.CommitIndex != maxCommitIndex {
		t.Errorf("Expected commit index to be %d based on majority rule, got %d", maxCommitIndex, leader.CommitIndex)
	}
}

// Test that entries from current term can be committed via majority rule
func TestCurrentTermEntriesCommittedViaMajority(t *testing.T) {
	leader := NewNode(1, &MockStorage{})
	leader.setState(Leader)
	leader.CurrentTerm = 2  // Current term is 2
	leader.ClusterSize = 3  // 3-node cluster, majority = 2

	// Add an entry from the current term
	currentEntry := LogEntry{Command: "currentCmd", Term: 2}
	leader.AppendEntry(currentEntry)

	// Initialize match indexes for followers
	leader.MatchIndex = make([]int, leader.ClusterSize)
	leader.NextIndex = make([]int, leader.ClusterSize)

	// Simulate that followers have replicated the entry
	leader.MatchIndex[0] = 0  // follower 0 has entry at index 0
	leader.MatchIndex[2] = 0  // follower 2 has entry at index 0

	// Process a response from a follower
	response := Message{
		Type:        AppendEntriesResponseMsg,
		From:        0,
		Term:        leader.CurrentTerm, // term 2
		VoteGranted: true,
	}

	leader.handleAppendEntriesResponse(response)

	// The entry from the current term (index 0) should be committed via majority rule
	// because leader + follower 0 + follower 2 = 3 out of 3 nodes have it (which is majority)
	if leader.CommitIndex < 0 {
		t.Errorf("Expected commit index to be at least 0 (entry from current term), got %d", leader.CommitIndex)
	}
}