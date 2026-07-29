#!/usr/bin/env bash
# Deploy qubership-logging-operator locally and verify app.kubernetes.io/* labels on created resources.
# Follows the official README (Helm repo add + install from repo) and Installation Guide (namespace labels).
# Prerequisites: kubectl, helm, cluster (e.g. kind) with kubectl context set.
# Optional: OPERATOR_IMAGE=custom image; USE_LOCAL_CHART=1 to install from local chart (default when repo is unreachable).
# Requires: jq (for label verification).

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
NAMESPACE="${NAMESPACE:-logging}"
# Official Helm repo (README: https://github.com/Netcracker/qubership-logging-operator#installation)
HELM_REPO_NAME="qubership-logging-operator"
HELM_REPO_URL="https://netcracker.github.io/qubership-logging-operator"
CHART_PATH="${CHART_PATH:-$REPO_ROOT/charts/qubership-logging-operator}"
VALUES_PATH="$SCRIPT_DIR/values-label-check.yaml"

echo "=== Checking cluster and tools ==="
if ! kubectl cluster-info &>/dev/null; then
  echo "Error: kubectl cluster-info failed. Ensure a cluster is running and kubeconfig is set."
  exit 1
fi
if ! helm version &>/dev/null; then
  echo "Error: helm not found."
  exit 1
fi
if ! command -v jq &>/dev/null; then
  echo "Error: jq not found. Install jq for label verification."
  exit 1
fi

echo "=== Creating namespace $NAMESPACE (with privileged PSS if supported) ==="
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
# Kubernetes 1.25+ often requires privileged PSS for logging agents (hostPath, etc.)
kubectl label namespace "$NAMESPACE" \
  pod-security.kubernetes.io/enforce=privileged \
  pod-security.kubernetes.io/enforce-version=latest \
  --overwrite 2>/dev/null || true

# Chart creates GrafanaDashboard, PodMonitor, PrometheusRule; install their CRDs first (from monitoring-operator).
if [[ -z "${SKIP_CRDS:-}" ]]; then
  echo "=== Installing monitoring CRDs (GrafanaDashboard, PodMonitor, PrometheusRule, ServiceMonitor) ==="
  MONITORING_CRDS_BASE="https://raw.githubusercontent.com/Netcracker/qubership-monitoring-operator/refs/heads/main/charts/qubership-monitoring-operator"
  kubectl apply -f "${MONITORING_CRDS_BASE}/charts/grafana-operator/crds/integreatly.org_grafanadashboards.yaml" 2>/dev/null || true
  kubectl apply -f "${MONITORING_CRDS_BASE}/charts/victoriametrics-operator/crds/monitoring.coreos.com_prometheusrules.yaml" 2>/dev/null || true
  kubectl apply -f "${MONITORING_CRDS_BASE}/charts/victoriametrics-operator/crds/monitoring.coreos.com_servicemonitors.yaml" 2>/dev/null || true
  kubectl apply -f "${MONITORING_CRDS_BASE}/charts/victoriametrics-operator/crds/monitoring.coreos.com_podmonitors.yaml" 2>/dev/null || true
fi

EXTRA_SET=""
if [[ -n "${OPERATOR_IMAGE:-}" ]]; then
  EXTRA_SET="--set operatorImage=$OPERATOR_IMAGE"
fi

if [[ -n "${USE_LOCAL_CHART:-}" ]]; then
  echo "=== Installing chart from local path: $CHART_PATH ==="
  helm upgrade --install qubership-logging-operator "$CHART_PATH" \
    -n "$NAMESPACE" \
    -f "$VALUES_PATH" \
    --create-namespace \
    $EXTRA_SET
else
  echo "=== Adding Helm repo (official README) ==="
  if ! helm repo add "$HELM_REPO_NAME" "$HELM_REPO_URL" 2>/dev/null; then
    if [[ -d "$CHART_PATH" ]]; then
      echo "Helm repo unreachable; using local chart: $CHART_PATH"
      echo "  (Set USE_LOCAL_CHART=1 to skip repo add next time.)"
      helm upgrade --install qubership-logging-operator "$CHART_PATH" \
        -n "$NAMESPACE" \
        -f "$VALUES_PATH" \
        --create-namespace \
        $EXTRA_SET
    else
      echo "Error: failed to add Helm repo and no local chart at $CHART_PATH"
      echo "  Check network and URL: $HELM_REPO_URL"
      echo "  Or set USE_LOCAL_CHART=1 and run from repo root to use local chart."
      exit 1
    fi
  else
    helm repo update
    echo "=== Installing chart from repo $HELM_REPO_NAME/qubership-logging-operator ==="
    helm upgrade --install qubership-logging-operator "$HELM_REPO_NAME/qubership-logging-operator" \
      -n "$NAMESPACE" \
      -f "$VALUES_PATH" \
      --create-namespace \
      $EXTRA_SET
  fi
fi

echo "=== Waiting for operator pod to be Ready (up to 120s) ==="
kubectl wait --for=condition=Ready pod -l name=logging-service-operator -n "$NAMESPACE" --timeout=120s 2>/dev/null || {
  echo "Operator pod not ready in time. Current pods:"
  kubectl get pods -n "$NAMESPACE"
  echo ""
  IMG=$(kubectl get pod -l name=logging-service-operator -n "$NAMESPACE" -o jsonpath='{.items[0].spec.containers[0].image}' 2>/dev/null || true)
  if [[ -n "$IMG" ]]; then
    echo "Operator image in use: $IMG"
  fi
  echo ""
  echo "If status is ImagePullBackOff: the cluster cannot pull the image."
  echo "  - For a local image (e.g. kind): build it, load it, then run with OPERATOR_IMAGE=qubership-logging-operator:local"
  echo "    e.g.  docker build -t qubership-logging-operator:local . && kind load docker-image qubership-logging-operator:local"
  echo "  - For the default repo image: ensure the cluster has network access to the chart's default registry."
  exit 1
}

echo "=== Waiting 30s for operator to reconcile (create DaemonSet/Deployment/Service) ==="
sleep 30

echo "=== Verifying labels on workloads and services ==="
FAILED=0

check_labels() {
  local kind=$1
  local name=$2
  local expected_labels="$3"
  echo "--- $kind/$name ---"
  local json
  json=$(kubectl get "$kind" "$name" -n "$NAMESPACE" -o json 2>/dev/null) || {
    echo "  (not found; may still be reconciling)"
    return
  }
  # Required labels: show value or MISSING
  echo "  Required (resource metadata):"
  for key in $expected_labels; do
    [[ -z "$key" ]] && continue
    if echo "$json" | jq -e ".metadata.labels[\"$key\"]" &>/dev/null; then
      val=$(echo "$json" | jq -r ".metadata.labels[\"$key\"]")
      echo "    $key: $val"
    else
      echo "    *** MISSING: $key ***"
      FAILED=1
    fi
  done
  # All other labels on the resource (sorted)
  echo "  All labels (resource metadata):"
  echo "$json" | jq -r '
    if .metadata.labels then
      .metadata.labels | to_entries | sort_by(.key)[] | "    \(.key): \(.value)"
    else
      "    (none)"
    end
  '
  # For workloads, also check spec.template.metadata.labels
  if [[ "$kind" == "Deployment" || "$kind" == "DaemonSet" || "$kind" == "StatefulSet" ]]; then
    local pod_required="name app.kubernetes.io/name app.kubernetes.io/component app.kubernetes.io/part-of app.kubernetes.io/managed-by app.kubernetes.io/technology app.kubernetes.io/instance app.kubernetes.io/version"
    echo "  Required (pod template):"
    for key in $pod_required; do
      if echo "$json" | jq -e ".spec.template.metadata.labels[\"$key\"]" &>/dev/null; then
        val=$(echo "$json" | jq -r ".spec.template.metadata.labels[\"$key\"]")
        echo "    $key: $val"
      else
        echo "    *** MISSING: $key ***"
        FAILED=1
      fi
    done
    echo "  All labels (pod template):"
    echo "$json" | jq -r '
      if .spec.template.metadata.labels then
        .spec.template.metadata.labels | to_entries | sort_by(.key)[] | "    \(.key): \(.value)"
      else
        "    (none)"
      end
    '
  fi
  echo ""
}

# Required labels per skill: name, app.kubernetes.io/name, part-of, managed-by; for workloads + instance, version, component, technology
REQUIRED_ALL="name app.kubernetes.io/name app.kubernetes.io/part-of app.kubernetes.io/managed-by"

check_labels Deployment logging-service-operator "$REQUIRED_ALL app.kubernetes.io/component app.kubernetes.io/instance app.kubernetes.io/version app.kubernetes.io/technology"
check_labels DaemonSet logging-fluentbit "$REQUIRED_ALL app.kubernetes.io/component app.kubernetes.io/technology"
check_labels Deployment events-reader "$REQUIRED_ALL app.kubernetes.io/component app.kubernetes.io/technology"
check_labels Service logging-service-operator "$REQUIRED_ALL app.kubernetes.io/component"
check_labels Service logging-fluentbit "$REQUIRED_ALL app.kubernetes.io/component"
check_labels Service events-reader "$REQUIRED_ALL app.kubernetes.io/component"

echo ""
if [[ $FAILED -eq 1 ]]; then
  echo "=== Some required labels were missing (see MISSING above). ==="
  exit 1
fi
echo "=== All checked resources have required labels. ==="
