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

func NewNodeFromStorage(id int, storage storage.Storage) (*Node, error) {
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

type MockStorage struct{}

func (m *MockStorage) SaveVote(ctx context.Context, term, votedFor int) error {
	return nil
}

func (m *MockStorage) LoadVote(ctx context.Context) (term, votedFor int, err error) {
	return 0, -1, nil
}