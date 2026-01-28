#!/usr/bin/env bash
set -euo pipefail

MAX_ITERS=100
LOGS_DIR="logs"
mkdir -p "$LOGS_DIR"

# Git setup check
if ! git config user.email > /dev/null 2>&1; then
  echo "❌ Git not configured. Run: git config user.email 'you@example.com' && git config user.name 'Your Name'"
  exit 1
fi

# Check if remote exists
if ! git remote get-url origin > /dev/null 2>&1; then
  echo "❌ Git remote 'origin' not configured. Run: git remote add origin <url>"
  exit 1
fi

PROMPT='
You are an autonomous engineering agent working inside a Go Raft implementation repository.

You MUST strictly follow QWEN.md and TODO.md.

Execute exactly ONE Ralph Wiggum iteration:
- Pick the next TODO in order
- Work on ONE invariant or task only
- Add or run tests where applicable
- Report honestly using the required format
- Do NOT continue to another TODO
- NEVER skip this iteration loop or move to next work item

Log Output Location:
- Write ALL iteration logs to logs/ directory only
- Do NOT write logs elsewhere
- Format: logs/iteration-{N}.log

Git Workflow:
- After completing work, commit with: git commit -m "iteration-{N}: <description>"
- Push with: git push origin master
- If git fails, report and stop
- Exit after push

If you are blocked, say BLOCKED and explain why so that the next agent can continue from there.

Output ONLY the iteration report to stdout.
'

for i in $(seq 1 $MAX_ITERS); do
  LOG_FILE="$LOGS_DIR/iteration-$i.log"
  
  echo "=============================="
  echo " Ralph Wiggum Iteration $i"
  echo "=============================="
  
  # Run qwen and capture output, redirecting to log file
  if ! qwen -p "$PROMPT" -y 2>&1 | tee "$LOG_FILE"; then
    echo "❌ qwen command failed on iteration $i"
    git add -A || true
    git commit -m "iteration-$i: qwen command failed" || true
    git push origin master || true
    exit 1
  fi
  
  # Check for blocking status
  if grep -qi "Result:[[:space:]]*.*Blocked" "$LOG_FILE"; then
    echo ""
    echo "🛑 Blocked at iteration $i. Stopping Ralph loop."
    git add -A || true
    git commit -m "iteration-$i: BLOCKED - stopping" || true
    git push origin master || true
    exit 0
  fi
  
  # Commit and push after each iteration
  echo ""
  echo "📦 Committing iteration $i..."
  if ! git add -A; then
    echo "❌ git add failed on iteration $i"
    exit 1
  fi
  
  COMMIT_MSG="iteration-$i: ralph loop checkpoint"
  if ! git commit -m "$COMMIT_MSG"; then
    echo "⚠️  Nothing to commit on iteration $i (may be normal)"
  fi
  
  echo "🚀 Pushing to remote..."
  if ! git push origin master; then
    echo "❌ git push failed on iteration $i"
    exit 1
  fi
  
  echo "✅ Iteration $i complete"
  echo ""
done

echo ""
echo "🏁 Completed $MAX_ITERS Ralph Wiggum iterations."
exit 0
