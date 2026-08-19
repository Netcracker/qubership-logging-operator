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

| Profile                 | Read offset database                        | Input buffer                              | Expected node disk writes                     |
| ----------------------- | ------------------------------------------- | ----------------------------------------- | --------------------------------------------- |
| `memory-only` (default) | Memory-backed `emptyDir`, limited to `32Mi` | Memory                                    | None from the offset database or input buffer |
| `persistent-offsets`    | `/var/lib/fluent-bit/state` on the node     | Memory                                    | Offset database writes only                   |
| `node-persistent`       | `/var/lib/fluent-bit/state` on the node     | `/var/lib/fluent-bit/storage` on the node | Offset database and buffered log writes       |

All profiles mount node logs at `/var/log` read-only. FluentBit accesses offset databases through
`/fluent-bit/state`. The `node-persistent` profile also mounts its buffer at `/fluent-bit/storage`.

Use `memory-only` when avoiding node filesystem writes is more important than preserving read offsets across Pod
replacement. Its offset database survives a container restart but is lost when Kubernetes replaces the Pod. After
that loss, each input follows its own initial read setting. For example, the container Tail input has
`Read_from_Head True`, while the systemd input has `Read_from_Tail On`.

Use `persistent-offsets` to reduce disk writes while preserving offsets when a Pod is replaced on the same node. The
offsets do not follow a Pod to another node and are lost with the node filesystem. Use `node-persistent` when buffered
logs must also survive Pod replacement and temporary output outages on the same node. Neither persistent profile
protects data from node loss.

The operator does not copy databases previously stored under `/var/log` into `/fluent-bit/state`. During the first
rollout after this path change, FluentBit creates new databases and applies each input's initial read setting. Plan
for possible duplicate records from inputs configured to read from the beginning.

`fluentbit.db.enabled: false` disables the read offset databases for all three profiles. The remaining behavior still
depends on each input's initial read setting and on whether the watched files are present after FluentBit restarts.

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
