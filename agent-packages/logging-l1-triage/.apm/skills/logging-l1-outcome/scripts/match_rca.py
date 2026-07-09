"""Deterministic RCA-case matcher for logging-l1-outcome.

Mirrors the intent skill's extract_signals.py. Parses the line-based
rca-cases.txt (one [<case-id>] section per known problem, with bare phrase or
"re:" regex lines) and returns the ids of cases whose pattern matches the ticket
text (plus any attachment text). Standard library only. A match is a HINT — the
model reads references/rca-cases.md for the matched case's cause and draft reply
and decides whether it truly applies.
"""
import argparse
import json
import os
import re

_WORD = re.compile(r"\w")


def default_cases_path():
    here = os.path.abspath(os.path.dirname(__file__))
    return os.path.join(here, "..", "references", "rca-cases.txt")


def _compile_phrase(phrase):
    parts = re.split(r"\s+", phrase.strip())
    body = r"\s+".join(re.escape(p) for p in parts)
    prefix = r"\b" if _WORD.match(phrase[0]) else ""
    suffix = r"\b" if _WORD.match(phrase[-1]) else ""
    return re.compile(prefix + body + suffix, re.IGNORECASE)


def load(path=None):
    """Parse rca-cases.txt into {case_id: [compiled regex, ...]}."""
    path = path or default_cases_path()
    cases = {}
    current = None
    with open(path, encoding="utf-8") as fh:
        for raw in fh:
            line = raw.strip()
            if not line or line.startswith("#"):
                continue
            if line.startswith("[") and line.endswith("]"):
                current = line[1:-1].strip()
                cases.setdefault(current, [])
                continue
            if current is None:
                continue
            if line.startswith("re:"):
                cases[current].append(re.compile(line[3:].strip(), re.IGNORECASE))
            else:
                cases[current].append(_compile_phrase(line))
    return cases


def match_text(text, cases):
    """Return ids of cases with at least one matching pattern, dedup, in order."""
    text = text or ""
    out = []
    seen = set()
    for cid, regexes in cases.items():
        if cid in seen:
            continue
        if any(rx.search(text) for rx in regexes):
            seen.add(cid)
            out.append(cid)
    return out


def main(argv=None):
    ap = argparse.ArgumentParser(description="RCA-case matcher for logging-l1-outcome.")
    ap.add_argument("ticket_file", help="file with ticket text (+ attachment text)")
    ap.add_argument("--cases", default=None, help="path to rca-cases.txt")
    args = ap.parse_args(argv)
    with open(args.ticket_file, encoding="utf-8") as fh:
        text = fh.read()
    print(json.dumps(match_text(text, load(args.cases)), ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
