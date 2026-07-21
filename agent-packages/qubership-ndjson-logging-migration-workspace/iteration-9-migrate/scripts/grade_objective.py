#!/usr/bin/env python3
"""Run objective gate checks for iteration-9 migrate grading (skill-creator supplement)."""

from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
WORKSPACE = ROOT.parent
GATES = WORKSPACE / "scripts/check_migration_gates.py"
CONFIG = json.loads((ROOT / "config.json").read_text(encoding="utf-8"))


def run_gates(worktree: Path, gate_args: list[str]) -> dict:
    cmd = [sys.executable, str(GATES), str(worktree), *gate_args]
    proc = subprocess.run(cmd, capture_output=True, text=True)
    return {
        "command": " ".join(cmd),
        "exit_code": proc.returncode,
        "stdout": proc.stdout,
        "stderr": proc.stderr,
        "passed": proc.returncode == 0,
    }


def main() -> int:
    if len(sys.argv) < 3:
        print(f"usage: {sys.argv[0]} <arm> <eval-name>", file=sys.stderr)
        print("  arm: with_skill | without_skill", file=sys.stderr)
        return 2
    arm, eval_name = sys.argv[1], sys.argv[2]
    ev = next(e for e in CONFIG["evals"] if e["name"] == eval_name)
    wt_key = "with_skill_worktree" if arm == "with_skill" else "baseline_worktree"
    worktree = Path(ev[wt_key])
    gate_args = ev.get("gate_args", [])
    result = run_gates(worktree, gate_args)
    out = {
        "iteration": CONFIG["iteration"],
        "stage": CONFIG["stage"],
        "eval": eval_name,
        "arm": arm,
        "worktree": str(worktree),
        **result,
    }
    print(json.dumps(out, indent=2))
    return 0 if result["passed"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
