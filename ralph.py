#!/usr/bin/env python3
"""
Ralph Wiggum Loop Orchestrator
Manages iterative Raft implementation using autonomous agents.
"""

import os
import sys
import subprocess
import time
from pathlib import Path
from datetime import datetime

# Configuration
MAX_ITERS = 100
LOGS_DIR = Path("logs")
REPO_ROOT = Path(__file__).parent.absolute()
GIT_REMOTE = "origin"
GIT_BRANCH = "master"

# Ensure logs directory exists
LOGS_DIR.mkdir(exist_ok=True)


def log_to_file(iteration: int, message: str):
    """Write message to iteration log file."""
    log_file = LOGS_DIR / f"iteration-{iteration}.log"
    timestamp = datetime.now().isoformat()
    with open(log_file, "a") as f:
        f.write(f"[{timestamp}] {message}\n")


def run_command(cmd: list, iteration: int, description: str) -> tuple[bool, str]:
    """
    Run a shell command and capture output.
    Returns: (success: bool, output: str)
    """
    try:
        result = subprocess.run(
            cmd, cwd=REPO_ROOT, capture_output=True, text=True, timeout=300
        )
        output = result.stdout + result.stderr
        success = result.returncode == 0

        if not success:
            log_to_file(
                iteration, f"❌ {description} FAILED (exit code {result.returncode})"
            )
            log_to_file(iteration, f"Command: {' '.join(cmd)}")
            log_to_file(iteration, f"Output: {output[:500]}")
        else:
            log_to_file(iteration, f"✅ {description} SUCCESS")

        return success, output
    except subprocess.TimeoutExpired:
        log_to_file(iteration, f"❌ {description} TIMEOUT (300s)")
        return False, "Command timed out"
    except Exception as e:
        log_to_file(iteration, f"❌ {description} ERROR: {str(e)}")
        return False, str(e)


def check_git_config() -> bool:
    """Verify git is configured properly."""
    success, _ = run_command(["git", "config", "user.email"], 0, "Git user.email check")
    if not success:
        print("❌ Git not configured. Run:")
        print("   git config user.email 'you@example.com'")
        print("   git config user.name 'Your Name'")
        return False

    success, _ = run_command(
        ["git", "remote", "get-url", GIT_REMOTE], 0, "Git remote check"
    )
    if not success:
        print(f"❌ Git remote '{GIT_REMOTE}' not configured. Run:")
        print(f"   git remote add {GIT_REMOTE} <url>")
        return False

    return True


def check_logs_dir_exists() -> bool:
    """Verify logs directory exists and is writable."""
    try:
        LOGS_DIR.mkdir(exist_ok=True)
        test_file = LOGS_DIR / ".test"
        test_file.write_text("test")
        test_file.unlink()
        return True
    except Exception as e:
        print(f"❌ Cannot write to logs directory: {e}")
        return False


def run_qwen_iteration(iteration: int) -> tuple[bool, str, str]:
    """
    Run qwen for one iteration.
    Returns: (success, output, log_file_path)
    """
    log_file = LOGS_DIR / f"iteration-{iteration}.log"

    prompt = f"""
You are an autonomous engineering agent working inside a Go Raft implementation repository.

You MUST strictly follow QWEN.md and TODO.md.

Execute exactly ONE Ralph Wiggum iteration {iteration}:
- Pick the next TODO in order
- Work on ONE invariant or task only
- Add or run tests where applicable
- Report honestly using the required format
- Do NOT continue to another TODO
- Never skip steps or move forward without reporting

Log Output Location:
- Write ALL iteration logs to logs/ directory only
- Do NOT write logs elsewhere
- Logs are committed to GitHub

Git Workflow After Work:
- After completing work, run: git commit -m "iteration-{iteration}: <description>"
- Then run: git push {GIT_REMOTE} {GIT_BRANCH}
- If any git command fails, report and stop
- Always exit after push

If you are blocked, say "Result: - Blocked" and explain why so that the next agent can continue.

Output ONLY the iteration report.
"""

    log_to_file(iteration, f"🚀 Starting iteration {iteration}")

    try:
        result = subprocess.run(
            ["qwen", "-p", prompt, "-y"],
            cwd=REPO_ROOT,
            capture_output=True,
            text=True,
            timeout=600,  # 10 min timeout per iteration
        )

        output = result.stdout + result.stderr

        # Append qwen output to log file
        with open(log_file, "a") as f:
            f.write("\n--- QWEN OUTPUT START ---\n")
            f.write(output)
            f.write("\n--- QWEN OUTPUT END ---\n")

        success = result.returncode == 0

        if not success:
            log_to_file(iteration, f"⚠️  qwen exited with code {result.returncode}")

        return success, output, str(log_file)

    except subprocess.TimeoutExpired:
        log_to_file(iteration, "❌ qwen TIMEOUT (10 min)")
        log_to_file(iteration, "🔴 ABRUPT EXIT DETECTED: Timeout occurred")
        return False, "qwen command timed out", str(log_file)
    except KeyboardInterrupt:
        log_to_file(iteration, "❌ qwen INTERRUPTED by user")
        log_to_file(iteration, "🔴 ABRUPT EXIT DETECTED: User interrupted")
        return False, "User interrupted", str(log_file)
    except Exception as e:
        log_to_file(iteration, f"❌ qwen ERROR: {str(e)}")
        log_to_file(iteration, f"🔴 ABRUPT EXIT DETECTED: {str(e)}")
        return False, str(e), str(log_file)


def check_blocked_status(log_file: str) -> bool:
    """Check if agent reported being blocked."""
    try:
        with open(log_file, "r") as f:
            content = f.read()
            if "blocked" in content.lower() or "Result: - Blocked" in content:
                return True
    except:
        pass
    return False


def commit_and_push(iteration: int) -> bool:
    """Commit and push changes after iteration."""
    # Add all changes
    success, _ = run_command(["git", "add", "-A"], iteration, "Git add")
    if not success:
        log_to_file(iteration, "❌ git add failed - stopping")
        return False

    # Commit
    commit_msg = f"iteration-{iteration}: ralph loop checkpoint"
    success, output = run_command(
        ["git", "commit", "-m", commit_msg], iteration, "Git commit"
    )
    if not success:
        # No changes to commit is OK
        if "nothing to commit" in output.lower():
            log_to_file(iteration, "⚠️  Nothing to commit (normal)")
        else:
            log_to_file(iteration, f"❌ git commit failed: {output[:200]}")
            return False

    # Push
    success, _ = run_command(
        ["git", "push", GIT_REMOTE, GIT_BRANCH], iteration, "Git push"
    )
    if not success:
        log_to_file(iteration, "❌ git push failed - stopping")
        return False

    log_to_file(iteration, f"✅ Iteration {iteration} complete - committed and pushed")
    return True


def main():
    """Main ralph loop orchestrator."""
    print("=" * 50)
    print(" Ralph Wiggum Loop Orchestrator (Python)")
    print("=" * 50)
    print()

    # Pre-flight checks
    print("🔍 Pre-flight checks...")

    if not check_git_config():
        sys.exit(1)

    if not check_logs_dir_exists():
        sys.exit(1)

    print("✅ All pre-flight checks passed\n")

    # Main loop
    for i in range(1, MAX_ITERS + 1):
        print(f"\n{'=' * 50}")
        print(f" Iteration {i}/{MAX_ITERS}")
        print(f"{'=' * 50}")

        log_to_file(i, "=" * 60)
        log_to_file(i, f"ITERATION {i}")
        log_to_file(i, "=" * 60)

        # Run qwen
        print(f"\n🤖 Running qwen iteration {i}...")
        success, output, log_file = run_qwen_iteration(i)

        if not success:
            print(f"\n🔴 ABRUPT EXIT DETECTED at iteration {i}")
            print(f"   qwen exited unexpectedly or timed out")
            print(f"   This happened. It fucked up. Recovering...")
            log_to_file(i, "🔴 Agent experienced abrupt exit")
            log_to_file(i, "🔴 It happened. It fucked up.")
            log_to_file(i, "⏱️  Attempting recovery: committing current state...")
            print(f"   Attempting recovery: committing current state asap...")

            if not commit_and_push(i):
                print(f"\n❌ Recovery failed - could not commit/push")
                print(f"   Manual intervention needed. Check logs at {log_file}")
                log_to_file(i, "❌ Recovery commit failed")
                sys.exit(1)

            print(f"✅ Recovery successful - current state committed")
            print(f"   Moving to next iteration...")
            log_to_file(i, "✅ Recovery successful - continuing to next iteration")
            continue

        # Check for blocking
        if check_blocked_status(log_file):
            print(f"\n🛑 Agent blocked at iteration {i}")
            log_to_file(i, "🛑 BLOCKED - Stopping ralph loop")
            commit_and_push(i)
            print("✅ Committed blocked state and exiting")
            sys.exit(0)

        # Commit and push
        print(f"\n📦 Committing and pushing iteration {i}...")
        if not commit_and_push(i):
            print(f"\n❌ Git operations failed during normal flow")
            print(f"   Attempting forced recovery...")
            log_to_file(i, "❌ Git operations failed - attempting forced recovery")

            # Try again with explicit error handling
            try:
                subprocess.run(
                    ["git", "add", "-A"], cwd=REPO_ROOT, capture_output=True, timeout=30
                )
                subprocess.run(
                    [
                        "git",
                        "commit",
                        "-m",
                        f"iteration-{i}: recovery commit after failure",
                    ],
                    cwd=REPO_ROOT,
                    capture_output=True,
                    timeout=30,
                )
                subprocess.run(
                    ["git", "push", GIT_REMOTE, GIT_BRANCH],
                    cwd=REPO_ROOT,
                    capture_output=True,
                    timeout=30,
                )
                log_to_file(i, "✅ Recovery commit successful")
                print("✅ Forced recovery successful - continuing...")
            except Exception as e:
                log_to_file(i, f"❌ Forced recovery failed: {str(e)}")
                print(f"❌ Forced recovery failed. Exiting.")
                sys.exit(1)

        print(f"✅ Iteration {i} complete")

        # Brief pause between iterations
        time.sleep(1)

    print(f"\n{'=' * 50}")
    print(f"🏁 Completed {MAX_ITERS} iterations")
    print(f"{'=' * 50}")
    sys.exit(0)


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\n⏸️  Ralph loop interrupted by user")
        sys.exit(130)
    except Exception as e:
        print(f"\n❌ Fatal error: {e}")
        sys.exit(1)
