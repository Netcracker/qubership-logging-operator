# Qubership VictoriaLogs chart

This chart deploys a single-node VictoriaLogs instance without VictoriaMetrics Operator. It can also create VMAuth,
Ingress, HTTPRoute, ServiceMonitor, and GrafanaDashboard resources.

## Install

```bash
helm upgrade --install victorialogs charts/qubership-victorialogs \
  --namespace logging \
  --create-namespace
```

The client Service is `victorialogs` by default and listens on port `9428`. Configure logging collectors with the
in-cluster endpoint `victorialogs.logging:9428` when both releases use the `logging` namespace.

The generated PVC has the Helm `keep` policy. Back up its data and delete it manually after uninstalling the release.
Changing `victorialogs.nameOverride` or `victorialogs.storage.existingClaim` does not migrate existing data.

On OpenShift, the selected Security Context Constraint controls the container UID, GID, and non-root policy. The chart
omits its default identity constraints while retaining its capability, privilege-escalation, and filesystem controls.

## External access

Ingress and HTTPRoute send traffic through VMAuth. Store the complete VMAuth `auth.yml` in an existing Secret to keep
credentials out of Helm values:

```yaml
victorialogs:
  ingress:
    install: true
    hosts:
      - host: logs.example.org
  vmauth:
    existingSecret: victorialogs-auth
    existingSecretKey: auth.yml
```

If `ingress.hosts` or `httpRoute.hostnames` is empty, set the root-level `CLOUD_PUBLIC_HOST` value. The chart generates
`vmauth-<namespace>.<CLOUD_PUBLIC_HOST>`.

You can instead set `victorialogs.vmauth.config.users` and let the chart create a Secret. Each user must use exactly one
authentication method: a non-empty `username` and `password` pair, or a non-empty `bearer_token`.

## Optional operator resources

`victorialogs.serviceMonitor.install` and `victorialogs.dashboard.install` default to `false`. Enable them only when the
corresponding Prometheus Operator or Grafana Operator v5 CRDs are installed.

<!-- markdownlint-disable line-length -->
## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| CLOUD_PUBLIC_HOST | string | `""` | Public DNS suffix used to generate external hosts when an explicit host is not set. |
| annotations | object | `{}` | Annotations applied to VictoriaLogs and VMAuth workloads. |
| labels | object | `{}` | Labels applied to all VictoriaLogs and VMAuth resources. |
| victorialogs.addIngressIgnoreAnnotation | bool | `true` | Add the gateway converter ignore annotation to Ingress when HTTPRoute is enabled. |
| victorialogs.affinity | object | `{}` | Affinity and anti-affinity rules for the VictoriaLogs Pod. |
| victorialogs.annotations | object | `{}` | Annotations for the VictoriaLogs StatefulSet. |
| victorialogs.dashboard.allowCrossNamespaceImport | bool | `true` | Allow the dashboard to target a Grafana instance in another namespace. |
| victorialogs.dashboard.annotations | object | `{}` | Annotations for the GrafanaDashboard. |
| victorialogs.dashboard.install | bool | `false` | Create a GrafanaDashboard v5 resource. The Grafana Operator v5 CRDs must be installed. |
| victorialogs.dashboard.instanceSelector | object | `{"matchLabels":{"app.kubernetes.io/component":"grafana","app.kubernetes.io/part-of":"monitoring"}}` | Label selector for the Grafana v5 instance that imports the dashboard. |
| victorialogs.dashboard.labels | object | `{}` | Additional labels for the GrafanaDashboard. |
| victorialogs.dockerImage | string | `""` | VictoriaLogs image. The chart uses docker.io/victoriametrics/victoria-logs:v1.51.0 when empty. |
| victorialogs.env | list | `[]` | Additional environment variables for the VictoriaLogs container. |
| victorialogs.envFrom | list | `[]` | Sources for additional VictoriaLogs environment variables. |
| victorialogs.extraArgs | object | `{"envflag.enable":true,"envflag.prefix":"VM_","http.shutdownDelay":"15s","loggerFormat":"json"}` | Additional VictoriaLogs command-line arguments. |
| victorialogs.extraVolumeMounts | list | `[]` | Additional volume mounts for the VictoriaLogs container. |
| victorialogs.extraVolumes | list | `[]` | Additional volumes for the VictoriaLogs Pod. |
| victorialogs.httpRoute | object | `{"annotations":{},"hostnames":[],"install":false,"labels":{},"parentRefs":[]}` | Gateway API HTTPRoute configuration for external access through VMAuth. |
| victorialogs.httpRoute.annotations | object | `{}` | Annotations for the HTTPRoute. |
| victorialogs.httpRoute.hostnames | list | `[]` | HTTPRoute hostnames. An empty list uses `vmauth-<namespace>.<CLOUD_PUBLIC_HOST>`. |
| victorialogs.httpRoute.install | bool | `false` | Create an HTTPRoute backed by VMAuth. Configure an authenticated user or `existingSecret`. |
| victorialogs.httpRoute.labels | object | `{}` | Additional labels for the HTTPRoute. |
| victorialogs.httpRoute.parentRefs | list | `[]` | Gateway listeners that accept the HTTPRoute. |
| victorialogs.imagePullPolicy | string | `"IfNotPresent"` | Image pull policy for the VictoriaLogs container. |
| victorialogs.imagePullSecrets | list | `[]` | Image pull Secrets for the VictoriaLogs Pod. |
| victorialogs.ingress | object | `{"annotations":{},"hosts":[],"ingressClassName":"","install":false,"labels":{},"tls":[]}` | Ingress configuration for external access through VMAuth. |
| victorialogs.ingress.annotations | object | `{}` | Annotations for the Ingress. |
| victorialogs.ingress.hosts | list | `[]` | Ingress hosts and paths. An empty list uses `vmauth-<namespace>.<CLOUD_PUBLIC_HOST>`. |
| victorialogs.ingress.ingressClassName | string | `""` | IngressClass assigned to the Ingress. |
| victorialogs.ingress.install | bool | `false` | Create an Ingress backed by VMAuth. Configure an authenticated user or `existingSecret`. |
| victorialogs.ingress.labels | object | `{}` | Additional labels for the Ingress. |
| victorialogs.ingress.tls | list | `[]` | TLS configuration for the Ingress. |
| victorialogs.install | bool | `true` | Deploy VictoriaLogs resources. |
| victorialogs.labels | object | `{}` | Additional labels for VictoriaLogs resources. |
| victorialogs.livenessProbe | object | `{"failureThreshold":10,"initialDelaySeconds":30,"periodSeconds":30,"tcpSocket":{"port":"http"},"timeoutSeconds":5}` | Liveness probe for the VictoriaLogs container. |
| victorialogs.nameOverride | string | `""` | Override the VictoriaLogs resource name. |
| victorialogs.nodeSelector | object | `{}` | Node selector for the VictoriaLogs Pod. |
| victorialogs.podAnnotations | object | `{}` | Annotations for the VictoriaLogs Pod. |
| victorialogs.podLabels | object | `{}` | Additional labels for the VictoriaLogs Pod. Selector labels take precedence. |
| victorialogs.podManagementPolicy | string | `"OrderedReady"` | Pod creation order for the VictoriaLogs StatefulSet. |
| victorialogs.podSecurityContext | object | `{"fsGroup":2000,"runAsGroup":1000,"runAsNonRoot":true,"runAsUser":1000,"seccompProfile":{"type":"RuntimeDefault"}}` | Security context for the VictoriaLogs Pod. |
| victorialogs.port | int | `9428` | HTTP listen port exposed by the VictoriaLogs container. |
| victorialogs.priorityClassName | string | `""` | PriorityClass assigned to the VictoriaLogs Pod. |
| victorialogs.readinessProbe | object | `{"failureThreshold":3,"httpGet":{"path":"/health","port":"http","scheme":"HTTP"},"initialDelaySeconds":5,"periodSeconds":5,"timeoutSeconds":5}` | Readiness probe for the VictoriaLogs container. |
| victorialogs.resources | object | `{}` | Compute resources for the VictoriaLogs container. |
| victorialogs.retentionPeriod | string | `"1"` | Data retention period. Supported units are h, d, w, and y. A value without a unit means months. |
| victorialogs.securityContext | object | `{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]},"readOnlyRootFilesystem":true,"runAsNonRoot":true}` | Security context for the VictoriaLogs container. |
| victorialogs.service | object | `{"annotations":{},"labels":{},"port":9428}` | VictoriaLogs client Service configuration. |
| victorialogs.service.annotations | object | `{}` | Annotations for the VictoriaLogs client and headless Services. |
| victorialogs.service.labels | object | `{}` | Additional labels for the VictoriaLogs client Service. |
| victorialogs.service.port | int | `9428` | Port exposed by the VictoriaLogs client Service. |
| victorialogs.serviceMonitor.annotations | object | `{}` | Annotations for the ServiceMonitor. |
| victorialogs.serviceMonitor.install | bool | `false` | Create a ServiceMonitor. The Prometheus Operator CRDs must be installed. |
| victorialogs.serviceMonitor.labels | object | `{}` | Additional labels for the ServiceMonitor. |
| victorialogs.serviceMonitor.metricRelabelings | list | `[]` | Prometheus metric relabeling rules. |
| victorialogs.serviceMonitor.relabelings | list | `[]` | Prometheus target relabeling rules. |
| victorialogs.serviceMonitor.scrapeInterval | string | `"30s"` | Prometheus scrape interval. |
| victorialogs.serviceMonitor.scrapeTimeout | string | `"10s"` | Prometheus scrape timeout. |
| victorialogs.startupProbe | object | `{}` | Optional startup probe for the VictoriaLogs container. |
| victorialogs.storage.accessModes | list | `["ReadWriteOnce"]` | Access modes for the generated PVC. |
| victorialogs.storage.annotations | object | `{}` | Annotations for the generated PVC. |
| victorialogs.storage.existingClaim | string | `""` | Mount this PVC instead of creating one. |
| victorialogs.storage.labels | object | `{}` | Labels for the generated PVC. |
| victorialogs.storage.mountPath | string | `"/storage"` | Data volume mount path. |
| victorialogs.storage.persistentVolume | string | `""` | Bind the generated PVC to this persistent volume. |
| victorialogs.storage.size | string | `"10Gi"` | Requested storage capacity for the generated PVC. |
| victorialogs.storage.storageClassName | string | `""` | StorageClass for the generated PVC. |
| victorialogs.storage.subPath | string | `""` | Optional subpath within the data volume. |
| victorialogs.terminationGracePeriodSeconds | int | `60` | Grace period for VictoriaLogs shutdown. |
| victorialogs.tmpVolume | object | `{"sizeLimit":"100Mi"}` | Temporary volume configuration for VictoriaLogs. |
| victorialogs.tmpVolume.sizeLimit | string | `"100Mi"` | Maximum capacity of the VictoriaLogs `/tmp` volume. |
| victorialogs.tolerations | list | `[]` | Tolerations for the VictoriaLogs Pod. |
| victorialogs.topologySpreadConstraints | list | `[]` | Topology spread constraints for the VictoriaLogs Pod. |
| victorialogs.updateStrategy | object | `{"type":"RollingUpdate"}` | Update strategy for the VictoriaLogs StatefulSet. |
| victorialogs.vmauth.affinity | object | `{}` | Affinity and anti-affinity rules for VMAuth Pods. |
| victorialogs.vmauth.annotations | object | `{}` | Annotations for the VMAuth Deployment. |
| victorialogs.vmauth.config | object | `{"users":[]}` | Native VMAuth configuration used to generate a Secret. Define at least one user before enabling external access unless existingSecret is set. |
| victorialogs.vmauth.dockerImage | string | `""` | VMAuth image. The chart uses docker.io/victoriametrics/vmauth:v1.147.0 when empty. |
| victorialogs.vmauth.env | list | `[]` | Additional environment variables for the VMAuth container. |
| victorialogs.vmauth.envFrom | list | `[]` | Sources for additional VMAuth environment variables. |
| victorialogs.vmauth.existingSecret | string | `""` | Name of an existing Secret that contains `auth.yml`. When empty, the chart generates a Secret from `config`. |
| victorialogs.vmauth.existingSecretKey | string | `"auth.yml"` | Key in `existingSecret` that contains the native VMAuth configuration. |
| victorialogs.vmauth.extraArgs | object | `{"configCheckInterval":"1m","envflag.enable":true,"envflag.prefix":"VM_","loggerFormat":"json"}` | Additional VMAuth command-line arguments. |
| victorialogs.vmauth.extraVolumeMounts | list | `[]` | Additional volume mounts for the VMAuth container. |
| victorialogs.vmauth.extraVolumes | list | `[]` | Additional volumes for VMAuth Pods. |
| victorialogs.vmauth.imagePullPolicy | string | `"IfNotPresent"` | Image pull policy for the VMAuth container. |
| victorialogs.vmauth.imagePullSecrets | list | `[]` | Image pull Secrets for VMAuth Pods. |
| victorialogs.vmauth.livenessProbe | object | `{"initialDelaySeconds":5,"periodSeconds":15,"tcpSocket":{"port":"http"},"timeoutSeconds":5}` | Liveness probe for the VMAuth container. |
| victorialogs.vmauth.nodeSelector | object | `{}` | Node selector for VMAuth Pods. |
| victorialogs.vmauth.podAnnotations | object | `{}` | Annotations for VMAuth Pods. |
| victorialogs.vmauth.podLabels | object | `{}` | Additional labels for VMAuth Pods. Selector labels take precedence. |
| victorialogs.vmauth.podSecurityContext | object | `{"runAsGroup":1000,"runAsNonRoot":true,"runAsUser":1000,"seccompProfile":{"type":"RuntimeDefault"}}` | Security context for VMAuth Pods. |
| victorialogs.vmauth.port | int | `8427` | HTTP listen port exposed by the VMAuth container. |
| victorialogs.vmauth.priorityClassName | string | `""` | PriorityClass assigned to VMAuth Pods. |
| victorialogs.vmauth.readinessProbe | object | `{"initialDelaySeconds":5,"periodSeconds":15,"tcpSocket":{"port":"http"}}` | Readiness probe for the VMAuth container. |
| victorialogs.vmauth.replicaCount | int | `1` | Number of VMAuth replicas. |
| victorialogs.vmauth.resources | object | `{}` | Compute resources for the VMAuth container. |
| victorialogs.vmauth.securityContext | object | `{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]},"readOnlyRootFilesystem":true,"runAsNonRoot":true}` | Security context for the VMAuth container. |
| victorialogs.vmauth.service | object | `{"annotations":{},"labels":{},"port":8427}` | VMAuth Service configuration. |
| victorialogs.vmauth.service.annotations | object | `{}` | Annotations for the VMAuth Service. |
| victorialogs.vmauth.service.labels | object | `{}` | Additional labels for the VMAuth Service. |
| victorialogs.vmauth.service.port | int | `8427` | Port exposed by the VMAuth Service. |
| victorialogs.vmauth.startupProbe | object | `{}` | Optional startup probe for the VMAuth container. |
| victorialogs.vmauth.terminationGracePeriodSeconds | int | `30` | Grace period for VMAuth shutdown. |
| victorialogs.vmauth.tmpVolume | object | `{"sizeLimit":"100Mi"}` | Temporary volume configuration for VMAuth. |
| victorialogs.vmauth.tmpVolume.sizeLimit | string | `"100Mi"` | Maximum capacity of the VMAuth `/tmp` volume. |
| victorialogs.vmauth.tolerations | list | `[]` | Tolerations for VMAuth Pods. |
| victorialogs.vmauth.topologySpreadConstraints | list | `[]` | Topology spread constraints for VMAuth Pods. |
<!-- markdownlint-enable line-length -->
