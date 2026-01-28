# Ralph Loop Setup Checklist

## ✅ Pre-Execution Checklist

Run this before starting the ralph loop:

```bash
# 1. Verify git configuration
git config user.email
git config user.name

# 2. Verify remote
git remote -v

# 3. Verify project compiles
go build ./...

# 4. Check current status
git status
```

## ✅ What Has Been Configured

### Python Ralph Loop (`ralph.py`)
- ✅ Robust error handling on abrupt exits
- ✅ Automatic recovery by committing state
- ✅ Continues to next iteration after failures
- ✅ Stops only on BLOCKED status or fatal errors
- ✅ All output logged to `logs/iteration-{N}.log`

### Git Workflow
- ✅ Commits after each iteration
- ✅ Pushes to `origin/master` after each commit
- ✅ Logs are part of audit trail in git history
- ✅ Pre-flight checks for git config and remote

### Documentation
- ✅ `QWEN.md` - Working agreement with engineering standards
- ✅ `TODO.md` - Invariant ladder (work breakdown)
- ✅ `README.md` - Project structure and quick start

### Project Structure
```
raft/
├── go.mod
├── README.md
├── QWEN.md
├── TODO.md
├── ralph.py (main orchestrator)
├── ralph.sh (backup bash version)
├── .gitignore
├── logs/ (iteration logs - committed to git)
└── internal/raft/
    ├── raft.go (core types & methods)
    └── raft_test.go (tests - to be created)
```

## 🚀 How to Run

```bash
cd /Users/rsaran/Projects/raft
python3 ralph.py
```

The loop will:
1. Run qwen for iteration 1
2. Commit with message: "iteration-1: ralph loop checkpoint"
3. Push to GitHub
4. Continue to iteration 2
5. Repeat until:
   - BLOCKED status detected → exit gracefully
   - MAX_ITERS (100) reached → exit gracefully
   - Fatal error → exit with error code 1

## 📝 Key Features

| Feature | Behavior |
|---------|----------|
| Abrupt Exit | Logs "🔴 It happened. It fucked up." then recovers |
| Normal Failure | Logs error, attempts recovery commit, continues |
| Blocked Status | Commits current state and exits cleanly |
| Git Failure | Attempts forced recovery then exits if still failing |
| Success | Commits, pushes, continues to next iteration |

## 🔍 Monitoring

Watch progress in real-time:
```bash
# Terminal 1: Watch logs
watch -n 1 "ls -lht logs/iteration-*.log | head -5"

# Terminal 2: Watch git commits
watch -n 1 "git log --oneline | head -10"

# Terminal 3: Run ralph loop
python3 ralph.py
```

## 🆘 Troubleshooting

### Ralph loop won't start
- Check: `git config user.email` and `git config user.name`
- Check: `git remote get-url origin`
- Check: `python3 --version` (needs Python 3.7+)

### Commits not pushing
- Check: `git push origin master` works manually
- Check: Network connectivity
- Check: GitHub credentials/SSH keys

### Logs not appearing
- Check: `logs/` directory is writable: `ls -ld logs/`
- Check: `touch logs/test.log` works

### Qwen command not found
- Check: `which qwen`
- Install: `brew install qwen` or appropriate package manager
