# Ralph Loop System - Final Configuration Summary

## System Ready for Production Use

This document summarizes the complete Ralph Wiggum loop orchestration system for iterative Raft implementation.

### Core Components

#### 1. Python Ralph Loop (`ralph.py`)
- Robust orchestrator with exception handling
- Detects abrupt exits (timeouts, interrupts)
- Auto-recovery with commit/push
- Never stops on errors—continues to next iteration
- Stops cleanly on BLOCKED status
- 10-minute timeout per qwen iteration

#### 2. Working Agreement (QWEN.md)
- 17 sections covering engineering standards
- Section 3: Explicit scope guidance
  - 1-2 hours per iteration minimum
  - 10+ lines of real code required
  - 1-3 files modified per iteration
  - Comprehensive test implementations
- Section 4: Comment policy - minimal, informative only
- Section 16: Concurrency testing with goroutines

#### 3. Work Breakdown (TODO.md)
- 6 major phases with 19 total TODOs
- Phase 0: Foundations (6 TODOs)
- Phase 1-6: Core Raft features
- Each TODO includes:
  - Specific implementation requirements
  - Multiple test scenarios (5-10 tests each)
  - Concurrency testing requirements
  - Clear invariants

#### 4. Agent Manual (AGENT_INSTRUCTIONS.md)
- Clear iteration scope guidance
- Examples of good vs bad iterations
- Complete code style guidelines
- Reporting format with examples
- Command reference

### Key Features

| Feature | Implementation |
|---------|-----------------|
| **Iteration Scope** | 1-2 hours minimum, 10+ LOC, 1-3 files |
| **Error Recovery** | Detects abrupt exits, logs, commits, continues |
| **Comments** | Minimal—only complex algorithms and design choices |
| **Concurrency Tests** | Mandatory goroutines with sync.WaitGroup |
| **Code Quality** | Production-grade Go from day one |
| **Logging** | All output to `logs/iteration-{N}.log` (committed) |
| **Git Workflow** | Auto-commit/push after each iteration |

### Example Good Iterations

**Iteration 1 (TODO-0.2):**
- Implement Node struct
- Implement state transitions
- Implement Step() message router
- Implement StartElection()
- Write 8+ tests
- ~2 hours, ~100+ lines of code

**Iteration 2 (TODO-0.3):**
- Write term monotonicity tests
- Implement term comparison in Step()
- Write concurrent tests (10 goroutines)
- Run with -race detector
- ~1.5 hours, ~80+ lines of test code

**Iteration 3 (TODO-0.5):**
- Implement vote granting logic
- Implement RequestVote RPC handling
- Write 5+ test scenarios
- Concurrent vote request tests
- ~2 hours, ~120+ lines of code

### Project Structure

```
raft/
├── go.mod                   # Module definition
├── README.md               # Quick start
├── QWEN.md                # Engineering standards (17 sections)
├── TODO.md                # Work breakdown (19 TODOs)
├── AGENT_INSTRUCTIONS.md   # Agent manual
├── RALPH_SETUP.md         # Setup guide
├── ralph.py               # Main orchestrator
├── ralph.sh               # Bash backup
├── .gitignore            # Git rules
├── logs/                  # Iteration logs (committed)
└── internal/raft/
    ├── raft.go           # Core implementation
    └── raft_test.go      # Tests (to be created)
```

### How to Use

```bash
# 1. Configure git (once)
git config user.email "you@example.com"
git config user.name "Your Name"

# 2. Start the ralph loop
cd /Users/rsaran/Projects/raft
python3 ralph.py

# Loop will:
# - Run qwen iteration 1
# - On success: commit, push, continue to iteration 2
# - On abrupt exit: recover, commit, continue
# - On BLOCKED: commit, push, stop cleanly
# - Repeat for up to 100 iterations
```

### Monitoring Progress

```bash
# Watch commits
watch -n 1 "git log --oneline | head -5"

# Check logs
ls -lht logs/iteration-*.log

# View specific iteration
cat logs/iteration-1.log

# Check git status
git status
```

### Expected Timeline

Phase 0 (Foundations): ~12-16 iterations, ~2-3 weeks
- 6 TODOs covering basic node structure, state transitions, term/vote logic
- Foundation for all subsequent phases

Phase 1-6: ~100+ iterations total
- ~4-6 months of continuous development
- Incremental progress on core Raft features
- Complete implementation by end

### Comment Guidelines

**Remove these (obvious code):**
```go
n.CurrentTerm++  // Increment term
n.VotedFor = 2   // Set vote
if x > y { }     // If x greater than y
```

**Keep these (informative):**
```go
// Raft paper: vote only for up-to-date candidates
// Candidate log >= follower log (by term first, then index)

// Atomically write to disk to prevent partial corruption
// followed by CRC32 verification on load

// Randomize [150ms, 300ms] to prevent election sync
timeout := 150 + rand.Intn(150)
```

### Testing Strategy

- Unit tests for individual functions
- Integration tests for interaction between components
- Concurrency tests using `sync.WaitGroup` and `sync.Mutex`
- Race detector enabled: `go test -race ./...`
- Retry with `-count=10` to catch flaky tests

### Success Criteria

An iteration is successful if:
1. ✅ Tests exist and pass
2. ✅ Invariant is enforced
3. ✅ No unrelated changes
4. ✅ Failure cases considered
5. ✅ Code is properly formatted
6. ✅ No TODO comments left
7. ✅ 10+ lines of real code added
8. ✅ Proper report filed

### Common Commands

```bash
# Compile and test
go build ./...
go test -race ./internal/raft/...
go test -race -count=10 ./internal/raft/...
go test -race -v ./internal/raft/...

# Git
git status
git diff
git add -A
git commit -m "iteration-{N}: description"
git push origin master

# Logs
ls -la logs/
tail -f logs/iteration-*.log
```

### Repository

- **URL:** https://github.com/Try3D/raft-ralph.git
- **Branch:** master
- **Status:** Clean, all changes committed
- **First iteration:** Ready to start

### Starting First Iteration

```
python3 ralph.py

Expected in Iteration 1:
- Agent reads TODO-0.1
- Runs: go build ./... and go vet ./...
- Verifies project compiles
- Reports results
- Commits: "iteration-1: verify project compiles"
- Pushes to GitHub
- Ralph loop continues to Iteration 2
```

---

**System is 100% ready. Start with: `python3 ralph.py`**
