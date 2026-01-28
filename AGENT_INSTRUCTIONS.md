# Agent Instructions for Ralph Loop

**READ THIS BEFORE EACH ITERATION**

## Your Role

You are an autonomous Go engineering agent working on a Raft consensus implementation.

## What You Must Do Each Iteration

1. **Read QWEN.md and TODO.md first** (they define how you work)
2. **Pick ONE TODO from TODO.md in order** (don't skip, don't do multiple)
3. **Do ONE small thing** - write tests, implement minimal code, or verify
4. **Run tests** - use `go test -race ./internal/raft`
5. **Commit and push** - use git to save your work
6. **Report results** - use the format from QWEN.md section 6
7. **Log everything** - write to `logs/iteration-{N}.log` only
8. **Exit after push** - don't continue looping

## Critical Rules

⚠️ **You MUST follow these or your work is invalid:**

1. **ONE TODO per iteration** - Pick the next TODO in TODO.md and work ONLY on that
2. **Tests first** - Write tests before or with implementation
3. **Use concurrency tests** - Spawn goroutines to simulate clients/nodes
4. **Log to logs/ only** - All your work output goes to `logs/iteration-{N}.log`
5. **Commit after working** - Run: `git commit -m "iteration-{N}: <description>"`
6. **Push after committing** - Run: `git push origin master`
7. **Report honestly** - Use exact format from QWEN.md section 6
8. **Never skip to next TODO** - Complete one fully before moving on
9. **Go best practices** - All code must be production-grade from day one

## The Ralph Wiggum Loop

Each iteration follows this pattern:

```
Pick ONE TODO
    ↓
Write/run tests for it
    ↓
Implement minimal code
    ↓
Run all tests: go test -race ./...
    ↓
Report results (QWEN.md section 6)
    ↓
git commit -m "iteration-{N}: ..."
    ↓
git push origin master
    ↓
EXIT (Python ralph loop handles next iteration)
```

## If You Get Blocked

Stop immediately and say:

```
Result: - Blocked

I cannot proceed because: [clear explanation]

Next steps: [what would unblock you]
```

The Python ralph loop will detect this and stop gracefully.

## Report Format (MANDATORY)

End every iteration with exactly this format:

```
✅ **What I Worked On**
- [Describe the single invariant/TODO]
- [List exact files or modules touched]

🧪 **Tests**
- [What test type: Unit/Integration/Property]
- [What behavior they verify]

📌 **Result**
- [One of: Fully working / Partially working / Blocked]

⚠️ **Learnings / Issues**
- [Confusion, assumptions, design concerns]
- [Places future work might break]
```

## Example Iteration 1

```
TODO: TODO-0.1 - Verify Project Compiles

Work Done:
- Ran: go build ./...
- Ran: go vet ./...
- Verified: internal/raft/raft.go compiles

Tests:
- Type: Unit
- Ran: go test -race ./internal/raft/...
- Result: PASS (0 tests yet, but code compiles)

✅ **What I Worked On**
- TODO-0.1: Verified Go project compiles
- Files: internal/raft/raft.go (no changes needed)

🧪 **Tests**
- Type: Compilation check
- Behavior: Ensures code has no syntax errors

📌 **Result**
- Fully working

⚠️ **Learnings / Issues**
- Project structure follows Go standards
- No import issues or unused variables
- Ready to proceed to TODO-0.2
```

## Key Directories

```
internal/raft/
├── raft.go          # Main implementation (already exists)
└── raft_test.go     # Tests (create/update this)

logs/
└── iteration-{N}.log # Your work log (created for you)
```

## Useful Commands

```bash
# Compile
go build ./...

# Run tests
go test -race ./internal/raft/...

# Check for race conditions
go test -race -count=10 ./internal/raft/...

# See what changed
git diff

# Commit
git commit -m "iteration-{N}: description"

# Push
git push origin master

# Check git status
git status
```

## What Success Looks Like

- ✅ Each iteration tackles ONE TODO
- ✅ Tests are written before/with implementation
- ✅ Code is production-grade Go
- ✅ Commits are made after each iteration
- ✅ No skipped steps
- ✅ Clear error reporting when blocked
- ✅ Audit trail in git with iteration logs
- ✅ Continuous progress on Raft implementation

## Remember

> Do the smallest thing that increases confidence in correctness.

That is always the right move.

Good luck! 🚀
