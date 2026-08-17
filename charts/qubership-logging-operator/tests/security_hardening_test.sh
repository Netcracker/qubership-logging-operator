#!/usr/bin/env bash
# Smoke test for container security hardening (readOnlyRootFilesystem, non-root,
# seccomp, dropped capabilities, mandatory /tmp mounts) across the Helm-rendered
# workloads (operator, integration-tests) and the legacy PSP/SCC policies.
#
# The FluentBit/FluentD/Graylog/EventsReader workloads are rendered by the
# operator itself (controllers/*/assets/*.yaml), not by this Helm chart, so
# they are out of scope for this script.
set -euo pipefail

CHART_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RENDERED="$(mktemp)"
OPENSHIFT_RENDERED="$(mktemp)"
trap 'rm -f "${RENDERED}" "${OPENSHIFT_RENDERED}"' EXIT

helm template "${CHART_DIR}" \
    --set integrationTests.install=true \
    --set victorialogs.install=true \
    --set victorialogs.storage.existingClaim=victorialogs \
    --set victorialogs.ingress.install=true \
    --set victorialogs.ingress.hosts[0].host=vmauth.test.local \
    --set victorialogs.ingress.hosts[0].paths[0].path=/ \
    --set victorialogs.ingress.hosts[0].paths[0].pathType=Prefix \
    --set victorialogs.vmauth.config.users[0].username=test \
    --set victorialogs.vmauth.config.users[0].password=test \
    --set graylog.install=true \
    --set createClusterAdminEntities=true \
    --set fluentbit.install=true \
    --set fluentbit.securityResources.install=true \
    --set fluentd.install=true \
    --set fluentd.securityResources.install=true \
    --set fluentbitHA.install=true \
    --set fluentbitHA.securityResources.install=true \
    --api-versions policy/v1beta1 \
    >"${RENDERED}"

fail=0

check_min_count() {
    local pattern="$1" min="$2" description="$3"
    local count
    count=$(grep -c -- "${pattern}" "${RENDERED}" || true)
    if [ "${count}" -lt "${min}" ]; then
        echo "FAIL: expected at least ${min} occurrence(s) of '${pattern}' (${description}), found ${count}"
        fail=1
    else
        echo "PASS: ${description} (${count} occurrence(s))"
    fi
}

check_absent() {
    local pattern="$1" description="$2"
    if grep -q -- "${pattern}" "${RENDERED}"; then
        echo "FAIL: found forbidden '${pattern}' (${description})"
        fail=1
    else
        echo "PASS: no ${description}"
    fi
}

check_following_line() {
    local pattern="$1" expected="$2" description="$3"
    if grep -A1 -- "${pattern}" "${RENDERED}" | grep -q -- "${expected}"; then
        echo "PASS: ${description}"
    else
        echo "FAIL: '${pattern}' is not followed by '${expected}' (${description})"
        fail=1
    fi
}

check_min_count "readOnlyRootFilesystem: true" 4 "readOnlyRootFilesystem enabled"
check_min_count "runAsNonRoot: true" 8 "runAsNonRoot enabled"
check_min_count "allowPrivilegeEscalation: false" 4 "allowPrivilegeEscalation disabled"
check_min_count "type: RuntimeDefault" 4 "seccompProfile RuntimeDefault"
check_min_count "- ALL" 4 "capabilities dropped (ALL)"
check_min_count "name: tmp" 8 "mandatory /tmp emptyDir volumes and mounts"
check_min_count "ephemeral-storage: 200Mi" 1 "operator ephemeral-storage limit"
check_min_count "runAsUser: 2001" 1 "operator runs with its image UID"
check_min_count "runAsGroup: 1000" 3 "workloads run with their image group"
check_following_line "pathPrefix: /var/log" "readOnly: false" \
    "Fluentd PSP permits its legacy position files under /var/log to remain writable"

check_absent "hostNetwork: true" "hostNetwork: true"
check_absent "hostPID: true" "hostPID: true"

helm template "${CHART_DIR}" \
    --set openshiftDeploy=true \
    --set integrationTests.install=true \
    --set victorialogs.install=true \
    --set victorialogs.storage.existingClaim=victorialogs \
    --set victorialogs.ingress.install=true \
    --set victorialogs.ingress.hosts[0].host=vmauth.test.local \
    --set victorialogs.ingress.hosts[0].paths[0].path=/ \
    --set victorialogs.ingress.hosts[0].paths[0].pathType=Prefix \
    --set victorialogs.vmauth.config.users[0].username=test \
    --set victorialogs.vmauth.config.users[0].password=test \
    --api-versions security.openshift.io/v1 \
    >"${OPENSHIFT_RENDERED}"

openshift_group_count=$(grep -c -- "runAsGroup: 1000" "${OPENSHIFT_RENDERED}" || true)
if [ "${openshift_group_count}" -lt 4 ]; then
    echo "FAIL: expected OpenShift workloads to retain GID 1000, found ${openshift_group_count} occurrence(s)"
    fail=1
else
    echo "PASS: OpenShift workloads retain GID 1000 (${openshift_group_count} occurrence(s))"
fi

if grep -q -- "runAsUser: 1000" "${OPENSHIFT_RENDERED}"; then
    echo "FAIL: OpenShift workload pins UID 1000 instead of using an SCC-assigned UID"
    fail=1
else
    echo "PASS: OpenShift workloads do not pin UID 1000"
fi

# Graylog's legacy PSP and SCC must permit a writable root filesystem for the
# setup init container. Application workloads are covered by the positive
# readOnlyRootFilesystem assertion above and controller-level unit tests.

if [ "${fail}" -ne 0 ]; then
    echo "Security hardening smoke test FAILED"
    exit 1
fi

echo "Security hardening smoke test PASSED"
