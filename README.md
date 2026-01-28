# Raft Consensus Implementation

A correct, production-grade Raft consensus implementation in Go, built incrementally using the Ralph Wiggum testing methodology.

## Project Structure

```
raft/
├── go.mod                 # Module definition
├── README.md             # This file
├── QWEN.md              # Engineering standards & working agreement
├── TODO.md              # Invariant ladder & work breakdown
├── ralph.py             # Ralph Wiggum loop (Python) - RECOMMENDED
├── ralph.sh             # Ralph Wiggum loop (Bash) - Alternative
├── .gitignore          # Git ignore rules
├── logs/                # Iteration logs (committed to GitHub)
├── internal/
│   └── raft/           # Core Raft library (private)
│       ├── raft.go     # Type definitions & implementation
│       └── raft_test.go # Tests (to be created)
├── cmd/
│   └── raft-server/    # Server binary (coming soon)
└── internal/storage/   # Storage layer (coming soon)
```

## Quick Start

### Setup

```bash
# Configure git
git config user.email "you@example.com"
git config user.name "Your Name"

# Add remote (if not done)
git remote add origin https://github.com/Try3D/raft-ralph.git

# Run ralph loop (Python version - recommended)
python3 ralph.py

# Alternative: Run ralph loop (Bash version)
bash ralph.sh
```

### Running Tests

```bash
# Run all tests with race detector
go test -race ./...

# Run with verbose output
go test -race -v ./...

# Run with retry for flaky tests
go test -race -count=10 ./...
```

## Working Agreement

**All work must follow:**
- `QWEN.md` - Engineering standards (project layout, error handling, testing, git workflow)
- `TODO.md` - Invariant ladder (work breakdown and ordering)

## Key Principles

- **Correctness > Robustness > Completeness > Performance**
- One invariant per iteration
- Tests drive correctness
- Production-grade Go from day one
- Commit and push after each iteration

## Iteration Logs

All iteration logs are in `logs/iteration-{N}.log`.

## Getting Help

- Read `QWEN.md` for working agreement
- Read `TODO.md` for what to work on next
- Check `logs/` for previous iteration details
