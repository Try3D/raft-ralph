# Raft Implementation Working Agreement

This document defines how you must work, not just what you must work on.

If you violate this, the work is considered invalid.

## 1. Core Principle

You are building a production-grade, correct Raft implementation.

**Correctness > Robustness > Completeness > Performance.**

You will proceed one invariant at a time, but every line of code must be industry-standard Go.

## 2. The Ralph Wiggum Loop (Mandatory)

You must always operate in the following loop:

1. Pick ONE small TODO or invariant
2. Understand it deeply
3. Write or update tests for it
4. Implement the minimal code required
5. Run all relevant tests
6. Report results honestly
7. Reflect on learnings or issues

You must not:
- Implement multiple features in one step
- Skip tests
- Assume correctness without verification
- Hide uncertainty

## 3. Scope Guidance (IMPORTANT)

**Each iteration should:**
- Be 1-2 hours of focused work
- Produce meaningful changes (not trivial commits)
- Add/modify 1-3 files
- Include 10+ lines of real code changes minimum
- Include actual test implementations, not stubs
- Verify the work with multiple test scenarios

**One iteration examples:**
- ✅ Add full test suite for one invariant + minimal implementation
- ✅ Implement a complete RPC handler + tests
- ✅ Build a storage interface + file-based implementation with tests
- ❌ Just write a type definition
- ❌ Just rename a variable
- ❌ Just add comments

**If the TODO feels too small after reading it:**
Split it mentally into subtasks and combine 2-3 into one iteration.

## 4. Engineering Standards (Strict)

You must adhere to the following Go industry best practices:

**Project Layout:**
- Use `internal/` for private library code.
- Use `cmd/` for application entry points.
- Keep the root directory clean (only `go.mod`, `README`, etc.).

**Code Quality:**
- Wrap errors with context: `fmt.Errorf("operation: %w", err)`
- **Never** ignore errors.
- Use strict typing. Avoid `interface{}` unless absolutely necessary.
- Use `context.Context` for long-running operations, I/O, and RPCs.
- Document exported types and functions.

**COMMENTS - MANDATORY DELETE POLICY:**
- **EVERY FILE MUST HAVE ALL EXISTING COMMENTS DELETED** (except file header)
- NO inline comments whatsoever (self-documenting code names)
- NO commented-out code blocks under any circumstances
- NO "TODO" comments left in code
- NEVER add comments while implementing
- Code must be readable through clear naming and structure alone
- EXCEPTION ONLY: Package-level doc comments for exported functions (required by Go linting)

**Before committing ANY code:**
1. Search for and DELETE every inline comment
2. Search for and DELETE every commented-out line
3. Verify: `grep -r "//" internal/raft/raft.go` returns ONLY doc comments (lines starting with package doc)
4. If violations found, fix and re-commit

**Robust File System Usage:**
- When implementing persistence, assume the file system is unreliable.
- Use `fsync` to guarantee durability.
- Handle partial writes and corruption (checksums).
- Use atomic file replacements for state updates where appropriate.
- Define clear `Storage` interfaces before implementing file IO.

**Testing & Verification:**
- Always use `go test -race ./...` to catch concurrency bugs.
- Write table-driven tests (`t.Run`) for complex logic.
- Ensure strict linting passes (assume `golangci-lint` is standard).
- Concurrency Testing: Use Go's concurrency primitives extensively. Spawn goroutines to simulate multiple clients, peer nodes, or network conditions.
- Use channels to model concurrent events and network partitions.

## 5. Scope Rules

**Allowed:**
- One major invariant
- Multiple related test cases
- One module/package
- Related behavior patterns
- Both unit AND integration tests in same iteration

**Forbidden:**
- "Implement leader election" (too big)
- "Finish replication" (too big)
- "Handle snapshots fully" (too big)
- Large refactors unless explicitly requested

**If the task feels big, split it before touching code.**

## 6. Definition of "Done"

A task is done only if:
- Tests exist and pass
- The invariant is enforced
- No unrelated behavior changed
- Failure cases are considered
- Code is properly formatted
- No TODO comments left

"Looks correct" is not a valid definition.

## 7. Testing Requirements

You must always state:
- What type of test you added or ran:
  - Unit
  - Integration
  - Property / chaos
- What invariant the test enforces
- If no test was written, explain why.

## 8. Reporting Format (MANDATORY)

At the end of every response, you must include:

✅ **What I Worked On**
- Describe the single invariant or TODO
- Mention the exact files or modules touched

🧪 **Tests**
- What tests were added or run
- What behavior they verify

📌 **Result**
- Fully working
- Partially working
- Blocked

Be honest. Partial progress is fine.

⚠️ **Learnings / Issues**
- Anything confusing
- Any assumption you had to make
- Any design smell you noticed
- Any place future work might break

This section is not optional.

## 9. When You Are Blocked

If you cannot proceed:
- Stop immediately
- Explain why you are blocked
- Suggest at least one concrete next step
- Do NOT guess or hallucinate a solution

## 10. Raft-Specific Rules

You must respect the Raft paper invariants:
- Term monotonicity
- Election safety
- Log matching
- Leader completeness
- State machine safety

If an implementation choice risks violating one, you must call it out explicitly.

## 11. No Silent Global Changes

You must not:
- Change public APIs
- Rename concepts
- Modify timing behavior
- Add background goroutines

Unless explicitly instructed.

If you did, you must report it.

## 12. Memory and Continuity

- Always update TODO.md with progress
- When starting work, read QWEN.md and TODO.md first
- Maintain continuity between sessions by documenting findings

## 13. Testing Protocol

- Run changes with robust testing each time
- Execute all relevant tests after each implementation
- Document test results in iteration logs
- Append testing conclusions to relevant test files
- Verify all existing tests continue to pass

## 14. Language & Style

- Be precise
- Be boring
- Be explicit
- Avoid "probably", "should", "seems"
- If something is uncertain, say so.

## 15. Logging & Git Workflow

**Logging:**
- ALL iteration logs MUST be written to `logs/` directory only.
- File format: `logs/iteration-{N}.log` where N is iteration number.
- Include comprehensive results, test runs, and error messages in logs.
- Logs ARE committed to GitHub for audit trail and history.

**Git Workflow (MANDATORY):**
- After EACH successful iteration, commit changes with format: `git commit -m "iteration-{N}: {brief description}"`
- Push to remote after each commit: `git push origin master`
- If any git command fails, stop immediately and report the error.
- Commit the following:
  - Source code changes
  - Test files
  - Iteration logs in `logs/` directory
- DO NOT commit:
  - Temporary files
  - IDE files
- Use `.gitignore` to exclude build artifacts and non-essential files.

**Exit Protocol:**
- After successful iteration: commit, push, then exit cleanly with code 0.
- On blocking error: log the issue to `logs/iteration-{N}.log`, commit, push, then exit with code 1.
- Never continue looping on git failures.

## 16. Concurrency Testing (Mandatory)

**Test Harness:**
- Use `sync.WaitGroup` to coordinate multiple goroutines.
- Spawn goroutines to simulate:
  - Multiple clients sending concurrent requests
  - Peer nodes in the cluster operating in parallel
  - Network partitions and message delays using channels
- Use channels to queue messages between "virtual threads".
- Always run tests with `go test -race -count=10 ./...` to catch data races.

**Example Pattern:**
```go
func TestConcurrentClients(t *testing.T) {
    const numClients = 5
    var wg sync.WaitGroup
    
    for i := 0; i < numClients; i++ {
        wg.Add(1)
        go func(clientID int) {
            defer wg.Done()
            // Simulate client behavior
        }(i)
    }
    wg.Wait()
}
```

## 17. Final Rule

If you are ever unsure what to do next:

👉 Do the smallest thing that increases confidence in correctness.

That is always the right move.

