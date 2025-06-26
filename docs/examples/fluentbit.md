# FluentBit Examples

FluentBit is a lightweight and high-performance log processor and forwarder. These examples demonstrate various FluentBit deployment scenarios, from basic setups to advanced configurations with custom processing.

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

| Parameter | Description | Use Case |
|-----------|-------------|----------|
| `fluentbit.install` | Enable FluentBit deployment | All scenarios |
| `fluentbit.systemLogType` | System log source type | `varlogsyslog`, `journald` |
| `fluentbit.containerLogging` | Enable container log collection | Container environments |
| `fluentbit.graylogOutput` | Enable Graylog output | Graylog integration |
| `fluentbit.customLuaScriptConf` | Custom Lua processing scripts | Advanced log processing |

## Use Cases

- **Simple Deployment**: Basic log collection and forwarding
- **High Availability**: Mission-critical environments requiring redundancy
- **Custom Processing**: Complex log transformation requirements
- **Hybrid Configurations**: Combining HA with custom processing logic 