# FluentBit Examples

FluentBit is a lightweight and high-performance log processor and forwarder.
These examples demonstrate various FluentBit deployment scenarios,
from basic setups to advanced configurations with custom processing.

## Simple Deployment

Basic FluentBit configuration for standard log collection:

```yaml title="fluentbit-simple-values.yaml"
--8<-- "examples/fluentbit/fluentbit-simple-values.yaml"
```

## High Availability with Aggregator

FluentBit deployment with aggregator for improved reliability and load distribution:

```yaml title="fluentbit-with-aggragator-values.yaml"
--8<-- "examples/fluentbit/fluentbit-with-aggragator-values.yaml"
```

## Storage Profiles

`fluentbit.storageProfile` controls the durability and disk I/O of the standard collector and the HA forwarder.

| Profile                 | Read offset database                                                   | Input buffer                              | Expected node disk writes                     |
| ----------------------- | ---------------------------------------------------------------------- | ----------------------------------------- | --------------------------------------------- |
| `memory-only` (default) | Memory-backed `emptyDir`, configurable with `memoryOnlyStateSizeLimit` | Memory                                    | None from the offset database or input buffer |
| `persistent-offsets`    | `/var/lib/fluent-bit/state` on the node                                | Memory                                    | Offset database writes only                   |
| `node-persistent`       | `/var/lib/fluent-bit/state` on the node                                | `/var/lib/fluent-bit/storage` on the node | Offset database and buffered log writes       |

All profiles mount node logs at `/var/log` read-only. FluentBit accesses offset databases through
`/fluent-bit/state`. The `node-persistent` profile also mounts its buffer at `/fluent-bit/storage`.

Use `memory-only` when avoiding node filesystem writes is more important than preserving read offsets across Pod
replacement. Its offset database survives a container restart but is lost when Kubernetes replaces the Pod. After
that loss, each input follows its own initial read setting. For example, the container Tail input has
`Read_from_Head True`, while the systemd input has `Read_from_Tail On`.

Set `fluentbit.memoryOnlyStateSizeLimit` to limit the memory-backed offset database volume. The default is `32Mi`.
The volume counts against the FluentBit container memory limit. If the volume reaches its size limit, FluentBit cannot
update its offset databases and reports filesystem errors. If the Pod reaches its memory limit first, Kubernetes may
terminate it with an out-of-memory error. Set both limits with enough headroom for the number of enabled inputs and
their database growth.

Use `persistent-offsets` to reduce disk writes while preserving offsets when a Pod is replaced on the same node. The
offsets do not follow a Pod to another node and are lost with the node filesystem. Use `node-persistent` when buffered
logs must also survive Pod replacement and temporary output outages on the same node. Neither persistent profile
protects data from node loss.

The operator does not copy databases previously stored under `/var/log` into `/fluent-bit/state`. During the first
rollout after this path change, FluentBit creates new databases and applies each input's initial read setting. Plan
for possible duplicate records from inputs configured to read from the beginning.

The upgrade also leaves the following legacy data on each node:

- Offset databases and their `-shm` and `-wal` files: `/var/log/containers.db`, `/var/log/messages.db`,
  `/var/log/syslog.db`, `/var/log/kube-apiserver.db`, `/var/log/openshift-apiserver.db`,
  `/var/log/kube-apiserver-audit.db`, `/var/log/kube-audit.db`, `/var/log/ocp-audit.db`, and
  `/var/log/audit/audit.db`.
- The systemd offset database `/var/log/flb-storage/journal.db` and buffered records under `/var/log/flb-storage`.

Remove the old offset databases only after the new FluentBit Pods are healthy. Treat `/var/log/flb-storage` separately:
it may contain records that were not delivered before the upgrade, so deleting it can lose logs. The operator does not
remove either location automatically.

The `persistent-offsets` and `node-persistent` profiles create `/var/lib/fluent-bit/state` on each node. The
`node-persistent` profile also creates `/var/lib/fluent-bit/storage`. Kubernetes does not remove these host directories
when you switch profiles or uninstall Logging. Retain them when you plan to return to a persistent profile, or remove
them as part of node maintenance after confirming that their offsets and buffered records are no longer needed.

`fluentbit.db.enabled: false` disables the read offset databases for all three profiles. The remaining behavior still
depends on each input's initial read setting and on whether the watched files are present after FluentBit restarts.

## Security Hardening Constraints

FluentBit uses a read-only container root filesystem, the `RuntimeDefault` seccomp profile, and a dedicated `emptyDir`
mounted at `/tmp`. These controls do not make mounted volumes read-only. The log collectors retain the access required
to read node logs and update their existing state files.

### Node log collector and HA forwarder

The standard FluentBit collector and the HA forwarder have the following intentional exceptions to the default
non-root hardening profile:

- They run as UID `0` because node log directories such as `/var/log/pods` can be owned by root with mode `0750`.
  A non-root process cannot traverse these directories reliably across supported platforms.
- They mount the node `/var/log` directory as a read-only `hostPath`. The `persistent-offsets` and `node-persistent`
  profiles add writable host paths under `/var/lib/fluent-bit` for offset databases and filesystem buffering.
- When `fluentbit.securityContextPrivileged` is `false`, the containers drop all Linux capabilities and add only
  `DAC_OVERRIDE`. This capability lets UID `0` read root-owned node logs after dropping the default capability set. It
  does not bypass a read-only volume mount.
- On OpenShift, the standard collector and HA forwarder pods use SELinux type `spc_t`. Node log and storage paths are
  protected by host SELinux labels that the default container type cannot access. The HA aggregator retains the
  default confined container type because it does not mount node paths.
- Setting `fluentbit.securityContextPrivileged` to `true` preserves the legacy privileged mode for environments that
  require it. In this mode, the main container does not explicitly disable privilege escalation or restrict its
  capabilities. The read-only root filesystem remains enabled. Use the default value, `false`, unless the node runtime
  requires privileged access.

Do not make these containers non-root, remove `DAC_OVERRIDE`, or remove their host paths without validating all
supported node layouts and upgrade scenarios.

The read-only `/var/log` host path is an explicit exception to the generic container-hardening rule that prohibits host
paths. The writable `/var/lib/fluent-bit` host paths are additional exceptions for persistent storage profiles.
Removing `/var/log` would prevent the DaemonSet from collecting node logs.

### HA aggregator

The HA aggregator does not read node files and has no `hostPath` exception. Its main container and config reload
sidecar run as non-root, disable privilege escalation, use a read-only root filesystem, and drop all Linux
capabilities. Kubernetes and OpenShift deployments use UID `1001`. The OpenShift UID must be explicit because the
upstream Fluent Bit image declares root as its default user; `runAsNonRoot: true` alone makes the kubelet reject it.

The aggregator keeps its filesystem buffer under `/fluent-bit/storage`. This path remains writable through the
existing `storage` volume:

- The default configuration uses an `emptyDir` for each aggregator replica.
- Enabling `fluentbit.aggregator.volume.bind` uses a per-replica PersistentVolumeClaim.

The default storage `emptyDir` does not define a Kubernetes `sizeLimit`. Adding a volume limit could change buffering
and disk I/O behavior, so storage sizing remains outside the container-hardening scope while that behavior is being
investigated. Control growth through Fluent Bit buffer settings or configure a suitably sized PVC.

The paths `/fluent-bit/state` and `/fluent-bit/storage` are part of the storage-profile behavior. Container hardening
does not change their persistence semantics.

### HA failover behavior

The forwarder sends records to the individual StatefulSet replicas through pod-specific DNS names. If an aggregator
pod is deleted or unavailable, FluentBit can log temporary DNS, connection, and chunk retry messages for that replica.
The remaining replica continues to receive records, and the forwarder retries buffered chunks until the unavailable
replica and its DNS record return.

Treat short-lived retry messages during a rollout or failover as expected. Investigate them when both aggregator
replicas are ready and the messages continue, the forwarder storage backlog keeps growing, or records stop reaching
the configured output.

## Custom Lua Script Processing

Advanced FluentBit configuration with custom Lua script for specialized log processing:

```yaml title="fluentbit-custom-lua-script-values.yaml"
--8<-- "examples/fluentbit/fluentbit-custom-lua-script-values.yaml"
```

This configuration demonstrates:

- Custom Lua script for date/time conversion to UTC
- Graylog output configuration
- Custom log processing logic
- Integration with external commands for date parsing

## High Availability with Custom Lua Scripts

Combined high availability and custom processing configuration:

```yaml title="fluentbit-ha-custom-lua-script-values.yaml"
--8<-- "examples/fluentbit/fluentbit-ha-custom-lua-script-values.yaml"
```

This advanced configuration provides:

- High availability setup with multiple instances
- Custom Lua scripts for log transformation
- Enhanced reliability and processing capabilities
- Scalable log processing pipeline

## Key Configuration Parameters

| Parameter                       | Description                     | Use Case                   |
| ------------------------------- | ------------------------------- | -------------------------- |
| `fluentbit.install`             | Enable FluentBit deployment     | All scenarios              |
| `fluentbit.systemLogType`       | System log source type          | `varlogsyslog`, `journald` |
| `fluentbit.containerLogging`    | Enable container log collection | Container environments     |
| `fluentbit.graylogOutput`       | Enable Graylog output           | Graylog integration        |
| `fluentbit.customLuaScriptConf` | Custom Lua processing scripts   | Advanced log processing    |

## Use Cases

- **Simple Deployment**: Basic log collection and forwarding
- **High Availability**: Mission-critical environments requiring redundancy
- **Custom Processing**: Complex log transformation requirements
- **Hybrid Configurations**: Combining HA with custom processing logic
