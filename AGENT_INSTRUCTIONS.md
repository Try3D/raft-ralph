# Agent Instructions for Ralph Loop

**READ THIS BEFORE EACH ITERATION**

## Your Role

You are an autonomous Go engineering agent working on a Raft consensus implementation.

## Iteration Scope (IMPORTANT)

**Each iteration must produce real, meaningful work:**

- 30 mins-1 hours of focused effort
- 50+ lines of actual code (not comments or stubs)
- 1-5 files modified/created
- Complete test implementation, not just test names
- Actual verification with multiple test scenarios
- Significant progress on ONE invariant

**Examples of good iterations:**
- ✅ Full test suite + implementation for term monotonicity
- ✅ Implement complete `handleRequestVote()` with 5+ tests
- ✅ Build Storage interface + file-based implementation with persistence tests
- ✅ Implement election timeout with Tick() and randomization logic

**Examples of bad iterations:**
- ❌ Just define a type
- ❌ Just rename variables
- ❌ Just add comments
- ❌ Stub test names without implementations
- ❌ Implement something that's not tested

## What You Must Do Each Iteration

1. **Read QWEN.md section 3 (Scope Guidance)** - understand iteration size
2. **Read QWEN.md and TODO.md first** - they define how you work
3. **Pick ONE TODO from TODO.md in order** - don't skip, don't do multiple
4. **Understand the invariant deeply** - what must be true?
5. **Write comprehensive tests** - not stubs, real test logic with assertions
6. **Implement minimal code** - just enough to satisfy tests
7. **Run tests with race detector** - `go test -race ./internal/raft`
8. **Commit and push** - save your work to GitHub
9. **Report results** - use format from QWEN.md section 8
10. **Log everything** - write to `logs/iteration-{N}.log` only
11. **Exit** - python ralph loop handles next iteration

## Critical Rules

⚠️ **You MUST follow these or your work is invalid:**

1. **ONE TODO per iteration** - Pick next TODO and work ONLY on that
2. **Real tests, not stubs** - Write actual test logic, not just func names
3. **Use concurrency tests** - Spawn goroutines to simulate clients/nodes using `sync.WaitGroup`
4. **Log to logs/ only** - All output goes to `logs/iteration-{N}.log`
5. **Commit after working** - Run: `git commit -m "iteration-{N}: <description>"`
6. **Push after committing** - Run: `git push origin master`
7. **Report honestly** - Use exact format from QWEN.md section 8
8. **Meaningful scope** - At least 1-2 hours of real work per iteration
9. **No comments except info** - Only comment complex logic, not obvious code
10. **Go best practices** - Production-grade from day one

## The Ralph Wiggum Loop Pattern

```
Read QWEN.md (scope guidance)
    ↓
Pick ONE TODO from TODO.md
    ↓
Write comprehensive tests
    ↓
Implement minimal code
    ↓
Run: go test -race ./...
    ↓
Report results using QWEN.md format
    ↓
git commit -m "iteration-{N}: ..."
    ↓
git push origin master
    ↓
EXIT (ralph.py continues to next iteration)
```

## If You Get Blocked

Stop immediately and say:

```
Result: - Blocked

Cannot proceed because: [clear explanation]

Next steps: [what would unblock you]
```

Python ralph loop detects this and stops gracefully.

## Report Format (MANDATORY)

End every iteration with exactly this:

```
✅ **What I Worked On**
- [Describe the invariant/TODO]
- [List exact files touched]

🧪 **Tests**
- [Test type: Unit/Integration/Property]
- [What behavior they verify]

📌 **Result**
- [One of: Fully working / Partially working / Blocked]

⚠️ **Learnings / Issues**
- [Confusion, assumptions, design concerns]
- [Places future work might break]
```

## Example: Good Iteration

**TODO-0.3: Term Monotonicity Tests**

Tests written:
- TestTermNeverDecreases() - set term to 5, receive term 3, verify term stays 5
- TestHigherTermForcesFollower() - become leader, get higher-term message, verify become follower
- TestMultipleConcurrentMessages() - 10 goroutines sending random-term messages, verify no races
- All pass with `go test -race -count=10`

Implementation:
- Updated Step() to compare message.Term with node.CurrentTerm
- If message.Term > node.CurrentTerm: update and step down
- Already had State field, no new fields needed
- 8 lines of code added

Files touched:
- internal/raft/raft.go (8 lines added)
- internal/raft/raft_test.go (40 lines added)

```
✅ **What I Worked On**
- TODO-0.3: Term monotonicity - verify term never decreases
- Files: internal/raft/raft.go (Step method), internal/raft/raft_test.go

🧪 **Tests**
- Type: Unit + Concurrency
- Verify: term only increases or stays same, no data races under concurrent messages
- 3 unit tests + 1 concurrent test, all pass

📌 **Result**
- Fully working

⚠️ **Learnings / Issues**
- Term comparison in Step() is critical - any higher term forces follower state
- Randomization in concurrent test helps catch race conditions
- Mutex not needed since no concurrent writes to same Node (tests serialize)
```

## Key Directories & Files

```
internal/raft/
├── raft.go          # Main implementation
└── raft_test.go     # Tests - create/update this

logs/
└── iteration-{N}.log  # Your work log
```

## Useful Commands

```bash
go build ./...
go vet ./...
go test -race ./internal/raft/...
go test -race -count=10 ./internal/raft/...
go test -race -v ./internal/raft/...

git status
git diff
git commit -m "iteration-{N}: description"
git push origin master
```

## Code Style

- Use table-driven tests for multiple cases
- Spawn goroutines with `sync.WaitGroup` for concurrency tests
- Use channels to simulate message queues
- No inline comments for obvious code
- Only comment: complex algorithms, non-obvious design choices
- Wrap errors with context: `fmt.Errorf("doing x: %w", err)`

## Remember

Do the smallest thing that increases confidence in correctness. That is always the right move.

Good luck! 🚀

