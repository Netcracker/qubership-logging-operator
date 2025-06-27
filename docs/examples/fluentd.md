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

## Key Configuration Parameters

| Parameter | Description | Values |
|-----------|-------------|--------|
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
