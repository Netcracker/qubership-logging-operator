#!/usr/bin/env bash
# Smell-check / inventory greps for qubership-ndjson-logging-migrate.
#
# Usage: smell-checks.sh [ROOT]
#   ROOT — component or repo root to scan (default: current directory).
#
# Single home for all check commands (inventory step, completion gates 2.2/3/4.1,
# review-pass re-check) so the copies cannot drift. Check meanings, production
# scopes, and misleading-zero caveats:
#   references/preformatted-message-patterns.md
# Gate targets: references/completion-gates.md
#
# Hits are review candidates, not failures; clean output is necessary but NEVER
# sufficient (see SKILL.md § Goal). Exit code is always 0.

set -u
ROOT="${1:-.}"
TOTAL=0

run_check() {
  local id="$1" title="$2" cmd="$3"
  echo
  echo "== ${id}: ${title}"
  local out
  out="$(eval "${cmd}" 2>/dev/null || true)"
  if [ -n "${out}" ]; then
    printf '%s\n' "${out}"
    local n
    n="$(printf '%s\n' "${out}" | wc -l)"
    echo "-- ${id}: ${n} hit(s)"
    TOTAL=$((TOTAL + n))
  else
    echo "-- ${id}: 0 hits"
  fi
}

# ---------------------------------------------------------------- Java scope
# Production scope = src/main/java subtrees when present, else whole ROOT.
JAVA_SCOPE=""
while IFS= read -r d; do
  JAVA_SCOPE="${JAVA_SCOPE} $(printf '%q' "${d}")"
done < <(find "${ROOT}" -type d -path '*/src/main/java' -not -path '*/.git/*' 2>/dev/null)
if [ -z "${JAVA_SCOPE}" ] && find "${ROOT}" -name '*.java' -not -path '*/.git/*' 2>/dev/null | head -1 | grep -q .; then
  JAVA_SCOPE="$(printf '%q' "${ROOT}")"
fi

if [ -n "${JAVA_SCOPE}" ]; then
  echo "Java scope:${JAVA_SCOPE}"
  JG="grep -rn --include='*.java'"

  run_check J1 "forbidden StructuredLog / per-call MDC.put for event fields" \
    "${JG} 'StructuredLog\|MDC\.put' ${JAVA_SCOPE}"

  run_check J2 "same-line SLF4J {} in log calls" \
    "${JG} -E 'log\.(info|debug|warn|error|trace)\([^)]*\{' ${JAVA_SCOPE}"

  run_check J3 "shared string constants still containing {} (misleading zero — stop and ask)" \
    "${JG} -E 'String\s+[A-Z][A-Z0-9_]*\s*=\s*\"[^\"]*\{\}' ${JAVA_SCOPE}"

  run_check J4 "preformatted message logs (log.warn(message), e.getMessage(), ...)" \
    "${JG} -E 'log\.(warn|error|debug|info)\((message|msg|aggregatedError|errorMsg|warn|e\.getMessage)' ${JAVA_SCOPE}"

  run_check J5 "text-block logs (open each hit: {} inside; not covered by J2)" \
    "${JG} -E 'log\.(info|debug|warn|error|trace)\(\"\"\"' ${JAVA_SCOPE}"

  run_check J6a "codemod residue field names (_get_ / _stream_ / e_get_message)" \
    "${JG} -E 'addKeyValue\(\"[^\"]*(_get_|_stream_|e_get_message)' ${JAVA_SCOPE}"

  run_check J6b "positional argN field names" \
    "${JG} '\"arg[0-9]\+\"' ${JAVA_SCOPE}"

  run_check J7 "illegal single-line text block on log lines (must not compile)" \
    "${JG} -E '\"\"\" [^\"]' ${JAVA_SCOPE} | grep -E 'log\.(at|info|debug|warn|error)'"

  run_check J8 "variable/expression as sole message argument" \
    "${JG} -E 'log\.(info|debug|warn|error|trace)\(\s*[^\"'\''{]' ${JAVA_SCOPE}"
else
  echo "Java scope: none found"
fi

# ------------------------------------------------------------------ Go scope
# Production scope: exclude _test.go, dev/, vendor/; ignore commented lines in review.
if find "${ROOT}" -name '*.go' -not -path '*/.git/*' 2>/dev/null | head -1 | grep -q .; then
  echo
  echo "Go scope: ${ROOT} (excluding _test.go, dev/, vendor/)"
  GG="grep -rn --include='*.go' --exclude='*_test.go' --exclude-dir=dev --exclude-dir=vendor"

  run_check G1 "formatted log calls log.*f( — include Trace" \
    "${GG} -E 'log\.(Trace|Debug|Info|Warn|Error|Fatal|Panic)f\(' $(printf '%q' "${ROOT}")"

  run_check G2 "residual printf verbs on non-f log calls (same-line; on any hit review the whole file)" \
    "${GG} -E 'log\.(Trace|Debug|Info|Warn|Error|Fatal|Panic)(C)?\(.*%[vTdoxXefg]' $(printf '%q' "${ROOT}")"

  run_check G3 "variable passed as message (logged preformatted)" \
    "${GG} -E 'log\.(Trace|Debug|Info|Warn|Error|Fatal|Panic)\([^\"'\'']' $(printf '%q' "${ROOT}")"
else
  echo
  echo "Go scope: none found"
fi

# -------------------------------------------------------------- Python scope
if find "${ROOT}" -name '*.py' -not -path '*/.git/*' 2>/dev/null | head -1 | grep -q .; then
  echo
  echo "Python scope: ${ROOT}"
  PG="grep -rn --include='*.py'"

  run_check P1 "non-literal logger message" \
    "${PG} -E 'logger\.(debug|info|warning|error|critical)\([^f\"'\'']' $(printf '%q' "${ROOT}")"

  run_check P2 "f-string logger calls" \
    "${PG} -E 'logger\.(debug|info|warning|error|critical)\(f\"' $(printf '%q' "${ROOT}")"
else
  echo
  echo "Python scope: none found"
fi

echo
echo "== TOTAL: ${TOTAL} hit(s) across all checks"
echo "Reminder: 0 hits everywhere is NOT 'migrated' — greps miss the fmt.Sprintf-then-%s dodge and"
echo "cannot see JSON placement. Run the placement probe, semantic gates, review pass, and smoke."
exit 0
