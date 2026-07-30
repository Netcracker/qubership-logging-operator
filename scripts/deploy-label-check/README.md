# Deploy locally and verify labels

Deploy `qubership-logging-operator` to a local cluster and verify that required
`app.kubernetes.io/*` labels are set on created resources (operator Deployment,
fluentbit DaemonSet, events-reader Deployment, and their Services).

The script follows:

- **[Official README – Installation](https://github.com/Netcracker/qubership-logging-operator#installation)**:
  add Helm repo, install the operator from the published chart
  (`qubership-logging-operator/qubership-logging-operator`).
- **[Installation Guide](https://netcracker.github.io/qubership-logging-operator/installation/)**:
  namespace preparation and `privileged` PSS labels for Kubernetes 1.25+.
- **Integration tests**: install monitoring CRDs (GrafanaDashboard, PodMonitor,
  PrometheusRule, ServiceMonitor) before the chart so the release applies
  successfully.

## Prerequisites

- **kubectl** – context pointing at a cluster (e.g. [kind](https://kind.sigs.k8s.io/),
  minikube, or existing cluster)
- **helm** 3.x
- **jq** – for label verification in the script
- **Network** – script adds the official Helm repo and installs monitoring CRDs
  from qubership-monitoring-operator. To skip CRDs: `SKIP_CRDS=1` (only if those
  CRDs are already installed).

## Quick run

From the **repository root**:

```bash
./scripts/deploy-label-check/deploy-and-verify-labels.sh
```

This will:

1. Create namespace `logging` (or `NAMESPACE` env) with privileged PSS labels
   (Installation Guide)
2. Install monitoring CRDs (GrafanaDashboard, PodMonitor, PrometheusRule,
   ServiceMonitor)
3. Add Helm repo `qubership-logging-operator` and install the chart from the
   repo (official README), with `scripts/deploy-label-check/values-label-check.yaml`
   (operator + fluentbit + cloud-events-reader only; no Graylog/Fluentd)
4. Wait for the operator pod to be Ready
5. Wait 30s for reconciliation (operator creates DaemonSet/Deployment/Services)
6. Verify required labels on: `Deployment/logging-service-operator`,
   `DaemonSet/logging-fluentbit`, `Deployment/events-reader`, and their Services

## Using a custom operator image

The operator Deployment uses `imagePullPolicy: IfNotPresent` (chart default).
For a locally built tag, load the image into the cluster (`kind load`, etc.) or
push to a registry the cluster can pull from.

**kind** (skip the `kind load` line if you use Rancher Desktop or another cluster)

```bash
docker build -t qubership-logging-operator:local .
kind load docker-image qubership-logging-operator:local   # only for kind; omit for Rancher Desktop

OPERATOR_IMAGE=qubership-logging-operator:local ./scripts/deploy-label-check/deploy-and-verify-labels.sh
```

**Rancher Desktop**
The cluster runs in a VM and may not see images you build on the host. Easiest
options:

1. **Push to a registry** (Docker Hub, ghcr.io, etc.) and use the full image name:

   ```bash
   docker build -t your-registry/qubership-logging-operator:local .
   docker push your-registry/qubership-logging-operator:local
   OPERATOR_IMAGE=your-registry/qubership-logging-operator:local ./scripts/deploy-label-check/deploy-and-verify-labels.sh
   ```

2. Or build the image **inside** Rancher Desktop’s Docker/containerd (use the
   same engine Rancher Desktop uses), then run the script with
   `OPERATOR_IMAGE=qubership-logging-operator:local` so the cluster uses that
   image with `imagePullPolicy=Never`.

## Using the local chart instead of the Helm repo

To install from the local chart (e.g. to test chart changes, or if the Helm repo
is unavailable):

```bash
USE_LOCAL_CHART=1 ./scripts/deploy-label-check/deploy-and-verify-labels.sh
```

If you see `Error: repo qubership-logging-operator not found`, the Helm repo may
be missing or unreachable; use `USE_LOCAL_CHART=1` to install from the chart in
this repo.

With a local image:

```bash
USE_LOCAL_CHART=1 OPERATOR_IMAGE=qubership-logging-operator:local ./scripts/deploy-label-check/deploy-and-verify-labels.sh
```

## Manual deploy (without the script)

Following the
[official README](https://github.com/Netcracker/qubership-logging-operator#installation)
and
[Installation Guide](https://netcracker.github.io/qubership-logging-operator/installation/):

```bash
# Namespace and labels (Installation Guide)
kubectl create namespace logging
kubectl label namespace logging pod-security.kubernetes.io/enforce=privileged pod-security.kubernetes.io/enforce-version=latest --overwrite

# Install monitoring CRDs (required for chart)
# See .github/workflows/integration-tests.yml for CRD URLs.

# Add repo and install (official README)
helm repo add qubership-logging-operator https://netcracker.github.io/qubership-logging-operator
helm repo update
helm upgrade --install qubership-logging-operator qubership-logging-operator/qubership-logging-operator \
  -n logging -f scripts/deploy-label-check/values-label-check.yaml --create-namespace
```

Then inspect labels, for example:

```bash
kubectl get deployment logging-service-operator -n logging -o jsonpath='{.metadata.labels}' | jq .
kubectl get daemonset logging-fluentbit -n logging -o jsonpath='{.spec.template.metadata.labels}' | jq .
```

## Values used

`values-label-check.yaml` enables only:

- **Operator** (always)
- **Fluentbit** (`fluentbit.install: true`)
- **Cloud Events Reader** (`cloudEventsReader.install: true`)

Graylog and Fluentd are disabled to avoid OpenSearch/Mongo and extra images.
Fluentbit is given dummy `graylogHost`/`graylogPort` so the LoggingService CR is
valid; logs may not be delivered until Graylog is configured.

## Deploying via a CR only (operator already running)

The operator reconciles **LoggingService** CRs (`logging.netcracker.com/v1`).
The Helm chart creates one such CR when you install it; that is why a full
install deploys Fluent Bit and cloud-events-reader.

If the operator is already running (e.g. you installed only the operator, or a
previous run left it in place) and you want it to deploy workloads **without**
re-running a full Helm install, create the CR manually:

```bash
kubectl apply -f scripts/deploy-label-check/loggingservice-cr.yaml -n logging
```

Use the same namespace as the operator (`logging` by default). The operator will
reconcile the CR and create the Fluent Bit DaemonSet and cloud-events-reader
Deployment. The file `loggingservice-cr.yaml` is a minimal LoggingService with
the same components as `values-label-check.yaml` (fluentbit + cloudEventsReader,
no Graylog/Fluentd).
