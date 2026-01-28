package raft

import (
	"encoding/json"
	"sync"
)

// MessageType defines the type of message being sent between nodes
type MessageType int

const (
	// RequestVoteMsg is used by candidates to gather votes
	RequestVoteMsg MessageType = iota
	// AppendEntriesMsg is used by leaders to replicate log entries
	AppendEntriesMsg
	// RequestVoteResponseMsg is the response to a RequestVoteMsg
	RequestVoteResponseMsg
	// AppendEntriesResponseMsg is the response to an AppendEntriesMsg
	AppendEntriesResponseMsg
)

// PersistentState represents the state that must be preserved on stable storage
// and must survive crashes and restarts
type PersistentState struct {
	CurrentTerm int
	VotedFor    int
}

// Message represents a message sent between Raft nodes
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

// LogEntry represents a single entry in the Raft log
type LogEntry struct {
	Command interface{}
	Term    int
}

// NodeState represents the state of a Raft node
type NodeState int

const (
	Follower NodeState = iota
	Candidate
	Leader
)

// IsValidState checks if the node is in a valid state
func (s NodeState) IsValidState() bool {
	return s == Follower || s == Candidate || s == Leader
}

// Node represents a single node in the Raft cluster
type Node struct {
	ID        int
	State     NodeState
	CurrentTerm int
	VotedFor  int // ID of the candidate this node voted for in CurrentTerm (or -1 if none)
	Log       []LogEntry
	CommitIndex int
	LastApplied int

	// For Leaders
	NextIndex  []int // For each server, index of the next log entry to send to that server
	MatchIndex []int // For each server, highest log entry known to be replicated on server

	// Mutex to protect concurrent access to node state
	mutex sync.RWMutex
}

// NewNode creates a new Raft node
func NewNode(id int) *Node {
	node := &Node{
		ID:          id,
		State:       Follower,
		CurrentTerm: 0,
		VotedFor:    -1, // -1 means no vote cast
		Log:         make([]LogEntry, 0),
		CommitIndex: 0,
		LastApplied: 0,
	}
	return node
}

// NewNodeWithState creates a new Raft node with existing persistent state
func NewNodeWithState(id int, persistentState PersistentState) *Node {
	return &Node{
		ID:          id,
		State:       Follower, // Start as follower after restart
		CurrentTerm: persistentState.CurrentTerm,
		VotedFor:    persistentState.VotedFor,
		Log:         make([]LogEntry, 0),
		CommitIndex: 0,
		LastApplied: 0,
	}
}

// GetPersistentState returns the current persistent state of the node
func (n *Node) GetPersistentState() PersistentState {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	return PersistentState{
		CurrentTerm: n.CurrentTerm,
		VotedFor:    n.VotedFor,
	}
}

// SaveToStorage simulates saving the persistent state to storage
// In a real implementation, this would write to disk
func (n *Node) SaveToStorage() ([]byte, error) {
	persistentState := n.GetPersistentState()
	data, err := json.Marshal(persistentState)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// LoadFromStorage simulates loading the persistent state from storage
// In a real implementation, this would read from disk
func LoadFromStorage(data []byte, id int) (*Node, error) {
	var persistentState PersistentState
	err := json.Unmarshal(data, &persistentState)
	if err != nil {
		return nil, err
	}

	return NewNodeWithState(id, persistentState), nil
}

// IsValidState checks if the node is in a valid state
func (n *Node) IsValidState() bool {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	return n.State.IsValidState()
}

// setState sets the node's state, ensuring it's always valid
func (n *Node) setState(newState NodeState) {
	if !newState.IsValidState() {
		// This should never happen in a correct implementation
		panic("attempting to set node to invalid state")
	}
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.State = newState
}

// TransitionToFollower transitions the node to follower state
func (n *Node) TransitionToFollower(term int) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.State = Follower
	if term > n.CurrentTerm {
		n.CurrentTerm = term
		n.VotedFor = -1
	}
}

// StartElection starts a new election by incrementing the term and voting for self
// This enforces the invariant that election always starts in a new term
func (n *Node) StartElection() {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	// Increment the term to start a new election
	n.CurrentTerm++

	// Vote for self in the new term
	n.VotedFor = n.ID

	// Transition to candidate state
	n.State = Candidate
}

// TransitionToCandidate transitions the node to candidate state without starting an election
// This is used internally when stepping down from leader to candidate
func (n *Node) TransitionToCandidate() {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if n.State == Leader {
		// When transitioning from leader to candidate, we should clear leader-specific state
		// This will be implemented in later TODOs
	}
	n.State = Candidate
}

// TransitionToLeader transitions the node to leader state
func (n *Node) TransitionToLeader() {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if n.State != Candidate {
		// Only candidates can become leaders
		return
	}
	n.State = Leader
	// Initialize leader-specific state
	// This will be implemented in later TODOs
}

// Step applies a message to the node, causing a state transition
func (n *Node) Step(msg Message) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	// If the message has a higher term, we must step down to follower
	// This enforces the invariant that no leader or candidate survives a higher term
	if msg.Term > n.CurrentTerm {
		n.CurrentTerm = msg.Term
		n.VotedFor = -1  // Reset vote when term changes
		n.State = Follower
	} else if msg.Term < n.CurrentTerm {
		// If the message has a lower term, we may need to respond appropriately
		// depending on the message type (e.g., reject vote requests with higher term)
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

	// Verify that the node is always in a valid state after processing the message
	if !n.State.IsValidState() {
		panic("node is in an invalid state after processing message")
	}
}

// handleRequestVote handles a RequestVote message
func (n *Node) handleRequestVote(msg Message) {
	// The Step method already acquired the lock, so we don't need to acquire it again here
	voteGranted := false

	// A node grants a vote if:
	// 1. The request's term is equal to the node's current term AND
	// 2. The node has not voted yet in this term OR has already voted for this candidate
	//
	// Note: In the Step function, if msg.Term > n.CurrentTerm, we already updated
	// n.CurrentTerm = msg.Term and n.VotedFor = -1, so the condition below still holds.

	if msg.Term == n.CurrentTerm && (n.VotedFor == -1 || n.VotedFor == msg.From) {
		// For now, we'll grant the vote (in later TODOs we'll add log comparison)
		// But only if the conditions above are met
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
	// In a real implementation, we would send the response message
	_ = response
}

// handleAppendEntries handles an AppendEntries message
func (n *Node) handleAppendEntries(msg Message) {
	// The Step method already acquired the lock, so we don't need to acquire it again here
	// For now, stub implementation - we'll implement the actual log replication logic in future TODOs
	response := Message{
		Type:        AppendEntriesResponseMsg,
		From:        n.ID,
		To:          msg.From,
		Term:        n.CurrentTerm,
		VoteGranted: false, // Actually means "success" for AppendEntriesResponse
	}
	// In a real implementation, we would send the response message
	// For now, we're just demonstrating the deterministic step function
	_ = response
}

// handleRequestVoteResponse handles a RequestVoteResponse message
func (n *Node) handleRequestVoteResponse(msg Message) {
	// For now, stub implementation - we'll implement the actual response handling in future TODOs
	// This is typically handled by the candidate that sent the RequestVote
}

// handleAppendEntriesResponse handles an AppendEntriesResponse message
func (n *Node) handleAppendEntriesResponse(msg Message) {
	// For now, stub implementation - we'll implement the actual response handling in future TODOs
	// This is typically handled by the leader that sent the AppendEntries
}