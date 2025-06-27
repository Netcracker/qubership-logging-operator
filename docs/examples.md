# Configuration Examples

This section contains practical configuration examples for different components of the Qubership Logging Operator. 
Each example demonstrates specific use cases and deployment scenarios.

## Available Examples

### [Cloud Events Reader](examples/cloud-events-reader.md)

Configuration examples for deploying and managing Kubernetes events collection:

- Simple deployment - Basic cloud events reader setup
- Node selector configuration - Targeted deployment on specific nodes

### [FluentBit](examples/fluentbit.md)

FluentBit agent configuration examples for various scenarios:

- Simple deployment - Basic FluentBit agent setup
- High availability - Scalable FluentBit deployment with redundancy
- Custom Lua scripts - Advanced log processing with custom scripts
- High availability with custom Lua - Combined HA and custom processing

### [FluentD](examples/fluentd.md)

FluentD agent configuration examples covering different runtime environments:

- CentOS with Docker - FluentD setup for CentOS-based Docker environments
- Containerd runtime - Configuration for containerd container runtime
- Custom input/filter - Advanced input sources and filtering rules
- Ubuntu with Docker - FluentD setup for Ubuntu-based Docker environments
- Docker runtime - Standard Docker environment configuration
- OpenShift Containerd - OpenShift-specific containerd setup
- Simple deployment - Basic FluentD agent setup
- Node selector - Targeted deployment configuration
- Without Graylog output - Alternative output configurations

### [Graylog](examples/graylog.md)

Graylog server configuration examples for different deployment patterns:

- Custom labels and annotations - Graylog with custom Kubernetes metadata
- Dynamic provisioning - Storage provisioning and scaling
- Migration to v5 - Upgrade procedures and compatibility
- Simple deployment - Basic Graylog server setup
- Static volume - Persistent storage configuration

## Usage

Each example includes:

- Complete YAML configuration files embedded directly in the documentation
- Configuration explanations and key parameters
- Common customization options
- Use case descriptions

Refer to the [Configuration Guide](configuration.md) for detailed parameter explanations.
