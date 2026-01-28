package raft

import (
	"context"
	"encoding/json"
	"sync"

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

	mutex sync.RWMutex
}

func NewNode(id int, storage storage.Storage) *Node {
	node := &Node{
		ID:          id,
		State:       Follower,
		CurrentTerm: 0,
		VotedFor:    -1,
		Log:         make([]LogEntry, 0),
		CommitIndex: 0,
		LastApplied: 0,
		Storage:     storage,
	}
	return node
}

func NewNodeWithState(id int, persistentState PersistentState, storage storage.Storage) *Node {
	return &Node{
		ID:          id,
		State:       Follower,
		CurrentTerm: persistentState.CurrentTerm,
		VotedFor:    persistentState.VotedFor,
		Log:         make([]LogEntry, 0),
		CommitIndex: 0,
		LastApplied: 0,
		Storage:     storage,
	}
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
}

func (n *Node) StartElection() {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.CurrentTerm++

	n.VotedFor = n.ID

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
	} else if msg.Term < n.CurrentTerm {
	}

	switch msg.Type {
	case RequestVoteMsg:
		n.handleRequestVote(msg)
	case AppendEntriesMsg:
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

func (n *Node) handleRequestVote(msg Message) {
	voteGranted := false

	if msg.Term == n.CurrentTerm && (n.VotedFor == -1 || n.VotedFor == msg.From) {
		n.VotedFor = msg.From
		voteGranted = true
	}

	response := Message{
		Type:        RequestVoteResponseMsg,
		From:        n.ID,
		To:          msg.From,
		Term:        n.CurrentTerm,
		VoteGranted: voteGranted,
	}
	_ = response
}

func (n *Node) handleAppendEntries(msg Message) {
	response := Message{
		Type:        AppendEntriesResponseMsg,
		From:        n.ID,
		To:          msg.From,
		Term:        n.CurrentTerm,
		VoteGranted: false,
	}
	_ = response
}

func (n *Node) handleRequestVoteResponse(msg Message) {
}

func (n *Node) handleAppendEntriesResponse(msg Message) {
}

// MockStorage implements the storage.Storage interface for testing
type MockStorage struct{}

func (m *MockStorage) SaveVote(ctx context.Context, term, votedFor int) error {
	return nil
}

func (m *MockStorage) LoadVote(ctx context.Context) (term, votedFor int, err error) {
	return 0, -1, nil
}