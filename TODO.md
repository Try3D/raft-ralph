# Raft Implementation Invariant Ladder

Define what to work on and in what order.

## Rules

- Pick exactly one TODO item
- Complete it according to QWEN.md
- Report results before moving forward
- Skipping steps is forbidden

## How to Use This File

For each TODO:
- Treat it as an invariant, not a feature
- Write tests before or alongside implementation
- Do not advance unless correctness is verified
- Each TODO requires meaningful work (1-2 hours minimum)
- Run robust testing after implementation
- Test files MUST be in `internal/raft/` package
- Use concurrent goroutines to simulate multiple clients and nodes

## Phase 0: Foundations

### TODO-0.1: Verify Project Compiles
- Run `go build ./...` and `go vet ./...`
- Ensure no unused imports or variables
- Check for syntax errors

**Invariant:**
- Code compiles cleanly
- No linting errors

### TODO-0.2: Core Node Types & Methods
- Define `Node` struct with persistent state
- Implement `TransitionToFollower()`, `TransitionToCandidate()`, `TransitionToLeader()`
- Implement `StartElection()` - increments term, votes for self, becomes candidate
- Implement `Step()` - main message handler that routes to sub-handlers
- Verify `Node` maintains exactly one valid state at all times

**Invariant:**
- Node is always in exactly one state (Follower/Candidate/Leader)
- State transitions are explicit and testable
- No goroutines or concurrent modification yet

### TODO-0.3: Term Monotonicity Tests
- Write comprehensive test suite for term behavior:
  - `TestTermNeverDecreases()` - term only increases or stays same
  - `TestHigherTermForcesFollower()` - any higher term message forces follower state
  - `TestTermUpdateWithMessage()` - receiving higher-term message updates local term
  - `TestMultipleConcurrentMessages()` - spawn 10 goroutines sending different-term messages, verify no data races and term monotonicity
- Run all tests with `go test -race`

**Invariant:**
- currentTerm never decreases
- Receiving lower/equal term messages does not change term
- Concurrent messages maintain term monotonicity

### TODO-0.4: Vote Persistence Interface
- Define `Storage` interface: `SaveVote(ctx context.Context, term, votedFor int) error` and `LoadVote(ctx context.Context) (term, votedFor int, err error)`
- Create typed errors: `ErrCorrupted`, `ErrIO`
- Define error wrapping patterns with context
- NO implementation yet - interface only

**Invariant:**
- Storage is decoupled from consensus logic
- All I/O goes through interface

### TODO-0.5: Vote Granting Logic
- Implement `handleRequestVote()` with proper vote granting rules:
  - Grant only if message term equals current term AND (not voted yet OR voted for same candidate)
  - Reset vote when stepping down to new term
- Add RPC response generation
- Write test suite:
  - `TestSingleVotePerTerm()` - cannot vote twice in same term
  - `TestVoteResetOnTermIncrease()` - vote resets when term increases
  - `TestVotingDifferentCandidates()` - cannot vote for two different candidates in same term
  - `TestConcurrentVoteRequests()` - spawn 5 concurrent vote requests with sync.Mutex protection, verify single vote grant
- Verify data races with `go test -race`

**Invariant:**
- Node votes at most once per term
- Vote only changes on term increase
- No double-voting possible

### TODO-0.6: File-Based Vote Storage Implementation
- Implement `Storage` interface using atomic file writes
- Use `fsync()` for durability after every write
- Implement CRC32 checksums to detect corruption
- Handle partial writes and corruption gracefully
- Write persistence tests:
  - `TestVotePersistsAcrossRestart()` - save, crash, load, verify
  - `TestCorruptionDetection()` - corrupt file, verify error on load
  - `TestAtomicWrites()` - concurrent writes don't corrupt state
- Place in `internal/storage/storage.go`

**Invariant:**
- Data is durable after write returns
- Corruption is detected on load
- No data loss on crashes

## Phase 1: Log Structure

### TODO-1.1: Log Entry Types & Append
- Define `LogEntry` with Command, Term, Index
- Implement `AppendEntry()` method - adds entry only if term matches
- Implement `GetLastLogIndexAndTerm()` helper
- Write tests:
  - `TestLogAppendIncrementsIndex()` - index increases by 1
  - `TestLogTermNeverChanges()` - entry term immutable after append
  - `TestLogCannotSkipIndices()` - cannot append at index 5 if only have 0-2
  - `TestConcurrentAppends()` - spawn goroutines appending entries, verify ordering
- Run with `go test -race`

**Invariant:**
- Log indices are contiguous
- Log entries never reorder
- Entry term never changes after append

### TODO-1.2: Request Vote RPC Full Behavior
- Update `handleRequestVote()` to check log term/index for candidate log comparison
- Implement: grant vote only if candidate log is at least as up-to-date as ours
- Generate proper `RequestVoteResponse` messages
- Write tests:
  - `TestVoteGrantedIfLogUpToDate()` - candidate with newer log gets vote
  - `TestVoteDeniedIfLogOutOfDate()` - candidate with older log denied
  - `TestLogComparisonByTerm()` - term takes precedence over index
  - `TestConcurrentVoteComparison()` - multiple nodes comparing logs concurrently

**Invariant:**
- Only up-to-date candidates can become leaders
- Candidate log must be >= follower log to receive vote

### TODO-1.3: Append Entries RPC Structure  
- Implement `handleAppendEntries()` with prefix matching
- Check if previous log entry matches (term and index)
- Accept or reject based on log matching property
- Generate proper `AppendEntriesResponse`
- Write tests:
  - `TestLogMatchingProperty()` - only accept entries with matching prefix
  - `TestConflictingEntriesRejected()` - reject if term mismatch at index
  - `TestEmptyAppendEntriesAsHeartbeat()` - handle heartbeat messages
  - `TestConcurrentLogReplication()` - spawn leader and 3 followers, replicate entries concurrently

**Invariant:**
- Log matching property is enforced
- Conflicting entries detected and rejected

## Phase 2: Elections & Timing

### TODO-2.1: Election Timeout & Tick Mechanism
- Implement `Tick()` method - increments election timeout counter
- Implement election timeout logic:
  - Randomized timeout (e.g., 150-300ms)
  - When timeout occurs: become candidate, start new election
  - Reset timeout when receiving heartbeat/entries from leader
- Write tests:
  - `TestTickIncrementsTimeout()` - each tick increments counter
  - `TestRandomizedTimeout()` - timeout varies between runs
  - `TestElectionStartsAfterTimeout()` - follower becomes candidate after timeout
  - `TestConcurrentTicksFromMultipleNodes()` - spawn goroutines calling Tick() on different nodes, verify independence

**Invariant:**
- Election timeout is randomized (not synchronized)
- Followers become candidates when timeout expires
- Timeout resets on heartbeat

### TODO-2.2: Candidate Vote Collection
- Implement vote counting logic for candidates
- Track votes received, grant leadership only on majority
- Become leader immediately on majority vote
- Write tests:
  - `TestCandidateCountsVotes()` - candidate tallies vote responses
  - `TestBecomeLeaderOnMajority()` - needs > n/2 votes
  - `TestCannotBecomeLeaderWithoutMajority()` - rejects < majority
  - `TestConcurrentVoteCollection()` - spawn 5 candidate goroutines collecting votes, verify majority calculation

**Invariant:**
- At most one leader per term (election safety)
- Leader requires majority of votes

## Phase 3: Log Replication & Commits

### TODO-3.1: Append Entries as Leader
- Implement leader sending AppendEntries to followers
- Initialize `nextIndex` and `matchIndex` for each follower
- Retry on failure (increment term, step down if needed)
- Write tests:
  - `TestLeaderSendsHeartbeats()` - leader periodically sends entries
  - `TestNextIndexDecremented()` - on rejection, decrement nextIndex
  - `TestMatchIndexUpdated()` - on success, update matchIndex
  - `TestConcurrentReplicationToMultipleFollowers()` - leader replicates to 5 followers concurrently

**Invariant:**
- Leader tracks replication progress per follower
- Log eventually replicates to all followers

### TODO-3.2: Commit Index Advancement
- Implement commit index logic:
  - Can advance commitIndex only if entry is from current term
  - Must be replicated on majority of servers
- Implement applying committed entries to state machine
- Write tests:
  - `TestCommitIndexNeverDecreases()` - monotonic advancement
  - `TestCommitOnlyFromCurrentTerm()` - safety: only current-term entries
  - `TestMajorityReplicationAdvancesCommit()` - commit when majority has entry
  - `TestApplyCommittedEntries()` - spawn state machine observer, verify entries applied in order

**Invariant:**
- Committed entries are replicated on majority
- Committed entries are never lost
- Entries applied exactly once

## Phase 4: Client Interface

### TODO-4.1: Command Submission & Response
- Implement `SubmitCommand()` - accept client commands
- Leader appends to log, followers reject
- Track which commands are committed
- Implement response notification when committed
- Write tests:
  - `TestLeaderAcceptsCommands()` - leader appends commands
  - `TestFollowerRejectsCommands()` - follower returns error
  - `TestCommandEventuallyCommitted()` - command committed within timeout
  - `TestConcurrentClientSubmissions()` - spawn 20 clients submitting commands, verify all eventually committed

**Invariant:**
- Only leader accepts new commands
- Commands are replicated before response
- Linearizable ordering

## Phase 5: Snapshots & Log Compaction

### TODO-5.1: Snapshot Mechanics
- Implement snapshot creation and metadata storage
- Implement log compaction - discard entries covered by snapshot
- Implement snapshot loading on restart
- Write tests:
  - `TestSnapshotCreatedCorrectly()` - snapshot contains correct data
  - `TestLogCompactionDiscards()` - entries removed after compaction
  - `TestSnapshotPlusLogRecovery()` - restart using snapshot + remaining log
  - `TestConcurrentSnapshotAndReplication()` - snapshot while replicating to followers

**Invariant:**
- Snapshot + log reconstructs full state
- Entries not discarded prematurely

## Phase 6: Member Changes

### TODO-6.1: Configuration Changes
- Implement config change entries in log
- Implement joint consensus for safe transitions
- Implement old/new config handling
- Write tests with membership changes

**Invariant:**
- No split brain during config changes
- Config changes are linearizable



## Phase 2: Raft State Transitions

### TODO-2.1: Valid Node States
- Follower
- Candidate
- Leader

**Invariant:**
- Node is always in exactly one state

### TODO-2.2: Step Down on Higher Term
- Any higher-term message forces follower state

**Invariant:**
- No leader or candidate survives higher term

## Phase 3: Elections (No Time Yet)

### TODO-3.1: Start Election
- Candidate increments term
- Votes for self

**Invariant:**
- Election always starts in a new term

### TODO-3.2: Vote Granting Rules
- Grant vote only if:
  - term is current
  - not already voted
  - candidate log is acceptable (stub for now)

**Invariant:**
- No double voting

### TODO-3.3: Leader Elected by Majority
- Count votes
- Become leader only on quorum

**Invariant:**
- At most one leader per term

## Phase 4: Time (Deterministic)

### TODO-4.1: Logical Ticks
- Introduce Tick()

**Invariant:**
- No wall-clock time
- Tests drive time forward

### TODO-4.2: Election Timeout
- Followers become candidates after timeout

**Invariant:**
- Randomized timeout
- No synchronized elections

## Phase 5: Log Structure (No Replication)

### TODO-5.1: Append-Only Log
- Index increases monotonically

**Invariant:**
- Entries never reorder
- Indices are contiguous

### TODO-5.2: Log Term Association
- Each entry stores term

**Invariant:**
- Entry term never changes after append

## Phase 6: Log Replication

### TODO-6.1: AppendEntries RPC Handling
- Accept entries only if prefix matches

**Invariant:**
- Log matching property enforced

### TODO-6.2: Conflict Resolution
- Delete conflicting entries

**Invariant:**
- Follower log becomes prefix of leader log

### TODO-6.3: Leader nextIndex / matchIndex
- Track replication progress

**Invariant:**
- Leader never skips indices

## Phase 7: Commit Semantics

### TODO-7.1: Commit Index Monotonicity
- commitIndex only moves forward

**Invariant:**
- No rollback of committed entries

### TODO-7.2: Majority Commit Rule
- Entry committed only if replicated on majority
- Must be from current term

**Invariant:**
- Leader completeness preserved

### TODO-7.3: Apply to FSM
- Apply only committed entries
- Apply exactly once

**Invariant:**
- State machine safety

## Phase 8: Snapshots

### TODO-8.1: Snapshot Creation
- FSM produces snapshot

**Invariant:**
- Snapshot represents prefix of log

### TODO-8.2: Log Compaction
- Discard entries covered by snapshot

**Invariant:**
- Snapshot + log reconstructs full state

### TODO-8.3: InstallSnapshot RPC
- Followers accept snapshots

**Invariant:**
- Follower state matches leader snapshot

## Phase 9: Membership Changes

### TODO-9.1: Configuration Log Entries
- Membership stored in log

**Invariant:**
- Config changes are replicated like commands

### TODO-9.2: Joint Consensus
- Old + new config overlap

**Invariant:**
- No split brain during transition

## Phase 10: Linearizable Reads

### TODO-10.1: ReadIndex Mechanism
- Leader verifies quorum before read

**Invariant:**
- Reads observe all committed writes

## Phase 11: Chaos & Faults

### TODO-11.1: Message Drop
- Transport drops messages

**Invariant:**
- Safety preserved

### TODO-11.2: Message Reordering & Duplication
- Deliver out-of-order and duplicate RPCs

**Invariant:**
- Idempotency and safety

### TODO-11.3: Crash & Restart
- Kill nodes mid-operation

**Invariant:**
- Recovery preserves correctness

## Phase 12: KV Store (Built on Raft)

### TODO-12.1: Command Encoding
- Define Set, Delete

**Invariant:**
- Deterministic command application

### TODO-12.2: KV FSM
- Apply commands to map

**Invariant:**
- All replicas converge

### TODO-12.3: Linearizable Get
- Use Raft read path

**Invariant:**
- Get reflects committed state

## Final Rule

If a TODO feels large:

👉 Split it before touching code.

Correct Raft emerges from patience, not heroics.
