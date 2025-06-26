# Cloud Events Reader Examples

Cloud Events Reader is a component that collects Kubernetes events and forwards them to the logging pipeline. These examples show different deployment scenarios for various operational requirements.

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
|-----------|-------------|---------|
| `cloudEventsReader.install` | Enable/disable Cloud Events Reader deployment | `false` |
| `cloudEventsReader.resources` | Resource requests and limits | - |
| `cloudEventsReader.nodeSelector` | Node selection criteria | - |

## Use Cases

- **Simple Deployment**: Standard Kubernetes clusters with default scheduling
- **Node Selector**: Clusters with dedicated nodes for logging components
- **Resource Constraints**: Environments requiring specific resource allocation 