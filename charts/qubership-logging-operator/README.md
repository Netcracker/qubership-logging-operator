

# qubership-logging-operator



![Version: 2.5.0](https://img.shields.io/badge/Version-2.5.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 14.30.0](https://img.shields.io/badge/AppVersion-14.30.0-informational?style=flat-square) 

A Helm chart for qubership-logging-operator










## Values

<table>
	<thead>
		<th>Key</th>
		<th>Type</th>
		<th>Default</th>
		<th>Description</th>
	</thead>
	<tbody>
		<tr>
			<td>affinity</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>If specified, the pod's scheduling constraints Ref: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#affinity-v1-core</td>
		</tr>
		<tr>
			<td>annotations</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Annotations is an unstructured key value map stored with a resource that may be set by external tools to store and retrieve arbitrary metadata. They are not queryable and should be preserved when modifying objects. Ref: https://kubernetes.io/docs/user-guide/annotations</td>
		</tr>
		<tr>
			<td>cloudEventsReader</td>
			<td>object</td>
			<td><pre lang="json">
{
  "affinity": {},
  "install": true,
  "resources": {
    "limits": {
      "cpu": "100m",
      "memory": "128Mi"
    },
    "requests": {
      "cpu": "100m",
      "memory": "128Mi"
    }
  }
}
</pre>
</td>
			<td>Mandatory values for Cloud Events Reader.</td>
		</tr>
		<tr>
			<td>cloudEventsReader.install</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Allow disabling create Cloud Events Reader during deploy</td>
		</tr>
		<tr>
			<td>cloudEventsReader.resources</td>
			<td>object</td>
			<td><pre lang="json">
{
  "limits": {
    "cpu": "100m",
    "memory": "128Mi"
  },
  "requests": {
    "cpu": "100m",
    "memory": "128Mi"
  }
}
</pre>
</td>
			<td>The resources describe to compute resource requests and limits for single Pods. Ref: https://kubernetes.io/docs/user-guide/compute-resources/</td>
		</tr>
		<tr>
			<td>cloudURL</td>
			<td>string</td>
			<td><pre lang="json">
"https://kubernetes.default.svc"
</pre>
</td>
			<td>Cloud address. Openshift cloud may require to set the current host address.</td>
		</tr>
		<tr>
			<td>createClusterAdminEntities</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Specifies whether a cluster-admin entities should be created.</td>
		</tr>
		<tr>
			<td>fluentbit</td>
			<td>object</td>
			<td><pre lang="">
{}
</pre>
</td>
			<td>Mandatory values for FluentBit.</td>
		</tr>
		<tr>
			<td>fluentbit.additionalVolumeMounts</td>
			<td>list</td>
			<td><pre lang="">
not set
</pre>
</td>
			<td>VolumeMounts specified will be appended to other VolumeMounts in the prometheus container, that are generated as a result of StorageSpec objects. Ref: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#volumemount-v1-core</td>
		</tr>
		<tr>
			<td>fluentbit.additionalVolumes</td>
			<td>list</td>
			<td><pre lang="">
not set
</pre>
</td>
			<td>Volumes specified will be appended to other volumes that are generated as a result of StorageSpec objects. Ref: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#volume-v1-core</td>
		</tr>
		<tr>
			<td>fluentbit.affinity</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>If specified, the pod's scheduling constraints Ref: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#affinity-v1-core</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.affinity</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>If specified, the pod's scheduling constraints. Ref: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#affinity-v1-core</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.configmapReload</td>
			<td>object</td>
			<td><pre lang="json">
{
  "resources": {
    "limits": {
      "cpu": "50m",
      "memory": "50Mi"
    },
    "requests": {
      "cpu": "10m",
      "memory": "10Mi"
    }
  }
}
</pre>
</td>
			<td>A docker image to use for FluentBit aggregator deployment. dockerImage: fluent/fluent-bit:3.0.6</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.configmapReload.resources</td>
			<td>object</td>
			<td><pre lang="json">
{
  "limits": {
    "cpu": "50m",
    "memory": "50Mi"
  },
  "requests": {
    "cpu": "10m",
    "memory": "10Mi"
  }
}
</pre>
</td>
			<td>The resources describe to compute resource requests and limits for single Pods. Ref: https://kubernetes.io/docs/user-guide/compute-resources/</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.customFilterConf</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>FluentBit custom filter configuration.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.customInputConf</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>FluentBit custom input configuration.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.customLuaScriptConf</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>FluentBit custom lua script configuration.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.customOutputConf</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>FluentBit custom output configuration.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.graylogOutput</td>
			<td>bool</td>
			<td><pre lang="">
true
</pre>
</td>
			<td>Flag for enabling Graylog output.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.graylogProtocol</td>
			<td>string</td>
			<td><pre lang="json">
"tcp"
</pre>
</td>
			<td>Graylog protocol: tcp/udp/etc</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.install</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allow disabling create FluentBit aggregator deployment during deploy</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.memBufLimit</td>
			<td>string</td>
			<td><pre lang="json">
"5M"
</pre>
</td>
			<td>The size limitation of backlog data See storage.backlog.mem_limit in https://docs.fluentbit.io/manual/administration/buffering-and-storage#service-section-configuration</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.multilineFirstLineRegexp</td>
			<td>string</td>
			<td><pre lang="json">
null
</pre>
</td>
			<td>The regexp for the first line of multiline filter</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.multilineOtherLinesRegexp</td>
			<td>string</td>
			<td><pre lang="json">
null
</pre>
</td>
			<td>The regexp for the other lines of multiline filter</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output</td>
			<td>object</td>
			<td><pre lang="json">
{
  "http": {
    "auth": {},
    "enabled": false,
    "routing": {
      "enabled": false
    },
    "tls": {
      "enabled": false,
      "verify": true
    }
  },
  "loki": {
    "auth": {},
    "enabled": false,
    "extraParams": "workers                2\nRetry_Limit            32\nstorage.total_limit_size  5000M\nnet.connect_timeout 20\n",
    "labelsMapping": "{\n    \"container\": \"container\",\n    \"pod\": \"pod\",\n    \"namespace\": \"namespace\",\n    \"stream\": \"stream\",\n    \"level\": \"level\",\n    \"hostname\": \"hostname\",\n    \"nodename\": \"nodename\",\n    \"request_id\": \"request_id\",\n    \"tenant_id\": \"tenant_id\",\n    \"addressTo\": \"addressTo\",\n    \"originating_bi_id\": \"originating_bi_id\",\n    \"spanId\": \"spanId\"\n}",
    "staticLabels": "job=fluentbit",
    "tls": {
      "enabled": false
    }
  },
  "otel": {
    "auth": {},
    "compress": "gzip",
    "enabled": false,
    "host": "",
    "target": "victorialogs",
    "tls": {
      "enabled": false
    }
  }
}
</pre>
</td>
			<td>Output configuration.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.http</td>
			<td>object</td>
			<td><pre lang="json">
{
  "auth": {},
  "enabled": false,
  "routing": {
    "enabled": false
  },
  "tls": {
    "enabled": false,
    "verify": true
  }
}
</pre>
</td>
			<td>Http output configuration.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.http.auth</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Authentication for Http. Secret name and key in the secret storing the parameter should be provided. Username/password or token can be stored in different secrets. </td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.http.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Flag for enabling Http output.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.http.routing.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Enables streams routing filters for Http output.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.http.tls</td>
			<td>object</td>
			<td><pre lang="json">
{
  "enabled": false,
  "verify": true
}
</pre>
</td>
			<td>TLS configuration for Http output.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.http.tls.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allows enabling TLS for Http output.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.http.tls.verify</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Force certificate validation.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.loki</td>
			<td>object</td>
			<td><pre lang="json">
{
  "auth": {},
  "enabled": false,
  "extraParams": "workers                2\nRetry_Limit            32\nstorage.total_limit_size  5000M\nnet.connect_timeout 20\n",
  "labelsMapping": "{\n    \"container\": \"container\",\n    \"pod\": \"pod\",\n    \"namespace\": \"namespace\",\n    \"stream\": \"stream\",\n    \"level\": \"level\",\n    \"hostname\": \"hostname\",\n    \"nodename\": \"nodename\",\n    \"request_id\": \"request_id\",\n    \"tenant_id\": \"tenant_id\",\n    \"addressTo\": \"addressTo\",\n    \"originating_bi_id\": \"originating_bi_id\",\n    \"spanId\": \"spanId\"\n}",
  "staticLabels": "job=fluentbit",
  "tls": {
    "enabled": false
  }
}
</pre>
</td>
			<td>Loki output configuration.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.loki.auth</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Authentication for Loki.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.loki.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Flag for enabling Loki output.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.loki.extraParams</td>
			<td>string</td>
			<td><pre lang="json">
"workers                2\nRetry_Limit            32\nstorage.total_limit_size  5000M\nnet.connect_timeout 20\n"
</pre>
</td>
			<td>Additional configuration parameters for Loki output. See docs: https://docs.fluentbit.io/manual/pipeline/outputs/loki#configuration-parameters</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.loki.labelsMapping</td>
			<td>string</td>
			<td><pre lang="json">
"{\n    \"container\": \"container\",\n    \"pod\": \"pod\",\n    \"namespace\": \"namespace\",\n    \"stream\": \"stream\",\n    \"level\": \"level\",\n    \"hostname\": \"hostname\",\n    \"nodename\": \"nodename\",\n    \"request_id\": \"request_id\",\n    \"tenant_id\": \"tenant_id\",\n    \"addressTo\": \"addressTo\",\n    \"originating_bi_id\": \"originating_bi_id\",\n    \"spanId\": \"spanId\"\n}"
</pre>
</td>
			<td>Labels mappings that defines how to extract labels from each log record. Value should contain a JSON object.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.loki.staticLabels</td>
			<td>string</td>
			<td><pre lang="json">
"job=fluentbit"
</pre>
</td>
			<td>Static labels that added as stream labels.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.loki.tls</td>
			<td>object</td>
			<td><pre lang="json">
{
  "enabled": false
}
</pre>
</td>
			<td>TLS configuration for Loki output.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.loki.tls.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allows enabling TLS for Loki output.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.otel</td>
			<td>object</td>
			<td><pre lang="json">
{
  "auth": {},
  "compress": "gzip",
  "enabled": false,
  "host": "",
  "target": "victorialogs",
  "tls": {
    "enabled": false
  }
}
</pre>
</td>
			<td>Otel output configuration.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.otel.auth</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Authentication for Otel. Secret name and key in the secret storing the parameter should be provided. Username/password or token can be stored in different secrets.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.otel.compress</td>
			<td>string</td>
			<td><pre lang="json">
"gzip"
</pre>
</td>
			<td>Payload compression mechanism for opentelemetry output. Allowed values: gzip, zstd.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.otel.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Flag for enabling opentelemetry output.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.otel.host</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>Otel host. Example: 10.10.10.10 or vlsingle-k8s.victorialogs</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.otel.target</td>
			<td>string</td>
			<td><pre lang="json">
"victorialogs"
</pre>
</td>
			<td>Otel output target. Set to "" if the logs should be sent to Otel collector or another opentelemetry API compatible storage.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.otel.tls</td>
			<td>object</td>
			<td><pre lang="json">
{
  "enabled": false
}
</pre>
</td>
			<td>TLS configuration for otel output.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.output.otel.tls.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allows enabling TLS for Otel output.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.replicas</td>
			<td>int</td>
			<td><pre lang="json">
2
</pre>
</td>
			<td>A number of replicas for FluentBit aggregator deployment.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.resources</td>
			<td>object</td>
			<td><pre lang="json">
{
  "limits": {
    "cpu": 2,
    "memory": "2Gi"
  },
  "requests": {
    "cpu": "500m",
    "memory": "512Mi"
  }
}
</pre>
</td>
			<td>The resources describe to compute resource requests and limits for single Pods. Ref: https://kubernetes.io/docs/user-guide/compute-resources/</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.securityResources</td>
			<td>object</td>
			<td><pre lang="json">
{
  "install": false,
  "name": "logging-fluentbit-aggregator"
}
</pre>
</td>
			<td>Allow creating security resources as PodSecurityPolicy, SecurityContextConstraints</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.startupTimeout</td>
			<td>int</td>
			<td><pre lang="json">
8
</pre>
</td>
			<td>The amount of time the operator waits for Aggregator pod(s) to start, in minutes</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.tls</td>
			<td>object</td>
			<td><pre lang="json">
{
  "enabled": false,
  "generateCerts": {}
}
</pre>
</td>
			<td>Fluent-bit configuration for TLS enabling for Graylog output. Works only for TCP Graylog protocol.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.tls.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allows enabling TLS for out-of-box Graylog GELF output managed by the operator.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.tls.generateCerts</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Configuration for generating certificates.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.tolerations</td>
			<td>list</td>
			<td><pre lang="json">
[]
</pre>
</td>
			<td>List of tolerations applied to FluentBit aggregator Pods.</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.totalLimitSize</td>
			<td>string</td>
			<td><pre lang="json">
"1024M"
</pre>
</td>
			<td>The size limitation of output buffer</td>
		</tr>
		<tr>
			<td>fluentbit.aggregator.volume</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Allows installing PVCs for Aggregator pods. By default storage of aggregator has emptyDir type</td>
		</tr>
		<tr>
			<td>fluentbit.configmapReload</td>
			<td>object</td>
			<td><pre lang="json">
{
  "resources": {
    "limits": {
      "cpu": "50m",
      "memory": "50Mi"
    },
    "requests": {
      "cpu": "10m",
      "memory": "10Mi"
    }
  }
}
</pre>
</td>
			<td>A docker image to use for FluentBit daemon set. dockerImage: fluent/fluent-bit:3.0.6</td>
		</tr>
		<tr>
			<td>fluentbit.configmapReload.resources</td>
			<td>object</td>
			<td><pre lang="json">
{
  "limits": {
    "cpu": "50m",
    "memory": "50Mi"
  },
  "requests": {
    "cpu": "10m",
    "memory": "10Mi"
  }
}
</pre>
</td>
			<td>A docker image to use for ConfigMap Reload daemon set. dockerImage: ghcr.io/jimmidyson/configmap-reload:v0.13.1 The resources describe to compute resource requests and limits for single Pods. Ref: https://kubernetes.io/docs/user-guide/compute-resources/</td>
		</tr>
		<tr>
			<td>fluentbit.containerLogging</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Enable input for container logs from /var/logs/containers for Docker or /var/log/pods for other engines.</td>
		</tr>
		<tr>
			<td>fluentbit.customFilterConf</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>FluentBit custom filter configuration.</td>
		</tr>
		<tr>
			<td>fluentbit.customInputConf</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>FluentBit custom input configuration.</td>
		</tr>
		<tr>
			<td>fluentbit.customLuaScriptConf</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>FluentBit custom lua script configuration.</td>
		</tr>
		<tr>
			<td>fluentbit.customOutputConf</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>FluentBit custom output configuration.</td>
		</tr>
		<tr>
			<td>fluentbit.excludePath</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>Allow to exclude some logs of pods/containers</td>
		</tr>
		<tr>
			<td>fluentbit.graylogOutput</td>
			<td>bool</td>
			<td><pre lang="">
true
</pre>
</td>
			<td>Flag for enabling Graylog output.</td>
		</tr>
		<tr>
			<td>fluentbit.graylogProtocol</td>
			<td>string</td>
			<td><pre lang="json">
"tcp"
</pre>
</td>
			<td>Graylog protocol: tcp/udp/etc</td>
		</tr>
		<tr>
			<td>fluentbit.install</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allow disabling create FluentBit during deploy</td>
		</tr>
		<tr>
			<td>fluentbit.kubeApiserverAuditLogging</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Enable input for Kubernetes APIServer audit logs from /var/log/kube-apiserver/audit.log for Kubernetes and /var/log/openshift-apiserver/audit.log for OpenShift.</td>
		</tr>
		<tr>
			<td>fluentbit.kubeAuditLogging</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Enable input for Kubernetes audit logs from /var/log/kubernetes/kube-apiserver-audit.log and /var/log/kubernetes/audit.log.</td>
		</tr>
		<tr>
			<td>fluentbit.memBufLimit</td>
			<td>string</td>
			<td><pre lang="json">
"5M"
</pre>
</td>
			<td>The size limitation of backlog data See storage.backlog.mem_limit in https://docs.fluentbit.io/manual/administration/buffering-and-storage#service-section-configuration</td>
		</tr>
		<tr>
			<td>fluentbit.multilineFirstLineRegexp</td>
			<td>string</td>
			<td><pre lang="json">
null
</pre>
</td>
			<td>The regexp for the first line of multiline filter</td>
		</tr>
		<tr>
			<td>fluentbit.multilineOtherLinesRegexp</td>
			<td>string</td>
			<td><pre lang="json">
null
</pre>
</td>
			<td>The regexp for the other lines of multiline filter</td>
		</tr>
		<tr>
			<td>fluentbit.output</td>
			<td>object</td>
			<td><pre lang="json">
{
  "http": {
    "auth": {},
    "compress": "gzip",
    "enabled": false,
    "host": "",
    "routing": {
      "enabled": false
    },
    "tls": {
      "enabled": false
    }
  },
  "loki": {
    "auth": {},
    "enabled": false,
    "extraParams": "workers                2\nRetry_Limit            32\nstorage.total_limit_size  5000M\nnet.connect_timeout 20\n",
    "labelsMapping": "{\n    \"container\": \"container\",\n    \"pod\": \"pod\",\n    \"namespace\": \"namespace\",\n    \"stream\": \"stream\",\n    \"level\": \"level\",\n    \"hostname\": \"hostname\",\n    \"nodename\": \"nodename\",\n    \"request_id\": \"request_id\",\n    \"tenant_id\": \"tenant_id\",\n    \"addressTo\": \"addressTo\",\n    \"originating_bi_id\": \"originating_bi_id\",\n    \"spanId\": \"spanId\"\n}",
    "staticLabels": "job=fluentbit",
    "tls": {
      "enabled": false,
      "verify": true
    }
  },
  "otel": {
    "auth": {},
    "compress": "gzip",
    "enabled": false,
    "host": "",
    "target": "victorialogs",
    "tls": {
      "enabled": false
    }
  }
}
</pre>
</td>
			<td>Output configuration.</td>
		</tr>
		<tr>
			<td>fluentbit.output.http</td>
			<td>object</td>
			<td><pre lang="json">
{
  "auth": {},
  "compress": "gzip",
  "enabled": false,
  "host": "",
  "routing": {
    "enabled": false
  },
  "tls": {
    "enabled": false
  }
}
</pre>
</td>
			<td>Flag for enabling HTTP output.</td>
		</tr>
		<tr>
			<td>fluentbit.output.http.auth</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Authentication for Http.</td>
		</tr>
		<tr>
			<td>fluentbit.output.http.compress</td>
			<td>string</td>
			<td><pre lang="json">
"gzip"
</pre>
</td>
			<td>Payload compression mechanism for Http output. Allowed values: gzip, snappy, zstd.</td>
		</tr>
		<tr>
			<td>fluentbit.output.http.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Flag for enabling Http output.</td>
		</tr>
		<tr>
			<td>fluentbit.output.http.host</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>Http host. Example: 10.10.10.10 or vlsingle-k8s.victorialogs</td>
		</tr>
		<tr>
			<td>fluentbit.output.http.routing.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Enables streams routing filters for Http output.</td>
		</tr>
		<tr>
			<td>fluentbit.output.http.tls</td>
			<td>object</td>
			<td><pre lang="json">
{
  "enabled": false
}
</pre>
</td>
			<td>TLS configuration for Http output.</td>
		</tr>
		<tr>
			<td>fluentbit.output.http.tls.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allows enabling TLS for Http output.</td>
		</tr>
		<tr>
			<td>fluentbit.output.loki</td>
			<td>object</td>
			<td><pre lang="json">
{
  "auth": {},
  "enabled": false,
  "extraParams": "workers                2\nRetry_Limit            32\nstorage.total_limit_size  5000M\nnet.connect_timeout 20\n",
  "labelsMapping": "{\n    \"container\": \"container\",\n    \"pod\": \"pod\",\n    \"namespace\": \"namespace\",\n    \"stream\": \"stream\",\n    \"level\": \"level\",\n    \"hostname\": \"hostname\",\n    \"nodename\": \"nodename\",\n    \"request_id\": \"request_id\",\n    \"tenant_id\": \"tenant_id\",\n    \"addressTo\": \"addressTo\",\n    \"originating_bi_id\": \"originating_bi_id\",\n    \"spanId\": \"spanId\"\n}",
  "staticLabels": "job=fluentbit",
  "tls": {
    "enabled": false,
    "verify": true
  }
}
</pre>
</td>
			<td>Loki output configuration.</td>
		</tr>
		<tr>
			<td>fluentbit.output.loki.auth</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Authentication for Loki.</td>
		</tr>
		<tr>
			<td>fluentbit.output.loki.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Flag for enabling Loki output.</td>
		</tr>
		<tr>
			<td>fluentbit.output.loki.extraParams</td>
			<td>string</td>
			<td><pre lang="json">
"workers                2\nRetry_Limit            32\nstorage.total_limit_size  5000M\nnet.connect_timeout 20\n"
</pre>
</td>
			<td>Additional configuration parameters for Loki output. See docs: https://docs.fluentbit.io/manual/pipeline/outputs/loki#configuration-parameters</td>
		</tr>
		<tr>
			<td>fluentbit.output.loki.labelsMapping</td>
			<td>string</td>
			<td><pre lang="json">
"{\n    \"container\": \"container\",\n    \"pod\": \"pod\",\n    \"namespace\": \"namespace\",\n    \"stream\": \"stream\",\n    \"level\": \"level\",\n    \"hostname\": \"hostname\",\n    \"nodename\": \"nodename\",\n    \"request_id\": \"request_id\",\n    \"tenant_id\": \"tenant_id\",\n    \"addressTo\": \"addressTo\",\n    \"originating_bi_id\": \"originating_bi_id\",\n    \"spanId\": \"spanId\"\n}"
</pre>
</td>
			<td>Labels mappings that defines how to extract labels from each log record. Value should contain a JSON object.</td>
		</tr>
		<tr>
			<td>fluentbit.output.loki.staticLabels</td>
			<td>string</td>
			<td><pre lang="json">
"job=fluentbit"
</pre>
</td>
			<td>Static labels that added as stream labels.</td>
		</tr>
		<tr>
			<td>fluentbit.output.loki.tls</td>
			<td>object</td>
			<td><pre lang="json">
{
  "enabled": false,
  "verify": true
}
</pre>
</td>
			<td>TLS configuration for Loki output.</td>
		</tr>
		<tr>
			<td>fluentbit.output.loki.tls.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allows enabling TLS for Loki output.</td>
		</tr>
		<tr>
			<td>fluentbit.output.loki.tls.verify</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Force certificate validation.</td>
		</tr>
		<tr>
			<td>fluentbit.output.otel</td>
			<td>object</td>
			<td><pre lang="json">
{
  "auth": {},
  "compress": "gzip",
  "enabled": false,
  "host": "",
  "target": "victorialogs",
  "tls": {
    "enabled": false
  }
}
</pre>
</td>
			<td>OpenTelemetry output configuration.</td>
		</tr>
		<tr>
			<td>fluentbit.output.otel.auth</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Authentication for Otel. Secret name and key in the secret storing the parameter should be provided. Username/password or token can be stored in different secrets.</td>
		</tr>
		<tr>
			<td>fluentbit.output.otel.compress</td>
			<td>string</td>
			<td><pre lang="json">
"gzip"
</pre>
</td>
			<td>Payload compression mechanism for opentelemetry output. Allowed values: gzip, zstd.</td>
		</tr>
		<tr>
			<td>fluentbit.output.otel.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Flag for enabling opentelemetry output.</td>
		</tr>
		<tr>
			<td>fluentbit.output.otel.host</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>Otel host. Example: 10.10.10.10 or vlsingle-k8s.victorialogs</td>
		</tr>
		<tr>
			<td>fluentbit.output.otel.target</td>
			<td>string</td>
			<td><pre lang="json">
"victorialogs"
</pre>
</td>
			<td>Otel output target. Set to "" if the logs should be sent to Otel collector or another opentelemetry API compatible storage.</td>
		</tr>
		<tr>
			<td>fluentbit.output.otel.tls</td>
			<td>object</td>
			<td><pre lang="json">
{
  "enabled": false
}
</pre>
</td>
			<td>TLS configuration for otel output.</td>
		</tr>
		<tr>
			<td>fluentbit.output.otel.tls.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allows enabling TLS for Otel output.</td>
		</tr>
		<tr>
			<td>fluentbit.podMonitor</td>
			<td>object</td>
			<td><pre lang="json">
{
  "scrapeInterval": "30s",
  "scrapeTimeout": "10s"
}
</pre>
</td>
			<td>Pod monitor for FluentBit</td>
		</tr>
		<tr>
			<td>fluentbit.podMonitor.scrapeInterval</td>
			<td>string</td>
			<td><pre lang="json">
"30s"
</pre>
</td>
			<td>Allow change metrics scrape interval</td>
		</tr>
		<tr>
			<td>fluentbit.podMonitor.scrapeTimeout</td>
			<td>string</td>
			<td><pre lang="json">
"10s"
</pre>
</td>
			<td>Allow change metrics scrape timeout</td>
		</tr>
		<tr>
			<td>fluentbit.prometheusRules</td>
			<td>object</td>
			<td><pre lang="json">
{
  "alertDelay": "3m",
  "dropRecordsRateThreshold": 10,
  "parseErrorRateThreshold": 20
}
</pre>
</td>
			<td>Prometheus alerting rules configuration.This section contains parameters for customizing alert thresholds and delays for Prometheus rules related to Fluent Bit metrics.</td>
		</tr>
		<tr>
			<td>fluentbit.prometheusRules.alertDelay</td>
			<td>string</td>
			<td><pre lang="json">
"3m"
</pre>
</td>
			<td>Alert delay to decrease false positive cases</td>
		</tr>
		<tr>
			<td>fluentbit.prometheusRules.dropRecordsRateThreshold</td>
			<td>int</td>
			<td><pre lang="json">
10
</pre>
</td>
			<td>The threshold value for the rate of dropped records that triggers the alert.</td>
		</tr>
		<tr>
			<td>fluentbit.prometheusRules.parseErrorRateThreshold</td>
			<td>int</td>
			<td><pre lang="json">
20
</pre>
</td>
			<td>The threshold value for the rate of parse errors that triggers the alert.</td>
		</tr>
		<tr>
			<td>fluentbit.resources</td>
			<td>object</td>
			<td><pre lang="json">
{
  "limits": {
    "cpu": "200m",
    "memory": "512Mi"
  },
  "requests": {
    "cpu": "50m",
    "memory": "128Mi"
  }
}
</pre>
</td>
			<td>The resources describe to compute resource requests and limits for single Pods. Ref: https://kubernetes.io/docs/user-guide/compute-resources/</td>
		</tr>
		<tr>
			<td>fluentbit.securityContextPrivileged</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allows specifying securityContext.privileged for FluentBit container.</td>
		</tr>
		<tr>
			<td>fluentbit.securityResources</td>
			<td>object</td>
			<td><pre lang="json">
{
  "install": false,
  "name": "logging-fluentbit"
}
</pre>
</td>
			<td>Allow creating security resources as PodSecurityPolicy, SecurityContextConstraints</td>
		</tr>
		<tr>
			<td>fluentbit.systemAuditLogging</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Enable input for system audit logs from /var/log/audit/audit.log.</td>
		</tr>
		<tr>
			<td>fluentbit.systemLogging</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Enable input for system logs from /var/log/messages, /var/log/syslog or /var/log/journal. Type of system logs can be chosen by systemLogType parameter.</td>
		</tr>
		<tr>
			<td>fluentbit.tls</td>
			<td>object</td>
			<td><pre lang="json">
{
  "enabled": false,
  "generateCerts": {}
}
</pre>
</td>
			<td>Fluent-bit configuration for TLS enabling for Graylog output. Works only for TCP Graylog protocol.</td>
		</tr>
		<tr>
			<td>fluentbit.tls.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allows enabling TLS for out-of-box Graylog GELF output managed by the operator.</td>
		</tr>
		<tr>
			<td>fluentbit.tls.generateCerts</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Configuration for generating certificates.</td>
		</tr>
		<tr>
			<td>fluentbit.tolerations</td>
			<td>list</td>
			<td><pre lang="json">
[
  {
    "key": "node-role.kubernetes.io/master",
    "operator": "Exists"
  },
  {
    "effect": "NoExecute",
    "operator": "Exists"
  },
  {
    "effect": "NoSchedule",
    "operator": "Exists"
  }
]
</pre>
</td>
			<td>List of tolerations applied to FluentBit Pods.</td>
		</tr>
		<tr>
			<td>fluentbit.totalLimitSize</td>
			<td>string</td>
			<td><pre lang="json">
"1024M"
</pre>
</td>
			<td>The size limitation of output buffer</td>
		</tr>
		<tr>
			<td>fluentd</td>
			<td>object</td>
			<td><pre lang="">
{}
</pre>
</td>
			<td>Mandatory values for FluentD.</td>
		</tr>
		<tr>
			<td>fluentd.additionalVolumeMounts</td>
			<td>object</td>
			<td><pre lang="">
not set
</pre>
</td>
			<td>VolumeMounts specified will be appended to other VolumeMounts in the prometheus container, that are generated as a result of StorageSpec objects. Ref: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#volumemount-v1-core</td>
		</tr>
		<tr>
			<td>fluentd.additionalVolumes</td>
			<td>object</td>
			<td><pre lang="">
not set
</pre>
</td>
			<td>Volumes specified will be appended to other volumes that are generated as a result of StorageSpec objects. Ref: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#volume-v1-core</td>
		</tr>
		<tr>
			<td>fluentd.affinity</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>If specified, the pod's scheduling constraints. Ref: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#affinity-v1-core</td>
		</tr>
		<tr>
			<td>fluentd.cloudEventsReaderFormat</td>
			<td>string</td>
			<td><pre lang="json">
"json"
</pre>
</td>
			<td>Allow to add filter to parse json of Kubernetes events logs from Cloud Events Reader</td>
		</tr>
		<tr>
			<td>fluentd.configmapReload.resources</td>
			<td>object</td>
			<td><pre lang="json">
{
  "limits": {
    "cpu": "50m",
    "memory": "50Mi"
  },
  "requests": {
    "cpu": "10m",
    "memory": "10Mi"
  }
}
</pre>
</td>
			<td>A docker image for configmap-reload. dockerImage: ghcr.io/jimmidyson/configmap-reload:v0.13.1 The resources describe to compute resource requests and limits for single Pods. Ref: https://kubernetes.io/docs/user-guide/compute-resources/</td>
		</tr>
		<tr>
			<td>fluentd.containerLogging</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Enable input for container logs from /var/logs/containers for Docker or /var/log/pods for other engines.</td>
		</tr>
		<tr>
			<td>fluentd.customFilterConf</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>FluentD custom filter configuration.</td>
		</tr>
		<tr>
			<td>fluentd.customInputConf</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>FluentD custom input configuration.</td>
		</tr>
		<tr>
			<td>fluentd.customOutputConf</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>FluentD custom output configuration.</td>
		</tr>
		<tr>
			<td>fluentd.fileStorage</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Flag for using file storage instead of memory</td>
		</tr>
		<tr>
			<td>fluentd.graylogProtocol</td>
			<td>string</td>
			<td><pre lang="json">
"tcp"
</pre>
</td>
			<td>Graylog protocol: tcp/udp</td>
		</tr>
		<tr>
			<td>fluentd.install</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Allow disabling create FluentD during deploy</td>
		</tr>
		<tr>
			<td>fluentd.kubeApiserverAuditLogging</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Enable input for Kubernetes APIServer audit logs from /var/log/kube-apiserver/audit.log for Kubernetes and /var/log/openshift-apiserver/audit.log for OpenShift.</td>
		</tr>
		<tr>
			<td>fluentd.kubeAuditLogging</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Enable input for Kubernetes audit logs from /var/log/kubernetes/kube-apiserver-audit.log and /var/log/kubernetes/audit.log.</td>
		</tr>
		<tr>
			<td>fluentd.output</td>
			<td>object</td>
			<td><pre lang="json">
{
  "http": {
    "auth": {},
    "compress": "gzip",
    "enabled": false,
    "host": "",
    "routing": {
      "enabled": false
    },
    "tls": {
      "enabled": false
    }
  },
  "loki": {
    "auth": {},
    "enabled": false,
    "labelsMapping": "stream $.stream\ncontainer $.container\npod $.pod\nnamespace $.namespace\nlevel $.level\nhostname $.hostname\nnodename $.kubernetes_host\nrequest_id $.request_id\ntenant_id $.tenant_id\naddressTo $.addressTo\noriginating_bi_id $.originating_bi_id\nspanId $.spanId",
    "staticLabels": "{\"job\":\"fluentd\"}",
    "tls": {
      "allCiphers": true,
      "enabled": false,
      "noVerify": false
    }
  }
}
</pre>
</td>
			<td>Output configuration.</td>
		</tr>
		<tr>
			<td>fluentd.output.http</td>
			<td>object</td>
			<td><pre lang="json">
{
  "auth": {},
  "compress": "gzip",
  "enabled": false,
  "host": "",
  "routing": {
    "enabled": false
  },
  "tls": {
    "enabled": false
  }
}
</pre>
</td>
			<td>Flag for enabling HTTP output.</td>
		</tr>
		<tr>
			<td>fluentd.output.http.auth</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Authentication for HTTP output. Secret name and key in the secret storing the parameter should be provided. Username/password or token can be stored in different secrets.</td>
		</tr>
		<tr>
			<td>fluentd.output.http.compress</td>
			<td>string</td>
			<td><pre lang="json">
"gzip"
</pre>
</td>
			<td>Payload compression mechanism for Http output. Allowed values: text/gzip.</td>
		</tr>
		<tr>
			<td>fluentd.output.http.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Flag for enabling Http output.</td>
		</tr>
		<tr>
			<td>fluentd.output.http.host</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>Http host. nExample: http://10.10.10.10:9428 or https://vlsingle-k8s.victorialogs:9428</td>
		</tr>
		<tr>
			<td>fluentd.output.http.routing.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Enables stream field propagation to http header which is used to route logs to different storage backends.</td>
		</tr>
		<tr>
			<td>fluentd.output.http.tls</td>
			<td>object</td>
			<td><pre lang="json">
{
  "enabled": false
}
</pre>
</td>
			<td>TLS configuration for Http output.</td>
		</tr>
		<tr>
			<td>fluentd.output.http.tls.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allows enabling TLS for Http output.</td>
		</tr>
		<tr>
			<td>fluentd.output.loki</td>
			<td>object</td>
			<td><pre lang="json">
{
  "auth": {},
  "enabled": false,
  "labelsMapping": "stream $.stream\ncontainer $.container\npod $.pod\nnamespace $.namespace\nlevel $.level\nhostname $.hostname\nnodename $.kubernetes_host\nrequest_id $.request_id\ntenant_id $.tenant_id\naddressTo $.addressTo\noriginating_bi_id $.originating_bi_id\nspanId $.spanId",
  "staticLabels": "{\"job\":\"fluentd\"}",
  "tls": {
    "allCiphers": true,
    "enabled": false,
    "noVerify": false
  }
}
</pre>
</td>
			<td>Loki output configuration.</td>
		</tr>
		<tr>
			<td>fluentd.output.loki.auth</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Authentication for Loki.</td>
		</tr>
		<tr>
			<td>fluentd.output.loki.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Flag for enabling Loki output.</td>
		</tr>
		<tr>
			<td>fluentd.output.loki.labelsMapping</td>
			<td>string</td>
			<td><pre lang="json">
"stream $.stream\ncontainer $.container\npod $.pod\nnamespace $.namespace\nlevel $.level\nhostname $.hostname\nnodename $.kubernetes_host\nrequest_id $.request_id\ntenant_id $.tenant_id\naddressTo $.addressTo\noriginating_bi_id $.originating_bi_id\nspanId $.spanId"
</pre>
</td>
			<td>Labels mappings that defines how to extract labels from each log record</td>
		</tr>
		<tr>
			<td>fluentd.output.loki.staticLabels</td>
			<td>string</td>
			<td><pre lang="json">
"{\"job\":\"fluentd\"}"
</pre>
</td>
			<td>Static labels that added as stream labels. Should be valid JSON string with key-value pairs, for example: '{"job":"fluentd","env":"production"}'.</td>
		</tr>
		<tr>
			<td>fluentd.output.loki.tls</td>
			<td>object</td>
			<td><pre lang="json">
{
  "allCiphers": true,
  "enabled": false,
  "noVerify": false
}
</pre>
</td>
			<td>TLS configuration for Loki output.</td>
		</tr>
		<tr>
			<td>fluentd.output.loki.tls.allCiphers</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Allows any ciphers to be used, may be insecure.</td>
		</tr>
		<tr>
			<td>fluentd.output.loki.tls.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allows enabling TLS for Loki output.</td>
		</tr>
		<tr>
			<td>fluentd.output.loki.tls.noVerify</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Force certificate validation.</td>
		</tr>
		<tr>
			<td>fluentd.podMonitor</td>
			<td>object</td>
			<td><pre lang="json">
{
  "scrapeInterval": "30s",
  "scrapeTimeout": "10s"
}
</pre>
</td>
			<td>Pod monitor for FluentD</td>
		</tr>
		<tr>
			<td>fluentd.podMonitor.scrapeInterval</td>
			<td>string</td>
			<td><pre lang="json">
"30s"
</pre>
</td>
			<td>Allow change metrics scrape interval</td>
		</tr>
		<tr>
			<td>fluentd.podMonitor.scrapeTimeout</td>
			<td>string</td>
			<td><pre lang="json">
"10s"
</pre>
</td>
			<td>Allow change metrics scrape timeout</td>
		</tr>
		<tr>
			<td>fluentd.resources</td>
			<td>object</td>
			<td><pre lang="json">
{
  "limits": {
    "cpu": "500m",
    "memory": "512Mi"
  },
  "requests": {
    "cpu": "100m",
    "memory": "128Mi"
  }
}
</pre>
</td>
			<td>The resources describe to compute resource requests and limits for single Pods. Ref: https://kubernetes.io/docs/user-guide/compute-resources/</td>
		</tr>
		<tr>
			<td>fluentd.securityContextPrivileged</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allows specifying securityContext.privileged for FluentD container.</td>
		</tr>
		<tr>
			<td>fluentd.securityResources</td>
			<td>object</td>
			<td><pre lang="json">
{
  "install": false,
  "name": "logging-fluentd"
}
</pre>
</td>
			<td>Allow creating security resources as PodSecurityPolicy, SecurityContextConstraints</td>
		</tr>
		<tr>
			<td>fluentd.systemAuditLogging</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Enable input for system audit logs from /var/log/audit/audit.log.</td>
		</tr>
		<tr>
			<td>fluentd.systemLogging</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Enable input for system logs from /var/log/messages, /var/log/syslog or /var/log/journal. Type of system logs can be chosen by systemLogType parameter.</td>
		</tr>
		<tr>
			<td>fluentd.tls</td>
			<td>object</td>
			<td><pre lang="json">
{
  "allCiphers": true,
  "enabled": false,
  "generateCerts": {},
  "noDefaultCA": false,
  "noVerify": false,
  "rescueSslErrors": false,
  "version": ":TLSv1_2"
}
</pre>
</td>
			<td>FluentD configuration for TLS enabling for Graylog output. Works only for TCP Graylog protocol.</td>
		</tr>
		<tr>
			<td>fluentd.tls.allCiphers</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Allows any ciphers to be used, may be insecure.</td>
		</tr>
		<tr>
			<td>fluentd.tls.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allows enabling TLS for out-of-box Graylog GELF output managed by the operator.</td>
		</tr>
		<tr>
			<td>fluentd.tls.generateCerts</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Configuration for generating certificates.</td>
		</tr>
		<tr>
			<td>fluentd.tls.noDefaultCA</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Prevents OpenSSL from using the systems CA store.</td>
		</tr>
		<tr>
			<td>fluentd.tls.noVerify</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Disable peer verification.</td>
		</tr>
		<tr>
			<td>fluentd.tls.rescueSslErrors</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Similar to rescue_network_errors in notifier.rb, allows SSL exceptions to be raised.</td>
		</tr>
		<tr>
			<td>fluentd.tls.version</td>
			<td>string</td>
			<td><pre lang="json">
":TLSv1_2"
</pre>
</td>
			<td>Any of :TLSv1, :TLSv1_1, :TLSv1_2.</td>
		</tr>
		<tr>
			<td>fluentd.tolerations</td>
			<td>list</td>
			<td><pre lang="json">
[
  {
    "key": "node-role.kubernetes.io/master",
    "operator": "Exists"
  },
  {
    "effect": "NoExecute",
    "operator": "Exists"
  },
  {
    "effect": "NoSchedule",
    "operator": "Exists"
  }
]
</pre>
</td>
			<td>List of tolerations applied to FluentD Pods.</td>
		</tr>
		<tr>
			<td>fluentd.totalLimitSize</td>
			<td>string</td>
			<td><pre lang="json">
"512MB"
</pre>
</td>
			<td>The size limitation of output buffer</td>
		</tr>
		<tr>
			<td>graylog</td>
			<td>object</td>
			<td><pre lang="">
{}
</pre>
</td>
			<td>Mandatory values for Graylog.</td>
		</tr>
		<tr>
			<td>graylog.affinity</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>If specified, the pod's scheduling constraints Ref: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#affinity-v1-core</td>
		</tr>
		<tr>
			<td>graylog.authProxy</td>
			<td>object</td>
			<td><pre lang="json">
{
  "authType": "ldap",
  "install": false,
  "ldap": {
    "baseDN": "",
    "bindDN": "",
    "bindPassword": "",
    "bindPasswordSecret": {
      "key": "bindPassword",
      "name": "graylog-auth-proxy-secret"
    },
    "disableReferrals": false,
    "overSsl": false,
    "searchFilter": "(cn=%(username)s)",
    "skipVerify": false,
    "startTls": false,
    "url": "ldap://localhost:389"
  },
  "logLevel": "INFO",
  "oauth": {
    "authorizationPath": "",
    "clientCredentialsSecret": {
      "key": "clientSecret",
      "name": "graylog-auth-proxy-secret"
    },
    "clientID": "",
    "clientSecret": "",
    "host": "",
    "rolesJsonpath": "realm_access.roles[*]",
    "scopes": "openid profile roles",
    "skipVerify": false,
    "tokenPath": "",
    "userJsonpath": "preferred_username",
    "userinfoPath": ""
  },
  "preCreatedUsers": "admin,auditViewer,operator,telegraf_operator,graylog-sidecar,graylog_api_th_user",
  "requestsTimeout": 30,
  "resources": {
    "limits": {
      "cpu": "200m",
      "memory": "512Mi"
    },
    "requests": {
      "cpu": "100m",
      "memory": "128Mi"
    }
  },
  "roleMapping": "[]",
  "rotationPassInterval": 3,
  "streamMapping": ""
}
</pre>
</td>
			<td>Settings for graylog-auth-proxy</td>
		</tr>
		<tr>
			<td>graylog.authProxy.authType</td>
			<td>string</td>
			<td><pre lang="json">
"ldap"
</pre>
</td>
			<td>Defines which type of authentication protocol will be chosen (LDAP or OAuth 2.0). Allowed values: ldap, oauth</td>
		</tr>
		<tr>
			<td>graylog.authProxy.install</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allows enabling graylog-auth-proxy installation.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.ldap</td>
			<td>object</td>
			<td><pre lang="json">
{
  "baseDN": "",
  "bindDN": "",
  "bindPassword": "",
  "bindPasswordSecret": {
    "key": "bindPassword",
    "name": "graylog-auth-proxy-secret"
  },
  "disableReferrals": false,
  "overSsl": false,
  "searchFilter": "(cn=%(username)s)",
  "skipVerify": false,
  "startTls": false,
  "url": "ldap://localhost:389"
}
</pre>
</td>
			<td>Configuration for LDAP or AD connection</td>
		</tr>
		<tr>
			<td>graylog.authProxy.ldap.baseDN</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>LDAP base DN.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.ldap.bindDN</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>LDAP bind DN.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.ldap.bindPassword</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>LDAP password for the bind DN. Will be stored in the secret with .ldap.bindPasswordSecret.name at key specified in the .ldap.bindPasswordSecret.key.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.ldap.bindPasswordSecret</td>
			<td>object</td>
			<td><pre lang="json">
{
  "key": "bindPassword",
  "name": "graylog-auth-proxy-secret"
}
</pre>
</td>
			<td>Allows to define a secret for bind password. Ref: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#secretkeyselector-v1-core</td>
		</tr>
		<tr>
			<td>graylog.authProxy.ldap.disableReferrals</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Sets ldap.OPT_REFERRALS to zero.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.ldap.overSsl</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Establish an LDAP session over SSL.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.ldap.searchFilter</td>
			<td>string</td>
			<td><pre lang="json">
"(cn=%(username)s)"
</pre>
</td>
			<td>LDAP filter for binding users.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.ldap.skipVerify</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allow skipping verification of the LDAP server's certificate.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.ldap.startTls</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Establish a STAR TLS protected session.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.ldap.url</td>
			<td>string</td>
			<td><pre lang="json">
"ldap://localhost:389"
</pre>
</td>
			<td>LDAP host to query.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.logLevel</td>
			<td>string</td>
			<td><pre lang="json">
"INFO"
</pre>
</td>
			<td>Logging level. Allowed values: DEBUG, INFO, WARNING, ERROR, CRITICAL</td>
		</tr>
		<tr>
			<td>graylog.authProxy.oauth</td>
			<td>object</td>
			<td><pre lang="json">
{
  "authorizationPath": "",
  "clientCredentialsSecret": {
    "key": "clientSecret",
    "name": "graylog-auth-proxy-secret"
  },
  "clientID": "",
  "clientSecret": "",
  "host": "",
  "rolesJsonpath": "realm_access.roles[*]",
  "scopes": "openid profile roles",
  "skipVerify": false,
  "tokenPath": "",
  "userJsonpath": "preferred_username",
  "userinfoPath": ""
}
</pre>
</td>
			<td>Configuration for OAuth 2.0 connection</td>
		</tr>
		<tr>
			<td>graylog.authProxy.oauth.authorizationPath</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>This path will be used to build URL for redirection to OAuth2 authorization server login page.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.oauth.clientCredentialsSecret</td>
			<td>object</td>
			<td><pre lang="json">
{
  "key": "clientSecret",
  "name": "graylog-auth-proxy-secret"
}
</pre>
</td>
			<td>Allows to define a Kubernetes Secret for OAuth client secret.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.oauth.clientID</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>OAuth2 Client ID for the proxy.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.oauth.clientSecret</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>OAuth2 Client Secret for the proxy. Will be stored in the secret with .oauth.clientCredentialsSecret.name at key specified in the .oauth.clientCredentialsSecret.key.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.oauth.host</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>OAuth2 authorization server host.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.oauth.rolesJsonpath</td>
			<td>string</td>
			<td><pre lang="json">
"realm_access.roles[*]"
</pre>
</td>
			<td>JSONPath (by jsonpath-ng) for taking information about entities (roles, groups, etc.) for Graylog roles and streams mapping from the JSON returned from OAuth2 server by using userinfo path. Configured for Keycloak server by default.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.oauth.scopes</td>
			<td>string</td>
			<td><pre lang="json">
"openid profile roles"
</pre>
</td>
			<td>OAuth2 scopes for the proxy separated by spaces. Configured for Keycloak server by default.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.oauth.skipVerify</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allow skipping verification of the OAuth2 authorization server's certificate.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.oauth.tokenPath</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>This path will be used to build URL for getting access token from OAuth2 authorization server.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.oauth.userJsonpath</td>
			<td>string</td>
			<td><pre lang="json">
"preferred_username"
</pre>
</td>
			<td>JSONPath (by jsonpath-ng) for taking username from the JSON returned from OAuth2 server by using userinfo path. Configured for Keycloak server by default.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.oauth.userinfoPath</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>This path will be used to build URL for getting information about current user from OAuth2 authorization server to get username and entities (roles, groups, etc.) for Graylog roles and streams mapping.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.preCreatedUsers</td>
			<td>string</td>
			<td><pre lang="json">
"admin,auditViewer,operator,telegraf_operator,graylog-sidecar,graylog_api_th_user"
</pre>
</td>
			<td>Comma separated pre-created users in Graylog for which you do not need to rotate passwords.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.requestsTimeout</td>
			<td>int</td>
			<td><pre lang="json">
30
</pre>
</td>
			<td>A global timeout parameter affects requests to LDAP server, OAuth server and Graylog server.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.resources</td>
			<td>object</td>
			<td><pre lang="json">
{
  "limits": {
    "cpu": "200m",
    "memory": "512Mi"
  },
  "requests": {
    "cpu": "100m",
    "memory": "128Mi"
  }
}
</pre>
</td>
			<td>The resources describe to compute resource requests and limits for graylog-auth-proxy container. Ref: https://kubernetes.io/docs/user-guide/compute-resources/</td>
		</tr>
		<tr>
			<td>graylog.authProxy.roleMapping</td>
			<td>string</td>
			<td><pre lang="json">
"[]"
</pre>
</td>
			<td>Filter for mapping Graylog roles between LDAP or OAuth and Graylog users. Example: '"CN=otrk_admins,OU=OTRK_Groups,OU=IRQA_LDAP,DC=testad,DC=local":["Admin"] | "CN=otrk_users,OU=OTRK_Groups,OU=IRQA_LDAP,DC=testad,DC=local":["Reader","Operator"] | ["Reader"]'</td>
		</tr>
		<tr>
			<td>graylog.authProxy.rotationPassInterval</td>
			<td>int</td>
			<td><pre lang="json">
3
</pre>
</td>
			<td>Interval in days between password rotation for non-pre-created users.</td>
		</tr>
		<tr>
			<td>graylog.authProxy.streamMapping</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>Filter for sharing Graylog streams between LDAP or OAuth and Graylog users. Example: '"CN=otrk_admins,OU=OTRK_Groups,OU=IRQA_LDAP,DC=testad,DC=local":["Default Stream/manage","all events/view"] | "CN=otrk_users,OU=OTRK_Groups,OU=IRQA_LDAP,DC=testad,DC=local":["All events"] | ["System logs/view"]'</td>
		</tr>
		<tr>
			<td>graylog.awsAccessKey</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>An accessKey for using S3-storage in the graylog-archiving-plugin</td>
		</tr>
		<tr>
			<td>graylog.awsSecretKey</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>A secretKey for using S3-storage in the graylog-archiving-plugin</td>
		</tr>
		<tr>
			<td>graylog.contentPack</td>
			<td>list</td>
			<td><pre lang="json">
[]
</pre>
</td>
			<td>Graylog content packs.</td>
		</tr>
		<tr>
			<td>graylog.contentPackPaths</td>
			<td>Deprecated</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>Use "contentPack" instead of "contentPackPaths". Graylog content pack path.</td>
		</tr>
		<tr>
			<td>graylog.createIngress</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Indicates if Ingress need to be created.</td>
		</tr>
		<tr>
			<td>graylog.graylogPersistentVolume</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>Graylog PVC name.</td>
		</tr>
		<tr>
			<td>graylog.graylogResources</td>
			<td>object</td>
			<td><pre lang="json">
{
  "limits": {
    "cpu": "1000m",
    "memory": "2048Mi"
  },
  "requests": {
    "cpu": "500m",
    "memory": "1536Mi"
  }
}
</pre>
</td>
			<td>The resources describe to compute resource requests and limits for single Pods. Ref: https://kubernetes.io/docs/user-guide/compute-resources/</td>
		</tr>
		<tr>
			<td>graylog.graylogStorageClassName</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>Graylog PVC storage class name.</td>
		</tr>
		<tr>
			<td>graylog.httpRoute</td>
			<td>object</td>
			<td><pre lang="json">
{
  "hostnames": [],
  "install": false,
  "parentRefs": [],
  "rules": []
}
</pre>
</td>
			<td>HTTPRoute-specific configuration through Gateway API.</td>
		</tr>
		<tr>
			<td>graylog.httpRoute.hostnames</td>
			<td>list</td>
			<td><pre lang="json">
[]
</pre>
</td>
			<td>Hostnames for graylog UI. If not set then host from ingress configuration will be used.</td>
		</tr>
		<tr>
			<td>graylog.httpRoute.install</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Enables / disables deploy of HTTPRoute for Graylog</td>
		</tr>
		<tr>
			<td>graylog.httpRoute.parentRefs</td>
			<td>list</td>
			<td><pre lang="json">
[]
</pre>
</td>
			<td>List of Gateway references this HTTPRoute should bind to. Each item describes a parent Gateway (via group, kind, name, namespace, and sectionName). If empty, the HTTPRoute is still created but will not be attached to any Gateway listeners.</td>
		</tr>
		<tr>
			<td>graylog.httpRoute.rules</td>
			<td>list</td>
			<td><pre lang="json">
[]
</pre>
</td>
			<td>Custom HTTPRoute rules to define routing behavior. If set, these rules are used in the HTTPRoute spec.rules block. If not set or empty, a single default rule is used:   - matches path with type PathPrefix and / value   - backendRefs to graylog:9000 Ref: https://gateway-api.sigs.k8s.io/reference/spec/#httprouterule</td>
		</tr>
		<tr>
			<td>graylog.ingressLabels</td>
			<td>object</td>
			<td><pre lang="json">
{
  "ingressAudienceType": "ops-user",
  "ingressType": "public-network"
}
</pre>
</td>
			<td>Netcracker labels for Ingress (only on kind: Ingress, not OpenShift Route).</td>
		</tr>
		<tr>
			<td>graylog.ingressLabels.ingressAudienceType</td>
			<td>string</td>
			<td><pre lang="json">
"ops-user"
</pre>
</td>
			<td>dev-user | ops-user | conf-user | end-user</td>
		</tr>
		<tr>
			<td>graylog.ingressLabels.ingressType</td>
			<td>string</td>
			<td><pre lang="json">
"public-network"
</pre>
</td>
			<td>private-network = intranet; public-network = internet</td>
		</tr>
		<tr>
			<td>graylog.initResources</td>
			<td>object</td>
			<td><pre lang="json">
{
  "limits": {
    "cpu": "100m",
    "memory": "256Mi"
  },
  "requests": {
    "cpu": "50m",
    "memory": "128Mi"
  }
}
</pre>
</td>
			<td>The resources describe to compute resource requests and limits for single Pods. Ref: https://kubernetes.io/docs/user-guide/compute-resources/</td>
		</tr>
		<tr>
			<td>graylog.install</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allow disabling create Graylog during deploy</td>
		</tr>
		<tr>
			<td>graylog.mongoPersistentVolume</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>Mongo PVC name.</td>
		</tr>
		<tr>
			<td>graylog.mongoResources</td>
			<td>object</td>
			<td><pre lang="json">
{
  "limits": {
    "cpu": "500m",
    "memory": "256Mi"
  },
  "requests": {
    "cpu": "500m",
    "memory": "256Mi"
  }
}
</pre>
</td>
			<td>The resources describe to compute resource requests and limits for single Pods. Ref: https://kubernetes.io/docs/user-guide/compute-resources/</td>
		</tr>
		<tr>
			<td>graylog.mongoStorageClassName</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>Mongo PVC storage class name.</td>
		</tr>
		<tr>
			<td>graylog.mongoUpgrade</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Activates automatic step-by-step upgrade of the MongoDB database. Can be used only for migration from Graylog 4 to 5.</td>
		</tr>
		<tr>
			<td>graylog.password</td>
			<td>string</td>
			<td><pre lang="json">
"admin"
</pre>
</td>
			<td>Graylog password.</td>
		</tr>
		<tr>
			<td>graylog.pathRepo</td>
			<td>string</td>
			<td><pre lang="json">
"/usr/share/opensearch/snapshots/graylog/"
</pre>
</td>
			<td>A pathRepo for graylog-archiving-plugin</td>
		</tr>
		<tr>
			<td>graylog.replicas</td>
			<td>int</td>
			<td><pre lang="json">
1
</pre>
</td>
			<td>A number of replicas for Graylog.</td>
		</tr>
		<tr>
			<td>graylog.s3Archive</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>A flag for using S3-storage in the graylog-archiving-plugin</td>
		</tr>
		<tr>
			<td>graylog.securityResources</td>
			<td>object</td>
			<td><pre lang="json">
{
  "install": false,
  "name": "logging-graylog"
}
</pre>
</td>
			<td>Allow creating security resources as PodSecurityPolicy, SecurityContextConstraints</td>
		</tr>
		<tr>
			<td>graylog.serviceMonitor</td>
			<td>object</td>
			<td><pre lang="json">
{
  "scrapeInterval": "30s",
  "scrapeTimeout": "10s"
}
</pre>
</td>
			<td>Service monitor for graylog</td>
		</tr>
		<tr>
			<td>graylog.serviceMonitor.scrapeInterval</td>
			<td>string</td>
			<td><pre lang="json">
"30s"
</pre>
</td>
			<td>Allow change metrics scrape interval</td>
		</tr>
		<tr>
			<td>graylog.serviceMonitor.scrapeTimeout</td>
			<td>string</td>
			<td><pre lang="json">
"10s"
</pre>
</td>
			<td>Allow change metrics scrape timeout</td>
		</tr>
		<tr>
			<td>graylog.storageSize</td>
			<td>string</td>
			<td><pre lang="json">
"2Gi"
</pre>
</td>
			<td>Storage size, e.g. '2Gi'</td>
		</tr>
		<tr>
			<td>graylog.streams</td>
			<td>list</td>
			<td><pre lang="json">
[]
</pre>
</td>
			<td>Configuration for enable streams. System and audit logs will be created by default if the section is empty. List of default streams:   "System logs", "Audit logs", "Kubernetes events", "Integration logs", "Access logs", "Nginx logs", "Bill Cycle logs".</td>
		</tr>
		<tr>
			<td>graylog.tls</td>
			<td>object</td>
			<td><pre lang="json">
{
  "http": {
    "enabled": false,
    "generateCerts": {}
  },
  "input": {
    "enabled": false,
    "generateCerts": {}
  }
}
</pre>
</td>
			<td>Configuration for TLS enabling for Graylog.</td>
		</tr>
		<tr>
			<td>graylog.tls.http</td>
			<td>object</td>
			<td><pre lang="json">
{
  "enabled": false,
  "generateCerts": {}
}
</pre>
</td>
			<td>Configuration for TLS for HTTP interface. Certificate and private key for Graylog server and CA certificates in the Graylog keystore can be specified in this section.</td>
		</tr>
		<tr>
			<td>graylog.tls.http.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allows enabling TLS for HTTP interface. If this parameter is true, each connection to and from the Graylog server except inputs will be secured by TLS, including API calls of the server to itself.</td>
		</tr>
		<tr>
			<td>graylog.tls.http.generateCerts</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Configuration for generating certificates for TLS for HTTP interface.</td>
		</tr>
		<tr>
			<td>graylog.tls.input</td>
			<td>object</td>
			<td><pre lang="json">
{
  "enabled": false,
  "generateCerts": {}
}
</pre>
</td>
			<td>Configuration for TLS for out-of-box GELF input.</td>
		</tr>
		<tr>
			<td>graylog.tls.input.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allows enabling TLS for out-of-box GELF input managed by the operator.</td>
		</tr>
		<tr>
			<td>graylog.tls.input.generateCerts</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Configuration for generating certificates for TLS for out-of-box GELF input.</td>
		</tr>
		<tr>
			<td>graylog.user</td>
			<td>string</td>
			<td><pre lang="json">
"admin"
</pre>
</td>
			<td>Graylog user.</td>
		</tr>
		<tr>
			<td>integrationTests</td>
			<td>object</td>
			<td><pre lang="json">
{
  "externalGraylogServer": "true",
  "graylogProtocol": "http",
  "install": false,
  "resources": {
    "limits": {
      "cpu": "200m",
      "memory": "256Mi"
    },
    "requests": {
      "cpu": "100m",
      "memory": "128Mi"
    }
  },
  "service": {
    "name": "logging-integration-tests-runner"
  },
  "sshKey": "",
  "statusWriting": {
    "customResourcePath": "logging.netcracker.com/v1/logging-operator/loggingservices/logging-service",
    "enabled": false,
    "isShortStatusMessage": true,
    "onlyIntegrationTests": true
  },
  "tags": "smoke",
  "timeoutBeforeStart": 100,
  "victorialogs": {
    "auth": {},
    "url": ""
  },
  "vmUser": ""
}
</pre>
</td>
			<td>Mandatory values for Integration Tests.</td>
		</tr>
		<tr>
			<td>integrationTests.externalGraylogServer</td>
			<td>string</td>
			<td><pre lang="json">
"true"
</pre>
</td>
			<td>Flag for specify kind of Graylog (internal in cloud or external on VM).</td>
		</tr>
		<tr>
			<td>integrationTests.graylogProtocol</td>
			<td>string</td>
			<td><pre lang="json">
"http"
</pre>
</td>
			<td>Graylog protocol.</td>
		</tr>
		<tr>
			<td>integrationTests.install</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Allow disabling create Integration Tests Pod during deploy</td>
		</tr>
		<tr>
			<td>integrationTests.resources</td>
			<td>object</td>
			<td><pre lang="json">
{
  "limits": {
    "cpu": "200m",
    "memory": "256Mi"
  },
  "requests": {
    "cpu": "100m",
    "memory": "128Mi"
  }
}
</pre>
</td>
			<td>The resources describe to compute resource requests and limits for Integration Tests Pod. Ref: https://kubernetes.io/docs/user-guide/compute-resources/</td>
		</tr>
		<tr>
			<td>integrationTests.service</td>
			<td>object</td>
			<td><pre lang="json">
{
  "name": "logging-integration-tests-runner"
}
</pre>
</td>
			<td>Service name for Integration Tests.</td>
		</tr>
		<tr>
			<td>integrationTests.sshKey</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>The parameter specifies the ssh key for login to VM.</td>
		</tr>
		<tr>
			<td>integrationTests.tags</td>
			<td>string</td>
			<td><pre lang="json">
"smoke"
</pre>
</td>
			<td>Tags for Integration Tests. Possible values: smoke, graylog</td>
		</tr>
		<tr>
			<td>integrationTests.timeoutBeforeStart</td>
			<td>int</td>
			<td><pre lang="json">
100
</pre>
</td>
			<td>The parameter specifies the timeout before the start of integration tests.</td>
		</tr>
		<tr>
			<td>integrationTests.victorialogs</td>
			<td>object</td>
			<td><pre lang="json">
{
  "auth": {},
  "url": ""
}
</pre>
</td>
			<td>VictoriaLogs connection settings</td>
		</tr>
		<tr>
			<td>integrationTests.victorialogs.auth</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Authentication for VictoriaLogs.</td>
		</tr>
		<tr>
			<td>integrationTests.victorialogs.url</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>The parameter specifies the url for VictoriaLogs. Example : https://vlsingle-example.victorialogs:9428</td>
		</tr>
		<tr>
			<td>integrationTests.vmUser</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>The parameter specifies the user for ssh login to VM. Example: ubuntu</td>
		</tr>
		<tr>
			<td>ipv6</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Set to 'true' to deploy to IPv6 environment.</td>
		</tr>
		<tr>
			<td>labels</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Map of string keys and values that can be used to organize and categorize (scope and select) objects. May match selectors of replication controllers and services. Ref: https://kubernetes.io/docs/user-guide/labels</td>
		</tr>
		<tr>
			<td>livenessProbe</td>
			<td>object</td>
			<td><pre lang="json">
{
  "failureThreshold": 3,
  "httpGet": {
    "path": "/health",
    "port": 8081
  },
  "initialDelaySeconds": 5,
  "periodSeconds": 10,
  "timeoutSeconds": 5
}
</pre>
</td>
			<td>Liveness probe for logging-operator</td>
		</tr>
		<tr>
			<td>logLevel</td>
			<td>string</td>
			<td><pre lang="json">
"info"
</pre>
</td>
			<td>The log level of operator logs.</td>
		</tr>
		<tr>
			<td>name</td>
			<td>string</td>
			<td><pre lang="json">
"logging"
</pre>
</td>
			<td>Provide a name of CR</td>
		</tr>
		<tr>
			<td>osKind</td>
			<td>string</td>
			<td><pre lang="json">
"centos"
</pre>
</td>
			<td>Operating system kind on cloud nodes: centos/rhel/oracle/ubuntu</td>
		</tr>
		<tr>
			<td>podMonitor</td>
			<td>object</td>
			<td><pre lang="json">
{
  "scrapeInterval": "30s",
  "scrapeTimeout": "10s"
}
</pre>
</td>
			<td>Pod monitor for logging-operator</td>
		</tr>
		<tr>
			<td>podMonitor.scrapeInterval</td>
			<td>string</td>
			<td><pre lang="json">
"30s"
</pre>
</td>
			<td>Allow change metrics scrape interval</td>
		</tr>
		<tr>
			<td>podMonitor.scrapeTimeout</td>
			<td>string</td>
			<td><pre lang="json">
"10s"
</pre>
</td>
			<td>Allow change metrics scrape timeout</td>
		</tr>
		<tr>
			<td>pprof</td>
			<td>object</td>
			<td><pre lang="json">
{
  "containerPort": 9180,
  "install": true,
  "service": {
    "annotations": {},
    "labels": {},
    "port": 9180,
    "portName": "pprof",
    "type": "ClusterIP"
  }
}
</pre>
</td>
			<td>logging-operator pprof</td>
		</tr>
		<tr>
			<td>pprof.containerPort</td>
			<td>int</td>
			<td><pre lang="json">
9180
</pre>
</td>
			<td>Port of pprof which use in container</td>
		</tr>
		<tr>
			<td>pprof.install</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Indicates if logging-operator has pprof enabled.</td>
		</tr>
		<tr>
			<td>pprof.service</td>
			<td>object</td>
			<td><pre lang="json">
{
  "annotations": {},
  "labels": {},
  "port": 9180,
  "portName": "pprof",
  "type": "ClusterIP"
}
</pre>
</td>
			<td>Service configuration for pprof service</td>
		</tr>
		<tr>
			<td>pprof.service.annotations</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Annotations set which will create in service</td>
		</tr>
		<tr>
			<td>pprof.service.labels</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Labels set which will create in service</td>
		</tr>
		<tr>
			<td>pprof.service.port</td>
			<td>int</td>
			<td><pre lang="json">
9180
</pre>
</td>
			<td>Port of pprof which use in service</td>
		</tr>
		<tr>
			<td>pprof.service.portName</td>
			<td>string</td>
			<td><pre lang="json">
"pprof"
</pre>
</td>
			<td>Port name of pprof which use in service</td>
		</tr>
		<tr>
			<td>pprof.service.type</td>
			<td>string</td>
			<td><pre lang="json">
"ClusterIP"
</pre>
</td>
			<td>Type of pprof service</td>
		</tr>
		<tr>
			<td>readinessProbe</td>
			<td>object</td>
			<td><pre lang="json">
{
  "failureThreshold": 3,
  "httpGet": {
    "path": "/ready",
    "port": 8081
  },
  "initialDelaySeconds": 3,
  "periodSeconds": 10,
  "timeoutSeconds": 5
}
</pre>
</td>
			<td>Readiness probe for logging-operator</td>
		</tr>
		<tr>
			<td>resources</td>
			<td>object</td>
			<td><pre lang="json">
{
  "limits": {
    "cpu": "150m",
    "memory": "100Mi"
  },
  "requests": {
    "cpu": "25m",
    "memory": "50Mi"
  }
}
</pre>
</td>
			<td>Resources for logging-operator</td>
		</tr>
	</tbody>
</table>





