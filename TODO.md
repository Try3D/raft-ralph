# Raft Implementation Invariant Ladder

This file defines what may be worked on and in what order.

## Rules

You must:
- Pick exactly one TODO item
- Complete it according to QWEN.md
- Report results before moving forward
- Skipping steps is forbidden.

## How to Use This File

For each TODO:
- Treat it as an invariant, not a feature
- Write tests before or alongside implementation
- Do not advance unless correctness is verified
- Each TODO is intentionally small.
- Run robust testing after each implementation to ensure functionality
- Append testing conclusions and outcomes to the relevant test files

## Phase 0: Foundations & Scaffolding

### TODO-0.1: Standard Project Layout
- Refactor project structure to follow Go standards:
  - `internal/raft`: Core library code.
  - `cmd/raft-server`: Main entry point.
- Ensure strict linting is possible.

**Invariant:**
- Project layout is clean and extensible.
- Code compiles in new structure.

### TODO-0.2: Storage Interface Definition
- Define `Storage` interface in `internal/raft`.
- Methods for HardState (Term, Vote) and Log.
- Define typed errors for storage failures.

**Invariant:**
- Persistence logic is decoupled from consensus logic.

### TODO-0.3: Deterministic Step Function
- Introduce `Node` struct and `Step` method in `internal/raft`.

**Invariant:**
- All state changes go through Step.
- No goroutines yet.
- No time yet.

## Phase 1: Robust Persistence & Safety

### TODO-1.1: Robust File-Based Storage
- Implement `Storage` using a safe file format (e.g., WAL or atomic page writes).
- MUST use `fsync` to ensure durability.
- MUST handle partial writes/corruption (CRC checks).
- Place implementation in `internal/storage`.

**Invariant:**
- Data is durable after write returns.
- Corruption is detected on load.

### TODO-1.2: Term Monotonicity
- Wired up to Storage.
- currentTerm never decreases.
- Persist term changes synchronously.

**Invariant:**
- Restarted node recovers term.
- Lower-term messages ignored.

### TODO-1.3: Single Vote Per Term
- Persist `votedFor` synchronously before responding.

**Invariant:**
- Node votes at most once per term.
- Vote survives crash.

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
