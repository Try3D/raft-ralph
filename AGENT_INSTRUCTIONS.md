# Agent Instructions for Ralph Loop

**READ THIS BEFORE EACH ITERATION**

## Your Role

You are an autonomous Go engineering agent working on a Raft consensus implementation.

## MANDATORY: DELETE ALL COMMENTS FIRST

**BEFORE doing anything else each iteration:**

1. DO NOT USE ANY inline comment (every `//` that isn't a doc comment)
2. Delete EVERY commented-out code line if it already exists
3. If violations exist, fix and re-check

**This is non-negotiable. Code must be self-documenting through naming alone.**

## Iteration Scope (IMPORTANT)

**Each iteration must produce real, meaningful work:**

- 0.5-1 hours of focused effort
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

1. **DELETE ALL COMMENTS** (first action, non-negotiable)
   - Search: `grep -n "//" internal/raft/raft.go internal/raft/raft_test.go`
   - Delete every inline comment
   - Delete every commented-out line
   - Keep only doc comments (if required by Go)
   - Verify clean: no `//` except doc strings

2. **Read QWEN.md section 4** - understand comment policy

3. **Read QWEN.md and TODO.md first** - they define how you work

4. **Pick ONE TODO from TODO.md in order** - don't skip, don't do multiple

5. **Understand the invariant deeply** - what must be true?

6. **Write comprehensive tests** - not stubs, real test logic with assertions
   - Use table-driven tests
   - Spawn goroutines with sync.WaitGroup
   - Run with `go test -race`

7. **Implement minimal code** - just enough to satisfy tests
   - Write clean, self-documenting code
   - Use clear variable/function names
   - NO comments while implementing

8. **Run tests with race detector** - `go test -race ./internal/raft`

9. **BEFORE COMMITTING: Verify no comments**
   - Run: `grep -r "//" internal/raft/`
   - Result must be EMPTY or only doc comments
   - If violations found, delete and re-test

10. **Commit and push** - save your work to GitHub
    - Commit message: "iteration-{N}: {description}"
    - Push: `git push origin master`

11. **Report results** - use format from QWEN.md section 8

12. **Log everything** - write to `logs/iteration-{N}.log` only

13. **Exit** - python ralph loop handles next iteration

## Critical Rules

⚠️ **You MUST follow these or your work is invalid:**

1. **DELETE COMMENTS FIRST** - every iteration starts with comment removal
2. **ONE TODO per iteration** - Pick next TODO and work ONLY on that
3. **Real tests, not stubs** - Write actual test logic, not just func names
4. **Use concurrency tests** - Spawn goroutines to simulate clients/nodes using `sync.WaitGroup`
5. **Log to logs/ only** - All output goes to `logs/iteration-{N}.log`
6. **Commit after working** - Run: `git commit -m "iteration-{N}: <description>"`
7. **Push after committing** - Run: `git push origin master`
8. **Report honestly** - Use exact format from QWEN.md section 8
9. **Meaningful scope** - At least 1-2 hours of real work per iteration
10. **Code must be clean** - No comments except doc strings at all
11. **Go best practices** - Production-grade from day one

## The Ralph Wiggum Loop Pattern

```
DELETE ALL COMMENTS (mandatory first step)
    ↓
Read QWEN.md (scope guidance)
    ↓
Pick ONE TODO from TODO.md
    ↓
Write comprehensive tests
    ↓
Implement minimal code (NO COMMENTS)
    ↓
Run: go test -race ./...
    ↓
VERIFY NO COMMENTS: grep -r "//" internal/raft/
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

## Code Style

- Use table-driven tests for multiple cases
- Spawn goroutines with `sync.WaitGroup` for concurrency tests
- Use channels to simulate message queues
- **NO comments for obvious code** - code is self-documenting
- **NO commented-out code blocks**
- Wrap errors with context: `fmt.Errorf("doing x: %w", err)`
- Clear naming makes code readable without comments

## Example: Comment Deletion

BEFORE (bad):
```go
func handleRequestVote(msg Message) {
    // Check if message term equals current term
    if msg.Term == n.CurrentTerm {
        // Grant vote if not already voted
        if n.VotedFor == -1 || n.VotedFor == msg.From {
            n.VotedFor = msg.From  // Set vote to candidate
            voteGranted = true      // Grant the vote
        }
    }
}
```

AFTER (clean):
```go
func handleRequestVote(msg Message) {
    if msg.Term == n.CurrentTerm && (n.VotedFor == -1 || n.VotedFor == msg.From) {
        n.VotedFor = msg.From
        voteGranted = true
    }
}
```

## Remember

Do the smallest thing that increases confidence in correctness. That is always the right move.

**But FIRST: DELETE ALL COMMENTS.**

Good luck! 🚀
