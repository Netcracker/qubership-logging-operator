# Graylog Examples

Graylog is a centralized log management platform that provides search, analysis, and alerting capabilities.
These examples demonstrate different Graylog deployment patterns for various operational requirements.

## Simple Deployment

Basic Graylog configuration with essential components:

```yaml title="graylog-simple-values.yaml"
--8<-- "examples/graylog/graylog-simple-values.yaml"
```

This comprehensive configuration includes:

- Graylog server with Elasticsearch integration
- FluentD agent for log collection
- Cloud Events Reader for Kubernetes events
- Resource allocation and node targeting
- Multi-component logging stack

## Storage Configurations

### Dynamic Provisioning

Graylog with dynamic storage provisioning for scalable deployments:

```yaml title="graylog-dynamic-provisioning-values.yaml"
--8<-- "examples/graylog/graylog-dynamic-provisioning-values.yaml"
```

### Static Volume Configuration

Graylog with predefined persistent storage:

```yaml title="graylog-static-volume-values.yaml"
--8<-- "examples/graylog/graylog-static-volume-values.yaml"
```

## Advanced Configurations

### Custom Labels and Annotations

Graylog deployment with custom Kubernetes metadata:

```yaml title="graylog-custom-labels-and-annotations-values.yaml"
--8<-- "examples/graylog/graylog-custom-labels-and-annotations-values.yaml"
```

This configuration demonstrates:

- Custom labels for resource organization
- Annotations for operational metadata
- Enhanced Kubernetes integration

## Migration and Upgrades

### Migration to Graylog v5

Configuration example for upgrading to Graylog version 5:

```yaml title="graylog-migration-to-v5.yaml"
--8<-- "examples/graylog/graylog-migration-to-v5.yaml"
```

This migration configuration includes:

- Version-specific parameters
- Compatibility settings
- Upgrade considerations

## Key Configuration Parameters

| Parameter | Description | Example |
| --------- | ----------- | ------- |
| `graylog.install` | Enable Graylog deployment | `true` |
| `graylog.host` | Graylog server hostname | `graylog.example.com` |
| `graylog.elasticsearchHost` | Elasticsearch connection URL | `http://user:pass@es:9200` |
| `graylog.resources` | Resource requests and limits | CPU/Memory specs |
| `graylog.persistence` | Storage configuration | PVC settings |
| `graylog.nodeSelector` | Node selection criteria | Label selectors |
| `createClusterAdminEntities` | Create cluster-wide resources | `true`/`false` |
| `osKind` | Operating system type | `centos`/`ubuntu`/`rhel` |
| `containerRuntimeType` | Container runtime | `docker`/`containerd`/`cri-o` |

## Integration Components

Most Graylog examples include integration with:

- **FluentD**: Log collection and forwarding
- **Cloud Events Reader**: Kubernetes events ingestion
- **Elasticsearch**: Search and storage backend

## Use Cases

- **Simple Deployment**: Complete logging stack for standard environments
- **Dynamic Storage**: Cloud environments with automatic provisioning
- **Static Storage**: On-premises with predefined storage
- **Custom Metadata**: Enhanced Kubernetes integration and organization
- **Version Migration**: Upgrading between Graylog versions
- **Resource Optimization**: Specific resource allocation requirements
