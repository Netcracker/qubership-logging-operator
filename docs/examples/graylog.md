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

## Container hardening

Graylog and MongoDB run as non-root users with a read-only root filesystem. Their application containers use the
`RuntimeDefault` seccomp profile, disable privilege escalation, and drop all Linux capabilities. Writable runtime
paths use bounded `emptyDir` volumes. Graylog and MongoDB data remain on their existing persistent volume claims.

The Graylog container adds back only `NET_BIND_SERVICE`. The upstream Graylog image assigns
`cap_net_bind_service=ep` to its Java binary, and Graylog exposes its default UDP input on port 514. Removing this
capability from the bounding set prevents Java from starting. Port 514 is an explicit exception to the hardening
profile's forbidden-port rule because changing it would break the existing Graylog input contract.

The `setup` init container remains root and keeps a writable image filesystem. It prepares ownership and permissions
on existing Graylog persistent volumes before the non-root application containers start. Kubernetes
PodSecurityPolicy and OpenShift SecurityContextConstraints apply one policy to both init and application containers,
so their global read-only-root-filesystem and required-drop-capabilities settings cannot be enabled without blocking
this setup step. The stricter controls are applied directly to every application container instead.

On Kubernetes, the setup step assigns the data volume to UID and GID `1100`, removes permissions for other users, and
uses mode `0660` for `directories.json`. OpenShift retains the legacy `0777` data-directory and `0666`
`directories.json` permissions until a restricted model is verified on OpenShift. The Graylog-specific SCC uses
`runAsUser: RunAsAny`, so application containers explicitly use their image UID and GID (`1100` for Graylog and
`1001` for MongoDB). Without those settings, the kubelet rejects these images when `runAsNonRoot` is enabled.
The setup container also normalizes existing MongoDB data to UID and GID `1001` with owner/group-only access. This is
required when upgrading a volume whose root directory already matches the pod `fsGroup`, but nested files do not.
MongoDB upgrade jobs use the same UID, GID, and `fsGroup` on Kubernetes and OpenShift. The Graylog SCC uses
`runAsUser: RunAsAny`, so omitting the numeric identity would leave the root-default MongoDB image incompatible with
`runAsNonRoot: true`.

The optional `download-plugins` init container runs as non-root UID `1001`, but it does not enforce a read-only root
filesystem, drop capabilities, or disable privilege escalation. Existing init containers are outside the hardening
scope.

## Use Cases

- **Simple Deployment**: Complete logging stack for standard environments
- **Dynamic Storage**: Cloud environments with automatic provisioning
- **Static Storage**: On-premises with predefined storage
- **Custom Metadata**: Enhanced Kubernetes integration and organization
- **Version Migration**: Upgrading between Graylog versions
- **Resource Optimization**: Specific resource allocation requirements
