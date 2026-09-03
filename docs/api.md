<!-- markdownlint-disable line-length -->
<!-- markdownlint-disable reference-links-images -->
# API Reference

## Packages
- [logging.netcracker.com/v1](#loggingnetcrackercomv1)


## logging.netcracker.com/v1

Package v1 contains API Schema definitions for the cache v1 API group

### Resource Types
- [LoggingService](#loggingservice)



#### Auth







_Appears in:_
- [HttpFluentbit](#httpfluentbit)
- [HttpFluentd](#httpfluentd)
- [LokiFluentbit](#lokifluentbit)
- [LokiFluentd](#lokifluentd)
- [OtelFluentbit](#otelfluentbit)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `user` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#secretkeyselector-v1-core)_ |  |  |  |


#### AuthProxy







_Appears in:_
- [Graylog](#graylog)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#resourcerequirements-v1-core)_ |  |  |  |
| `bindPasswordSecret` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#secretkeyselector-v1-core)_ |  |  |  |
| `ca` _[CA](#ca)_ | CA contains selectors for the Secret containing TLS certificate for LDAP database or OAuth authentication server |  |  |
| `cert` _[Cert](#cert)_ | Cert contains selectors for the Secret containing TLS certificate for client authentication<br />to LDAP database or OAuth authentication server |  |  |
| `key` _[Key](#key)_ | Key contains selectors for the Secret containing TLS private key for client authentication<br />to LDAP database or OAuth authentication server |  |  |
| `image` _string_ |  |  |  |
| `install` _boolean_ |  |  |  |


#### CA







_Appears in:_
- [AuthProxy](#authproxy)
- [Certificates](#certificates)
- [FluentbitTLS](#fluentbittls)
- [FluentdHttpTLS](#fluentdhttptls)
- [FluentdLokiTLS](#fluentdlokitls)
- [FluentdTLS](#fluentdtls)
- [InputGraylogTLS](#inputgraylogtls)
- [TLS](#tls)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretName` _string_ |  |  |  |
| `secretKey` _string_ |  |  |  |


#### Cert







_Appears in:_
- [AuthProxy](#authproxy)
- [Certificates](#certificates)
- [FluentbitTLS](#fluentbittls)
- [FluentdHttpTLS](#fluentdhttptls)
- [FluentdLokiTLS](#fluentdlokitls)
- [FluentdTLS](#fluentdtls)
- [HTTPGraylogTLS](#httpgraylogtls)
- [InputGraylogTLS](#inputgraylogtls)
- [TLS](#tls)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretName` _string_ |  |  |  |
| `secretKey` _string_ |  |  |  |


#### Certificates







_Appears in:_
- [FluentbitTLS](#fluentbittls)
- [FluentdHttpTLS](#fluentdhttptls)
- [FluentdLokiTLS](#fluentdlokitls)
- [FluentdTLS](#fluentdtls)
- [InputGraylogTLS](#inputgraylogtls)
- [TLS](#tls)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ca` _[CA](#ca)_ |  |  |  |
| `cert` _[Cert](#cert)_ |  |  |  |
| `key` _[Key](#key)_ |  |  |  |


#### CloudEventsReader



CloudEventsReader contains EventsReader-specific configuration



_Appears in:_
- [LoggingServiceSpec](#loggingservicespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#resourcerequirements-v1-core)_ |  |  |  |
| `dockerImage` _string_ |  |  |  |
| `priorityClassName` _string_ |  |  |  |
| `nodeSelectorKey` _string_ |  |  |  |
| `nodeSelectorValue` _string_ |  |  |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#affinity-v1-core)_ |  |  |  |
| `labels` _object (keys:string, values:string)_ |  |  |  |
| `annotations` _object (keys:string, values:string)_ |  |  |  |
| `args` _string array_ |  |  |  |
| `install` _boolean_ |  |  |  |


#### ConfigmapReload







_Appears in:_
- [Fluentbit](#fluentbit)
- [FluentbitAggregator](#fluentbitaggregator)
- [Fluentd](#fluentd)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `dockerImage` _string_ |  |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#resourcerequirements-v1-core)_ |  |  |  |


#### ContentPackPathHTTPConfig







_Appears in:_
- [Graylog](#graylog)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `tls` _[HTTPConfig](#httpconfig)_ |  |  |  |
| `url` _string_ |  |  |  |


#### Credentials







_Appears in:_
- [HTTPConfig](#httpconfig)



#### Fluentbit



Fluentbit contains Fluentbit-specific configuration



_Appears in:_
- [LoggingServiceSpec](#loggingservicespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#resourcerequirements-v1-core)_ |  |  |  |
| `annotations` _object (keys:string, values:string)_ |  |  |  |
| `labels` _object (keys:string, values:string)_ |  |  |  |
| `extraFields` _object (keys:string, values:string)_ |  |  |  |
| `aggregator` _[FluentbitAggregator](#fluentbitaggregator)_ |  |  |  |
| `memBufLimit` _string_ |  |  |  |
| `systemLogType` _string_ |  |  |  |
| `graylogHost` _string_ |  |  |  |
| `nodeSelectorValue` _string_ |  |  |  |
| `graylogProtocol` _string_ |  |  |  |
| `dockerImage` _string_ |  |  |  |
| `configmapReload` _[ConfigmapReload](#configmapreload)_ |  |  |  |
| `priorityClassName` _string_ |  |  |  |
| `totalLimitSize` _string_ |  |  |  |
| `customInputConf` _string_ |  |  |  |
| `customFilterConf` _string_ |  |  |  |
| `customOutputConf` _string_ |  |  |  |
| `customLuaScriptConf` _object (keys:string, values:string)_ |  |  |  |
| `logLevel` _string_ |  |  |  |
| `multilineFirstLineRegexp` _string_ |  |  |  |
| `nodeSelectorKey` _string_ |  |  |  |
| `multilineOtherLinesRegexp` _string_ |  |  |  |
| `tls` _[FluentbitTLS](#fluentbittls)_ |  |  |  |
| `additionalVolumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#volume-v1-core) array_ |  |  |  |
| `additionalVolumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#volumemount-v1-core) array_ |  |  |  |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#toleration-v1-core) array_ |  |  |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#affinity-v1-core)_ |  |  |  |
| `graylogPort` _integer_ |  |  |  |
| `securityContextPrivileged` _boolean_ |  |  |  |
| `watchKubernetesMetadata` _boolean_ |  |  |  |
| `mockKubeData` _boolean_ |  |  |  |
| `systemLogging` _boolean_ |  |  |  |
| `billCycleConf` _boolean_ |  |  |  |
| `graylogOutput` _boolean_ |  |  |  |
| `systemAuditLogging` _boolean_ |  |  |  |
| `kubeAuditLogging` _boolean_ |  |  |  |
| `kubeApiserverAuditLogging` _boolean_ |  |  |  |
| `containerLogging` _boolean_ |  |  |  |
| `excludePath` _string_ |  |  |  |
| `output` _[OutputFluentbit](#outputfluentbit)_ |  |  |  |
| `flush` _integer_ | Flush is an interval in seconds to flush records to the outputs.<br />Increasing the interval reduces the amount of produced chunks and, as a result, the disk load. |  | Minimum: 1 <br /> |
| `storageProfile` _string_ | StorageProfile selects where Fluentbit keeps input read offsets and buffered logs.<br />The default "memory-only" profile avoids writes to the node filesystem. The "persistent-offsets"<br />profile stores read offsets on the node and buffers logs in memory. The "node-persistent" profile<br />stores both read offsets and buffered logs on the node. |  | Enum: [memory-only persistent-offsets node-persistent] <br /> |
| `memoryOnlyStateSizeLimit` _string_ | MemoryOnlyStateSizeLimit limits the memory-backed volume that stores read offset databases for the<br />"memory-only" profile. The default is "32Mi". |  |  |
| `db` _[FluentbitDB](#fluentbitdb)_ | DB contains settings of the SQLite database which the input plugins use to keep the read offsets |  |  |


#### FluentbitAggregator



FluentbitAggregator contains Fluentbit-aggregator-specific configuration



_Appears in:_
- [Fluentbit](#fluentbit)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `labels` _object (keys:string, values:string)_ |  |  |  |
| `volume` _[Volume](#volume)_ |  |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#resourcerequirements-v1-core)_ |  |  |  |
| `annotations` _object (keys:string, values:string)_ |  |  |  |
| `extraFields` _object (keys:string, values:string)_ |  |  |  |
| `memBufLimit` _string_ |  |  |  |
| `dockerImage` _string_ |  |  |  |
| `configmapReload` _[ConfigmapReload](#configmapreload)_ |  |  |  |
| `nodeSelectorValue` _string_ |  |  |  |
| `multilineOtherLinesRegexp` _string_ |  |  |  |
| `graylogHost` _string_ |  |  |  |
| `multilineFirstLineRegexp` _string_ |  |  |  |
| `graylogProtocol` _string_ |  |  |  |
| `nodeSelectorKey` _string_ |  |  |  |
| `priorityClassName` _string_ |  |  |  |
| `totalLimitSize` _string_ |  |  |  |
| `customFilterConf` _string_ |  |  |  |
| `customOutputConf` _string_ |  |  |  |
| `customLuaScriptConf` _object (keys:string, values:string)_ |  |  |  |
| `tls` _[FluentbitTLS](#fluentbittls)_ |  |  |  |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#toleration-v1-core) array_ |  |  |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#affinity-v1-core)_ |  |  |  |
| `startupTimeout` _integer_ |  |  |  |
| `replicas` _integer_ |  |  |  |
| `graylogPort` _integer_ |  |  |  |
| `install` _boolean_ |  |  |  |
| `securityContextPrivileged` _boolean_ |  |  |  |
| `graylogOutput` _boolean_ |  |  |  |
| `output` _[OutputFluentbit](#outputfluentbit)_ |  |  |  |


#### FluentbitDB



FluentbitDB contains settings of the SQLite database which the Fluentbit input plugins
use to keep the position of the read files.
Details https://docs.fluentbit.io/manual/pipeline/inputs/tail



_Appears in:_
- [Fluentbit](#fluentbit)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled defines whether the input plugins keep the read offsets in the database file.<br />Disabling it removes all the database writes to the storage, but Fluentbit loses the read offsets<br />on the restart. Every input then starts from the position set by its own Read_from_Head<br />or Read_from_Tail option: the container log inputs re-read the existing records, and the system<br />and audit log inputs skip the records written while Fluentbit was down. Enabled by default. |  |  |
| `journalMode` _string_ | JournalMode sets the journal mode of the database. Allowed values are<br />"wal", "delete", "truncate", "persist", "memory" and "off" in the lower or the upper case. |  | Enum: [wal delete truncate persist memory off WAL DELETE TRUNCATE PERSIST MEMORY OFF] <br /> |
| `sync` _string_ | Sync sets the synchronization mode of the database. Allowed values are<br />"off", "normal", "full" and "extra" in the lower or the upper case. |  | Enum: [off normal full extra OFF NORMAL FULL EXTRA] <br /> |
| `locking` _boolean_ | Locking sets the exclusive access mode to the database file. Enabled by default. |  |  |


#### FluentbitHTTPRouting

_Underlying type:_ _[struct{Enabled bool "json:\"enabled,omitempty\""; HeaderTag string "json:\"headerTag,omitempty\""}](#struct{enabled-bool-"json:\"enabled,omitempty\"";-headertag-string-"json:\"headertag,omitempty\""})_





_Appears in:_
- [HttpFluentbit](#httpfluentbit)



#### FluentbitHttpTLS

_Underlying type:_ _[struct{Certificates "json:\",inline\""; FluentbitTLSParams "json:\",inline\""}](#struct{certificates-"json:\",inline\"";-fluentbittlsparams-"json:\",inline\""})_





_Appears in:_
- [HttpFluentbit](#httpfluentbit)
- [OtelFluentbit](#otelfluentbit)



#### FluentbitLokiTLS

_Underlying type:_ _[struct{Certificates "json:\",inline\""; FluentbitTLSParams "json:\",inline\""}](#struct{certificates-"json:\",inline\"";-fluentbittlsparams-"json:\",inline\""})_





_Appears in:_
- [LokiFluentbit](#lokifluentbit)



#### FluentbitTLS







_Appears in:_
- [Fluentbit](#fluentbit)
- [FluentbitAggregator](#fluentbitaggregator)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `generateCerts` _[GenerateCerts](#generatecerts)_ |  |  |  |
| `ca` _[CA](#ca)_ |  |  |  |
| `cert` _[Cert](#cert)_ |  |  |  |
| `key` _[Key](#key)_ |  |  |  |
| `keyPasswd` _string_ |  |  |  |
| `enabled` _boolean_ |  |  |  |
| `verify` _boolean_ |  |  |  |


#### FluentbitTLSParams







_Appears in:_
- [FluentbitTLS](#fluentbittls)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `keyPasswd` _string_ |  |  |  |
| `enabled` _boolean_ |  |  |  |
| `verify` _boolean_ |  |  |  |


#### Fluentd



Fluentd contains Fluentd-specific configuration



_Appears in:_
- [LoggingServiceSpec](#loggingservicespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#resourcerequirements-v1-core)_ |  |  |  |
| `annotations` _object (keys:string, values:string)_ |  |  |  |
| `labels` _object (keys:string, values:string)_ |  |  |  |
| `extraFields` _object (keys:string, values:string)_ |  |  |  |
| `customFilterConf` _string_ |  |  |  |
| `systemLogType` _string_ |  |  |  |
| `cloudEventsReaderFormat` _string_ |  |  |  |
| `graylogHost` _string_ |  |  |  |
| `dockerImage` _string_ |  |  |  |
| `configmapReload` _[ConfigmapReload](#configmapreload)_ |  |  |  |
| `graylogProtocol` _string_ |  |  |  |
| `priorityClassName` _string_ |  |  |  |
| `multilineFirstLineRegexp` _string_ |  |  |  |
| `logLevel` _string_ |  |  |  |
| `totalLimitSize` _string_ |  |  |  |
| `customInputConf` _string_ |  |  |  |
| `nodeSelectorKey` _string_ |  |  |  |
| `customOutputConf` _string_ |  |  |  |
| `nodeSelectorValue` _string_ |  |  |  |
| `tls` _[FluentdTLS](#fluentdtls)_ |  |  |  |
| `additionalVolumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#volumemount-v1-core) array_ |  |  |  |
| `excludePath` _string array_ |  |  |  |
| `additionalVolumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#volume-v1-core) array_ |  |  |  |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#toleration-v1-core) array_ |  |  |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#affinity-v1-core)_ |  |  |  |
| `queueLimitLength` _integer_ |  |  |  |
| `graylogPort` _integer_ |  |  |  |
| `billCycleConf` _boolean_ |  |  |  |
| `systemLogging` _boolean_ |  |  |  |
| `systemAuditLogging` _boolean_ |  |  |  |
| `kubeAuditLogging` _boolean_ |  |  |  |
| `kubeApiserverAuditLogging` _boolean_ |  |  |  |
| `containerLogging` _boolean_ |  |  |  |
| `watchKubernetesMetadata` _boolean_ |  |  |  |
| `securityContextPrivileged` _boolean_ |  |  |  |
| `useFileStorage` _boolean_ |  |  |  |
| `graylogOutput` _boolean_ |  |  |  |
| `graylogBufferFlushInterval` _string_ |  |  |  |
| `compress` _string_ |  |  |  |
| `mockKubeData` _boolean_ |  |  |  |
| `output` _[OutputFluentd](#outputfluentd)_ |  |  |  |


#### FluentdHTTPRouting







_Appears in:_
- [HttpFluentd](#httpfluentd)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `logCategoryHeader` _string_ |  |  |  |


#### FluentdHttpTLS







_Appears in:_
- [HttpFluentd](#httpfluentd)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ca` _[CA](#ca)_ |  |  |  |
| `cert` _[Cert](#cert)_ |  |  |  |
| `key` _[Key](#key)_ |  |  |  |
| `enabled` _boolean_ |  |  |  |
| `verifyMode` _string_ |  |  |  |
| `version` _string_ |  |  |  |
| `ciphers` _string_ |  |  |  |


#### FluentdHttpTLSParams







_Appears in:_
- [FluentdHttpTLS](#fluentdhttptls)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `verifyMode` _string_ |  |  |  |
| `version` _string_ |  |  |  |
| `ciphers` _string_ |  |  |  |


#### FluentdLokiTLS







_Appears in:_
- [LokiFluentd](#lokifluentd)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ca` _[CA](#ca)_ |  |  |  |
| `cert` _[Cert](#cert)_ |  |  |  |
| `key` _[Key](#key)_ |  |  |  |
| `version` _string_ |  |  |  |
| `enabled` _boolean_ |  |  |  |
| `noDefaultCA` _boolean_ |  |  |  |
| `allCiphers` _boolean_ |  |  |  |
| `rescueSslErrors` _boolean_ |  |  |  |
| `noVerify` _boolean_ |  |  |  |


#### FluentdTLS







_Appears in:_
- [Fluentd](#fluentd)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `generateCerts` _[GenerateCerts](#generatecerts)_ |  |  |  |
| `ca` _[CA](#ca)_ |  |  |  |
| `cert` _[Cert](#cert)_ |  |  |  |
| `key` _[Key](#key)_ |  |  |  |
| `version` _string_ |  |  |  |
| `enabled` _boolean_ |  |  |  |
| `noDefaultCA` _boolean_ |  |  |  |
| `allCiphers` _boolean_ |  |  |  |
| `rescueSslErrors` _boolean_ |  |  |  |
| `noVerify` _boolean_ |  |  |  |


#### FluentdTLSParams







_Appears in:_
- [FluentdLokiTLS](#fluentdlokitls)
- [FluentdTLS](#fluentdtls)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `version` _string_ |  |  |  |
| `enabled` _boolean_ |  |  |  |
| `noDefaultCA` _boolean_ |  |  |  |
| `allCiphers` _boolean_ |  |  |  |
| `rescueSslErrors` _boolean_ |  |  |  |
| `noVerify` _boolean_ |  |  |  |


#### GenerateCerts



GenerateCerts define settings for cert-manager.



_Appears in:_
- [FluentbitTLS](#fluentbittls)
- [FluentdTLS](#fluentdtls)
- [HTTPGraylogTLS](#httpgraylogtls)
- [InputGraylogTLS](#inputgraylogtls)
- [TLS](#tls)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretName` _string_ |  |  |  |
| `enabled` _boolean_ |  |  |  |


#### Graylog



Graylog contains Graylog-specific configuration



_Appears in:_
- [LoggingServiceSpec](#loggingservicespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `graylogResources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#resourcerequirements-v1-core)_ |  |  |  |
| `mongoResources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#resourcerequirements-v1-core)_ |  |  |  |
| `initResources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#resourcerequirements-v1-core)_ |  |  |  |
| `mongoDBUpgrade` _[MongoDBUpgrade](#mongodbupgrade)_ |  |  |  |
| `authProxy` _[AuthProxy](#authproxy)_ |  |  |  |
| `tls` _[GraylogTLS](#graylogtls)_ |  |  |  |
| `openSearch` _[OpenSearch](#opensearch)_ |  |  |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#affinity-v1-core)_ |  |  |  |
| `annotations` _object (keys:string, values:string)_ |  |  |  |
| `labels` _object (keys:string, values:string)_ |  |  |  |
| `nodeSelectorValue` _string_ |  |  |  |
| `pathRepo` _string_ |  |  |  |
| `priorityClassName` _string_ |  |  |  |
| `dockerImage` _string_ |  |  |  |
| `mongoDBImage` _string_ |  |  |  |
| `logLevel` _string_ |  |  |  |
| `contentDeployPolicy` _string_ |  |  |  |
| `javaOpts` _string_ |  |  |  |
| `contentPackPaths` _string_ |  |  |  |
| `customPluginsPaths` _string_ |  |  |  |
| `host` _string_ |  |  |  |
| `initSetupImage` _string_ |  |  |  |
| `nodeSelectorKey` _string_ |  |  |  |
| `initContainerDockerImage` _string_ |  |  |  |
| `graylogSecretName` _string_ |  |  |  |
| `contentPacks` _[ContentPackPathHTTPConfig](#contentpackpathhttpconfig) array_ |  |  |  |
| `streams` _[Stream](#stream) array_ |  |  |  |
| `processbufferProcessors` _integer_ |  |  |  |
| `outputbufferProcessorThreadsMaxPoolSize` _integer_ |  |  |  |
| `ringSize` _integer_ |  |  |  |
| `elasticsearchMaxTotalConnectionsPerRoute` _integer_ |  |  |  |
| `elasticsearchMaxTotalConnections` _integer_ |  |  |  |
| `outputBatchSize` _integer_ |  |  |  |
| `inputbufferRingSize` _integer_ |  |  |  |
| `outputbufferProcessors` _integer_ |  |  |  |
| `maxSize` _integer_ |  |  |  |
| `inputbufferProcessors` _integer_ |  |  |  |
| `startupTimeout` _integer_ |  |  |  |
| `indexShards` _integer_ |  |  |  |
| `indexReplicas` _integer_ |  |  |  |
| `maxNumberOfIndices` _integer_ |  |  |  |
| `logsRotationSizeGb` _integer_ |  |  |  |
| `inputPort` _integer_ |  |  |  |
| `replicas` _integer_ |  |  |  |
| `s3Archive` _boolean_ |  |  |  |


#### GraylogTLS







_Appears in:_
- [Graylog](#graylog)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `http` _[HTTPGraylogTLS](#httpgraylogtls)_ |  |  |  |
| `input` _[InputGraylogTLS](#inputgraylogtls)_ |  |  |  |


#### HTTPConfig







_Appears in:_
- [ContentPackPathHTTPConfig](#contentpackpathhttpconfig)
- [OpenSearch](#opensearch)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `credentials` _[Credentials](#credentials)_ |  |  |  |
| `tlsConfig` _[TLSConfig](#tlsconfig)_ |  |  |  |


#### HTTPGraylogTLS







_Appears in:_
- [GraylogTLS](#graylogtls)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `generateCerts` _[GenerateCerts](#generatecerts)_ |  |  |  |
| `cert` _[Cert](#cert)_ |  |  |  |
| `key` _[Key](#key)_ |  |  |  |
| `cacerts` _string_ |  |  |  |
| `keyFilePassword` _string_ |  |  |  |
| `enabled` _boolean_ |  |  |  |
| `insecureSkipVerify` _boolean_ |  |  |  |


#### HttpFluentbit







_Appears in:_
- [OutputFluentbit](#outputfluentbit)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `routing` _[FluentbitHTTPRouting](#fluentbithttprouting)_ |  |  |  |
| `host` _string_ |  |  |  |
| `port` _integer_ |  |  |  |
| `uri` _string_ |  |  |  |
| `auth` _[Auth](#auth)_ |  |  |  |
| `compress` _string_ |  |  |  |
| `tls` _[FluentbitHttpTLS](#fluentbithttptls)_ |  |  |  |
| `jsonDateFormat` _string_ |  |  |  |
| `format` _string_ |  |  |  |
| `extraParams` _string_ |  |  |  |


#### HttpFluentd







_Appears in:_
- [OutputFluentd](#outputfluentd)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `routing` _[FluentdHTTPRouting](#fluentdhttprouting)_ |  |  |  |
| `host` _string_ |  |  |  |
| `path` _string_ |  |  |  |
| `compress` _string_ |  |  |  |
| `headers` _object (keys:string, values:string)_ |  |  |  |
| `auth` _[Auth](#auth)_ |  |  |  |
| `tls` _[FluentdHttpTLS](#fluentdhttptls)_ |  |  |  |
| `extraParams` _string_ |  |  |  |


#### InputGraylogTLS







_Appears in:_
- [GraylogTLS](#graylogtls)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `generateCerts` _[GenerateCerts](#generatecerts)_ |  |  |  |
| `ca` _[CA](#ca)_ |  |  |  |
| `cert` _[Cert](#cert)_ |  |  |  |
| `key` _[Key](#key)_ |  |  |  |
| `keyFilePassword` _string_ |  |  |  |
| `enabled` _boolean_ |  |  |  |
| `insecureSkipVerify` _boolean_ |  |  |  |


#### Key







_Appears in:_
- [AuthProxy](#authproxy)
- [Certificates](#certificates)
- [FluentbitTLS](#fluentbittls)
- [FluentdHttpTLS](#fluentdhttptls)
- [FluentdLokiTLS](#fluentdlokitls)
- [FluentdTLS](#fluentdtls)
- [HTTPGraylogTLS](#httpgraylogtls)
- [InputGraylogTLS](#inputgraylogtls)
- [TLS](#tls)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretName` _string_ |  |  |  |
| `secretKey` _string_ |  |  |  |


#### LoggingService



LoggingService is the Schema for the loggingservices API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `logging.netcracker.com/v1` | | |
| `kind` _string_ | `LoggingService` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[LoggingServiceSpec](#loggingservicespec)_ |  |  |  |
| `status` _[LoggingServiceStatus](#loggingservicestatus)_ |  |  |  |


#### LoggingServiceCondition



LoggingServiceCondition contains description of status of LoggingService



_Appears in:_
- [LoggingServiceStatus](#loggingservicestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ |  |  |  |
| `reason` _string_ |  |  |  |
| `message` _string_ |  |  |  |
| `lastTransitionTime` _string_ |  |  |  |
| `status` _boolean_ |  |  |  |




#### LoggingServiceSpec



LoggingServiceSpec defines the desired state of LoggingService



_Appears in:_
- [LoggingService](#loggingservice)
- [LoggingServiceParameters](#loggingserviceparameters)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `graylog` _[Graylog](#graylog)_ |  |  |  |
| `fluentd` _[Fluentd](#fluentd)_ |  |  |  |
| `fluentbit` _[Fluentbit](#fluentbit)_ |  |  |  |
| `cloudEventsReader` _[CloudEventsReader](#cloudeventsreader)_ |  |  |  |
| `monitoringAgentLoggingPlugin` _[MonitoringAgentLoggingPlugin](#monitoringagentloggingplugin)_ |  |  |  |
| `cloudURL` _string_ |  |  |  |
| `osKind` _string_ |  |  |  |
| `containerRuntimeType` _string_ |  |  |  |
| `ipv6` _boolean_ |  |  |  |
| `openshiftDeploy` _boolean_ |  |  |  |


#### LoggingServiceStatus



LoggingServiceStatus defines the observed state of LoggingService



_Appears in:_
- [LoggingService](#loggingservice)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[LoggingServiceCondition](#loggingservicecondition) array_ |  |  |  |


#### LokiFluentbit







_Appears in:_
- [OutputFluentbit](#outputfluentbit)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `host` _string_ |  |  |  |
| `tenant` _string_ |  |  |  |
| `auth` _[Auth](#auth)_ |  |  |  |
| `staticLabels` _string_ |  |  |  |
| `labelsMapping` _string_ |  |  |  |
| `tls` _[FluentbitLokiTLS](#fluentbitlokitls)_ |  |  |  |
| `extraParams` _string_ |  |  |  |


#### LokiFluentd







_Appears in:_
- [OutputFluentd](#outputfluentd)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `host` _string_ |  |  |  |
| `tenant` _string_ |  |  |  |
| `auth` _[Auth](#auth)_ |  |  |  |
| `staticLabels` _string_ |  |  |  |
| `labelsMapping` _string_ |  |  |  |
| `tls` _[FluentdLokiTLS](#fluentdlokitls)_ |  |  |  |
| `extraParams` _string_ |  |  |  |


#### MongoDBUpgrade



MongoDBUpgrade is used for the sequential MongoDB upgrading from 3.6 to 5.0



_Appears in:_
- [Graylog](#graylog)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `mongoDBImage40` _string_ |  |  |  |
| `mongoDBImage42` _string_ |  |  |  |
| `mongoDBImage44` _string_ |  |  |  |


#### MonitoringAgentLoggingPlugin



MonitoringAgentLoggingPlugin contains MonitoringAgentLoggingPlugin-specific configuration



_Appears in:_
- [LoggingServiceSpec](#loggingservicespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#resourcerequirements-v1-core)_ |  |  |  |
| `annotations` _object (keys:string, values:string)_ |  |  |  |
| `labels` _object (keys:string, values:string)_ |  |  |  |
| `influxDBName` _string_ |  |  |  |
| `influxDBSecretName` _string_ |  |  |  |
| `influxDBHost` _string_ |  |  |  |
| `nodeSelectorKey` _string_ |  |  |  |
| `nodeSelectorValue` _string_ |  |  |  |
| `saSecret` _string_ |  |  |  |
| `saSecretVolume` _string_ |  |  |  |
| `priorityClassName` _string_ |  |  |  |
| `dockerImage` _string_ |  |  |  |
| `influxDBPort` _integer_ |  |  |  |
| `influxDBMode` _boolean_ |  |  |  |


#### OpenSearch







_Appears in:_
- [Graylog](#graylog)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `tls` _[HTTPConfig](#httpconfig)_ |  |  |  |
| `url` _string_ |  |  |  |


#### OtelFluentbit







_Appears in:_
- [OutputFluentbit](#outputfluentbit)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `host` _string_ |  |  |  |
| `port` _integer_ |  |  |  |
| `logsUri` _string_ |  |  |  |
| `target` _string_ |  |  |  |
| `logSuppressInterval` _integer_ |  |  |  |
| `auth` _[Auth](#auth)_ |  |  |  |
| `compress` _string_ |  |  |  |
| `tls` _[FluentbitHttpTLS](#fluentbithttptls)_ |  |  |  |
| `extraParams` _string_ |  |  |  |


#### OutputFluentbit







_Appears in:_
- [Fluentbit](#fluentbit)
- [FluentbitAggregator](#fluentbitaggregator)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `loki` _[LokiFluentbit](#lokifluentbit)_ |  |  |  |
| `http` _[HttpFluentbit](#httpfluentbit)_ |  |  |  |
| `otel` _[OtelFluentbit](#otelfluentbit)_ |  |  |  |


#### OutputFluentd







_Appears in:_
- [Fluentd](#fluentd)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `loki` _[LokiFluentd](#lokifluentd)_ |  |  |  |
| `http` _[HttpFluentd](#httpfluentd)_ |  |  |  |


#### Release







_Appears in:_
- [LoggingServiceParameters](#loggingserviceparameters)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `Namespace` _string_ |  |  |  |


#### Stream







_Appears in:_
- [Graylog](#graylog)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `rotationStrategy` _string_ |  |  |  |
| `rotationPeriod` _string_ |  |  |  |
| `maxSize` _integer_ |  |  |  |
| `maxNumberOfIndices` _integer_ |  |  |  |
| `install` _boolean_ |  |  |  |


#### TLS







_Appears in:_
- [FluentbitTLS](#fluentbittls)
- [FluentdTLS](#fluentdtls)
- [InputGraylogTLS](#inputgraylogtls)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `generateCerts` _[GenerateCerts](#generatecerts)_ |  |  |  |
| `ca` _[CA](#ca)_ |  |  |  |
| `cert` _[Cert](#cert)_ |  |  |  |
| `key` _[Key](#key)_ |  |  |  |


#### TLSConfig







_Appears in:_
- [HTTPConfig](#httpconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ca` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#secretkeyselector-v1-core)_ |  |  |  |
| `cert` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#secretkeyselector-v1-core)_ |  |  |  |
| `key` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#secretkeyselector-v1-core)_ |  |  |  |
| `insecureSkipVerify` _boolean_ |  |  |  |


#### Volume







_Appears in:_
- [FluentbitAggregator](#fluentbitaggregator)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `storageClassName` _string_ |  |  |  |
| `storageSize` _string_ |  |  |  |
| `bind` _boolean_ |  |  |  |


