# Installation parameters

The configurable parameters for installation are described below.

## Table of Contents

* [Installation parameters](#installation-parameters)
  * [Table of Contents](#table-of-contents)
  * [Root](#root)
  * [Graylog](#graylog)
    * [Graylog TLS](#graylog-tls)
    * [OpenSearch](#opensearch)
    * [ContentPacks](#contentpacks)
    * [Graylog Streams](#graylog-streams)
    * [Graylog Auth Proxy](#graylog-auth-proxy)
      * [Graylog Auth Proxy LDAP](#graylog-auth-proxy-ldap)
      * [Graylog Auth Proxy OAuth](#graylog-auth-proxy-oauth)
  * [FluentBit](#fluentbit)
    * [FluentBit Aggregator](#fluentbit-aggregator)
    * [FluentBit TLS](#fluentbit-tls)
  * [FluentD](#fluentd)
    * [FluentD TLS](#fluentd-tls)
  * [Cloud Events Reader](#cloud-events-reader)
  * [Integration tests](#integration-tests)

## Root

This is a common section that contains some generic parameters.

All parameters in the table below should be specified on the first (root) level, e.g.:

```yaml
name: logging-service
containerRuntimeType: containerd
...
```

<!-- markdownlint-disable line-length -->
| Parameter                    | Type                                                                                                             | Mandatory | Default value                    | Description                                                                                                                                                     |
| ---------------------------- | ---------------------------------------------------------------------------------------------------------------- | --------- | -------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `name`                       | string                                                                                                           | no        | `logging-service`                | Name of new custom resource                                                                                                                                     |
| `namespace`                  | string                                                                                                           | no        |                                  | Cloud namespace to deploy logging service                                                                                                                       |
| `cloudURL`                   | string                                                                                                           | no        | `https://kubernetes.default.svc` | Address of Kubernetes APIServer                                                                                                                                 |
| `osKind`                     | string                                                                                                           | no        | `centos`                         | Operating system kind on Cloud nodes. Possible values: `centos` / `rhel` / `oracle` / `ubuntu`. It defines the [logs location](./log-location.md).              |
| `ipv6`                       | boolean                                                                                                          | no        | `false`                          | Set to true when deploying in an `IPv6` environment                                                                                                             |
| `containerRuntimeType`       | String                                                                                                           | no        | `docker`                         | Container runtime software used in the cloud environment. Possible values: `docker` / `cri-o` / `containerd`. Currently, it only differentiates                 |
| `createClusterAdminEntities` | boolean                                                                                                          | no        | `true`                           | Set to `true` to create logging service entities that require cluster-admin privileges. The user running the deployment must have cluster-admin privileges.     |
| `operatorImage`              | string                                                                                                           | no        | `-`                              | Docker image of logging-operator                                                                                                                                |
| `skipMetricsService`         | boolean                                                                                                          | no        | `-`                              | Set to `true` to skip metrics Service and ServiceMonitor creation                                                                                               |
| `nodeSelectorKey`            | string                                                                                                           | no        | `-`                              | NodeSelector key                                                                                                                                                |
| `nodeSelectorValue`          | string                                                                                                           | no        | `-`                              | NodeSelector value                                                                                                                                              |
| `affinity`                   | [core/v1.Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podaffinityterm-v1-core) | no        | `-`                              | Specifies the pod\'s scheduling constraints                                                                                                                     |
| `annotations`                | map                                                                                                              | no        | `{}`                             | Allows to specify additional annotations                                                                                                                        |
| `labels`                     | map                                                                                                              | no        | `{}`                             | Allows to specify additional labels                                                                                                                             |
| `pprof.install`              | boolean                                                                                                          | no        | `true`                           | Enables pprof for collecting profiling data.                                                                                                                    |
| `pprof.containerPort`        | int                                                                                                              | no        | `9180`                           | prot for pprof container.                                                                                                                                       |
| `pprof.service.type`         | string                                                                                                           | no        | `ClusterIP`                      | pprof service type.                                                                                                                                             |
| `pprof.service.port`         | int                                                                                                              | no        | `9100`                           | pprof port which is used in service                                                                                                                             |
| `pprof.service.portName`     | string                                                                                                           | no        | `http`                           | pprof port name which is used in service.                                                                                                                       |
| `pprof.service.annotations`  | map[string]string                                                                                                | no        | `{}`                             | Allows to specify additional annotations in service                                                                                                             |
| `pprof.service.labels`       | map[string]string                                                                                                | no        | `{}`                             | Allows to specify additional labels in service                                                                                                                  |
| `priorityClassName`          | string                                                                                                           | no        | `-`                              | Pod priority class. Indicates the importance of a Pod relative to other Pods and prevents it from being evicted.                                                |
<!-- markdownlint-enable line-length -->

Examples:

**Note:** This is only an example of the parameters format, not a recommended value.

```yaml
name: logging-service
namespace: logging
operatorImage: ghcr.io/netcracker/qubership-logging-operator:main

cloudURL: https://kubernetes.default.svc
osKind: ubuntu
ipv6: false
containerRuntimeType: containerd

createClusterAdminEntities: true

skipMetricsService: false
pprof:
  install: true
  containerPort: 9180
  service:
    type: ClusterIP
    port: 9180
    protName: pprof
    annotations: {}
    labels: {}

nodeSelectorKey: kubernetes.io/os
nodeSelectorValue: linux
```

[Back to TOC](#table-of-contents)

## Graylog

The `graylog` section contains parameters to enable and configure the Graylog deployment in the cloud.

All parameters described below should be specified under the `graylog` section as follows:

```yaml
graylog:
  install: true
  #...
```

<!-- markdownlint-disable line-length -->
| Parameter                                  | Type                                                                                                                   | Mandatory | Default value                                                                   | Description                                                                                                                                                                                           |
| ------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------- | --------- | ------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `install`                                  | boolean                                                                                                                | no        | `false`                                                                         | Enable the Graylog deployment in the cloud                                                                                                                                                            |
| `dockerImage`                              | string                                                                                                                 | no        | `-`                                                                             | Image used for Graylog container                                                                                                                                                                      |
| `initSetupImage`                           | string                                                                                                                 | no        | `-`                                                                             | Image for the init container of Graylog                                                                                                                                                               |
| `initContainerDockerImage`                 | string                                                                                                                 | no        | `-`                                                                             | Image used to initialize plugins for Graylog                                                                                                                                                          |
| `initResources`                            | [core/v1.Resources](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core) | no        | `{requests: {cpu: 50m, memory: 128Mi}, limits: {cpu: 100m, memory: 256Mi}}`     | The resources for init container; describes compute resources requests and limits                                                                                                                     |
| `replicas`                                 | integer                                                                                                                | no        | `1`                                                                             | Number of Graylog pods                                                                                                                                                                                |
| `mongoDBImage`                             | string                                                                                                                 | no        | `-`                                                                             | Image of MongoDB to be used for Graylog deployment                                                                                                                                                    |
| `mongoUpgrade`                             | string                                                                                                                 | no        | `false`                                                                         | Activates automatic, step-by-step MongoDB database upgrade. Intended for migration from Graylog 4 to 5  only                                                                                          |
| `mongoDBUpgrade.mongoDBImage40`            | string                                                                                                                 | no        | `-`                                                                             | Image of MongoDB 4.0 to use for Graylog deployment. Used for migration from MongoDB 3.6 to 5.x                                                                                                        |
| `mongoDBUpgrade.mongoDBImage42`            | string                                                                                                                 | no        | `-`                                                                             | Image of MongoDB 4.2 to use for Graylog deployment. Used for migration from MongoDB 3.6 to 5.x                                                                                                        |
| `mongoDBUpgrade.mongoDBImage44`            | string                                                                                                                 | no        | `-`                                                                             | Image of MongoDB 4.0 to use for Graylog deployment. Used for migration from MongoDB 3.6 to 5.x                                                                                                        |
| `mongoPersistentVolume`                    | string                                                                                                                 | no        | `-`                                                                             | MongoDB Persistent Volume (PV) name. Used to claim an existing PVs                                                                                                                                    |
| `mongoStorageClassName`                    | string                                                                                                                 | no        | `-`                                                                             | MongoDB Persistent Volume Claim (PVC) storage class name. Used in case of dynamic storage provisioning                                                                                                |
| `mongoResources`                           | [core/v1.Resources](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core) | no        | `{requests: {cpu: 500m, memory: 256Mi}, limits: {cpu: 500m, memory: 256Mi}}`    | Describes compute resources requests and limits MongoDB container                                                                                                                                     |
| `annotations`                              | map                                                                                                                    | no        | `{}`                                                                            | Allows to specify additional annotations for Graylog pod                                                                                                                                              |
| `labels`                                   | map                                                                                                                    | no        | `{}`                                                                            | Allows to specify additional labels for Graylog pod                                                                                                                                                   |
| `graylogResources`                         | [core/v1.Resources](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core) | no        | `{requests: {cpu: 500m, memory: 1536Mi}, limits: {cpu: 1000m, memory: 2048Mi}}` | Describes compute resources requests and limits for Graylog container                                                                                                                                 |
| `graylogPersistentVolume`                  | string                                                                                                                 | no        | `-`                                                                             | Graylog Persistent Volume (PV) name. Used to claim an existing PVs                                                                                                                                    |
| `graylogStorageClassName`                  | string                                                                                                                 | no        | `""`                                                                            | Graylog Persistent Volume Claim (PVC) storage class name. Used in case of dynamic storage provisioning                                                                                                |
| `storageSize`                              | string                                                                                                                 | no        | `2Gi`                                                                           | Graylog Persistent Volume size. Used for `journald` cache                                                                                                                                             |
| `priorityClassName`                        | string                                                                                                                 | no        | `-`                                                                             | Pod priority class. Indicates the importance of a Pod relative to other Pods and prevents it from being evicted.                                                                                      |
| `nodeSelectorKey`                          | string                                                                                                                 | no        | `-`                                                                             | Key of `nodeSelector`                                                                                                                                                                                 |
| `nodeSelectorValue`                        | string                                                                                                                 | no        | `-`                                                                             | Value of `nodeSelector`                                                                                                                                                                               |
| `affinity`                                 | [core/v1.Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podaffinityterm-v1-core)       | no        | `-`                                                                             | Specifies the pod\'s scheduling constraints                                                                                                                                                           |
| `securityResources.install`                | boolean                                                                                                                | no        | `false`                                                                         | Enables creation of security resources such as PodSecurityPolicy and SecurityContextConstraints                                                                                                       |
| `securityResources.name`                   | string                                                                                                                 | no        | `logging-graylog`                                                               | Specifies the name for PodSecurityPolicy and SecurityContextConstraints objects                                                                                                                       |
| `logLevel`                                 | string                                                                                                                 | no        | `INFO`                                                                          | Sets the Graylog log level for it's logs                                                                                                                                                              |
| `indexReplicas`                            | integer                                                                                                                | no        | `1`                                                                             | Number of OpenSearch/Elasticsearch replicas used per index                                                                                                                                            |
| `indexShards`                              | integer                                                                                                                | no        | `5`                                                                             | Number of OpenSearch/Elasticsearch shards used per index                                                                                                                                              |
| `elasticsearchHost`                        | string                                                                                                                 | yes       | `-`                                                                             | OpenSearch/Elasticsearch host with schema, credentials and port. For example: `http://user:password@elasticsearch.elasticsearch.svc:9200`                                                             |
| `elasticsearchMaxTotalConnections`         | integer                                                                                                                | no        | `100`                                                                           | Maximum number of total connections to OpenSearch/Elasticsearch                                                                                                                                       |
| `elasticsearchMaxTotalConnectionsPerRoute` | integer                                                                                                                | no        | `100`                                                                           | Maximum number of connections per OpenSearch/Elasticsearch route (normally this means per OpenSearch/Elasticsearch server)                                                                            |
| `createIngress`                            | boolean                                                                                                                | no        | `true`                                                                          | Enables or disables the creation of an Ingress resource for Graylog                                                                                                                                   |
| `host`                                     | string                                                                                                                 | no        | `-`                                                                             | The Graylog host for Ingress and Route. For example: `https://graylog-service.kubernetes.test.org/`                                                                                                   |
| `ingressClassName`                         | string                                                                                                                 | no        | `-`                                                                             | Name of an IngressClass that will be used to create Ingress                                                                                                                                           |
| `inputPort`                                | string                                                                                                                 | no        | `12201`                                                                         | Port used by the default Graylog input                                                                                                                                                                |
| `graylogSecretName`                        | string                                                                                                                 | no        | `graylog-secret`                                                                | The name of Kubernetes Secret that store Graylog super admin credentials and OpenSearch/Elasticsearch connection string                                                                               |
| `contentDeployPolicy`                      | string                                                                                                                 | no        | `only-create`                                                                   | Strategy for applying default and new configurations during Graylog provisioning. Available values: `only-create`, `force-update`                                                                     |
| `logsRotationSizeGb`                       | integer                                                                                                                | no        | `20`                                                                            | Set maximum size of logs in `All messages` stream                                                                                                                                                     |
| `maxNumberOfIndices`                       | integer                                                                                                                | no        | `20`                                                                            | Set maximum number of indices                                                                                                                                                                         |
| `javaOpts`                                 | string                                                                                                                 | no        | `-`                                                                             | Graylog JVM options. For example: `-Xms1024m -Xmx1024m`                                                                                                                                               |
| `contentPacks`                             | [loggingservice/v11.ContentPackPathHTTPConfig](#contentpacks)                                                          | no        | `{}`                                                                            | Links to Graylog\'s Content Packs.                                                                                                                                                                    |
| `contentPackPaths`                         | string                                                                                                                 | no        | `-`                                                                             | Links to Graylog\'s Content Packs. To specify some Context Packs use comma (`,`) as a separator                                                                                                       |
| `customPluginsPaths`                       | string                                                                                                                 | no        | `-`                                                                             | Graylog plugins path                                                                                                                                                                                  |
| `startupTimeout`                           | integer                                                                                                                | no        | `10`                                                                            | Time in minutes that the operator waits for a Graylog pod to start                                                                                                                                    |
| `ringSize`                                 | integer                                                                                                                | no        | `262144`                                                                        | Total size of ring buffers. Must be a power of 2 (512, 1024, 2048, ...)                                                                                                                               |
| `inputbufferRingSize`                      | integer                                                                                                                | no        | `131072`                                                                        | Size of input ring buffers. Must be a power of 2 (512, 1024, 2048, ...)                                                                                                                               |
| `inputbufferProcessors`                    | integer                                                                                                                | no        | `3`                                                                             | The number of cores/processes to process Input Buffer                                                                                                                                                 |
| `processbufferProcessors`                  | integer                                                                                                                | no        | `6`                                                                             | The number of cores/processes to process Processing Buffer                                                                                                                                            |
| `outputbufferProcessors`                   | integer                                                                                                                | no        | `6`                                                                             | The number of cores/processes to process Output Buffer                                                                                                                                                |
| `outputbufferProcessorThreadsMaxPoolSize`  | integer                                                                                                                | no        | `33`                                                                            | The maximum number of threads to allow in the pool                                                                                                                                                    |
| `outputBatchSize`                          | integer                                                                                                                | no        | `1000`                                                                          | Batch size for the OpenSearch/Elasticsearch output. This is the **maximum** number of messages the OpenSearch/Elasticsearch output module will get at once and write to Elasticsearch in a batch call |
| `openSearch`                               | [loggingservice/v11.OpenSearch](#opensearch)                                                                           | no        | `{}`                                                                            | Configuration of OpenSearch.                                                                                                                                                                          |
| `streams`                                  | [loggingservice/v11.GraylogStream](#graylog-streams)                                                                   | no        | `{}`                                                                            | Configuration of Graylog Streams. System and audit logs will be created by default if the section is empty.                                                                                           |
| `tls`                                      | [loggingservice/v11.GraylogTLS](#graylog-tls)                                                                          | no        | `{}`                                                                            | Configuration of Graylog HTTPS/TLS for WebUI and default Inputs                                                                                                                                       |
| `authProxy`                                | [loggingservice/v11.GraylogAuthProxy](#graylog-auth-proxy)                                                             | no        | `{}`                                                                            | Configuration of Graylog auth-proxy that allows use LDAP integration with Graylog groups                                                                                                              |
| `user`                                     | string                                                                                                                 | no        | `admin`                                                                         | Username of Graylog super-admin user. Can't be empty                                                                                                                                                  |
| `password`                                 | string                                                                                                                 | no        | `admin`                                                                         | Password of Graylog super-admin user. Can't be empty                                                                                                                                                  |
| `s3Archive`                                | boolean                                                                                                                | no        | `false`                                                                         | Enables the use of S3 storage in the `graylog-archiving-plugin`                                                                                                                                       |
| `awsAccessKey`                             | string                                                                                                                 | no        | `""`                                                                            | AccessKey for using S3 storage in the `graylog-archiving-plugin`                                                                                                                                      |
| `awsSecretKey`                             | string                                                                                                                 | no        | `""`                                                                            | AccessKey for using S3 storage in the `graylog-archiving-plugin`                                                                                                                                      |
| `pathRepo`                                 | string                                                                                                                 | no        | `/usr/share/opensearch/snapshots/graylog/`                                      | Path in OpenSearch/Elasticsearch where data snapshots will be stored. These data will be uploaded to S3 later. Used in `graylog-archiving-plugin`                                                     |
| `serviceMonitor.scrapeInterval`            | string                                                                                                                 | no        | `30s`                                                                           | Sets metrics scrape interval                                                                                                                                                                          |
| `serviceMonitor.scrapeTimeout`             | string                                                                                                                 | no        | `10s`                                                                           | Sets metrics scrape timeout                                                                                                                                                                           |
<!-- markdownlint-enable line-length -->

Examples:

**Note:** This is only an example of the parameters format, not a recommended value.

```yaml
graylog:
  install: true
  dockerImage: graylog/graylog:5.2.12
  createIngress: true

  # Init image settings
  initSetupImage: alpine:3.21
  initContainerDockerImage: graylog-plugins-init-container:main
  initResources:
    requests:
      cpu: 50m
      memory: 128Mi
    limits:
      cpu: 100m
      memory: 256Mi

  # MongoDB sidecar settings
  mongoDBImage: mongo:5.0.31
  mongoUpgrade: true
  mongoDBUpgrade:
    mongoDBImage40: mongo:4.0.28
    mongoDBImage42: mongo:4.2.22
    mongoDBImage44: mongo:4.4.17
  mongoPersistentVolume: pv-mongodb
  mongoStorageClassName: cinder
  mongoResources:
    requests:
      cpu: 500m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 256Mi

  # Graylog deployment settings
  annotations:
    custom/annotation: value
  labels:
    app.kubernetes.io/part-of: logging
  graylogResources:
    requests:
      cpu: 500m
      memory: 1536Mi
    limits:
      cpu: 1000m
      memory: 2048Mi
  graylogPersistentVolume: pv-graylog
  graylogStorageClassName: nginx
  storageSize: 5Gi
  graylogSecretName: graylog-secret
  priorityClassName: system-cluster-critical
  startupTimeout: 10
  nodeSelectorKey: kubernetes.io/os
  nodeSelectorValue: linux
  host: https://graylog-service.kubernetes.test.org/
  ingressClassName: nginx
  securityResources:
    install: true
    name: logging-graylog

  # Graylog settings
  logLevel: INFO
  indexReplicas: 1
  indexShards: 5
  elasticsearchHost: http://user:password@elasticsearch.elasticsearch.svc:9200
  elasticsearchMaxTotalConnections: 100
  elasticsearchMaxTotalConnectionsPerRoute: 100
  inputPort: 12201
  contentDeployPolicy: force-update
  logsRotationSizeGb: 20
  maxNumberOfIndices: 20
  javaOpts: "-Xms1024m -Xmx2048"
  contentPackPaths: http://nexus.test.org/raw/custom-content-pack.zip
  customPluginsPaths: /path/to/plugins

  # Graylog performance settings and buffer sizes
  ringSize: 262144
  inputbufferRingSize: 131072
  inputbufferProcessors: 3
  processbufferProcessors: 6
  outputbufferProcessors: 6
  outputbufferProcessorThreadsMaxPoolSize: 33
  outputBatchSize: 1000

  # Graylog super admin credentials
  user: admin
  password: admin

  # Graylog archiving plugins settings. To store archives in S3
  s3Archive: true
  awsAccessKey: s3_access_key
  awsSecretKey: s3_secret_key
  pathRepo: /usr/share/opensearch/snapshots/graylog/

  serviceMonitor:
    scrapeInterval: 30s
    scrapeTimeout: 10s

  streams:
    ...
  tls:
    ...
  authProxy:
    ...
```

[Back to TOC](#table-of-contents)

### Graylog TLS

The `graylog.tls` section defines parameters enabling TLS for both the Graylog WebUI and the default Inputs.
It includes two subsections:

* `http` for securing the WebUI
* `input` for securing Graylog Inputs.

All parameters for the Graylog WebUI must be specified under the `graylog.tls.http` section, as shown in the example below:

```yaml
graylog:
  tls:
    http:
      enabled: true
      #...
```

<!-- markdownlint-disable line-length -->
| Parameter                         | Type    | Mandatory | Default value | Description                                                                                                                                                                                                    |
| --------------------------------- | ------- | --------- | ------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `enabled`                         | boolean | no        | `false`       | Enables TLS for the HTTP interface. When set to `true`, each connection to and from the Graylog server - except for Inputs - are secured with TLS, including the server’s own API calls to itself              |
| `cacerts`                         | string  | no        | `-`           | Specifies the name of the Secret with CA certificates for a custom CA store. If provided, all certificates from the Secret are added to the Java keystore which can also be used for TLS in the custom inputs. |
| `keyFilePassword`                 | string  | no        | `-`           | The password used to unlock the private key securing the HTTP interface                                                                                                                                        |
| `cert.secretKey`                  | string  | no        | `-`           | Name of the Secret containing the certificate                                                                                                                                                                  |
| `cert.secretName`                 | string  | no        | `-`           | Key (filename) within the Secret that stores the certificate                                                                                                                                                   |
| `key.secretKey`                   | string  | no        | `-`           | Name of the Secret containing the private key for the certificate                                                                                                                                              |
| `key.secretName`                  | string  | no        | `-`           | Key (filename) within the Secret that stores the private key for the certificate                                                                                                                               |
| `generateCerts.enabled`           | boolean | no        | `-`           | Enables integration with `cert-manager` for automatic certificate generation. This parameter is mutually exclusive with `cert` and `key` parameters                                                            |
| `generateCerts.secretName`        | string  | no        | `-`           | Name of the Secret where certificates generated by `cert-manager` will be stored                                                                                                                               |
| `generateCerts.clusterIssuerName` | string  | no        | `-`           | Issuer that will be used to generate certificates                                                                                                                                                              |
| `generateCerts.duration`          | string  | no        | `-`           | Sets certificates validity period                                                                                                                                                                              |
| `generateCerts.renewBefore`       | string  | no        | `-`           | Specifies the number of days prior to the certificate expiration date when they will be reissued                                                                                                               |
<!-- markdownlint-enable line-length -->

Examples:

**Note:** This is only an example of the parameters format, not a recommended value.

```yaml
graylog:
  tls:
    http:
      enabled: true
      cacerts: secret-ca
      keyFilePassword: changeit
      cert:
        secretName: graylog-http-tls-assets-0
        secretkey: graylog-http.crt
      key:
        secretName: graylog-http-tls-assets-0
        secretkey: graylog-http.key
      generateCerts:
        enabled: true
        secretName: graylog-http-cert-manager-tls
        clusterIssuerName: ""
        duration: 365
        renewBefore: 15
```

All parameters for the Graylog's Inputs must be specified under the `graylog.tls.input` section,
as shown in the example below:

```yaml
graylog:
  tls:
    input:
      enabled: true
      #...
```

<!-- markdownlint-disable line-length -->
| Parameter                         | Type    | Mandatory | Default value | Description                                                                                                                            |
| --------------------------------- | ------- | --------- | ------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `enabled`                         | boolean | no        | `false`       | Enables TLS for out-of-box GELF input managed by the operator                                                                          |
| `keyFilePassword`                 | string  | no        | `-`           | The password used to unlock the private key securing the Graylog input                                                                 |
| `ca.secretName`                   | string  | no        | `-`           | Name of the Kubernetes Secret with CA certificate. It's mutually exclusive with `generateCerts` section                                |
| `ca.secretKey`                    | string  | no        | `-`           | Key (filename) within the Secret that stores the CA certificate                                                                        |
| `cert.secretKey`                  | string  | no        | `-`           | Name of the Secret containing the certificate. Mutually exclusive with `generateCerts` parameters                                      |
| `cert.secretName`                 | string  | no        | `-`           | Key (filename) within the Secret that stores the certificate                                                                           |
| `key.secretKey`                   | string  | no        | `-`           | Name of the Secret containing the private key for the certificate. Mutually exclusive with `generateCerts` parameters                  |
| `key.secretName`                  | string  | no        | `-`           | Key (filename) in the Secret with the private key for the certificate                                                                  |
| `generateCerts.enabled`           | boolean | no        | `-`           | Enables integration with the `cert-manager` for automatic certificates generation. Mutually exclusive with `cert` and `key` parameters |
| `generateCerts.secretName`        | string  | no        | `-`           | Name of the Secret where certificates generated by `cert-manager` will be stored                                                       |
| `generateCerts.clusterIssuerName` | string  | no        | `-`           | Issuer that will be used to generate certificates                                                                                      |
| `generateCerts.duration`          | string  | no        | `-`           | Sets certificates validity period                                                                                                      |
| `generateCerts.renewBefore`       | string  | no        | `-`           | Specifies the number of days prior to the certificate expiration date when they will be reissued                                       |
<!-- markdownlint-enable line-length -->

Examples:

**Note:** This is only an example of the parameters format, not a recommended value.

```yaml
graylog:
  tls:
    input:
      enabled: true
      keyFilePassword: changeit
      # Certificates from Kubernetes Secrets
      ca:
        secretName: graylog-input-tls-assets-0
        secretKey: ca.crt
      cert:
        secretName: graylog-input-tls-assets-0
        secretKey: graylog-input.crt
      key:
        secretName: graylog-input-tls-assets-0
        secretKey: graylog-input.key

      # Integration with cert-manager
      generateCerts:
        enabled: true
        secretName: graylog-input-cert-manager-tls
        clusterIssuerName: ""
        duration: 365
        renewBefore: 15
```

[Back to TOC](#table-of-contents)

### OpenSearch

The `opensearch` section contains OpenSearch HTTP parameters.

All parameters for OpenSearch must be specified under the `graylog.openSearch` section as shown below:

```yaml
graylog:
  openSearch:
    http:
      credentials:
        ...
      tlsConfig:
        ...
    url:
```

<!-- markdownlint-disable line-length -->
| Parameter                           | Type               | Mandatory | Default value | Description                                                                                        |
| ----------------------------------- | ------------------ | --------- | ------------- | -------------------------------------------------------------------------------------------------- |
| `http.credentials.username`         | *SecretKeySelector | no        | `-`           | The secret that contains the username for Basic authentication                                     |
| `http.credentials.password`         | *SecretKeySelector | no        | `-`           | The secret that contains the password for Basic authentication                                     |
| `http.tlsConfig.ca`                 | *SecretKeySelector | no        | `-`           | Secret name and key where the CA is stored.                                                        |
| `http.tlsConfig.cert`               | *SecretKeySelector | no        | `-`           | Secret name and key where the certificate is stored.                                               |
| `http.tlsConfig.key`                | *SecretKeySelector | no        | `-`           | Secret name and key where the private key is stored.                                               |
| `http.tlsConfig.insecureSkipVerify` | boolean            | no        | `-`           | InsecureSkipVerify controls whether a client verifies the server's certificate chain and hostname. |
| `url`                               | string             | no        | `-`           | OpenSearch host                                                                                    |
<!-- markdownlint-enable line-length -->

Examples:

**Note:** This is only an example of the parameters format, not a recommended value.

```yaml
graylog:
  openSearch:
    http:
      credentials:
        username:
          name: openSearch-credentials-secret
          key: httpRequestUsername
        password:
          name: openSearch-credentials-secret
          key: httpRequestPassword
      tlsConfig:
        ca:
          name: secret-certificate
          key: cert-ca.pem
        cert:
          name: secret-certificate
          key: cert.crt
        key:
          name: secret-certificate
          key: cert.key
        insecureSkipVerify: false
    url: openSearch host
```

[Back to TOC](#table-of-contents)

### ContentPacks

The `contentPacks` section contains graylog content packs parameters.

All parameters for Content Packs must be specified under the `graylog.contentPacks` section as shown below:

```yaml
graylog:
  contentPacks:
    - http:
        credentials:
          ...
        tlsConfig:
          ...
      url:
    - http:
        credentials:
           ...
        tlsConfig:
           ...
      url:
```

<!-- markdownlint-disable line-length -->
| Parameter                           | Type               | Mandatory | Default value | Description                                                                                        |
| ----------------------------------- | ------------------ | --------- | ------------- | -------------------------------------------------------------------------------------------------- |
| `http.credentials.username`         | *SecretKeySelector | no        | `-`           | The secret that contains the username for Basic authentication                                     |
| `http.credentials.password`         | *SecretKeySelector | no        | `-`           | The secret that contains the password for Basic authentication                                     |
| `http.tlsConfig.ca`                 | *SecretKeySelector | no        | `-`           | Secret name and key where the CA is stored.                                                        |
| `http.tlsConfig.cert`               | *SecretKeySelector | no        | `-`           | Secret name and key where the certificate is stored.                                               |
| `http.tlsConfig.key`                | *SecretKeySelector | no        | `-`           | Secret name and key where the private key is stored.                                               |
| `http.tlsConfig.insecureSkipVerify` | boolean            | no        | `-`           | InsecureSkipVerify controls whether a client verifies the server's certificate chain and hostname. |
| `url`                               | string             | no        | `-`           | Content pack URL                                                                                   |
<!-- markdownlint-enable line-length -->

Examples:

**Note:** This is only an example of the parameters format, not a recommended value.

```yaml
graylog:
  contentPacks:
    - http:
        credentials:
          username:
            name: contentPack-credentials-secret
            key: httpRequestUsername
          password:
            name: contentPack-credentials-secret
            key: httpRequestPassword
        tlsConfig:
          ca:
            name: secret-certificate
            key: cert-ca.pem
          cert:
            name: secret-certificate
            key: cert.crt
          key:
            name: secret-certificate
            key: cert.key
          insecureSkipVerify: false
      url: contentPack url
    - http:
        ...
```

[Back to TOC](#table-of-contents)

### Graylog Streams

The `graylog.streams` section contains parameters to enable, disable or modify the retention strategy for the default
Graylog's Streams.

All parameters for Graylog streams must be specified under the `graylog.streams` section as shown below:

```yaml
graylog:
  streams:
    - name: "System logs"
      #...
```

<!-- markdownlint-disable line-length -->
| Parameter          | Type    | Mandatory | Default value | Description                                                                                                                              |
| ------------------ | ------- | --------- | ------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| `install`          | boolean | no        | `-`           | Enable or disable stream                                                                                                                 |
| `name`             | string  | no        | `-`           | The title of a Graylog's Stream. Available logs are `System logs`, `Audit logs`, `Access logs`, `Integration logs` and `Bill Cycle logs` |
| `rotationStrategy` | string  | no        | `sizeBased`   | Specifies the rotation strategy for the Stream’s IndexSet. Available values: `sizeBased`, `timeBased`                                    |
| `rotationPeriod`   | string  | no        | `-`           | Sets the rotation period for the Stream's IndexSet if `rotationStrategy` is `sizeBased`. The parameter must be set as ISO 8601 Duration  |
<!-- markdownlint-enable line-length -->

Examples:

**Note:** This is only an example of the parameters format, not a recommended value.

```yaml
graylog:
  streams:
    - name: "Audit logs"
      install: true
      rotationStrategy: "timeBased"
      rotationPeriod: "P1M"
    - name: "Integration logs"
      install: false
      rotationStrategy: "timeBased"
      rotationPeriod: "P1M"
    - name: "Access logs"
      install: false
      rotationStrategy: "timeBased"
      rotationPeriod: "P1M"
    - name: "Nginx logs"
      install: false
      rotationStrategy: "timeBased"
      rotationPeriod: "P1M"
    - name: "Bill Cycle logs"
      install: false
      rotationStrategy: "timeBased"
      rotationPeriod: "P1M15D"
```

[Back to TOC](#table-of-contents)

### Graylog Auth Proxy

The `graylog.authProxy` section includes parameters to enable and configure the Graylog authentication proxy.
This proxy facilitates user authentication and authorization for the Graylog server by integrating with third-party
identity providers, such as `Active Directory` or OAuth authorization services like `Keycloak`.

All parameters for authProxy must be specified under the `graylog.authProxy` section as shown below:

```yaml
graylog:
  authProxy:
    install: true
    #...
```

<!-- markdownlint-disable line-length -->
| Parameter              | Type                                                                                                                   | Mandatory | Default value                                                                      | Description                                                                                                    |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------------- | --------- | ---------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| `install`              | boolean                                                                                                                | no        | `false`                                                                            | Enable `graylog-auth-proxy` deployment                                                                         |
| `logLevel`             | string                                                                                                                 | no        | `INFO`                                                                             | Logging level. Allowed values: `DEBUG`, `INFO`, `WARNING`, `ERROR`, `CRITICAL`                                 |
| `image`                | string                                                                                                                 | yes       | `-`                                                                                | Image of `graylog-auth-proxy`                                                                                  |
| `preCreatedUsers`      | string                                                                                                                 | no        | `admin,auditViewer,operator,telegraf_operator,graylog-sidecar,graylog_api_th_user` | Comma-separated list of pre-created Graylog users for whom password rotation is not required                   |
| `rotationPassInterval` | integer                                                                                                                | no        | `3`                                                                                | Interval in days between password rotations for users not listed as pre-created                                |
| `roleMapping`          | string                                                                                                                 | no        | `'[]'`                                                                             | Filter used to map Graylog roles to LDAP users based on the memberOf attribute                                 |
| `streamMapping`        | string                                                                                                                 | no        | `""`                                                                               | Filter used to share Graylog streams between LDAP and Graylog users based on the memberOf attribute            |
| `resources`            | [core/v1.Resources](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core) | no        | `{}`                                                                               | Describes compute resources requests and limits for `graylog-auth-proxy` container                             |
| `requestsTimeout`      | float                                                                                                                  | no        | 30                                                                                 | A global timeout parameter affects requests to LDAP server, OAuth server and Graylog server                    |
| `authType`             | string                                                                                                                 | no        | `ldap`                                                                             | Defines which type of authentication protocol will be chosen (LDAP or OAuth 2.0). Allowed values: ldap, oauth  |
| `ldap`                 | [loggingservice/v11.GraylogAuthProxyLDAP](#graylog-auth-proxy-ldap)                                                    | no        | `-`                                                                                | Configuration for LDAP or AD connection                                                                        |
| `oauth`                | [loggingservice/v11.GraylogAuthProxyOAuth](#graylog-auth-proxy-oauth)                                                  | no        | `-`                                                                                | Configuration for OAuth 2.0 connection                                                                         |
<!-- markdownlint-enable line-length -->

Examples:

**Note:** This is only an example of the parameters format, not a recommended value.

```yaml
graylog:
  authProxy:
    install: true
    image: graylog-auth-proxy:main
    resources:
      requests:
        cpu: 100m
        memory: 128Mi
      limits:
        cpu: 200m
        memory: 256Mi

    preCreatedUsers: 3
    roleMapping: '["Reader"]'
    streamMapping: ''
    requestsTimeout: 30

    authType: 'ldap'

    ldap:
      ...
```

[Back to TOC](#table-of-contents)

#### Graylog Auth Proxy LDAP

The `graylog.authProxy.ldap` section contains parameters to configure LDAP provider for `graylog-auth-proxy`.

All parameters for `LDAP` auth proxy must be specified under the `graylog.authProxy.ldap` section as shown below:

```yaml
graylog:
  authProxy:
    authType: "ldap"
    ldap:
      ...
```

<!-- markdownlint-disable line-length -->
| Parameter                 | Type    | Mandatory | Default value               | Description                                                                                                         |
| ------------------------- | ------- | --------- | --------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `url`                     | string  | yes       | `-`                         | LDAP host to query users and their data                                                                             |
| `startTls`                | boolean | no        | `false`                     | Enables establishing a `STARTTLS` protected session                                                                 |
| `overSsl`                 | boolean | no        | `false`                     | Enables establishing an LDAP session over `SSL`                                                                     |
| `skipVerify`              | boolean | no        | `false`                     | Allows skipping verification of the LDAP server's certificate                                                       |
| `ca.secretName`           | string  | no        | `-`                         | Name of the Kubernetes Secret with CA certificate                                                                   |
| `ca.secretKey`            | string  | no        | `-`                         | Key (filename) in the Secret with CA certificate                                                                    |
| `cert.secretName`         | string  | no        | `-`                         | Name of the Kubernetes Secret with the client certificate                                                           |
| `cert.secretKey`          | string  | no        | `-`                         | Key (filename) in the Secret with the client certificate                                                            |
| `key.secretName`          | string  | no        | `-`                         | Name of the Kubernetes Secret with the private key for the client certificate                                       |
| `key.secretKey`           | string  | no        | `-`                         | Key (filename) in the Secret with the private key for the client certificate                                        |
| `disableReferrals`        | boolean | no        | `false`                     | Sets `ldap.OPT_REFERRALS` to zero                                                                                   |
| `searchFilter`            | string  | no        | `(cn=%(username)s)`         | LDAP filter for binding users                                                                                       |
| `baseDN`                  | string  | yes       | `-`                         | LDAP base DN                                                                                                        |
| `bindDN`                  | string  | yes       | `-`                         | LDAP bind DN                                                                                                        |
| `bindPassword`            | string  | yes       | `-`                         | LDAP password for the bind DN. Mutually exclusive with `bindPasswordSecret` parameter                               |
| `bindPasswordSecret.name` | string  | no        | `graylog-auth-proxy-secret` | Kubernetes Secret name with LDAP password for the bind DN. Mutually exclusive with `bindPassword` parameter         |
| `bindPasswordSecret.key`  | string  | no        | `bindPassword`              | Field in the Kubernetes Secret with LDAP password for the bind DN. Mutually exclusive with `bindPassword` parameter |
<!-- markdownlint-enable line-length -->

Examples:

**Note:** This is only an example of the parameters format, not a recommended value.

```yaml
graylog:
  authProxy:
    authType: "ldap"
    ldap:
      url: ldaps://openldap.test.org:636
      startTls: false
      overSsl: true
      skipVerify: false
      ca:
        secretName: graylog-auth-proxy-ldap-ca
        secretKey: ca.crt
      disableReferrals: false
      searchFilter: (cn=%(username)s)

      baseDN: cn=admin,dc=example,dc=com
      bindDN: dc=example,dc=com
      bindPassword: very_secret_password
      bindPasswordSecret:
        name: graylog-auth-proxy-credentials
        key: password
```

[Back to TOC](#table-of-contents)

#### Graylog Auth Proxy OAuth

The `graylog.authProxy.oauth` section contains parameters to configure OAuth provider for `graylog-auth-proxy`.

All parameters for `OAuth` auth proxy must be specified under the `graylog.authProxy.oauth` section as shown below:

```yaml
graylog:
  authProxy:
    authType: "oauth"
    oauth:
      ...
```

<!-- markdownlint-disable line-length -->
| Parameter                      | Type    | Mandatory | Default value               | Description                                                                                                                                                                                                                                                        |
| ------------------------------ | ------- | --------- | --------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `host`                         | string  | yes       | `-`                         | OAuth2 authorization server host                                                                                                                                                                                                                                   |
| `authorizationPath`            | string  | yes       | `-`                         | Path used to construct the URL for redirecting to the OAuth2 authorization server login page.                                                                                                                                                                      |
| `tokenPath`                    | string  | yes       | `-`                         | Path used to construct URL for getting access token from OAuth2 authorization server                                                                                                                                                                               |
| `userinfoPath`                 | string  | yes       | `-`                         | Path used to construct URL for getting information about current user from OAuth2 authorization server to get username and entities (roles, groups, etc.) for Graylog roles and streams mapping                                                                    |
| `redirectUri`                  | string  | no        | `-`                         | URI to redirect to after successful login on the OAuth2 authorization server. By default, it uses the graylog.host value (the same host as in the Graylog Ingress) with the /code path. Ensure that your OAuth server allows this URI as a valid redirect URI.     |
| `clientID`                     | string  | yes       | `-`                         | OAuth2 Client ID for the proxy                                                                                                                                                                                                                                     |
| `clientSecret`                 | string  | yes       | `-`                         | OAuth2 Client Secret for the proxy. Will be stored in the secret with .oauth.clientCredentialsSecret.name at key specified in the .oauth.clientCredentialsSecret.key.                                                                                              |
| `scopes`                       | string  | no        | `openid profile roles`      | OAuth2 scopes for the proxy separated by spaces. Configured for Keycloak server by default                                                                                                                                                                         |
| `userJsonpath`                 | string  | no        | `preferred_username`        | JSONPath expression (using jsonpath-ng) used to extract the username from the JSON response returned by the OAuth2 server via the userinfo endpoint. By default, it is configured for a Keycloak server.                                                           |
| `rolesJsonpath`                | string  | no        | `realm_access.roles[*]`     | JSONPath (jsonpath-ng) used to extract information about entities (roles, groups, etc.) for Graylog role and stream mapping from the JSON response returned by the OAuth2 server via the userinfo endpoint. By default, it is configured for a Keycloak server.    |
| `skipVerify`                   | boolean | no        | `false`                     | Allows skipping verification of the OAuth server's certificate                                                                                                                                                                                                     |
| `ca.secretName`                | string  | no        | `-`                         | Name of the Kubernetes Secret with the CA certificate                                                                                                                                                                                                              |
| `ca.secretKey`                 | string  | no        | `-`                         | Key (filename) in the Secret with the CA certificate                                                                                                                                                                                                               |
| `cert.secretName`              | string  | no        | `-`                         | Name of the Kubernetes Secret with the client certificate                                                                                                                                                                                                          |
| `cert.secretKey`               | string  | no        | `-`                         | Key (filename) in the Secret with the client certificate                                                                                                                                                                                                           |
| `key.secretName`               | string  | no        | `-`                         | Name of the Kubernetes Secret with the private key for the client certificate                                                                                                                                                                                      |
| `key.secretKey`                | string  | no        | `-`                         | Key (filename) in the Secret with the private key for the client certificate                                                                                                                                                                                       |
| `clientCredentialsSecret.name` | string  | no        | `graylog-auth-proxy-secret` | Kubernetes Secret name with the OAuth client secret. Mutually exclusive with the `clientSecret` parameter                                                                                                                                                          |
| `clientCredentialsSecret.key`  | string  | no        | `clientSecret`              | Field in the Kubernetes Secret with the OAuth client secret. Mutually exclusive with the `clientSecret` parameter                                                                                                                                                  |
<!-- markdownlint-enable line-length -->

Examples:

**Note:**  This is only an example of the parameters format, not a recommended value.

```yaml
graylog:
  authProxy:
    authType: "oauth"
    oauth:
      host: https://keycloak.server.com
      authorizationPath: /realms/test-realm/protocol/openid-connect/auth
      tokenPath: /realms/test-realm/protocol/openid-connect/token
      userinfoPath: /realms/test-realm/protocol/openid-connect/userinfo
      skipVerify: false
      ca:
        secretName: graylog-auth-proxy-oauth-ca
        secretKey: ca.crt

      clientID: graylog-auth-proxy
      clientSecret: <client-secret>
      scopes: "openid profile roles"
      userJsonpath: "preferred_username"
      rolesJsonpath: "realm_access.roles[*]"
      clientCredentialsSecret:
        name: graylog-auth-proxy-secret
        key: clientSecret
```

[Back to TOC](#table-of-contents)

## FluentBit

The `fluentbit` section contains parameters to enable and configure FluentBit logging agent.

All parameters for Fluentbit must be specified under the `fluentbit` section as shown below:

```yaml
fluentbit:
  install: true
  #...
```

<!-- markdownlint-disable line-length -->
| Parameter                         | Type                                                                                                                              | Mandatory  | Default value                                                                      | Description                                                                                                                                                                              |
| --------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- | ---------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `install`                         | boolean                                                                                                                           | no         | `false`                                                                            | Flag for installation `logging-fluentbit`                                                                                                                                                |
| `dockerImage`                     | string                                                                                                                            | no         | `-`                                                                                | Docker image of FluentBit                                                                                                                                                                |
| `configmapReload.dockerImage`     | string                                                                                                                            | no         | `-`                                                                                | Docker image of configmap_reload for FluentBit                                                                                                                                           |
| `configmapReload.resources`       | [core/v1.Resources](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core)            | no         | `{requests: {cpu: 10m, memory: 10Mi}, limits {cpu: 50m, memory: 50Mi}}`            | Describes resources requests and limits for `configmap_reload` container                                                                                                                 |
| `nodeSelectorKey`                 | string                                                                                                                            | no         | `-`                                                                                | NodeSelector key, can be multiple by OR condition, separated by comma, usually `role`                                                                                                    |
| `nodeSelectorValue`               | string                                                                                                                            | no         | `-`                                                                                | NodeSelector value, can be multiple by OR condition, separated by comma, usually `compute`                                                                                               |
| `tolerations`                     | [core/v1.Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core)                     | no         | `[]`                                                                               | List of tolerations applied to FluentBit Pods                                                                                                                                            |
| `affinity`                        | [core/v1.Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podaffinityterm-v1-core)                  | no         | `-`                                                                                | Specifies the pod\'s scheduling constraints                                                                                                                                              |
| `resources`                       | [core/v1.Resources](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core)            | no         | `{requests: {cpu: 50m, memory: 128Mi}, limits {cpu: 200m, memory: 512Mi}}`         | Describes compute resources requests and limits for the `fluentbit` container                                                                                                            |
| `graylogOutput`                   | boolean                                                                                                                           | no         | `true`                                                                             | Enables Graylog output                                                                                                                                                                   |
| `graylogHost`                     | string                                                                                                                            | no         | `-`                                                                                | Graylog host                                                                                                                                                                             |
| `graylogPort`                     | integer                                                                                                                           | no         | `12201`                                                                            | Graylog port                                                                                                                                                                             |
| `graylogProtocol`                 | string                                                                                                                            | no         | `tcp`                                                                              | The Graylog protocol. Available values: `tcp`, `udp`                                                                                                                                     |
| `extraFields`                     | map[string]string                                                                                                                 | no         | `-`                                                                                | Appends additional custom fields/labels to each log message by applying a filter based on the `record_modifier` plugin.                                                                  |
| `customInputConf`                 | string                                                                                                                            | no         | `-`                                                                                | Custom input configuration                                                                                                                                                               |
| `customFilterConf`                | string                                                                                                                            | no         | `-`                                                                                | Custom filter configuration                                                                                                                                                              |
| `customLuaScriptConf`             | map[string]string                                                                                                                 | no         | `-`                                                                                | Set of custom Lua scripts                                                                                                                                                                |
| `customOutputConf`                | string                                                                                                                            | no         | `-`                                                                                | Custom output configuration                                                                                                                                                              |
| `multilineFirstLineRegexp`        | string                                                                                                                            | no         | `/^(\\[\\d{4}\\-\\d{2}\\-\\d{2}).*/`                                               | Custom regular expression for the first line of multiline filter                                                                                                                         |
| `multilineOtherLinesRegexp`       | string                                                                                                                            | no         | `/^(?!\\[\\d{4}\\-\\d{2}\\-\\d{2}).*/`                                             | Custom regular expression for the other lines of multiline filter                                                                                                                        |
| `billCycleConf`                   | boolean                                                                                                                           | no         | `false`                                                                            | Filter for bil-cycle-logs stream                                                                                                                                                         |
| `securityContextPrivileged`       | boolean                                                                                                                           | no         | `false`                                                                            | Specifies securityContext.privileged for fluentbit container                                                                                                                             |
| `systemLogging`                   | boolean                                                                                                                           | no         | `false`                                                                            | Enables collecting of system logs                                                                                                                                                        |
| `systemLogType`                   | string                                                                                                                            | no         | `varlogmessages`                                                                   | Type of system logs to collect. Available values: `varlogmessages`, `varlogsyslog` and `systemd`                                                                                         |
| `systemAuditLogging`              | boolean                                                                                                                           | no         | `true`                                                                             | Enables input for system audit logs from `/var/log/audit/audit.log`.                                                                                                                     |
| `kubeAuditLogging`                | boolean                                                                                                                           | no         | `true`                                                                             | Enables input for Kubernetes audit logs from `/var/log/kubernetes/kube-apiserver-audit.log` and `/var/log/kubernetes/audit.log`.                                                         |
| `kubeApiserverAuditLogging`       | boolean                                                                                                                           | no         | `true`                                                                             | Enables input for Kubernetes APIServer audit logs from `/var/log/kube-apiserver/audit.log` for K8S and `/var/log/openshift-apiserver/audit.log` for OpenShift                            |
| `containerLogging`                | boolean                                                                                                                           | no         | `true`                                                                             | Enables input for container logs from `/var/logs/containers` for Docker or `/var/log/pods` for other engines.                                                                            |
| `totalLimitSize`                  | string                                                                                                                            | no         | `1024M`                                                                            | The size limitation of output buffer                                                                                                                                                     |
| `memBufLimit`                     | string                                                                                                                            | no         | `1024M`                                                                            | Limit of allowed storage for chucks of logs before sending                                                                                                                               |
| `additionalVolumes`               | [core/v1.PersistentVolumeSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#persistentvolumespec-v1-core) | no         | `{}`                                                                               | Additional volumes for FluentBit                                                                                                                                                         |
| `additionalVolumeMounts`          | [core/v1.VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volumemount-v1-core)                   | no         | `{}`                                                                               | Allows volume-mounts for FluentBit                                                                                                                                                       |
| `securityResources.install`       | boolean                                                                                                                           | no         | `false`                                                                            | Enables creating security resources as `PodSecurityPolicy`, `SecurityContextConstraints`                                                                                                 |
| `securityResources.name`          | string                                                                                                                            | no         | `logging-fluentbit`                                                                | Specifies the name for `PodSecurityPolicy` and `SecurityContextConstraints` objects.                                                                                                     |
| `podMonitor.scrapeTimeout`        | string                                                                                                                            | no         | `30s`                                                                              | Defines metrics scrape interval                                                                                                                                                          |
| `podMonitor.scrapeInterval`       | string                                                                                                                            | no         | `10s`                                                                              | Defines metrics scrape timeout                                                                                                                                                           |
| `tls`                             | [loggingservice/v11.FluentBitTLS](#fluentbit-tls)                                                                                 | no         | `{}`                                                                               | TLS configuration for FluentBit Graylog Output                                                                                                                                           |
| `annotations`                     | map                                                                                                                               | no         | `{}`                                                                               | Specifies the list of additional annotations                                                                                                                                             |
| `labels`                          | map                                                                                                                               | no         | `{}`                                                                               | Specifies the list of additional labels                                                                                                                                                  |
| `priorityClassName`               | string                                                                                                                            | no         | `-`                                                                                | Pod priority. Indicates the importance of a Pod relative to other Pods and prevents it from evicting.                                                                                    |
| `excludePath`                     | string                                                                                                                            | no         | `-`                                                                                | One or more shell patterns, separated by commas, to exclude files matching specific criteria, e.g: *.gz,*.zip.                                                                           |
| `output.loki.enabled`             | boolean                                                                                                                           | no         | `false`                                                                            | Enables Loki output                                                                                                                                                                      |
| `output.loki.host`                | string                                                                                                                            | no         | `-`                                                                                | Loki host                                                                                                                                                                                |
| `output.loki.tenant`              | string                                                                                                                            | no         | `-`                                                                                | Loki tenant ID                                                                                                                                                                           |
| `output.loki.auth.token.name`     | string                                                                                                                            | no         | `-`                                                                                | Authentication for Loki with token. Name of the secret where token is stored                                                                                                             |
| `output.loki.auth.token.key`      | string                                                                                                                            | no         | `-`                                                                                | Authentication for Loki with token. Name of key in the secret where token is stored                                                                                                      |
| `output.loki.auth.user.name`      | string                                                                                                                            | no         | `-`                                                                                | Basic authentication credentials for Loki. Name of the secret where username is stored                                                                                                   |
| `output.loki.auth.user.key`       | string                                                                                                                            | no         | `-`                                                                                | Basic authentication credentials for Loki. Name of key in the secret where username is stored                                                                                            |
| `output.loki.auth.password.name`  | string                                                                                                                            | no         | `-`                                                                                | Basic authentication credentials for Loki. Name of the secret where password is stored                                                                                                   |
| `output.loki.auth.password.key`   | string                                                                                                                            | no         | `-`                                                                                | Basic authentication credentials for Loki. Name of key in the secret where password is stored                                                                                            |
| `output.loki.staticLabels`        | string                                                                                                                            | no         | `job=fluentbit`                                                                    | Static labels that added as stream labels                                                                                                                                                |
| `output.loki.labelsMapping`       | string                                                                                                                            | no         | See example below                                                                  | Labels mappings that defines how to extract labels from each log record. Value should contain a JSON object                                                                              |
| `output.loki.extraParams`         | string                                                                                                                            | no         | See example below                                                                  | Additional configuration parameters for Loki output. See docs: [Loki output: configuration parameters](https://docs.fluentbit.io/manual/pipeline/outputs/loki#configuration-parameters). |
| `output.loki.tls.enabled`         | boolean                                                                                                                           | no         | `false`                                                                            | Flag to enable TLS connection for Loki output                                                                                                                                            |
| `output.loki.tls.ca.secretName`   | string                                                                                                                            | no         | `-`                                                                                | Name of Secret with Loki CA certificate                                                                                                                                                  |
| `output.loki.tls.ca.secretKey`    | string                                                                                                                            | no         | `-`                                                                                | Key (filename) in the Secret with Loki CA certificate                                                                                                                                    |
| `output.loki.tls.cert.secretName` | string                                                                                                                            | no         | `-`                                                                                | Name of Secret with Loki certificate                                                                                                                                                     |
| `output.loki.tls.cert.secretKey`  | string                                                                                                                            | no         | `-`                                                                                | Key (filename) in the Secret with Loki certificate                                                                                                                                       |
| `output.loki.tls.key.secretName`  | string                                                                                                                            | no         | `-`                                                                                | Name of Secret with key                                                                                                                                                                  |
| `output.loki.tls.key.secretKey`   | string                                                                                                                            | no         | `-`                                                                                | Key (filename) in the Secret with key                                                                                                                                                    |
| `output.loki.tls.verify`          | boolean                                                                                                                           | no         | `true`                                                                             | Force certificate validation                                                                                                                                                             |
| `output.loki.tls.keyPasswd`       | boolean                                                                                                                           | no         | `-`                                                                                | Optional password for private key file                                                                                                                                                   |
| `output.http.enabled`             | boolean                                                                                                                           | no         | `false`                                                                            | Enables `http` output.                                                                                                                                                                   |
| `output.http.host`                | string                                                                                                                            | no         | `-`                                                                                | Http host. Example: vlsingle-k8s.victorialogs                                                                                                                                            |
| `output.http.port`                | integer                                                                                                                           | no         | `9428`                                                                             | Http server port                                                                                                                                                                         |
| `output.http.uri`                 | string                                                                                                                            | no         | `/insert/jsonline?_stream_fields=stream&_msg_field=short_message&_time_field=time` | HTTP URI for the target web server                                                                                                                                                       |
| `output.http.auth.token.name`     | string                                                                                                                            | no         | `-`                                                                                | Authentication for http with token. Name of the secret where token is stored                                                                                                             |
| `output.http.auth.token.key`      | string                                                                                                                            | no         | `-`                                                                                | Authentication for http with token. Name of the key in the secret where token is stored                                                                                                  |
| `output.http.auth.user.name`      | string                                                                                                                            | no         | `-`                                                                                | Basic authentication credentials for http. Name of the secret where username is stored                                                                                                   |
| `output.http.auth.user.key`       | string                                                                                                                            | no         | `-`                                                                                | Basic authentication credentials for http. Name of key in the secret where username is stored                                                                                            |
| `output.http.auth.password.name`  | string                                                                                                                            | no         | `-`                                                                                | Basic authentication credentials for http. Name of the secret where password is stored                                                                                                   |
| `output.http.auth.password.key`   | string                                                                                                                            | no         | `-`                                                                                | Basic authentication credentials for http. Name of key in the secret where password is stored                                                                                            |
| `output.http.tls.keyPasswd`       | boolean                                                                                                                           | no         | `-`                                                                                | Optional password for private key file                                                                                                                                                   |
| `output.http.extraParams`         | string                                                                                                                            | no         | See example below                                                                  | Additional configuration parameters for http output. See docs: [http output: configuration parameters](https://docs.fluentbit.io/manual/pipeline/outputs/http#configuration-parameters)  |
| `output.http.compress`            | string                                                                                                                            | no         | `-`                                                                                | Payload compression mechanism. Allowed values: `gzip`, `snappy`, `zstd`. Disabled by default                                                                                             |
| `output.http.format`              | string                                                                                                                            | no         | `-`                                                                                | Data format to be used in the HTTP request body. Supported formats: `gelf`, `json`, `json_stream`, `json_lines`, `msgpack`.                                                              |
| `output.http.jsonDateFormat`      | string                                                                                                                            | no         | `iso8601`                                                                          | Format of the date. Supported formats: double, epoch, epoch_ms, iso8601, java_sql_timestamp.                                                                                             |
| `output.http.tls.enabled`         | boolean                                                                                                                           | no         | `false`                                                                            | Flag to enable TLS connection for http output                                                                                                                                            |
| `output.http.tls.ca.name`         | string                                                                                                                            | no         | `-`                                                                                | Name of Secret with http CA certificate                                                                                                                                                  |
| `output.http.tls.ca.key`          | string                                                                                                                            | no         | `-`                                                                                | Key (filename) in the Secret with http CA certificate                                                                                                                                    |
| `output.http.tls.cert.name`       | string                                                                                                                            | no         | `-`                                                                                | Name of Secret with http certificate                                                                                                                                                     |
| `output.http.tls.cert.key`        | string                                                                                                                            | no         | `-`                                                                                | Key (filename) in the Secret with http certificate                                                                                                                                       |
| `output.http.tls.key.name`        | string                                                                                                                            | no         | `-`                                                                                | Name of Secret with key                                                                                                                                                                  |
| `output.http.tls.key.key`         | string                                                                                                                            | no         | `-`                                                                                | Key (filename) in the Secret with private key                                                                                                                                            |
| `output.http.tls.verify`          | boolean                                                                                                                           | no         | `true`                                                                             | Force certificate validation                                                                                                                                                             |
| `output.otel.enabled`             | boolean                                                                                                                           | no         | `false`                                                                            | Enables `otel` output.                                                                                                                                                                   |
| `output.otel.host`                | string                                                                                                                            | no         | `-`                                                                                | Otel host. Opentelemetry-collector or victorialogs. Example: vlsingle-k8s.victorialogs                                                                                                   |
| `output.otel.port`                | integer                                                                                                                           | no         | `9428`                                                                             | Otel server port                                                                                                                                                                         |
| `output.otel.target`              | string                                                                                                                            | no         | `victorialogs`                                                                     | Target server for logs ingestion. Used to switch the output configuration depending on a specific storage. If set to `victorialogs` it includes victorialogs specific headers.           |
| `output.otel.logsUri`             | string                                                                                                                            | no         | `/insert/opentelemetry/v1/logs`                                                    | URI for logs ingestion                                                                                                                                                                   |
| `output.otel.auth.token.name`     | string                                                                                                                            | no         | `-`                                                                                | Authentication for otel with token. Name of the secret where token is stored                                                                                                             |
| `output.otel.auth.token.key`      | string                                                                                                                            | no         | `-`                                                                                | Authentication for otel with token. Name of the key in the secret where token is stored                                                                                                  |
| `output.otel.auth.user.name`      | string                                                                                                                            | no         | `-`                                                                                | Basic authentication credentials for otel. Name of the secret where username is stored                                                                                                   |
| `output.otel.auth.user.key`       | string                                                                                                                            | no         | `-`                                                                                | Basic authentication credentials for otel. Name of key in the secret where username is stored                                                                                            |
| `output.otel.auth.password.name`  | string                                                                                                                            | no         | `-`                                                                                | Basic authentication credentials for otel. Name of the secret where password is stored                                                                                                   |
| `output.otel.auth.password.key`   | string                                                                                                                            | no         | `-`                                                                                | Basic authentication credentials for otel. Name of key in the secret where password is stored                                                                                            |
| `output.otel.tls.keyPasswd`       | boolean                                                                                                                           | no         | `-`                                                                                | Optional password for private key file                                                                                                                                                   |
| `output.otel.extraParams`         | string                                                                                                                            | no         | See example below                                                                  | Additional configuration parameters for otel output. See docs: [Opentelemetry output](https://docs.fluentbit.io/manual/data-pipeline/outputs/opentelemetry)                              |
| `output.otel.compress`            | string                                                                                                                            | no         | `-`                                                                                | Payload compression mechanism. Allowed values: `gzip` and `zstd`. Disabled by default                                                                                                    |
| `output.otel.logSuppressInterval` | integer                                                                                                                           | no         | `-`                                                                                | Suppresses log messages from output plugin that appear similar within a specified time interval. 0 - no suppression.                                                                     |
| `output.otel.tls.enabled`         | boolean                                                                                                                           | no         | `false`                                                                            | Flag to enable TLS connection for otel output                                                                                                                                            |
| `output.otel.tls.ca.name`         | string                                                                                                                            | no         | `-`                                                                                | Name of Secret with otel CA certificate                                                                                                                                                  |
| `output.otel.tls.ca.key`          | string                                                                                                                            | no         | `-`                                                                                | Key (filename) in the Secret with otel CA certificate                                                                                                                                    |
| `output.otel.tls.cert.name`       | string                                                                                                                            | no         | `-`                                                                                | Name of Secret with otel certificate                                                                                                                                                     |
| `output.otel.tls.cert.key`        | string                                                                                                                            | no         | `-`                                                                                | Key (filename) in the Secret with otel certificate                                                                                                                                       |
| `output.otel.tls.key.name`        | string                                                                                                                            | no         | `-`                                                                                | Name of Secret with key                                                                                                                                                                  |
| `output.otel.tls.key.key`         | string                                                                                                                            | no         | `-`                                                                                | Key (filename) in the Secret with private key                                                                                                                                            |
| `output.otel.tls.verify`          | boolean                                                                                                                           | no         | `true`                                                                             | Force certificate validation                                                                                                                                                             |
<!-- markdownlint-enable line-length -->

Examples:

**Note:** This is only an example of the parameters format, not a recommended value.

```yaml
fluentbit:
  install: true
  dockerImage: fluent/fluent-bit:4.0.0

  graylogOutput: true
  graylogHost: graylog.logging.svc
  graylogPort: 12201
  graylogProtocol: tcp

  extraFields:
    foo_key: foo_value
    bar_key: bar_value
  systemLogging: true
  systemLogType: varlogmessages
  systemAuditLogging: true
  kubeAuditLogging: true
  kubeApiserverAuditLogging: true
  containerLogging: true

  customInputConf: |-
    [INPUT]
      Name   random
  customFilterConf: |-
    [FILTER]
      Name record_modifier
      Match *
      Record testField fluent-bit
  customOutputConf: |-
    [OUTPUT]
      Name null
      Match fluent.*
  customLuaScriptConf:
    "script1.lua": |-
      function()
        ...
      end
    "script2.lua": |-
      function()
        ...
      end

  multilineFirstLineRegexp: "/^(\\[\\d{4}\\-\\d{2}\\-\\d{2}).*/"
  multilineOtherLinesRegexp: "/^(?!\\[\\d{4}\\-\\d{2}\\-\\d{2}).*/"
  billCycleConf: true

  securityContextPrivileged: false
  nodeSelectorKey: kubernetes.io/os
  nodeSelectorValue: linux
  tolerations:
  - key: node-role.kubernetes.io/master
    operator: Exists
  - operator: Exists
    effect: NoExecute
  - operator: Exists
    effect: NoSchedule

  totalLimitSize: 1024M
  memBufLimit: 1024M

  # FluentBit additional volumes
  additionalVolumes:
    - name: dockervolume
      hostPath:
        path: /var/lib/docker
        type: Directory
  additionalVolumeMounts:
    - name: dockervolume
      mountPath: /var/log/docker

  excludePath:
    /var/log/pods/mongo_cnfrs2*/cnfrs2/*.log,
    /var/log/pods/mongo_cnfrs0*/cnfrs0/*.log,
    /var/log/pods/mongo_cnfrs1*/cnfrs1/*.log
```

Example of FluentBit configuration with Loki output enabled:

```yaml
fluentbit:
  install: true
  dockerImage: fluent/fluent-bit:4.0.0

  graylogOutput: false

  output:
    loki:
      enabled: true
      host: loki-write.loki.svc
      tenant: dev-cloud-1
      auth:
        token:
          name: loki-secret
          key: token
        user:
          name: loki-secret
          key: user
        password:
          name: loki-secret
          key: password
      staticLabels: job=fluentbit
      labelsMapping: |-
        {
            "container": "container",
            "pod": "pod",
            "namespace": "namespace",
            "stream": "stream",
            "level": "level",
            "hostname": "hostname",
            "nodename": "nodename",
            "request_id": "request_id",
            "tenant_id": "tenant_id",
            "addressTo": "addressTo",
            "originating_bi_id": "originating_bi_id",
            "spanId": "spanId"
        }
      tls:
        enabled: true
        ca:
          secretName: secret-ca
          secretKey: ca.crt
        cert:
          secretName: secret-cert
          secretKey: certificate.crt
        key:
          secretName: secret-key
          secretKey: privateKey.key
        verify: true
        keyPasswd: secretKeyPassword
      # See docs: https://docs.fluentbit.io/manual/pipeline/outputs/loki#configuration-parameters
      extraParams: |
          workers                2
          Retry_Limit            32
          storage.total_limit_size  5000M
          net.connect_timeout 20
```

Example of FluentBit configuration with HTTP output enabled:

```yaml
fluentbit:
  install: true
  dockerImage: fluent/fluent-bit:4.0.0

  graylogOutput: false

  output:
    http:
      enabled: true
      host: vlsingle-k8s.victorialogs
      port: 9428
      auth:
        token:
          name: http-secret
          key: token
        user:
          name: http-secret
          key: user
        password:
          name: http-secret
          key: password
      tls:
        enabled: true
        ca:
          secretName: secret-ca
          secretKey: ca.crt
        cert:
          secretName: secret-cert
          secretKey: certificate.crt
        key:
          secretName: secret-key
          secretKey: privateKey.key
        verify: true
        keyPasswd: secretKeyPassword
      # See docs: https://docs.fluentbit.io/manual/pipeline/outputs/http#configuration-parameters
      extraParams: |
          workers          2
          header           AccountID 12
          header           ProjectID 23
```

Example of FluentBit configuration with Opentelemetry output enabled:

```yaml
fluentbit:
  install: true
  dockerImage: fluent/fluent-bit:4.0.0

  graylogOutput: false

  output:
    otel:
      enabled: true
      host: vlsingle-k8s.victorialogs
      port: 9428
      logsUri: /api/v1/logs
      auth:
        token:
          name: otel-secret
          key: token
        user:
          name: otel-secret
          key: user
        password:
          name: otel-secret
          key: password
      compress: zstd
      logSuppressInterval: 5
      tls:
        enabled: true
        ca:
          secretName: secret-ca
          secretKey: ca.crt
        cert:
          secretName: secret-cert
          secretKey: certificate.crt
        key:
          secretName: secret-key
          secretKey: privateKey.key
        verify: true
        keyPasswd: secretKeyPassword
      # See docs: https://docs.fluentbit.io/manual/pipeline/outputs/http#configuration-parameters
      extraParams: |
          workers          2
          header           AccountID 12
          header           ProjectID 23
```

[Back to TOC](#table-of-contents)

### FluentBit Aggregator

The `fluentbit.aggregator` section contains parameters to enable and configure the FluentBit aggregator.
It can be used to balance the load from FluentBit to Graylog and provide an ability to store logs in case of Graylog
unavailability.

All parameters for `Fluentbit Aggregator` must be specified under the `fluentbit.aggregator` section as shown below:

```yaml
fluentbit:
  #...
  aggregator:
    install: true
    #...
```

<!-- markdownlint-disable line-length -->
| Parameter                         | Type                                                                                                                   | Mandatory  | Default value                                                                      | Description                                                                                                                                                                                                                       |
| --------------------------------- | ---------------------------------------------------------------------------------------------------------------------- | ---------- | ---------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `install`                         | boolean                                                                                                                | no         | `false`                                                                            | Allows to install FluentBit aggregator                                                                                                                                                                                            |
| `dockerImage`                     | string                                                                                                                 | no         | `-`                                                                                | Docker image of FluentBit aggregator                                                                                                                                                                                              |
| `configmapReload.dockerImage`     | string                                                                                                                 | no         | `-`                                                                                | Docker image of configmap_reload for FluentBit aggregator                                                                                                                                                                         |
| `configmapReload.resources`       | [core/v1.Resources](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core) | no         | `{requests: {cpu: 10m, memory: 10Mi}, limits {cpu: 50m, memory: 50Mi}}`            | Describes compute resources requests and limits for `configmap_reload` container                                                                                                                                                  |
| `replicas`                        | integer                                                                                                                | no         | `2`                                                                                | Number of FluentBit aggregator pods                                                                                                                                                                                               |
| `graylogOutput`                   | boolean                                                                                                                | no         | `true`                                                                             | Flag for using Graylog output                                                                                                                                                                                                     |
| `graylogHost`                     | string                                                                                                                 | no         | `-`                                                                                | Points to Graylog host. The parameter is used if aggregator is enabled                                                                                                                                                            |
| `graylogPort`                     | integer                                                                                                                | no         | `12201`                                                                            | Graylog port. The parameter is used if aggregator is enabled                                                                                                                                                                      |
| `graylogProtocol`                 | string                                                                                                                 | no         | `tcp`                                                                              | The Graylog protocol. Possible values: tcp/udp. The parameter is used if aggregator is enabled                                                                                                                                    |
| `extraFields`                     | map[string]string                                                                                                      | no         | `-`                                                                                | Adds additional custom fields/labels to every log message by using filter based on record_modifier plugin.                                                                                                                        |
| `customFilterConf`                | string                                                                                                                 | no         | `-`                                                                                | Custom filter configuration. The parameter is used if aggregator is enabled                                                                                                                                                       |
| `customOutputConf`                | string                                                                                                                 | no         | `-`                                                                                | Custom output configuration. The parameter is used if aggregator is enabled                                                                                                                                                       |
| `customLuaScriptConf`             | map[string]string                                                                                                      | no         | `-`                                                                                | Set of custom Lua scripts                                                                                                                                                                                                         |
| `multilineFirstLineRegexp`        | string                                                                                                                 | no         | `/^(\\[\\d{4}\\-\\d{2}\\-\\d{2}).*/`                                               | Custom regular expression for the first line of multiline filter                                                                                                                                                                  |
| `multilineOtherLinesRegexp`       | string                                                                                                                 | no         | `/^(?!\\[\\d{4}\\-\\d{2}\\-\\d{2}).*/`                                             | Custom regular expression for the other lines of multiline filter                                                                                                                                                                 |
| `totalLimitSize`                  | string                                                                                                                 | no         | `1024M`                                                                            | The size limitation of output buffer                                                                                                                                                                                              |
| `memBufLimit`                     | string                                                                                                                 | no         | `5M`                                                                               | Limit of allowed storage for chucks of logs before sending.                                                                                                                                                                       |
| `startupTimeout`                  | integer                                                                                                                | no         | `8`                                                                                | Time the operator waits for Aggregator pod(s) to start, in minutes                                                                                                                                                                |
| `tolerations`                     | [core/v1.Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core)          | no         | `[]`                                                                               | List of tolerations applied to FluentBit Pods                                                                                                                                                                                     |
| `affinity`                        | [core/v1.Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podaffinityterm-v1-core)       | no         | `-`                                                                                | It specifies the pod\'s scheduling constraints                                                                                                                                                                                    |
| `resources`                       | [core/v1.Resources](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core) | no         | `{requests: {cpu: 500m, memory: 512Mi}, limits {cpu: 2000m, memory: 2048Mi}}`      | Describes compute resources requests and limits for fluentbit container                                                                                                                                                           |
| `volume.bind`                     | boolean                                                                                                                | no         | `false`                                                                            | Allows installing PVCs for Aggregator pods                                                                                                                                                                                        |
| `volume.storageClassName`         | string                                                                                                                 | no         | `""`                                                                               | Aggregator PVC storage class name                                                                                                                                                                                                 |
| `volume.storageSize`              | string                                                                                                                 | no         | `2Gi`                                                                              | Storage size limit of PVC                                                                                                                                                                                                         |
| `securityResources.install`       | boolean                                                                                                                | no         | `false`                                                                            | Enables creating security resources as `PodSecurityPolicy`, `SecurityContextConstraints`                                                                                                                                          |
| `securityResources.name`          | string                                                                                                                 | no         | `logging-fluentbit-aggregator`                                                     | Defines a name of `PodSecurityPolicy`, `SecurityContextConstraints` objects                                                                                                                                                       |
| `podMonitor.scrapeInterval`       | string                                                                                                                 | no         | `30s`                                                                              | Defines metrics scrape interval                                                                                                                                                                                                   |
| `podMonitor.scrapeTimeout`        | string                                                                                                                 | no         | `10s`                                                                              | Defines metrics scrape timeout                                                                                                                                                                                                    |
| `tls`                             | [loggingservice/v11.FluentBitTLS](#fluentbit-tls)                                                                      | no         | `{}`                                                                               | TLS configuration for FluentBit Graylog Output                                                                                                                                                                                    |
| `priorityClassName`               | string                                                                                                                 | no         | `-`                                                                                | Pod priority. Priority indicates the importance of a Pod relative to other Pods and prevents them from evicting.                                                                                                                  |
| `output.loki.enabled`             | boolean                                                                                                                | no         | false                                                                              | Flag for enabling Loki output                                                                                                                                                                                                     |
| `output.loki.host`                | string                                                                                                                 | no         | `-`                                                                                | Loki host                                                                                                                                                                                                                         |
| `output.loki.tenant`              | string                                                                                                                 | no         | `-`                                                                                | Loki tenant ID                                                                                                                                                                                                                    |
| `output.loki.auth.token.name`     | string                                                                                                                 | no         | `-`                                                                                | Authentication for Loki with token. Name of the secret where token is stored                                                                                                                                                      |
| `output.loki.auth.token.key`      | string                                                                                                                 | no         | `-`                                                                                | Authentication for Loki with token. Name of key in the secret where token is stored                                                                                                                                               |
| `output.loki.auth.user.name`      | string                                                                                                                 | no         | `-`                                                                                | Basic authentication credentials for Loki. Name of the secret where username is stored                                                                                                                                            |
| `output.loki.auth.user.key`       | string                                                                                                                 | no         | `-`                                                                                | Basic authentication credentials for Loki. Name of key in the secret where username is stored                                                                                                                                     |
| `output.loki.auth.password.name`  | string                                                                                                                 | no         | `-`                                                                                | Basic authentication credentials for Loki. Name of the secret where password is stored                                                                                                                                            |
| `output.loki.auth.password.key`   | string                                                                                                                 | no         | `-`                                                                                | Basic authentication credentials for Loki. Name of key in the secret where password is stored                                                                                                                                     |
| `output.loki.staticLabels`        | string                                                                                                                 | no         | `job=fluentbit`                                                                    | Static labels that added as stream labels                                                                                                                                                                                         |
| `output.loki.labelsMapping`       | string                                                                                                                 | no         | See example below                                                                  | Labels mappings that defines how to extract labels from each log record. Value should contain a JSON object                                                                                                                       |
| `output.loki.extraParams`         | string                                                                                                                 | no         | See example below                                                                  | Additional configuration parameters for Loki output. See docs: [https://docs.fluentbit.io/manual/pipeline/outputs/loki#configuration-parameters](https://docs.fluentbit.io/manual/pipeline/outputs/loki#configuration-parameters) |
| `output.loki.tls.enabled`         | boolean                                                                                                                | no         | `false`                                                                            | Flag to enable TLS connection for Loki output                                                                                                                                                                                     |
| `output.loki.tls.ca.secretName`   | string                                                                                                                 | no         | `-`                                                                                | Name of Secret with Loki CA certificate                                                                                                                                                                                           |
| `output.loki.tls.ca.secretKey`    | string                                                                                                                 | no         | `-`                                                                                | Key (filename) in the Secret with Loki CA certificate                                                                                                                                                                             |
| `output.loki.tls.cert.secretName` | string                                                                                                                 | no         | `-`                                                                                | Name of Secret with Loki certificate                                                                                                                                                                                              |
| `output.loki.tls.cert.secretKey`  | string                                                                                                                 | no         | `-`                                                                                | Key (filename) in the Secret with Loki certificate                                                                                                                                                                                |
| `output.loki.tls.key.secretName`  | string                                                                                                                 | no         | `-`                                                                                | Name of Secret with key                                                                                                                                                                                                           |
| `output.loki.tls.key.secretKey`   | string                                                                                                                 | no         | `-`                                                                                | Key (filename) in the Secret with key                                                                                                                                                                                             |
| `output.loki.tls.verify`          | boolean                                                                                                                | no         | `true`                                                                             | Force certificate validation                                                                                                                                                                                                      |
| `output.loki.tls.keyPasswd`       | boolean                                                                                                                | no         | `-`                                                                                | Optional password for private key file                                                                                                                                                                                            |
| `output.http.enabled`             | boolean                                                                                                                | no         | `false`                                                                            | Enables `http` output.                                                                                                                                                                                                            |
| `output.http.host`                | string                                                                                                                 | no         | `-`                                                                                | Http host. Example: vlsingle-k8s.victorialogs                                                                                                                                                                                     |
| `output.http.port`                | integer                                                                                                                | no         | `9428`                                                                             | Http server port                                                                                                                                                                                                                  |
| `output.http.uri`                 | string                                                                                                                 | no         | `/insert/jsonline?_stream_fields=stream&_msg_field=short_message&_time_field=time` | HTTP URI for the target web server                                                                                                                                                                                                |
| `output.http.auth.token.name`     | string                                                                                                                 | no         | `-`                                                                                | Authentication for http with token. Name of the secret where token is stored                                                                                                                                                      |
| `output.http.auth.token.key`      | string                                                                                                                 | no         | `-`                                                                                | Authentication for http with token. Name of the key in the secret where token is stored                                                                                                                                           |
| `output.http.auth.user.name`      | string                                                                                                                 | no         | `-`                                                                                | Basic authentication credentials for http. Name of the secret where username is stored                                                                                                                                            |
| `output.http.auth.user.key`       | string                                                                                                                 | no         | `-`                                                                                | Basic authentication credentials for http. Name of key in the secret where username is stored                                                                                                                                     |
| `output.http.auth.password.name`  | string                                                                                                                 | no         | `-`                                                                                | Basic authentication credentials for http. Name of the secret where password is stored                                                                                                                                            |
| `output.http.auth.password.key`   | string                                                                                                                 | no         | `-`                                                                                | Basic authentication credentials for http. Name of key in the secret where password is stored                                                                                                                                     |
| `output.http.tls.keyPasswd`       | boolean                                                                                                                | no         | `-`                                                                                | Optional password for private key file                                                                                                                                                                                            |
| `output.http.extraParams`         | string                                                                                                                 | no         | See example below                                                                  | Additional configuration parameters for http output. See docs: [http output: configuration parameters](https://docs.fluentbit.io/manual/pipeline/outputs/http#configuration-parameters)                                           |
| `output.http.compress`            | string                                                                                                                 | no         | `-`                                                                                | Payload compression mechanism. Allowed values: `gzip`, `snappy`, `zstd`. Disabled by default                                                                                                                                      |
| `output.http.format`              | string                                                                                                                 | no         | `-`                                                                                | Data format to be used in the HTTP request body. Supported formats: `gelf`, `json`, `json_stream`, `json_lines`, `msgpack`.                                                                                                       |
| `output.http.jsonDateFormat`      | string                                                                                                                 | no         | `iso8601`                                                                          | Format of the date. Supported formats: double, epoch, epoch_ms, iso8601, java_sql_timestamp.                                                                                                                                      |
| `output.http.tls.enabled`         | boolean                                                                                                                | no         | `false`                                                                            | Flag to enable TLS connection for http output                                                                                                                                                                                     |
| `output.http.tls.ca.name`         | string                                                                                                                 | no         | `-`                                                                                | Name of Secret with http CA certificate                                                                                                                                                                                           |
| `output.http.tls.ca.key`          | string                                                                                                                 | no         | `-`                                                                                | Key (filename) in the Secret with http CA certificate                                                                                                                                                                             |
| `output.http.tls.cert.name`       | string                                                                                                                 | no         | `-`                                                                                | Name of Secret with http certificate                                                                                                                                                                                              |
| `output.http.tls.cert.key`        | string                                                                                                                 | no         | `-`                                                                                | Key (filename) in the Secret with http certificate                                                                                                                                                                                |
| `output.http.tls.key.name`        | string                                                                                                                 | no         | `-`                                                                                | Name of Secret with key                                                                                                                                                                                                           |
| `output.http.tls.key.key`         | string                                                                                                                 | no         | `-`                                                                                | Key (filename) in the Secret with private key                                                                                                                                                                                     |
| `output.http.tls.verify`          | boolean                                                                                                                | no         | `true`                                                                             | Force certificate validation                                                                                                                                                                                                      |
| `output.otel.enabled`             | boolean                                                                                                                | no         | `false`                                                                            | Enables `otel` output.                                                                                                                                                                                                            |
| `output.otel.host`                | string                                                                                                                 | no         | `-`                                                                                | Otel host. Opentelemetry-collector or victorialogs. Example: vlsingle-k8s.victorialogs                                                                                                                                            |
| `output.otel.port`                | integer                                                                                                                | no         | `9428`                                                                             | Otel server port                                                                                                                                                                                                                  |
| `output.otel.target`              | string                                                                                                                 | no         | `victorialogs`                                                                     | Target server for logs ingestion. Used to switch the output configuration depending on a specific storage. If set to `victorialogs` it includes victorialogs specific headers.                                                    |
| `output.otel.logsUri`             | string                                                                                                                 | no         | `/insert/opentelemetry/v1/logs`                                                    | URI for logs ingestion                                                                                                                                                                                                            |
| `output.otel.auth.token.name`     | string                                                                                                                 | no         | `-`                                                                                | Authentication for otel with token. Name of the secret where token is stored                                                                                                                                                      |
| `output.otel.auth.token.key`      | string                                                                                                                 | no         | `-`                                                                                | Authentication for otel with token. Name of the key in the secret where token is stored                                                                                                                                           |
| `output.otel.auth.user.name`      | string                                                                                                                 | no         | `-`                                                                                | Basic authentication credentials for otel. Name of the secret where username is stored                                                                                                                                            |
| `output.otel.auth.user.key`       | string                                                                                                                 | no         | `-`                                                                                | Basic authentication credentials for otel. Name of key in the secret where username is stored                                                                                                                                     |
| `output.otel.auth.password.name`  | string                                                                                                                 | no         | `-`                                                                                | Basic authentication credentials for otel. Name of the secret where password is stored                                                                                                                                            |
| `output.otel.auth.password.key`   | string                                                                                                                 | no         | `-`                                                                                | Basic authentication credentials for otel. Name of key in the secret where password is stored                                                                                                                                     |
| `output.otel.tls.keyPasswd`       | boolean                                                                                                                | no         | `-`                                                                                | Optional password for private key file                                                                                                                                                                                            |
| `output.otel.extraParams`         | string                                                                                                                 | no         | See example below                                                                  | Additional configuration parameters for otel output. See docs: [Opentelemetry output](https://docs.fluentbit.io/manual/data-pipeline/outputs/opentelemetry)                                                                       |
| `output.otel.compress`            | string                                                                                                                 | no         | `-`                                                                                | Payload compression mechanism. Allowed values: `gzip` and `zstd`.                                                                                                                                                                 |
| `output.otel.logSuppressInterval` | integer                                                                                                                | no         | `-`                                                                                | Suppresses log messages from output plugin that appear similar within a specified time interval. 0 - no suppression.                                                                                                              |
| `output.otel.tls.enabled`         | boolean                                                                                                                | no         | `false`                                                                            | Flag to enable TLS connection for otel output                                                                                                                                                                                     |
| `output.otel.tls.ca.name`         | string                                                                                                                 | no         | `-`                                                                                | Name of Secret with otel CA certificate                                                                                                                                                                                           |
| `output.otel.tls.ca.key`          | string                                                                                                                 | no         | `-`                                                                                | Key (filename) in the Secret with otel CA certificate                                                                                                                                                                             |
| `output.otel.tls.cert.name`       | string                                                                                                                 | no         | `-`                                                                                | Name of Secret with otel certificate                                                                                                                                                                                              |
| `output.otel.tls.cert.key`        | string                                                                                                                 | no         | `-`                                                                                | Key (filename) in the Secret with otel certificate                                                                                                                                                                                |
| `output.otel.tls.key.name`        | string                                                                                                                 | no         | `-`                                                                                | Name of Secret with key                                                                                                                                                                                                           |
| `output.otel.tls.key.key`         | string                                                                                                                 | no         | `-`                                                                                | Key (filename) in the Secret with private key                                                                                                                                                                                     |
| `output.otel.tls.verify`          | boolean                                                                                                                | no         | `true`                                                                             | Force certificate validation                                                                                                                                                                                                      |
<!-- markdownlint-enable line-length -->

Examples:

**Note:** This is only an example of the parameters format, not a recommended value.

```yaml
fluentbit:
  aggregator:
    install: true
    dockerImage: fluent/fluent-bit:4.0.0
    replicas: 2

    tolerations:
    - key: node-role.kubernetes.io/master
      operator: Exists
    - operator: Exists
      effect: NoExecute
    - operator: Exists
      effect: NoSchedule

    graylogOutput: true
    graylogHost: graylog.logging.svc
    graylogPort: 12201
    graylogProtocol: tcp

    extraFields:
      foo_key: foo_value
      bar_key: bar_value
    customFilterConf: |-
      [FILTER]
        Name record_modifier
        Match *
        Record testField fluent-bit
    customOutputConf: |-
      [OUTPUT]
        Name null
        Match fluent.*
    customLuaScriptConf:
      "script1.lua": |-
        function()
          ...
        end
      "script2.lua": |-
        function()
          ...
        end

    multilineFirstLineRegexp: "/^(\\[\\d{4}\\-\\d{2}\\-\\d{2}).*/"
    multilineOtherLinesRegexp: "/^(?!\\[\\d{4}\\-\\d{2}\\-\\d{2}).*/"
    totalLimitSize: 1024M
    memBufLimit: 5M

    volume:
      bind: true
      storageClassName: cinder
      storageSize: 200Gi
```

Example of FluentBit HA configuration with Loki output enabled:

```yaml
fluentbit:
  install: true
  dockerImage: fluent/fluent-bit:4.0.0

  aggregator:
    install: true
    dockerImage: fluent/fluent-bit:4.0.0
    replicas: 2
    graylogOutput: false
    output:
      loki:
        enabled: true
        host: loki-write.loki.svc
        tenant: dev-cloud-1
        auth:
          token:
            name: loki-secret
            key: token
          user:
            name: loki-secret
            key: user
          password:
            name: loki-secret
            key: password
        staticLabels: job=fluentbit
        labelsMapping: |-
          {
              "container": "container",
              "pod": "pod",
              "namespace": "namespace",
              "stream": "stream",
              "level": "level",
              "hostname": "hostname",
              "nodename": "nodename",
              "request_id": "request_id",
              "tenant_id": "tenant_id",
              "addressTo": "addressTo",
              "originating_bi_id": "originating_bi_id",
              "spanId": "spanId"
          }
        tls:
          enabled: true
          ca:
            secretName: secret-ca
            secretKey: ca.crt
          cert:
            secretName: secret-cert
            secretKey: certificate.crt
          key:
            secretName: secret-key
            secretKey: privateKey.key
          verify: true
          keyPasswd: secretKeyPassword
        # See docs: https://docs.fluentbit.io/manual/pipeline/outputs/loki#configuration-parameters
        extraParams: |
            workers                2
            Retry_Limit            32
            storage.total_limit_size  5000M
            net.connect_timeout 20
```

Example of FluentBit HA configuration with HTTP output enabled:

```yaml
fluentbit:
  install: true
  dockerImage: fluent/fluent-bit:4.0.0

  aggregator:
    install: true
    dockerImage: fluent/fluent-bit:4.0.0
    replicas: 2
    graylogOutput: false
    output:
      http:
        enabled: true
        host: vlsingle-k8s.victorialogs
        port: 9428
        uri: /insert/jsonline?_stream_fields=stream&_msg_field=short_message&_time_field=time
        auth:
          token:
            name: http-secret
            key: token
          user:
            name: http-secret
            key: user
          password:
            name: http-secret
            key: password
        tls:
          enabled: true
          ca:
            secretName: secret-ca
            secretKey: ca.crt
          cert:
            secretName: secret-cert
            secretKey: certificate.crt
          key:
            secretName: secret-key
            secretKey: privateKey.key
          verify: true
          keyPasswd: secretKeyPassword
        # See docs: https://docs.fluentbit.io/manual/pipeline/outputs/http#configuration-parameters
        extraParams: |
            workers          2
            header           AccountID 12
            header           ProjectID 23
```

Example of FluentBit HA configuration with Opentelemetry output enabled:

```yaml
fluentbit:
  install: true
  dockerImage: fluent/fluent-bit:4.0.0

  aggregator:
    install: true
    dockerImage: fluent/fluent-bit:4.0.0
    replicas: 2
    graylogOutput: false
    output:
      otel:
        enabled: true
        host: vlsingle-k8s.victorialogs
        port: 9428
        logsUri: /api/v1/logs
        auth:
          token:
            name: otel-secret
            key: token
          user:
            name: otel-secret
            key: user
          password:
            name: otel-secret
            key: password
        compress: zstd
        logSuppressInterval: 5
        tls:
          enabled: true
          ca:
            secretName: secret-ca
            secretKey: ca.crt
          cert:
            secretName: secret-cert
            secretKey: certificate.crt
          key:
            secretName: secret-key
            secretKey: privateKey.key
          verify: true
          keyPasswd: secretKeyPassword
        # See docs: https://docs.fluentbit.io/manual/pipeline/outputs/http#configuration-parameters
        extraParams: |
            workers          2
            header           AccountID 12
            header           ProjectID 23
```

[Back to TOC](#table-of-contents)

### FluentBit TLS

The `fluentbit.tls` or `fluentbit.aggregator.tls` section contains parameters to configure TLS for
FluentBit Graylog Output.

All parameters described for fluentbit TLS must be specified under the `fluentbit.tls` section or `fluentbit.aggregator.tls`
as shown below:

```yaml
fluentbit:
  tls:
    enable: true
    #...
```

or

```yaml
fluentbit:
  aggregator:
    tls:
      enable: true
      #...
```

<!-- markdownlint-disable line-length -->
| Parameter                         | Type    | Mandatory | Default value | Description                                                                                                                     |
| --------------------------------- | ------- | --------- | ------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `enabled`                         | boolean | no        | `false`       | Enables TLS for FluentBit                                                                                                       |
| `verify`                          | boolean | no        | `true`        | Enables certificate validation                                                                                                  |
| `keyPasswd`                       | string  | no        | `-`           | Password for private key file                                                                                                   |
| `ca.secretName`                   | string  | no        | `-`           | Name of the Kubernetes Secret with the CA certificate. Mutually exclusive with the `generateCerts` section                      |
| `ca.secretKey`                    | string  | no        | `-`           | Key (filename) in the Secret with the CA certificate                                                                            |
| `cert.secretName`                 | string  | no        | `-`           | Name of the Kubernetes Secret with the client certificate. Mutually exclusive with the `generateCerts` section                  |
| `cert.secretKey`                  | string  | no        | `-`           | Key (filename) in the Secret with the client certificate                                                                        |
| `key.secretName`                  | string  | no        | `-`           | Name of the Kubernetes Secret with the key for the client certificate. Mutually exclusive with the `generateCerts` section      |
| `key.secretKey`                   | string  | no        | `-`           | Key (filename) in the Secret with the key for the client certificate                                                            |
| `generateCerts.enabled`           | boolean | no        | `-`           | Enables integration with the `cert-manager` to generate certificates. Mutually exclusive with `ca`, `cert` and `key` parameters |
| `generateCerts.secretName`        | string  | no        | `-`           | Secret name with certificates automatically generated by `cert-manager`                                                         |
| `generateCerts.clusterIssuerName` | string  | no        | `-`           | Issuer that will be used to generate certificates                                                                               |
| `generateCerts.duration`          | string  | no        | `-`           | Sets certificates validity period                                                                                               |
| `generateCerts.renewBefore`       | string  | no        | `-`           | Sets the number of days before the certificates expiration date when they will be reissued                                      |
<!-- markdownlint-enable line-length -->

Examples:

**Note:** This is only an example of the parameters format, not a recommended value.

```yaml
fluentbit:
  tls:
    enabled: true

    verify: true
    keyPasswd: secret

    # Certificates from Kubernetes Secrets
    ca:
      secretName: fluentbit-tls-assets-0
      secretKey: ca.crt
    cert:
      secretName: fluentbit-tls-assets-0
      secretKey: cert.crt
    key:
      secretName: fluentbit-tls-assets-0
      secretKey: key.crt

    # Integration with cert-manager
    generateCerts:
      enabled: true
      secretName: fluentbit-cert-manager-tls-assets-0
      clusterIssuerName: ""
      duration: 365
      renewBefore: 15
```

[Back to TOC](#table-of-contents)

## FluentD

The `fluentd` section contains parameters to configure FluentD logging agent.

All parameters for `FluentD` must be specified under the `fluentd` section as shown below:

```yaml
fluentd:
  install: true
  #...
```

<!-- markdownlint-disable line-length -->
| Parameter                         | Type                                                                                                                              | Mandatory | Default value                                                                    | Description                                                                                                                                                                                                                                                                                            |
| --------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- | --------- | -------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `install`                         | boolean                                                                                                                           | no        | `true`                                                                           | Flag for installation `logging-fluentd`                                                                                                                                                                                                                                                                |
| `dockerImage`                     | string                                                                                                                            | no        | `-`                                                                              | Docker image of FluentD                                                                                                                                                                                                                                                                                |
| `configmapReload.dockerImage`     | string                                                                                                                            | no        | `-`                                                                              | Docker image of configmap_reload for FluentD                                                                                                                                                                                                                                                           |
| `configmapReload.resources`       | [core/v1.Resources](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core)            | no        | `{requests: {cpu: 10m, memory: 10Mi}, limits {cpu: 50m, memory: 50Mi}}`          | Describes compute resources requests and limits for `configmap-reload` container                                                                                                                                                                                                                       |
| `ip_v6`                           | boolean                                                                                                                           | no        | `false`                                                                          | Flag for using IPv6 environment                                                                                                                                                                                                                                                                        |
| `nodeSelectorKey`                 | string                                                                                                                            | no        | `-`                                                                              | NodeSelector key, can be multiple by OR condition, separated by comma, usually `role`                                                                                                                                                                                                                  |
| `nodeSelectorValue`               | string                                                                                                                            | no        | `-`                                                                              | NodeSelector value, can be multiple by OR condition, separated by comma, usually `compute`                                                                                                                                                                                                             |
| `tolerations`                     | [core/v1.Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core)                     | no        | `[]`                                                                             | List of tolerations applied to FluentD Pods                                                                                                                                                                                                                                                            |
| `affinity`                        | [core/v1.Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podaffinityterm-v1-core)                  | no        | `-`                                                                              | It specifies the pod\'s scheduling constraints                                                                                                                                                                                                                                                         |
| `resources`                       | [core/v1.Resources](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core)            | no        | `{requests: {cpu: 100m, memory: 128Mi}, limits: {cpu: 500m, memory: 512Mi}}`     | Describes compute resources requests and limits for `fluentd` container                                                                                                                                                                                                                                |
| `graylogOutput`                   | boolean                                                                                                                           | no        | `true`                                                                           | Flag for using Graylog output                                                                                                                                                                                                                                                                          |
| `graylogHost`                     | string                                                                                                                            | no        | `-`                                                                              | Points to Graylog host                                                                                                                                                                                                                                                                                 |
| `graylogPort`                     | integer                                                                                                                           | no        | `12201`                                                                          | Graylog port                                                                                                                                                                                                                                                                                           |
| `graylogProtocol`                 | string                                                                                                                            | no        | `tcp`                                                                            | The Graylog protocol. Available values: `tcp`, `udp`. **Note:** the liveness probe for FluentD pods always returns success if the udp protocol is selected                                                                                                                                             |
| `graylogBufferFlushInterval`      | string                                                                                                                            | no        | `5s`                                                                             | Interval of buffer flush                                                                                                                                                                                                                                                                               |
| `esHost`                          | string                                                                                                                            | no        | `-`                                                                              | **Deprecated** Points to Elasticsearch host                                                                                                                                                                                                                                                            |
| `esPort`                          | integer                                                                                                                           | no        | `-`                                                                              | **Deprecated** Elasticsearch port                                                                                                                                                                                                                                                                      |
| `esUsername`                      | string                                                                                                                            | no        | `-`                                                                              | **Deprecated** Username for Elasticsearch authentication                                                                                                                                                                                                                                               |
| `esPassword`                      | string                                                                                                                            | no        | `-`                                                                              | **Deprecated** Password for Elasticsearch authentication                                                                                                                                                                                                                                               |
| `extraFields`                     | map[string]string                                                                                                                 | no        | `-`                                                                              | Adds additional custom fields/labels to every log message by using filter based on record_transformer plugin. This parameter will override existing fields if their keys match those specified in `extraFields`.                                                                                       |
| `customInputConf`                 | string                                                                                                                            | no        | `-`                                                                              | FluentD custom input configuration                                                                                                                                                                                                                                                                     |
| `customFilterConf`                | string                                                                                                                            | no        | `-`                                                                              | FluentD custom filter configuration                                                                                                                                                                                                                                                                    |
| `customOutputConf`                | string                                                                                                                            | no        | `-`                                                                              | FluentD custom output configuration                                                                                                                                                                                                                                                                    |
| `multilineFirstLineRegexp`        | string                                                                                                                            | no        | `/(^\\[\\d{4}\\-\\d{2}\\-\\d{2})\|(^\\{\")\|(^*0m\\d{2}\\:\\d{2}\\:\\d{2})/`     | FluentD custom regular expression for multiline filter                                                                                                                                                                                                                                                 |
| `billCycleConf`                   | boolean                                                                                                                           | no        | `false`                                                                          | FluentD filter for bil-cycle-logs stream                                                                                                                                                                                                                                                               |
| `watchKubernetesMetadata`         | boolean                                                                                                                           | no        | `true`                                                                           | Set up a watch on pods on the API server for updates to metadata                                                                                                                                                                                                                                       |
| `securityContextPrivileged`       | boolean                                                                                                                           | no        | `false`                                                                          | Allows specifying securityContext.privileged for FluentD container                                                                                                                                                                                                                                     |
| `additionalVolumes`               | [core/v1.PersistentVolumeSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#persistentvolumespec-v1-core) | no        | `false`                                                                          | Additional volumes for FluentD                                                                                                                                                                                                                                                                         |
| `additionalVolumeMounts`          | [core/v1.VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volumemount-v1-core)                   | no        | `false`                                                                          | Allows volume-mounts for FluentD                                                                                                                                                                                                                                                                       |
| `totalLimitSize`                  | string                                                                                                                            | no        | `512MB`                                                                          | The size limitation of output buffer                                                                                                                                                                                                                                                                   |
| `fileStorage`                     | boolean                                                                                                                           | no        | `false`                                                                          | Flag for using file storage instead of memory                                                                                                                                                                                                                                                          |
| `compress`                        | string                                                                                                                            | no        | `text`                                                                           | If `gzip` is set, Fluentd compresses data records before writing to buffer chunks.                                                                                                                                                                                                                     |
| `queueLimitLength`                | integer                                                                                                                           | no        | `-`                                                                              | **Deprecated** Max length of buffers queue                                                                                                                                                                                                                                                             |
| `systemLogging`                   | boolean                                                                                                                           | no        | `false`                                                                          | Enable system log                                                                                                                                                                                                                                                                                      |
| `systemLogType`                   | string                                                                                                                            | no        | `varlogmessages`                                                                 | Set type of system log. Available values `varlogmessages`, `varlogsyslog` and `systemd`                                                                                                                                                                                                                |
| `systemAuditLogging`              | boolean                                                                                                                           | no        | `true`                                                                           | Enable input for system audit logs from `/var/log/audit/audit.log`.                                                                                                                                                                                                                                    |
| `kubeAuditLogging`                | boolean                                                                                                                           | no        | `true`                                                                           | Enable input for Kubernetes audit logs from `/var/log/kubernetes/kube-apiserver-audit.log` and `/var/log/kubernetes/audit.log`.                                                                                                                                                                        |
| `kubeApiserverAuditLogging`       | boolean                                                                                                                           | no        | `true`                                                                           | Enable input for Kubernetes APIServer audit logs from `/var/log/kube-apiserver/audit.log` for Kubernetes and `/var/log/openshift-apiserver/audit.log` for OpenShift.                                                                                                                                   |
| `containerLogging`                | boolean                                                                                                                           | no        | `true`                                                                           | Enable input for container logs from `/var/logs/containers` for Docker or `/var/log/pods` for other engines.                                                                                                                                                                                           |
| `securityResources.install`       | boolean                                                                                                                           | no        | `false`                                                                          | Enable creating security resources as `PodSecurityPolicy`, `SecurityContextConstraints`                                                                                                                                                                                                                |
| `securityResources.name`          | string                                                                                                                            | no        | `logging-fluentd`                                                                | Set a name of `PodSecurityPolicy`, `SecurityContextConstraints` objects                                                                                                                                                                                                                                |
| `podMonitor.scrapeInterval`       | string                                                                                                                            | no        | `30s`                                                                            | Set metrics scrape interval                                                                                                                                                                                                                                                                            |
| `podMonitor.scrapeTimeout`        | string                                                                                                                            | no        | `10s`                                                                            | Set metrics scrape timeout                                                                                                                                                                                                                                                                             |
| `tls`                             | [loggingservice/v11.FluentDTLS](#fluentd-tls)                                                                                     | no        | `{}`                                                                             | TLS configuration  for FluentD Graylog Output                                                                                                                                                                                                                                                          |
| `excludePath`                     | []string                                                                                                                          | no        | `[]`                                                                             | Path to exclude logs. It can contain multiple values.                                                                                                                                                                                                                                                  |
| `annotations`                     | map                                                                                                                               | no        | `{}`                                                                             | Allows to specify list of additional annotations                                                                                                                                                                                                                                                       |
| `labels`                          | map                                                                                                                               | no        | `{}`                                                                             | Allows to specify list of additional labels                                                                                                                                                                                                                                                            |
| `priorityClassName`               | string                                                                                                                            | no        | `-`                                                                              | Pod priority. Indicates the importance of a Pod relative to other Pods and prevents it from being evicted                                                                                                                                                                                              |
| `cloudEventsReaderFormat`         | string                                                                                                                            | no        | `json`                                                                           | Allows to add filter to parse JSON of Kubernetes events logs from Cloud Events Reader. Possible value is `json`. If other value is set, no additional parsing configuration will be added in FluentBit/Fluentd                                                                                         |
| `output.loki.enabled`             | boolean                                                                                                                           | no        | false                                                                            | Enables Loki output                                                                                                                                                                                                                                                                                    |
| `output.loki.host`                | string                                                                                                                            | no        | `-`                                                                              | Loki host                                                                                                                                                                                                                                                                                              |
| `output.loki.tenant`              | string                                                                                                                            | no        | `-`                                                                              | Loki tenant ID                                                                                                                                                                                                                                                                                         |
| `output.loki.auth.token.name`     | string                                                                                                                            | no        | `-`                                                                              | Authentication for Loki with token. Name of the secret where token is stored                                                                                                                                                                                                                           |
| `output.loki.auth.token.key`      | string                                                                                                                            | no        | `-`                                                                              | Authentication for Loki with token. Name of the key in the secret where token is stored                                                                                                                                                                                                                |
| `output.loki.auth.user.name`      | string                                                                                                                            | no        | `-`                                                                              | Basic authentication credentials for Loki. Name of the secret where username is stored                                                                                                                                                                                                                 |
| `output.loki.auth.user.key`       | string                                                                                                                            | no        | `-`                                                                              | Basic authentication credentials for Loki. Name of the key in the secret where username is stored                                                                                                                                                                                                      |
| `output.loki.auth.password.name`  | string                                                                                                                            | no        | `-`                                                                              | Basic authentication credentials for Loki. Name of the secret where password is stored                                                                                                                                                                                                                 |
| `output.loki.auth.password.key`   | string                                                                                                                            | no        | `-`                                                                              | Basic authentication credentials for Loki. Name of key in the secret where password is stored                                                                                                                                                                                                          |
| `output.loki.staticLabels`        | string                                                                                                                            | no        | `{"job":"fluentd"}`                                                              | Static labels that added as stream labels                                                                                                                                                                                                                                                              |
| `output.loki.labelsMapping`       | string                                                                                                                            | no        | See example below                                                                | Labels mappings that defines how to extract labels from each log record                                                                                                                                                                                                                                |
| `output.loki.extraParams`         | string                                                                                                                            | no        | `-`                                                                              | Additional configuration parameters for Loki output. Buffer can be configured here and other available parameters of Fluentd Loki output plugin. See all the parameters here: [fluent/plugin/out_loki.rb](https://github.com/grafana/loki/blob/main/clients/cmd/fluentd/lib/fluent/plugin/out_loki.rb) |
| `output.loki.tls.enabled`         | boolean                                                                                                                           | no        | `false`                                                                          | Flag to enable TLS connection for Loki output                                                                                                                                                                                                                                                          |
| `output.loki.tls.ca.secretName`   | string                                                                                                                            | no        | `-`                                                                              | Name of Secret with Loki CA certificate                                                                                                                                                                                                                                                                |
| `output.loki.tls.ca.secretKey`    | string                                                                                                                            | no        | `-`                                                                              | Key (filename) in the Secret with Loki CA certificate                                                                                                                                                                                                                                                  |
| `output.loki.tls.cert.secretName` | string                                                                                                                            | no        | `-`                                                                              | Name of Secret with Loki certificate                                                                                                                                                                                                                                                                   |
| `output.loki.tls.cert.secretKey`  | string                                                                                                                            | no        | `-`                                                                              | Key (filename) in the Secret with the Loki certificate                                                                                                                                                                                                                                                 |
| `output.loki.tls.key.secretName`  | string                                                                                                                            | no        | `-`                                                                              | Name of Secret with key                                                                                                                                                                                                                                                                                |
| `output.loki.tls.key.secretKey`   | string                                                                                                                            | no        | `-`                                                                              | Key (filename) in the Secret with key                                                                                                                                                                                                                                                                  |
| `output.loki.tls.allCiphers`      | boolean                                                                                                                           | no        | `true`                                                                           | Allows any ciphers to be used, may be insecure                                                                                                                                                                                                                                                         |
| `output.loki.tls.version`         | boolean                                                                                                                           | no        | `-`                                                                              | Any of `:TLSv1`, `:TLSv1_1`, `:TLSv1_2`                                                                                                                                                                                                                                                                |
| `output.loki.tls.noVerify`        | boolean                                                                                                                           | no        | `false`                                                                          | Force certificate validation                                                                                                                                                                                                                                                                           |
| `output.http.enabled`             | boolean                                                                                                                           | no        | `false`                                                                          | Flag for enabling Http output.                                                                                                                                                                                                                                                                         |
| `output.http.host`                | string                                                                                                                            | no        | `-`                                                                              | Http host. Example: `http://10.10.10.10:9428` or `https://vlsingle-k8s.victorialogs:9428` <!-- skip-link-check -->                                                                                                                                                                                     |
| `output.http.path`                | string                                                                                                                            | no        | '/insert/jsonline'                                                               | Http path for URL to ingest logs.                                                                                                                                                                                                                                                                      |
| `output.http.auth.token.name`     | string                                                                                                                            | no        | `-`                                                                              | The name of the secret with the token used for authorization.                                                                                                                                                                                                                                          |
| `output.http.auth.token.key`      | string                                                                                                                            | no        | `-`                                                                              | The key in the secret storing the token. Username/password or token can be stored in different secrets.                                                                                                                                                                                                |
| `output.http.auth.user.name`      | string                                                                                                                            | no        | `-`                                                                              | The name of the secret storing the username. Username/password or token can be stored in different secrets.                                                                                                                                                                                            |
| `output.http.auth.user.key`       | string                                                                                                                            | no        | `-`                                                                              | The key in the secret storing storing the username. Username/password or token can be stored in different secrets.                                                                                                                                                                                     |
| `output.http.auth.password.name`  | string                                                                                                                            | no        | `-`                                                                              | The name of the secret storing the password. Username/password or token can be stored in different secrets.                                                                                                                                                                                            |
| `output.http.auth.password.key`   | string                                                                                                                            | no        | `-`                                                                              | The key in the secret storing the password. Username/password or token can be stored in different secrets.                                                                                                                                                                                             |
| `output.http.compress`            | string                                                                                                                            | no        | text                                                                             | Payload compression mechanism for Http output. Allowed values: text/gzip.                                                                                                                                                                                                                              |
| `output.http.headers`             | map[string]string                                                                                                                 | no        | `{"VL-Msg-Field": "log", "VL-Time-Field": "time", "VL-Stream-Fields": "stream"}` | Additional headers for HTTP output.                                                                                                                                                                                                                                                                    |
| `output.http.tls.enabled`         | boolean                                                                                                                           | no        | `false`                                                                          | Flag for enabling / disabling TLS for Http output.                                                                                                                                                                                                                                                     |
| `output.http.tls.ca.secretName`   | string                                                                                                                            | no        | `-`                                                                              | The name of the secret with http server (e.g. Victorialogs)  CA.                                                                                                                                                                                                                                       |
| `output.http.tls.ca.secretKey`    | string                                                                                                                            | no        | `-`                                                                              | The key in the secret with http server (e.g. Victorialogs) CA.                                                                                                                                                                                                                                         |
| `output.http.tls.cert.secretName` | string                                                                                                                            | no        | `-`                                                                              | The name of the secret storing http server (e.g. Victorialogs) certificate.                                                                                                                                                                                                                            |
| `output.http.tls.cert.secretKey`  | string                                                                                                                            | no        | `-`                                                                              | The key in the secret storing http server (e.g. Victorialogs) certificate.                                                                                                                                                                                                                             |
| `output.http.tls.key.secretName`  | string                                                                                                                            | no        | `-`                                                                              | The secret name with a private key for the http server certificate.                                                                                                                                                                                                                                    |
| `output.http.tls.key.secretKey`   | string                                                                                                                            | no        | `-`                                                                              | The key in the secret storing a private key for the http server certificate.                                                                                                                                                                                                                           |
| `output.http.tls.ciphers`         | string                                                                                                                            | no        | `-`                                                                              | The cipher suites configuration of TLS. Allows any ciphers to be used, may be insecure.                                                                                                                                                                                                                |
| `output.http.tls.verifyMode`      | string                                                                                                                            | no        | `-`                                                                              | Enable / disable tls verification. Allowed values: peer / none.                                                                                                                                                                                                                                        |
| `output.http.tls.version`         | string                                                                                                                            | no        | ':TLSv1_2'                                                                       | The default version of TLS. TLSv1_3(since 1.19.0)/TLSv1_2/TLSv1_1                                                                                                                                                                                                                                      |
| `output.http.extraParams`         | string                                                                                                                            | no        | `-`                                                                              | Additional configuration parameters for Http output. Buffer can be configured here and other available parameters of FluentD Http output plugin. See all the parameters in [Fluentd Http Output Plugin](https://docs.fluentd.org/output/http)                                                          |

<!-- markdownlint-enable line-length -->

Examples:

**Note:** This is only an example of the parameters format, not a recommended value.

```yaml
fluentd:
  install: true
  dockerImage: qubership-fluentd:main
  ip_v6: false

  nodeSelectorKey: kubernetes.io/os
  nodeSelectorValue: linux
  excludePath:
    - "/var/log/pods/openshift-dns_dns-default*/dns/*.log"
  tolerations:
    - key: node-role.kubernetes.io/master
      operator: Exists
    - operator: Exists
      effect: NoExecute
    - operator: Exists
      effect: NoSchedule

  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi

  # Graylog input settings
  systemLogging: true
  systemLogType: varlogmessages
  systemAuditLogging: true
  kubeAuditLogging: true
  kubeApiserverAuditLogging: true
  containerLogging: true

  # Graylog output settings
  graylogOutput: true
  graylogHost: graylog.logging.svc
  graylogPort: 12201
  graylogProtocol: tcp

  # Custom FluentD configurations
  extraFields:
    foo_key: foo_value
    bar_key: bar_value
  customInputConf: |-
    <source>
      custom_input_configuration
    </source>
  customFilterConf: |-
    <filter raw.kubernetes.var.log.**>
      custom_filter_configuration
    </filter>
  customOutputConf: |-
    <store ignore_error>
      custom_output_configuration
    </store>

  multilineFirstLineRegexp: /(^\\[\\d{4}\\-\\d{2}\\-\\d{2})\|(^\\{\")\|(^*0m\\d{2}\\:\\d{2}\\:\\d{2})/
  billCycleConf: true

  watchKubernetesMetadata: true
  securityContextPrivileged: true
  totalLimitSize: 1GB

  # FluentD additional volumes
  additionalVolumes:
    - name: dockervolume
      hostPath:
        path: /var/lib/docker
        type: Directory
  additionalVolumeMounts:
    - name: dockervolume
      mountPath: /var/log/docker

  # FluentD TLS Graylog Output settings
  tls:
    ...
```

Example of FluentD HA configuration with Loki output enabled:

```yaml
fluentd:
  install: true
  dockerImage: qubership-fluentd:main
  output:
    loki:
      enabled: true
      host: https://loki-write.loki.svc:3100
      tenant: dev-env-tenant-1
      auth:
        token:
          name: loki-secret
          key: token
        user:
          name: loki-secret
          key: user
        password:
          name: loki-secret
          key: password
      staticLabels: {"job":"fluentd"}
      labelsMapping: |-
        stream $.stream
        container $.container
        pod $.pod
        namespace $.namespace
        level $.level
        hostname $.hostname
        nodename $.kubernetes_host
        request_id $.request_id
        tenant_id $.tenant_id
        addressTo $.addressTo
        originating_bi_id $.originating_bi_id
        spanId $.spanId
      tls:
        enabled: true
        ca:
          secretName: secret-ca
          secretKey: ca.crt
        cert:
          secretName: secret-cert
          secretKey: certificate.crt
        key:
          secretName: secret-key
          secretKey: privateKey.key
        allCiphers: true
        version: ":TLSv1_2"
        noVerify: false
      # Buffer can be configured here and other available parameters of Fluentd Loki output plugin.
      # See all the parameters here: https://github.com/grafana/loki/blob/main/clients/cmd/fluentd/lib/fluent/plugin/out_loki.rb
      extraParams: |
        extract_kubernetes_labels false
        remove_keys []
        custom_headers header:value
```

Example of FluentD configuration with HTTP output enabled:

```yaml
fluentd:
  install: true
  dockerImage: qubership-fluentd:main
  output:
    http:
      enabled: true
      host: https://vlsingle-k8s.victorialogs:9428
      path: /insert/jsonline
      headers: {"VL-Msg-Field": "log", "VL-Time-Field": "time", "VL-Stream-Fields": "stream"}
      auth:
        token:
          name: http-secret
          key: token
        user:
          name: http-secret
          key: user
        password:
          name: http-secret
          key: password
      tls:
        enabled: true
        ca:
          secretName: secret-ca
          secretKey: ca.crt
        cert:
          secretName: secret-cert
          secretKey: certificate.crt
        key:
          secretName: secret-key
          secretKey: privateKey.key
        verify: false
      # Buffer can be configured here and other available parameters of Fluentd HTTP output plugin.
      # See all the parameters here: https://docs.fluentd.org/output/http
      extraParams: |
        error_response_as_unrecoverable  true
        http_method                      post
```

[Back to TOC](#table-of-contents)

### FluentD TLS

The `fluentd.tls` section contains parameters to configure TLS for FluentD Graylog Output.

All parameters related to FluentD TLS must be specified under the `fluentd.tls` section as shown below:

```yaml
fluentd:
  tls:
    enabled: true
    #...
```

<!-- markdownlint-disable line-length -->
| Parameter                         | Type    | Mandatory | Default value | Description                                                                                                                 |
| --------------------------------- | ------- | --------- | ------------- | --------------------------------------------------------------------------------------------------------------------------- |
| `enabled`                         | boolean | no        | `false`       | Enables TLS for FluentD output in Graylog                                                                                   |
| `noDefaultCA`                     | boolean | no        | `false`       | Prevents OpenSSL from using the systems CA store                                                                            |
| `version`                         | string  | no        | `:TLSv1_2`    | TLS version. Available values `:TLSv1`, `:TLSv1_1`, `:TLSv1_2`                                                              |
| `allCiphers`                      | boolean | no        | `true`        | Allows any ciphers to be used, may be insecure                                                                              |
| `rescueSslErrors`                 | boolean | no        | `false`       | Similar to the rescue_network_errors in notifier.rb, allows SSL exceptions to be raised                                     |
| `noVerify`                        | boolean | no        | `false`       | Disables peer verification                                                                                                  |
| `ca.secretName`                   | string  | no        | `-`           | Name of Kubernetes Secret with CA certificate. Mutually exclusive with `generateCerts` section                              |
| `ca.secretKey`                    | string  | no        | `-`           | Key (filename) in the Secret with CA certificate                                                                            |
| `cert.secretName`                 | string  | no        | `-`           | Name of Kubernetes Secret with client certificate. Mutually exclusive with `generateCerts` section                          |
| `cert.secretKey`                  | string  | no        | `-`           | Key (filename) in the Secret with client certificate                                                                        |
| `key.secretName`                  | string  | no        | `-`           | Name of Kubernetes Secret with key for the client certificate. Mutually exclusive with `generateCerts` section              |
| `key.secretKey`                   | string  | no        | `-`           | Key (filename) in the Secret with key for the client certificate                                                            |
| `generateCerts.enabled`           | boolean | no        | `-`           | Enables integration with `cert-manager` to generate certificates. Mutually exclusive with `ca`, `cert` and `key` parameters |
| `generateCerts.secretName`        | string  | no        | `-`           | Secret name with certificates that will generate by `cert-manager`                                                          |
| `generateCerts.clusterIssuerName` | string  | no        | `-`           | Issuer that will be used to generate certificates                                                                           |
| `generateCerts.duration`          | string  | no        | `-`           | Sets certificates validity period                                                                                           |
| `generateCerts.renewBefore`       | string  | no        | `-`           | Sets the number of days before the certificates expiration day for which they will be reissued                              |

<!-- markdownlint-enable line-length -->

Examples:

**Note:** This is only an example of the parameters format, not a recommended value.

```yaml
fluentd:
  tls:
    enabled: true

    noDefaultCA: false
    version: ":TLSv1_2"
    allCiphers: true
    rescueSslErrors: false
    noVerify: false

    # Certificates from Kubernetes Secrets
    ca:
      secretName: fluentd-tls-assets-0
      secretKey: ca.crt
    cert:
      secretName: fluentd-tls-assets-0
      secretKey: cert.crt
    key:
      secretName: fluentd-tls-assets-0
      secretKey: key.crt

    # Integration with cert-manager
    generateCerts:
      enabled: true
      secretName: fluentd-cert-manager-tls
      clusterIssuerName: ""
      duration: 365
      renewBefore: 15
```

[Back to TOC](#table-of-contents)

## Cloud Events Reader

The `cloudEventsReader` section contains parameters to configure cloud-events-reader that collects and exposes
Kubernetes/OpenShift events.

All parameters for `Cloud Events Reader` must be specified under the `cloudEventsReader` section as shown below:

```yaml
cloudEventsReader:
  install: true
  #...
```

<!-- markdownlint-disable line-length -->
| Parameter           | Type                                                                                                                   | Mandatory | Default value                                                                | Description                                                                                                |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------- | --------- | ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `install`           | boolean                                                                                                                | no        | `true`                                                                       | Flag for installation `cloud-events-reader`                                                                |
| `dockerImage`       | string                                                                                                                 | no        | `-`                                                                          | Docker image of Cloud Events Reader                                                                        |
| `resources`         | [core/v1.Resources](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core) | no        | `{requests: {cpu: 100m, memory: 128Mi}, limits: {cpu: 100m, memory: 128Mi}}` | Describes compute resources requests and limits for single Pods                                            |
| `nodeSelectorKey`   | string                                                                                                                 | no        | `-`                                                                          | NodeSelector key, usually `role`                                                                           |
| `nodeSelectorValue` | string                                                                                                                 | no        | `-`                                                                          | NodeSelector value, usually `compute`                                                                      |
| `affinity`          | [core/v1.Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podaffinityterm-v1-core)       | no        | `-`                                                                          | Specifies the pod\'s scheduling constraints                                                                |
| `annotations`       | map                                                                                                                    | no        | `{}`                                                                         | Allows to specify list of additional annotations                                                           |
| `labels`            | map                                                                                                                    | no        | `{}`                                                                         | Allows to specify list of additional labels                                                                |
| `priorityClassName` | string                                                                                                                 | no        | `-`                                                                          | Pod priority. Indicates the importance of a Pod relative to other Pods and prevents it from being evicted. |
| `args`              | []string                                                                                                               | no        | `-`                                                                          | Command line arguments for Cloud Events Reader                                                             |
<!-- markdownlint-enable line-length -->

More information about setting `args` described in [user-guides/cloud-events](user-guides/cloud-events.md).

Examples:

**Note:** This is only an example of the parameters format, not a recommended value.

```yaml
cloudEventsReader:
  install: true
  dockerImage: ghcr.io/netcracker/qubership-kube-events-reader:main
  resources:
    requests:
      cpu:
      memory:
    limits:
      cpu:
      memory:
  nodeSelectorKey: kubernetes.io/os
  nodeSelectorValue: linux
```

[Back to TOC](#table-of-contents)

## Integration tests

The `integrationTests` section contains parameters to enable integration tests that can verify deployment of
Graylog, FluentBit or FluentD.

All parameters described below should be specified under a section `integrationTests` as the following:

```yaml
integrationTests:
  install: true
  #...
```

<!-- markdownlint-disable line-length -->
| Parameter                               | Type                                                                                                                   | Mandatory | Default value                                                                     | Description                                                                                                                                                                                                           |
| --------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- | --------- | --------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `install`                               | boolean                                                                                                                | no        | `false`                                                                           | Enable `integration-tests`                                                                                                                                                                                            |
| `image`                                 | string                                                                                                                 | no        | `-`                                                                               | Docker image of `integration-tests`                                                                                                                                                                                   |
| `service.name`                          | string                                                                                                                 | no        | `logging-integration-tests-runner`                                                | The name of Logging integration tests service                                                                                                                                                                         |
| `tags`                                  | string                                                                                                                 | no        | `smoke`                                                                           | The tags used to select which test cases to run                                                                                                                                                                       |
| `externalGraylogServer`                 | string                                                                                                                 | no        | `true`                                                                            | The kind of Graylog for testing                                                                                                                                                                                       |
| `graylogProtocol`                       | string                                                                                                                 | no        | `-`                                                                               | Graylog protocol                                                                                                                                                                                                      |
| `graylogHost`                           | string                                                                                                                 | no        | `-`                                                                               | The hostname of Graylog                                                                                                                                                                                               |
| `graylogPort`                           | integer                                                                                                                | no        | `80`                                                                              | The Graylog HTTP port                                                                                                                                                                                                 |
| `affinity`                              | [core/v1.Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podaffinityterm-v1-core)       | no        | `-`                                                                               | Specifies the pod\'s scheduling constraints                                                                                                                                                                           |
| `resources`                             | [core/v1.Resources](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core) | no        | `{requests: {cpu: 100m, memory: 128Mi}, limits: {cpu: 200m, memory: 256Mi}}`      | Describes compute resources requests and limits for Integration Tests container                                                                                                                                       |
| `statusWriting.enabled`                 | boolean                                                                                                                | no        | false                                                                             | Enable Store tests status to `LoggingService` custom resource                                                                                                                                                         |
| `statusWriting.isShortStatusMessage`    | boolean                                                                                                                | no        | true                                                                              | The size of integration test status message                                                                                                                                                                           |
| `statusWriting.onlyIntegrationTests`    | boolean                                                                                                                | no        | true                                                                              | Deploy only integration tests without any component (component was installed before).                                                                                                                                 |
| `statusWriting.customResourcePath`      | string                                                                                                                 | no        | `logging.netcracker.com/v1/logging-operator/loggingservices/logging-service`      | Path to the Custom Resource for writing integration test status, taken from the k8s entity selfLink without the `apis` prefix and namespace. Format: `<group>/<apiversion>/<namespace>/<plural>/<customResourceName>` |
| `annotations`                           | map                                                                                                                    | no        | `{}`                                                                              | Allows to specify list of additional annotations                                                                                                                                                                      |
| `labels`                                | map                                                                                                                    | no        | `{}`                                                                              | Allows to specify list of additional labels                                                                                                                                                                           |
| `priorityClassName`                     | string                                                                                                                 | no        | `-`                                                                               | Pod priority. Indicates the importance of a Pod relative to other Pods and prevents it from being evicted.                                                                                                            |
| `vmUser`                                | string                                                                                                                 | no        | `-`                                                                               | User for SSH login to VM.                                                                                                                                                                                             |
| `sshKey`                                | string                                                                                                                 | no        | `-`                                                                               | SSH key for SSH login to VM.                                                                                                                                                                                          |
| `victorialogs.url`                      | string                                                                                                                 | no        | `-`                                                                               | Victorialogs URL. Example: http:/<victorialogs_host>:<victorialogs_port>                                                                                                                                              |
| `victorialogs.auth.token.secretName`    | string                                                                                                                 | no        | `-`                                                                               | Name of the secret with token for authorization in Victorialogs                                                                                                                                                       |
| `victorialogs.auth.token.key`           | string                                                                                                                 | no        | `-`                                                                               | The key in the secret with token for authorization in Victorialogs                                                                                                                                                    |
| `victorialogs.auth.user.secretName`     | string                                                                                                                 | no        | `-`                                                                               | Name of the secret with username for authorization in Victorialogs                                                                                                                                                    |
| `victorialogs.auth.user.key`            | string                                                                                                                 | no        | `-`                                                                               | The key in the secret with username for authorization in Victorialogs                                                                                                                                                 |
| `victorialogs.auth.password.secretName` | string                                                                                                                 | no        | `-`                                                                               | Name of the secret with password for authorization in Victorialogs                                                                                                                                                    |
| `victorialogs.auth.password.key`        | string                                                                                                                 | no        | `-`                                                                               | The key in the secret with password for authorization in Victorialogs                                                                                                                                                 |

<!-- markdownlint-enable line-length -->

Examples:

**Note:** This is only an example of the parameters format, not a recommended value.

```yaml
integrationTests:
  install: true
  image: qubership-logging-integration-tests:main

  service:
    name: logging-integration-tests-runner
  tags: smoke
  externalGraylogServer: true
  graylogHost: 1.2.3.4
  graylogPort: 80
  vmUser: ubuntu
  sshKey: |
    -----BEGIN RSA PRIVATE KEY-----
    ................................
    -----END RSA PRIVATE KEY-----

  statusWriting:
    enabled: true
    isShortStatusMessage: false
    onlyIntegrationTests: false
    customResourcePath: logging.netcracker.com/v1/logging-operator/loggingservices/logging-service
```

[Back to TOC](#table-of-contents)
