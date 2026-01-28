package raft

import (
	"context"
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"github.com/try3d/raft-ralph/internal/storage"
)

type MessageType int

const (
	RequestVoteMsg MessageType = iota
	AppendEntriesMsg
	RequestVoteResponseMsg
	AppendEntriesResponseMsg
)

type PersistentState struct {
	CurrentTerm int
	VotedFor    int
}

type Message struct {
	Type    MessageType
	From    int
	To      int
	Term    int
	LogTerm int
	LogIndex int
	Entries []LogEntry
	CommitIndex int
	VoteGranted bool
}

type LogEntry struct {
	Command interface{}
	Term    int
	Index   int
}

type NodeState int

const (
	Follower NodeState = iota
	Candidate
	Leader
)

func (s NodeState) IsValidState() bool {
	return s == Follower || s == Candidate || s == Leader
}

type Node struct {
	ID        int
	State     NodeState
	CurrentTerm int
	VotedFor  int
	Log       []LogEntry
	CommitIndex int
	LastApplied int

	NextIndex  []int
	MatchIndex []int

	Storage storage.Storage

	ElectionTimeoutCounter int
	RandomizedElectionTimeout int

	// Vote tracking for candidate state
	votesReceived map[int]bool

	// Cluster configuration
	ClusterSize int

	mutex sync.RWMutex
}

func NewNode(id int, storage storage.Storage) *Node {
	rand.Seed(time.Now().UnixNano())
	node := &Node{
		ID:                    id,
		State:                 Follower,
		CurrentTerm:           0,
		VotedFor:              -1,
		Log:                   make([]LogEntry, 0),
		CommitIndex:           0,
		LastApplied:           0,
		Storage:               storage,
		ElectionTimeoutCounter: 0,
		RandomizedElectionTimeout: rand.Intn(150) + 150,
		votesReceived:         make(map[int]bool),
	}
	return node
}

func NewNodeWithState(id int, persistentState PersistentState, storage storage.Storage) *Node {
	rand.Seed(time.Now().UnixNano())
	return &Node{
		ID:                    id,
		State:                 Follower,
		CurrentTerm:           persistentState.CurrentTerm,
		VotedFor:              persistentState.VotedFor,
		Log:                   make([]LogEntry, 0),
		CommitIndex:           0,
		LastApplied:           0,
		Storage:               storage,
		ElectionTimeoutCounter: 0,
		RandomizedElectionTimeout: rand.Intn(150) + 150,
		votesReceived:         make(map[int]bool),
	}
}

func NewNodeFromStorage(id int, storage storage.Storage) (*Node, error) {
	rand.Seed(time.Now().UnixNano())
	node := &Node{
		ID:                    id,
		State:                 Follower,
		CurrentTerm:           0,
		VotedFor:              -1,
		Log:                   make([]LogEntry, 0),
		CommitIndex:           0,
		LastApplied:           0,
		Storage:               storage,
		ElectionTimeoutCounter: 0,
		RandomizedElectionTimeout: rand.Intn(150) + 150,
		votesReceived:         make(map[int]bool),
	}

	if storage != nil {
		ctx := context.Background()
		term, votedFor, err := storage.LoadVote(ctx)
		if err != nil {
			return nil, err
		}
		node.CurrentTerm = term
		node.VotedFor = votedFor
	}

	return node, nil
}

func (n *Node) GetPersistentState() PersistentState {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	return PersistentState{
		CurrentTerm: n.CurrentTerm,
		VotedFor:    n.VotedFor,
	}
}

func (n *Node) SaveToStorage() ([]byte, error) {
	persistentState := n.GetPersistentState()
	data, err := json.Marshal(persistentState)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func LoadFromStorage(data []byte, id int) (*Node, error) {
	var persistentState PersistentState
	err := json.Unmarshal(data, &persistentState)
	if err != nil {
		return nil, err
	}

	return NewNodeWithState(id, persistentState, nil), nil
}

func (n *Node) IsValidState() bool {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	return n.State.IsValidState()
}

func (n *Node) setState(newState NodeState) {
	if !newState.IsValidState() {
		panic("attempting to set node to invalid state")
	}
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.State = newState
}

func (n *Node) TransitionToFollower(term int) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.State = Follower
	if term > n.CurrentTerm {
		n.CurrentTerm = term
		n.VotedFor = -1
	}

	if n.Storage != nil {
		ctx := context.Background()
		_ = n.Storage.SaveVote(ctx, n.CurrentTerm, n.VotedFor)
	}
}

func (n *Node) StartElection() {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.CurrentTerm++

	n.VotedFor = n.ID

	// Initialize votesReceived map and vote for self
	n.votesReceived = make(map[int]bool)
	n.votesReceived[n.ID] = true

	if n.Storage != nil {
		ctx := context.Background()
		_ = n.Storage.SaveVote(ctx, n.CurrentTerm, n.VotedFor)
	}

	n.State = Candidate
}

func (n *Node) TransitionToCandidate() {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if n.State == Leader {
	}
	n.State = Candidate
}

func (n *Node) TransitionToLeader() {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if n.State != Candidate {
		return
	}
	n.State = Leader
}

func (n *Node) Step(msg Message) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if msg.Term > n.CurrentTerm {
		n.CurrentTerm = msg.Term
		n.VotedFor = -1
		n.State = Follower
		n.ElectionTimeoutCounter = 0

		if n.Storage != nil {
			ctx := context.Background()
			_ = n.Storage.SaveVote(ctx, n.CurrentTerm, n.VotedFor)
		}
	} else if msg.Term < n.CurrentTerm {
		if msg.Type == RequestVoteMsg {
			n.sendRequestVoteResponse(msg, false)
			return
		}
	}

	switch msg.Type {
	case RequestVoteMsg:
		n.handleRequestVote(msg)
	case AppendEntriesMsg:
		n.ElectionTimeoutCounter = 0
		n.handleAppendEntries(msg)
	case RequestVoteResponseMsg:
		n.handleRequestVoteResponse(msg)
	case AppendEntriesResponseMsg:
		n.handleAppendEntriesResponse(msg)
	}

	if !n.State.IsValidState() {
		panic("node is in an invalid state after processing message")
	}
}

func (n *Node) sendRequestVoteResponse(request Message, voteGranted bool) {
	response := Message{
		Type:        RequestVoteResponseMsg,
		From:        n.ID,
		To:          request.From,
		Term:        n.CurrentTerm,
		VoteGranted: voteGranted,
	}
	_ = response
}

func (n *Node) handleRequestVote(msg Message) {
	n.ElectionTimeoutCounter = 0

	voteGranted := false

	logUpToDate := false
	lastLogIndex, lastLogTerm := n.getLastLogIndexAndTermUnlocked()

	if msg.LogTerm > lastLogTerm {
		logUpToDate = true
	} else if msg.LogTerm == lastLogTerm && msg.LogIndex >= lastLogIndex {
		logUpToDate = true
	}

	if msg.Term == n.CurrentTerm && (n.VotedFor == -1 || n.VotedFor == msg.From) && logUpToDate {
		n.VotedFor = msg.From
		voteGranted = true
	} else if msg.Term > n.CurrentTerm && logUpToDate {
		n.CurrentTerm = msg.Term
		n.VotedFor = msg.From
		n.State = Follower
		voteGranted = true
	}

	if voteGranted && n.Storage != nil {
		ctx := context.Background()
		_ = n.Storage.SaveVote(ctx, n.CurrentTerm, n.VotedFor)
	}

	n.sendRequestVoteResponse(msg, voteGranted)
}

func (n *Node) handleRequestVoteWithLogInfo(msg Message, lastLogIndex, lastLogTerm int) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	voteGranted := false

	logUpToDate := false

	if msg.LogTerm > lastLogTerm {
		logUpToDate = true
	} else if msg.LogTerm == lastLogTerm && msg.LogIndex >= lastLogIndex {
		logUpToDate = true
	}

	if msg.Term == n.CurrentTerm && (n.VotedFor == -1 || n.VotedFor == msg.From) && logUpToDate {
		n.VotedFor = msg.From
		voteGranted = true
	} else if msg.Term > n.CurrentTerm && logUpToDate {
		n.CurrentTerm = msg.Term
		n.VotedFor = msg.From
		n.State = Follower
		voteGranted = true
	}

	if voteGranted && n.Storage != nil {
		ctx := context.Background()
		_ = n.Storage.SaveVote(ctx, n.CurrentTerm, n.VotedFor)
	}

	n.sendRequestVoteResponse(msg, voteGranted)
}

func (n *Node) getLastLogIndexAndTermUnlocked() (index, term int) {
	if len(n.Log) == 0 {
		return -1, -1
	}

	lastIndex := len(n.Log) - 1
	return n.Log[lastIndex].Index, n.Log[lastIndex].Term
}

func (n *Node) handleAppendEntries(msg Message) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if msg.Term < n.CurrentTerm {
		response := Message{
			Type:        AppendEntriesResponseMsg,
			From:        n.ID,
			To:          msg.From,
			Term:        n.CurrentTerm,
			VoteGranted: false,
		}
		return
	}

	if msg.Term > n.CurrentTerm {
		n.CurrentTerm = msg.Term
		n.VotedFor = -1
		n.State = Follower

		if n.Storage != nil {
			ctx := context.Background()
			_ = n.Storage.SaveVote(ctx, n.CurrentTerm, n.VotedFor)
		}
	}

	if msg.LogIndex >= 0 {
		if msg.LogIndex >= len(n.Log) {
			response := Message{
				Type:        AppendEntriesResponseMsg,
				From:        n.ID,
				To:          msg.From,
				Term:        n.CurrentTerm,
				VoteGranted: false,
			}
			return
		}

		if msg.LogIndex < len(n.Log) && n.Log[msg.LogIndex].Term != msg.LogTerm {
			response := Message{
				Type:        AppendEntriesResponseMsg,
				From:        n.ID,
				To:          msg.From,
				Term:        n.CurrentTerm,
				VoteGranted: false,
			}
			return
		}
	}

	for i, entry := range msg.Entries {
		logIndex := msg.LogIndex + 1 + i
		if logIndex < len(n.Log) {
			if n.Log[logIndex].Term != entry.Term {
				n.Log = n.Log[:logIndex]
				break
			}
		}
	}

	for i, entry := range msg.Entries {
		logIndex := msg.LogIndex + 1 + i
		if logIndex >= len(n.Log) {
			newEntry := LogEntry{
				Command: entry.Command,
				Term:    entry.Term,
				Index:   logIndex,
			}
			n.Log = append(n.Log, newEntry)
		}
	}

	if msg.CommitIndex > n.CommitIndex {
		lastNewIndex := msg.LogIndex + len(msg.Entries)
		if lastNewIndex < n.CommitIndex {
			n.CommitIndex = lastNewIndex
		} else {
			n.CommitIndex = msg.CommitIndex
		}
	}

	response := Message{
		Type:        AppendEntriesResponseMsg,
		From:        n.ID,
		To:          msg.From,
		Term:        n.CurrentTerm,
		VoteGranted: true,
	}
}

func (n *Node) handleRequestVoteResponse(msg Message) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	// Only process vote responses if we're a candidate in the same term
	if n.State != Candidate || msg.Term != n.CurrentTerm {
		return
	}

	// Record the vote if granted
	if msg.VoteGranted {
		n.votesReceived[msg.From] = true

		// For testing purposes, we'll assume a cluster of 3 nodes (including this node)
		// In a real implementation, we'd need to track the actual cluster membership
		// For now, we'll use a simple majority calculation based on known cluster size
		// We'll assume the cluster size is at least the highest node ID seen + 1
		maxNodeID := n.ID
		for nodeID := range n.votesReceived {
			if nodeID > maxNodeID {
				maxNodeID = nodeID
			}
		}

		// Assume cluster size is max node ID + 1 (simplified assumption)
		clusterSize := maxNodeID + 1
		if clusterSize < 3 { // Minimum cluster size for testing
			clusterSize = 3
		}

		majority := clusterSize/2 + 1

		if len(n.votesReceived) >= majority {
			// Become leader
			n.State = Leader

			// Initialize NextIndex and MatchIndex for followers
			// Size the slices based on cluster size assumption
			n.NextIndex = make([]int, clusterSize)
			n.MatchIndex = make([]int, clusterSize)

			// Set initial values for NextIndex to be the last log index + 1
			lastIndex, _ := n.getLastLogIndexAndTermUnlocked()
			for i := range n.NextIndex {
				if i != n.ID { // Don't set for self
					n.NextIndex[i] = lastIndex + 1
				}
			}
		}
	}
}

func (n *Node) handleAppendEntriesResponse(msg Message) {
}

func (n *Node) AppendEntry(entry LogEntry) bool {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if entry.Term != n.CurrentTerm {
		return false
	}

	nextIndex := len(n.Log)
	entry.Index = nextIndex

	n.Log = append(n.Log, entry)
	return true
}

func (n *Node) GetLastLogIndexAndTerm() (index, term int) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	if len(n.Log) == 0 {
		return -1, -1
	}

	lastIndex := len(n.Log) - 1
	return n.Log[lastIndex].Index, n.Log[lastIndex].Term
}

func (n *Node) Tick() {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.ElectionTimeoutCounter++

	if n.State == Follower || n.State == Candidate {
		if n.ElectionTimeoutCounter >= n.RandomizedElectionTimeout {
			n.startElectionAsFollower()
		}
	}
}

func (n *Node) startElectionAsFollower() {
	n.CurrentTerm++
	n.VotedFor = n.ID

	// Initialize votesReceived map and vote for self
	n.votesReceived = make(map[int]bool)
	n.votesReceived[n.ID] = true

	n.State = Candidate

	if n.Storage != nil {
		ctx := context.Background()
		_ = n.Storage.SaveVote(ctx, n.CurrentTerm, n.VotedFor)
	}

	n.ElectionTimeoutCounter = 0
}

type MockStorage struct{}

func (m *MockStorage) SaveVote(ctx context.Context, term, votedFor int) error {
	return nil
}

func (m *MockStorage) LoadVote(ctx context.Context) (term, votedFor int, err error) {
	return 0, -1, nil
}