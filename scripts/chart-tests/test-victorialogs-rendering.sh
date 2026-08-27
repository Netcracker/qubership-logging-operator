#!/usr/bin/env bash

set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
chart="$repo_root/charts/qubership-victorialogs"

assert_contains() {
    local expected=$1
    local rendered_output=$2
    local context=$3
    if ! grep -Fq -- "$expected" <<<"$rendered_output"; then
        echo "Expected $context to contain: $expected"
        echo "$rendered_output"
        exit 1
    fi
}

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

expect_schema_failure() {
    local expected_error=$1
    shift
    local validation_output
    if validation_output=$(helm template logging "$chart" "$@" 2>&1); then
        echo "The chart accepted values that should fail schema validation: $*."
        echo "$validation_output"
        exit 1
    fi
    assert_contains "$expected_error" "$validation_output" "the schema validation error"
}

expect_schema_failure "additional properties 'storageConfig' not allowed" \
    --set victorialogs.storageConfig.size=1Gi
expect_schema_failure "at '/victorialogs/service/port': got string, want integer" \
    --set-string victorialogs.service.port=not-a-port
expect_schema_failure "additional properties 'limit' not allowed" \
    --set victorialogs.tmpVolume.limit=64Mi
expect_schema_failure "at '/victorialogs/vmauth/tmpVolume/sizeLimit': got boolean, want string" \
    --set victorialogs.vmauth.tmpVolume.sizeLimit=true

pvc=$(helm template logging "$chart" --show-only templates/pvc.yaml \
    --set victorialogs.install=true)
assert_contains 'helm.sh/resource-policy: keep' "$pvc" "the PVC"

services=$(helm template logging "$chart" \
    --show-only templates/service.yaml \
    --show-only templates/service-headless.yaml \
    --set victorialogs.install=true)
assert_count 2 'type: ClusterIP' "$services" "the Services"
assert_count 1 'clusterIP: None' "$services" "the Services"

statefulset=$(helm template logging "$chart" --show-only templates/statefulset.yaml \
    --set victorialogs.install=true \
    --set-string 'victorialogs.podLabels.app\.kubernetes\.io/component=invalid')
assert_contains 'serviceName: victorialogs-headless' "$statefulset" "the StatefulSet"
assert_contains '--retentionPeriod=1' "$statefulset" "the StatefulSet"
assert_contains 'app.kubernetes.io/component: victorialogs' "$statefulset" "the StatefulSet"
assert_count 2 'runAsNonRoot: true' "$statefulset" "the StatefulSet security contexts"
assert_count 1 'runAsUser: 1000' "$statefulset" "the StatefulSet Pod security context"
assert_count 1 'runAsGroup: 1000' "$statefulset" "the StatefulSet Pod security context"
assert_count 1 'type: RuntimeDefault' "$statefulset" "the StatefulSet Pod security context"
assert_count 1 'allowPrivilegeEscalation: false' "$statefulset" "the StatefulSet container security context"
assert_count 1 'readOnlyRootFilesystem: true' "$statefulset" "the StatefulSet container security context"
assert_count 1 'mountPath: /tmp' "$statefulset" "the StatefulSet temporary volume mount"
assert_count 1 'sizeLimit: 100Mi' "$statefulset" "the StatefulSet temporary volume"
assert_count 1 'cpu: 50m' "$statefulset" "the StatefulSet CPU request"
assert_count 1 'memory: 64Mi' "$statefulset" "the StatefulSet memory request"
assert_count 1 'ephemeral-storage: 100Mi' "$statefulset" "the StatefulSet storage request"
assert_count 1 'memory: 512Mi' "$statefulset" "the StatefulSet memory limit"
if grep -Fq 'app.kubernetes.io/component: invalid' <<<"$statefulset"; then
    echo "Selector label app.kubernetes.io/component was overridden by podLabels."
    exit 1
fi

custom_tmp_statefulset=$(helm template logging "$chart" --show-only templates/statefulset.yaml \
    --set-string victorialogs.tmpVolume.sizeLimit=64Mi)
assert_count 1 'sizeLimit: 64Mi' "$custom_tmp_statefulset" "the customized StatefulSet temporary volume"

openshift_statefulset=$(helm template logging "$chart" --show-only templates/statefulset.yaml \
    --api-versions security.openshift.io/v1)
if grep -Eq '^[[:space:]]+(fsGroup|runAsGroup|runAsNonRoot|runAsUser):' <<<"$openshift_statefulset"; then
    echo "The VictoriaLogs StatefulSet rendered identity constraints that must be controlled by the OpenShift SCC."
    echo "$openshift_statefulset"
    exit 1
fi
assert_count 1 'type: RuntimeDefault' "$openshift_statefulset" "the OpenShift StatefulSet Pod security context"
assert_count 1 'readOnlyRootFilesystem: true' "$openshift_statefulset" \
    "the OpenShift StatefulSet container security context"
assert_count 1 'mountPath: /tmp' "$openshift_statefulset" "the OpenShift StatefulSet temporary volume mount"

dashboard=$(helm template logging "$chart" --namespace logging \
    --show-only templates/grafanadashboard.yaml \
    --set victorialogs.install=true \
    --set victorialogs.dashboard.install=true)
assert_contains 'apiVersion: grafana.integreatly.org/v1beta1' "$dashboard" "the dashboard"
assert_contains 'allowCrossNamespaceImport: true' "$dashboard" "the dashboard"
assert_contains 'app.kubernetes.io/component: grafana' "$dashboard" "the dashboard"
assert_contains 'app.kubernetes.io/part-of: monitoring' "$dashboard" "the dashboard"
if grep -Fq 'apiVersion: integreatly.org/v1alpha1' <<<"$dashboard"; then
    echo "VictoriaLogs rendered a Grafana Operator v4 dashboard resource."
    exit 1
fi

default_render=$(helm template logging "$chart")
if grep -Eq '^kind: (GrafanaDashboard|ServiceMonitor)$' <<<"$default_render"; then
    echo "Optional GrafanaDashboard or ServiceMonitor resource rendered by default."
    echo "$default_render"
    exit 1
fi

vmauth=$(helm template logging "$chart" --namespace logging \
    --show-only templates/vmauth-secret.yaml \
    --show-only templates/vmauth-deployment.yaml \
    --show-only templates/vmauth-service.yaml \
    --show-only templates/ingress.yaml \
    --show-only templates/httproute.yaml \
    --set victorialogs.install=true \
    --set victorialogs.ingress.install=true \
    --set victorialogs.httpRoute.install=true \
    --set-string CLOUD_PUBLIC_HOST=apps.example.com \
    --set-string victorialogs.vmauth.config.users[0].username=viewer \
    --set-string victorialogs.vmauth.config.users[0].password=strong-password \
    --set-string 'victorialogs.vmauth.podLabels.app\.kubernetes\.io/component=invalid')

assert_count 2 'vmauth-logging.apps.example.com' "$vmauth" "the external routes"
assert_contains 'name: vmauth-victorialogs' "$vmauth" "the VMAuth resources"
assert_contains 'kind: Secret' "$vmauth" "the generated VMAuth configuration"
assert_contains 'port: 8427' "$vmauth" "the VMAuth resources"
assert_contains '-auth.config=/etc/vmauth/auth.yml' "$vmauth" "the VMAuth Deployment"
assert_contains 'mountPath: /etc/vmauth' "$vmauth" "the VMAuth Deployment"
assert_contains 'readOnly: true' "$vmauth" "the VMAuth Deployment"
assert_contains 'secretName: vmauth-victorialogs' "$vmauth" "the VMAuth Deployment"
assert_count 2 'runAsNonRoot: true' "$vmauth" "the VMAuth security contexts"
assert_count 1 'runAsUser: 1000' "$vmauth" "the VMAuth Pod security context"
assert_count 1 'runAsGroup: 1000' "$vmauth" "the VMAuth Pod security context"
assert_count 1 'type: RuntimeDefault' "$vmauth" "the VMAuth Pod security context"
assert_count 1 'allowPrivilegeEscalation: false' "$vmauth" "the VMAuth container security context"
assert_count 1 'readOnlyRootFilesystem: true' "$vmauth" "the VMAuth container security context"
assert_count 1 'mountPath: /tmp' "$vmauth" "the VMAuth temporary volume mount"
assert_count 1 'sizeLimit: 100Mi' "$vmauth" "the VMAuth temporary volume"
assert_count 1 'cpu: 20m' "$vmauth" "the VMAuth CPU request"
assert_count 1 'memory: 32Mi' "$vmauth" "the VMAuth memory request"
assert_count 1 'ephemeral-storage: 100Mi' "$vmauth" "the VMAuth storage request"
assert_count 1 'memory: 128Mi' "$vmauth" "the VMAuth memory limit"
if grep -Eq '^[[:space:]]+env(From)?:' <<<"$vmauth"; then
    echo "VMAuth rendered environment variables instead of a read-only Secret file."
    exit 1
fi

custom_tmp_vmauth=$(helm template logging "$chart" --show-only templates/vmauth-deployment.yaml \
    --set victorialogs.ingress.install=true \
    --set-string CLOUD_PUBLIC_HOST=apps.example.com \
    --set-string victorialogs.vmauth.existingSecret=external-vmauth-config \
    --set-string victorialogs.vmauth.tmpVolume.sizeLimit=32Mi)
assert_count 1 'sizeLimit: 32Mi' "$custom_tmp_vmauth" "the customized VMAuth temporary volume"
vmauth_config=$(sed -n 's/^  auth.yml: "\(.*\)"$/\1/p' <<<"$vmauth" | base64 --decode)
assert_contains 'password: strong-password' "$vmauth_config" "the generated auth.yml"
assert_contains 'url_prefix: http://victorialogs:9428/' "$vmauth_config" "the generated auth.yml"
if grep -Fq 'app.kubernetes.io/component: invalid' <<<"$vmauth"; then
    echo "Selector label app.kubernetes.io/component was overridden in the VMAuth Pod."
    exit 1
fi

openshift_vmauth=$(helm template logging "$chart" --show-only templates/vmauth-deployment.yaml \
    --api-versions security.openshift.io/v1 \
    --set victorialogs.ingress.install=true \
    --set-string CLOUD_PUBLIC_HOST=apps.example.com \
    --set-string victorialogs.vmauth.existingSecret=external-vmauth-config)
if grep -Eq '^[[:space:]]+(fsGroup|runAsGroup|runAsNonRoot|runAsUser):' <<<"$openshift_vmauth"; then
    echo "The VMAuth Deployment rendered identity constraints that must be controlled by the OpenShift SCC."
    echo "$openshift_vmauth"
    exit 1
fi
assert_count 1 'type: RuntimeDefault' "$openshift_vmauth" "the OpenShift VMAuth Pod security context"
assert_count 1 'readOnlyRootFilesystem: true' "$openshift_vmauth" \
    "the OpenShift VMAuth container security context"
assert_count 1 'mountPath: /tmp' "$openshift_vmauth" "the OpenShift VMAuth temporary volume mount"

explicit_hosts=$(helm template logging "$chart" --namespace logging \
    --show-only templates/ingress.yaml \
    --show-only templates/httproute.yaml \
    --set victorialogs.install=true \
    --set victorialogs.ingress.install=true \
    --set-string victorialogs.ingress.hosts[0].host=ingress.example.org \
    --set victorialogs.httpRoute.install=true \
    --set-string victorialogs.httpRoute.hostnames[0]=route.example.org \
    --set-string victorialogs.vmauth.config.users[0].bearer_token=secret)

assert_contains 'ingress.example.org' "$explicit_hosts" "the explicit Ingress host"
assert_contains 'path: /' "$explicit_hosts" "the per-host default Ingress path"
assert_contains 'pathType: Prefix' "$explicit_hosts" "the per-host default Ingress path type"
assert_contains 'route.example.org' "$explicit_hosts" "the explicit HTTPRoute hostname"

existing_secret=$(helm template logging "$chart" --namespace logging \
    --set victorialogs.ingress.install=true \
    --set-string CLOUD_PUBLIC_HOST=apps.example.com \
    --set-string victorialogs.vmauth.existingSecret=external-vmauth-config \
    --set-string victorialogs.vmauth.existingSecretKey=custom.yml)
assert_contains 'secretName: external-vmauth-config' "$existing_secret" "the existing Secret reference"
assert_contains 'key: "custom.yml"' "$existing_secret" "the existing Secret key reference"
if grep -Fq 'kind: Secret' <<<"$existing_secret"; then
    echo "The chart generated a VMAuth Secret while existingSecret was set."
    echo "$existing_secret"
    exit 1
fi

if helm template logging "$chart" \
    --set victorialogs.install=true \
    --set victorialogs.ingress.install=true \
    --set-string CLOUD_PUBLIC_HOST=apps.example.com >/dev/null 2>&1; then
    echo "VictoriaLogs Ingress rendered without an authenticated VMAuth user."
    exit 1
fi

expect_vmauth_auth_failure() {
    local case_name=$1
    shift
    if helm template logging "$chart" \
        --set victorialogs.install=true \
        --set victorialogs.ingress.install=true \
        --set-string CLOUD_PUBLIC_HOST=apps.example.com \
        "$@" >/dev/null 2>&1; then
        echo "VMAuth accepted invalid authentication configuration: $case_name."
        exit 1
    fi
}

expect_vmauth_auth_failure "name without credentials" \
    --set-string victorialogs.vmauth.config.users[0].name=viewer
expect_vmauth_auth_failure "username without password" \
    --set-string victorialogs.vmauth.config.users[0].username=viewer
expect_vmauth_auth_failure "password without username" \
    --set-string victorialogs.vmauth.config.users[0].password=secret
expect_vmauth_auth_failure "empty password" \
    --set-string victorialogs.vmauth.config.users[0].username=viewer \
    --set-string victorialogs.vmauth.config.users[0].password=
expect_vmauth_auth_failure "multiple authentication methods" \
    --set-string victorialogs.vmauth.config.users[0].username=viewer \
    --set-string victorialogs.vmauth.config.users[0].password=secret \
    --set-string victorialogs.vmauth.config.users[0].bearer_token=secret
expect_vmauth_auth_failure "helper validation without schema" \
    --skip-schema-validation \
    --set-string victorialogs.vmauth.config.users[0].username=viewer
