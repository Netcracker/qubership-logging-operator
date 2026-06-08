"""Stage-1 signal extraction for logging-l1-classification.

Deterministic. Parses the line-based signal dictionary (intent-signals.txt)
and matches ticket text. Standard library only — no third-party deps — so the
skill runs wherever python3 is present. Produces hints only; the model
(stage 2) always decides the label.

Dictionary format:
  # comment (only at start of a line)
  [problem] / [consultation] / [feature_request] / [security]   intent sections
  [component.<name>]                                            stack-component sections
  [platform.<name>]                                             substrate sections (vm / kubernetes)
  [phase.<name>]                                                lifecycle sections (deploy)
  [symptom.<name>] / [symptom.<component>.<spec>] / [symptom.deploy.<spec>]
  [topic.<name>]
  <phrase>        a literal phrase (case-insensitive, whitespace-flexible,
                  word boundaries at word-char edges)
  re: <regex>     a raw regex, compiled case-insensitive, verbatim
"""
import argparse
import json
import os
import re

INTENT_CATEGORIES = ("problem", "consultation", "feature_request", "security")

LABEL_NAMESPACES = ("component", "platform", "phase", "symptom", "topic")

_WORD = re.compile(r"\w")


def default_signals_path():
    here = os.path.abspath(os.path.dirname(__file__))
    return os.path.join(here, "..", "references", "signals.txt")


def _compile_phrase(phrase):
    parts = re.split(r"\s+", phrase.strip())
    body = r"\s+".join(re.escape(p) for p in parts)
    prefix = r"\b" if _WORD.match(phrase[0]) else ""
    suffix = r"\b" if _WORD.match(phrase[-1]) else ""
    return re.compile(prefix + body + suffix, re.IGNORECASE)


def load(path=None):
    """Parse the dictionary into {section_name: [compiled regex, ...]}."""
    path = path or default_signals_path()
    sections = {}
    current = None
    with open(path, encoding="utf-8") as fh:
        for raw in fh:
            line = raw.strip()
            if not line or line.startswith("#"):
                continue
            if line.startswith("[") and line.endswith("]"):
                current = line[1:-1].strip()
                sections.setdefault(current, [])
                continue
            if current is None:
                continue
            if line.startswith("re:"):
                sections[current].append(re.compile(line[3:].strip(), re.IGNORECASE))
            else:
                sections[current].append(_compile_phrase(line))
    return sections


def _dedup(seq):
    seen = set()
    out = []
    for item in seq:
        if item not in seen:
            seen.add(item)
            out.append(item)
    return out


def match_text(text, sections):
    text = text or ""
    result = {}
    for cat in INTENT_CATEGORIES:
        hits = []
        for rx in sections.get(cat, []):
            hits.extend(m.group(0) for m in rx.finditer(text))
        result[cat] = _dedup(hits)
    for ns in LABEL_NAMESPACES:
        prefix = ns + "."
        labels = []
        for name, regexes in sections.items():
            if not name.startswith(prefix):
                continue
            label = name[len(prefix):]
            if any(rx.search(text) for rx in regexes):
                labels.append(label)
        result[ns] = _dedup(labels)
    return result


def main(argv=None):
    ap = argparse.ArgumentParser(
        description="Stage-1 signal extraction for logging-l1-classification."
    )
    ap.add_argument("ticket_file", help="file holding one ticket's text (Summary + Description)")
    ap.add_argument("--signals", default=None, help="path to intent-signals.txt")
    args = ap.parse_args(argv)
    with open(args.ticket_file, encoding="utf-8") as fh:
        text = fh.read()
    print(json.dumps(match_text(text, load(args.signals)), ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
