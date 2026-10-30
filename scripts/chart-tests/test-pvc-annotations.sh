#!/usr/bin/env bash

set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
chart="$repo_root/charts/qubership-logging-operator"

assert_count() {
    local expected_count=$1
    local expected_text=$2
    local rendered_output=$3
    local context=$4
    local actual_count
    actual_count=$(grep -Fc -- "$expected_text" <<<"$rendered_output" || true)
    if [[ $actual_count -ne $expected_count ]]; then
        echo "Expected $context to contain '$expected_text' $expected_count time(s), found $actual_count."
        echo "$rendered_output"
        exit 1
    fi
}

render_pvcs() {
    helm template logging "$chart" \
        --show-only templates/graylog/pvc-graylog.yaml \
        --show-only templates/graylog/pvc-mongo.yaml \
        --set graylog.install=true \
        "$@"
}

pvcs=$(render_pvcs --set-string 'pvc.metadata.annotations.argocd\.argoproj\.io/sync-options=Prune=false')
assert_count 1 'app.kubernetes.io/name: graylog-claim' "$pvcs" "the PVCs"
assert_count 1 'app.kubernetes.io/name: mongo-claim' "$pvcs" "the PVCs"
assert_count 2 'argocd.argoproj.io/sync-options: Prune=false' "$pvcs" "the PVCs"

conflicting_pvcs=$(render_pvcs \
    --set-string graylog.annotations.owner=graylog \
    --set-string pvc.metadata.annotations.owner=pvc)
assert_count 2 'owner: pvc' "$conflicting_pvcs" "the PVCs with conflicting annotations"
assert_count 0 'owner: graylog' "$conflicting_pvcs" "the PVCs with conflicting annotations"

default_pvcs=$(render_pvcs)
assert_count 2 'kind: PersistentVolumeClaim' "$default_pvcs" "the default PVCs"
assert_count 0 'annotations:' "$default_pvcs" "the default PVCs"

# A non-empty graylog.annotations exposes a merge that mutates .Values and leaks into other resources.
whole_chart=$(helm template logging "$chart" --set graylog.install=true \
    --set-string graylog.annotations.owner=graylog \
    --set-string pvc.metadata.annotations.leak-check=leakval)
assert_count 2 'leak-check: leakval' "$whole_chart" "the whole chart"
