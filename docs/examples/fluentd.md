# FluentD Examples

FluentD is a data collector for unified logging layer.
These examples cover various runtime environments and deployment scenarios,
from simple setups to complex multi-platform configurations.

## Simple Deployment

Basic FluentD configuration suitable for most Kubernetes environments:

```yaml title="fluentd-simple-values.yaml"
--8<-- "examples/fluentd/fluentd-simple-values.yaml"
```

## Container Runtime Configurations

### Docker Runtime

Standard Docker environment configuration:

```yaml title="fluentd-docker-runtime-values.yaml"
--8<-- "examples/fluentd/fluentd-docker-runtime-values.yaml"
```

### Containerd Runtime

Configuration optimized for containerd container runtime:

```yaml title="fluentd-containerd-runtime-values.yaml"
--8<-- "examples/fluentd/fluentd-containerd-runtime-values.yaml"
```

### OpenShift with Containerd

OpenShift-specific configuration with containerd runtime:

```yaml title="fluentd-openshift-containerd-values.yaml"
--8<-- "examples/fluentd/fluentd-openshift-containerd-values.yaml"
```

## Operating System Specific

### CentOS with Docker

Optimized configuration for CentOS-based Docker environments:

```yaml title="fluentd-centos-docker-values.yaml"
--8<-- "examples/fluentd/fluentd-centos-docker-values.yaml"
```

### Ubuntu with Containerd

Configuration tailored for Ubuntu systems with containerd:

```yaml title="fluentd-ubuntu-containerd-values.yaml"
--8<-- "examples/fluentd/fluentd-ubuntu-containerd-values.yaml"
```

## Advanced Configurations

### Custom Input and Filter

Advanced configuration with custom input sources and filtering rules:

```yaml title="fluentd-custom-input-filter-values.yaml"
--8<-- "examples/fluentd/fluentd-custom-input-filter-values.yaml"
```

### Node Selector Deployment

Targeted deployment using node selectors:

```yaml title="fluentd-with-node-selector-values.yaml"
--8<-- "examples/fluentd/fluentd-with-node-selector-values.yaml"
```

### Without Graylog Output

Configuration for alternative output destinations:

```yaml title="fluentd-without-graylog-output-values.yaml"
--8<-- "examples/fluentd/fluentd-without-graylog-output-values.yaml"
```

## Security Hardening Constraints

FluentD uses the `RuntimeDefault` seccomp profile, a read-only container root filesystem, and a dedicated `emptyDir`
mounted at `/tmp`. The config reload sidecar runs as non-root, disables privilege escalation, and drops all Linux
capabilities.

The main FluentD container has three intentional exceptions to the generic non-root hardening profile:

- It runs as UID `0` because node log directories such as `/var/log/pods` can be owned by root with mode `0750`.
  A non-root process cannot traverse these directories reliably across supported platforms.
- It uses a writable `hostPath` mount for `/var/log` to read node logs and update position files in their legacy
  locations.
- On OpenShift, it uses SELinux type `spc_t` because the default container type cannot read node logs labeled
  `var_log_t`. This setting applies to the pod because containers in a pod share one SELinux label.

FluentD keeps each position file at its original path under `/var/log`, such as `/var/log/es-containers.log.pos` for
container logs. These node-local files survive DaemonSet pod replacement on the same node. Replacing a node loses its
position files, which matches the behavior before hardening. The writable `/var/log` mount is an exception to the
generic rule that prohibits writable host paths; making it read-only would make position files ephemeral.

FluentD writes temporary runtime files and the optional Graylog file buffer under `/tmp`. When
`fluentd.securityContextPrivileged` is `false`, the main container disables privilege escalation and drops all Linux
capabilities. It does not require `DAC_OVERRIDE` because UID `0` owns the position files and their parent directories.

Setting `fluentd.securityContextPrivileged` to `true` preserves the legacy privileged mode for environments that need
it. In this mode, the main container does not explicitly disable privilege escalation or restrict its capabilities.
The container root filesystem remains read-only. Use the default value, `false`, unless the node runtime requires
privileged access.

Do not make the main container non-root or remove the `/var/log` host path without validating all supported node log
layouts. These settings are explicit exceptions to the generic container-hardening rules.

### File buffer limit

Setting `fluentd.fileStorage` to `true` stores the Graylog buffer under `/tmp/fluentd/buffer`. The hardening profile
limits the `/tmp` `emptyDir` to `100Mi`, while the default FluentD `totalLimitSize` is `512MB`. The volume limit is
therefore reached before FluentD's configured buffer limit when the output remains unavailable.

The file buffer is ephemeral and is lost when the pod is replaced. This behavior predates container hardening. Mount a
dedicated persistent volume through `additionalVolumes` and `additionalVolumeMounts` when buffered records must survive
pod replacement.

Set `fluentd.totalLimitSize` to at most `100Mi` when you enable file storage, accounting for temporary runtime files in
the same volume. Use `additionalVolumes` and `additionalVolumeMounts` with a dedicated writable volume when the
deployment requires a larger persistent buffer.

## Key Configuration Parameters

| Parameter | Description | Values |
| --------- | ----------- | ------ |
| `fluentd.install` | Enable FluentD deployment | `true`/`false` |
| `fluentd.graylogHost` | Graylog server hostname/IP | Hostname or IP |
| `fluentd.graylogPort` | Graylog input port | Port number (default: 12201) |
| `fluentd.resources` | Resource requests and limits | CPU/Memory specifications |
| `fluentd.nodeSelector` | Node selection criteria | Key-value pairs |
| `containerRuntimeType` | Container runtime type | `docker`/`cri-o`/`containerd` |
| `osKind` | Operating system type | `centos`/`rhel`/`oracle`/`ubuntu` |

## Use Cases

- **Simple Deployment**: Standard Kubernetes clusters with basic logging needs
- **Container Runtime Specific**: Environments with specific container runtime requirements
- **OS-Specific**: Optimized configurations for different operating systems
- **Custom Processing**: Advanced log processing and routing requirements
- **Node Targeting**: Specific node deployment requirements
- **Alternative Outputs**: Non-Graylog output destinations
