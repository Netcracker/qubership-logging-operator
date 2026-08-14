# Cloud Events Reader Examples

Cloud Events Reader is a component that collects Kubernetes events and forwards them to the logging pipeline.
These examples show different deployment scenarios for various operational requirements.

## Simple Deployment

Basic Cloud Events Reader configuration suitable for most environments:

```yaml title="cloud-events-reader-simple-values.yaml"
--8<-- "examples/cloud-events-reader/cloud-events-reader-simple-values.yaml"
```

This configuration provides:

- Basic resource allocation (100m CPU, 128Mi memory)
- Standard event collection from Kubernetes API
- Minimal resource footprint

## Node Selector Configuration

Targeted deployment on specific nodes using node selectors:

```yaml title="cloud-events-reader-with-nodeSelector-values.yaml"
--8<-- "examples/cloud-events-reader/cloud-events-reader-with-nodeSelector-values.yaml"
```

This configuration adds:

- Node selector for targeted deployment
- Same resource allocation as simple deployment
- Useful for dedicated logging nodes or specific node pools

## Key Configuration Parameters

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `cloudEventsReader.install` | Enable/disable Cloud Events Reader deployment | `false` |
| `cloudEventsReader.resources` | Resource requests and limits | - |
| `cloudEventsReader.nodeSelector` | Node selection criteria | - |

## Container hardening

Cloud Events Reader runs with UID and GID 1000 on Kubernetes. On OpenShift, the platform assigns a non-root UID while
the workload retains primary GID 1000. The container uses a read-only root filesystem, disables privilege escalation,
drops all Linux capabilities, and uses the `RuntimeDefault` seccomp profile. A 100 MiB `emptyDir` mounted at `/tmp`
provides its only writable runtime path.

The service account can only get, list, and watch Kubernetes Events through the `core/v1` and `events.k8s.io` APIs. It
does not receive the built-in `view` role and cannot read pods, ConfigMaps, or Secrets.

## Use Cases

- **Simple Deployment**: Standard Kubernetes clusters with default scheduling
- **Node Selector**: Clusters with dedicated nodes for logging components
- **Resource Constraints**: Environments requiring specific resource allocation
