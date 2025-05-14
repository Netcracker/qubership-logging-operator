{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "fluentd.name" -}}
  {{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "fluentd.fullname" -}}
  {{- if .Values.fullnameOverride -}}
    {{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
  {{- else -}}
    {{- $name := default .Chart.Name .Values.nameOverride -}}
    {{- if contains $name .Release.Name -}}
      {{- .Release.Name | trunc 63 | trimSuffix "-" -}}
    {{- else -}}
      {{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "fluentd.chart" -}}
  {{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "helm-chart.serviceAccountName" -}}
  {{- if .Values.serviceAccount.create -}}
    {{ default (include "helm-chart.fullname" .) .Values.serviceAccount.name }}
  {{- else -}}
    {{ default "default" .Values.serviceAccount.name }}
  {{- end -}}
{{- end -}}

{{/*
Check the major version of Graylog and return 'true' if it equal 5
*/}}
{{- define "graylog.isMajorVersion5" -}}
  {{- if regexMatch "^*:5\\.[0-9]+\\.[0-9]+$" (include "graylog.image" . ) -}}
true
  {{- end -}}
{{- end -}}

{{/*
Return true if generateCerts in TLS is enabled for Graylog HTTP.
*/}}
{{- define "graylog.http.generateCerts.enabled" -}}
  {{- if .Values.graylog.install -}}
    {{- if .Values.graylog.tls -}}
      {{- if .Values.graylog.tls.http -}}
        {{- if .Values.graylog.tls.http.generateCerts -}}
          {{- if .Values.graylog.tls.http.generateCerts.enabled -}}
true
          {{- end -}}
        {{- end -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Return true if generateCerts in TLS is enabled for Graylog input.
*/}}
{{- define "graylog.input.generateCerts.enabled" -}}
  {{- if .Values.graylog.install -}}
    {{- if .Values.graylog.tls -}}
      {{- if .Values.graylog.tls.input -}}
        {{- if .Values.graylog.tls.input.generateCerts -}}
          {{- if .Values.graylog.tls.input.generateCerts.enabled -}}
true
          {{- end -}}
        {{- end -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Return true if generateCerts in TLS is enabled for Fluentd.
*/}}
{{- define "fluentd.generateCerts.enabled" -}}
  {{- if .Values.fluentd.install -}}
    {{- if .Values.fluentd.tls -}}
      {{- if .Values.fluentd.tls.generateCerts -}}
        {{- if .Values.fluentd.tls.generateCerts.enabled -}}
true
        {{- end -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Return true if generateCerts in TLS is enabled for Fluent-bit.
*/}}
{{- define "fluentbit.generateCerts.enabled" -}}
  {{- if .Values.fluentbit.install -}}
    {{- if .Values.fluentbit.tls -}}
      {{- if .Values.fluentbit.tls.generateCerts -}}
        {{- if .Values.fluentbit.tls.generateCerts.enabled -}}
true
        {{- end -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Return true if generateCerts in TLS is enabled for Fluent-bit aggregator.
*/}}
{{- define "fluentbit.aggregator.generateCerts.enabled" -}}
  {{- if .Values.fluentbit.install -}}
    {{- if .Values.fluentbit.aggregator -}}
      {{- if .Values.fluentbit.aggregator.install -}}
        {{- if .Values.fluentbit.aggregator.tls -}}
          {{- if .Values.fluentbit.aggregator.tls.generateCerts -}}
            {{- if .Values.fluentbit.aggregator.tls.generateCerts.enabled -}}
true
            {{- end -}}
          {{- end -}}
        {{- end -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Return true if cacerts for HTTP TLS in Graylog is present and mounted.
*/}}
{{- define "graylog.cacerts.present" -}}
  {{- if .Values.graylog.tls.http.cacerts -}}
true
  {{- else -}}
    {{- if ( include "graylog.http.generateCerts.enabled" . ) -}}
true
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Return true if cert for HTTP TLS in Graylog is present and mounted.
*/}}
{{- define "graylog.cert.present" -}}
  {{- if .Values.graylog.tls.http.cert -}}
    {{- if and .Values.graylog.tls.http.cert.secretName .Values.graylog.tls.http.cert.secretKey }}
true
    {{- else -}}
      {{- if ( include "graylog.http.generateCerts.enabled" . ) -}}
true
      {{- end -}}
    {{- end -}}
  {{- else -}}
    {{- if ( include "graylog.http.generateCerts.enabled" . ) -}}
true
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Return true if key for HTTP TLS in Graylog is present and mounted.
*/}}
{{- define "graylog.key.present" -}}
  {{- if .Values.graylog.tls.http.key -}}
    {{- if and .Values.graylog.tls.http.key.secretName .Values.graylog.tls.http.key.secretKey }}
true
    {{- else -}}
      {{- if (include "graylog.http.generateCerts.enabled" . ) -}}
true
      {{- end -}}
    {{- end -}}
  {{- else -}}
    {{- if (include "graylog.http.generateCerts.enabled" . ) -}}
true
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Set default value for graylog host if not specified in Values.
*/}}
{{- define "graylog.host" -}}
  {{- if .Values.graylog.install -}}
    {{- if not .Values.graylog.host -}}
      {{- if .Values.CLOUD_PUBLIC_HOST -}}
        {{- printf "%s-%s.%s/" "https://graylog" .Values.NAMESPACE (trimSuffix "/" .Values.CLOUD_PUBLIC_HOST) -}}
      {{- end -}}
    {{- else -}}
      {{- $host := trimSuffix "/" .Values.graylog.host -}}
      {{- printf "%s/" $host -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Return secretName if generateCerts in TLS is enabled for Graylog input.
*/}}
{{- define "graylog.secretName" -}}
  {{- if .Values.graylog.install -}}
    {{- if .Values.graylog.tls -}}
      {{- if .Values.graylog.tls.input -}}
        {{- if .Values.graylog.tls.input.cert -}}
            {{- printf "%s" (trimSuffix "/" .Values.graylog.tls.input.cert.secretName) -}}
        {{- else -}}
          {{- if .Values.graylog.tls.input.generateCerts -}}
            {{- if .Values.graylog.tls.input.generateCerts.enabled -}}
              {{- printf "%s" (trimSuffix "/" .Values.graylog.tls.input.generateCerts.secretName) -}}
            {{- end -}}
          {{- end -}}
        {{- end -}}
      {{- end -}}
      {{- if .Values.graylog.tls.http -}}
        {{- if .Values.graylog.tls.http.cert -}}
          {{- printf "%s" (trimSuffix "/" .Values.graylog.tls.http.cert.secretName) -}}
        {{- else -}}
          {{- if .Values.graylog.tls.http.generateCerts -}}
            {{- if .Values.graylog.tls.http.generateCerts.enabled -}}
              {{- printf "%s" (trimSuffix "/" .Values.graylog.tls.http.generateCerts.secretName) -}}
            {{- end -}}
          {{- end -}}
        {{- end -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
{{- end -}}


{{- define "logging.monitoredImages" -}}
{{- end -}}

{{/******************************************************************************************************************/}}

{{/*
Find a logging-operator image in various places.
Image can be found from:
* specified by user from .Values.operatorImage
* default value
*/}}
{{- define "logging-operator.image" -}}
  {{- if .Values.operatorImage -}}
    {{- printf "%s" .Values.operatorImage -}}
  {{- else -}}
    {{- print "ghcr.io/netcracker/qubership-logging-operator:main" -}}
  {{- end -}}
{{- end -}}

{{/*
Find a graylog image in various places.
Image can be found from:
* specified by user from .Values.operatorImage
* default value
*/}}
{{- define "graylog.image" -}}
  {{- if .Values.graylog.dockerImage -}}
    {{- printf "%s" .Values.graylog.dockerImage -}}
  {{- else -}}
    {{- print "docker.io/graylog/graylog:5.2.12" -}}
  {{- end -}}
{{- end -}}

{{/*
Find a fluentd image in various places.
Image can be found from:
* specified by user from .Values.fluentd.dockerImage
* default value
*/}}
{{- define "fluentd.image" -}}
  {{- if .Values.fluentd.dockerImage -}}
    {{- printf "%s" .Values.fluentd.dockerImage -}}
  {{- else -}}
    {{- print "ghcr.io/netcracker/qubership-fluentd:main" -}}
  {{- end -}}
{{- end -}}

{{/*
Find a Fluentd ConfigMap reload image in various places.
Image can be found from:
* specified by user from Values.fluentd.configmapReload.image
* default value
*/}}
{{- define "fluentd.configmapReload.image" -}}
  {{- if .Values.fluentd.configmapReload.dockerImage -}}
    {{- printf "%s" .Values.fluentd.configmapReload.dockerImage -}}
  {{- else -}}
    {{- print "ghcr.io/jimmidyson/configmap-reload:v0.15.0" -}}
  {{- end -}}
{{- end -}}

{{/*
Find a FluentBit image in various places.
Image can be found from:
* specified by user from .Values.fluentbit.dockerImage
* default value
*/}}
{{- define "fluentbit.image" -}}
  {{- if .Values.fluentbit.dockerImage -}}
    {{- printf "%s" .Values.fluentbit.dockerImage -}}
  {{- else -}}
    {{- print "docker.io/fluent/fluent-bit:4.0.1" -}}
  {{- end -}}
{{- end -}}

{{/*
Find a FluentBit ConfigMap reload image in various places.
Image can be found from:
* specified by user from .Values.fluentbit.configmapReload.dockerImage
* default value
*/}}
{{- define "fluentbit.configmapReload.image" -}}
  {{- if .Values.fluentbit.configmapReload.dockerImage -}}
    {{- printf "%s" .Values.fluentbit.configmapReload.dockerImage -}}
  {{- else -}}
    {{- print "ghcr.io/jimmidyson/configmap-reload:v0.15.0" -}}
  {{- end -}}
{{- end -}}

{{/*
Find a cloud-events-reader image in various places.
Image can be found from:
* specified by user from .Values.cloudEventsReader.dockerImage
* default value
*/}}
{{- define "cloud-events-reader.image" -}}
  {{- if .Values.cloudEventsReader.dockerImage -}}
    {{- printf "%s" .Values.cloudEventsReader.dockerImage -}}
  {{- else -}}
    {{- print "ghcr.io/netcracker/qubership-kube-events-reader:main" -}}
  {{- end -}}
{{- end -}}

{{/*
Find a logging-integration-tests image in various places.
Image can be found from:
* specified by user from .Values.integrationTests.image
* default value
*/}}
{{- define "logging-integration-tests.image" -}}
  {{- if .Values.integrationTests.image -}}
    {{- printf "%s" .Values.integrationTests.image -}}
  {{- else -}}
    {{- print "ghcr.io/netcracker/qubership-logging-integration-tests:main" -}}
  {{- end -}}
{{- end -}}

{{/*
Find a graylog-plugins-init-container image in various places.
Image can be found from:
* specified by user from .Values.graylog.initContainerDockerImage
* default value
*/}}
{{- define "graylog-plugins-init.image" -}}
  {{- if .Values.graylog.initContainerDockerImage -}}
    {{- printf "%s" .Values.graylog.initContainerDockerImage -}}
  {{- else -}}
    {{- print "ghcr.io/netcracker/qubership-graylog-plugins-init:main" -}}
  {{- end -}}
{{- end -}}

{{/*
Find a mongodb image in various places.
Image can be found from:
* specified by user from .Values.graylog.mongodbImage
* default value
*/}}
{{- define "mongodb.image" -}}
  {{- if .Values.graylog.mongodbImage -}}
    {{- printf "%s" .Values.graylog.mongodbImage -}}
  {{- else -}}
    {{- print "docker.io/mongo:5.0.31" -}}
  {{- end -}}
{{- end -}}

{{/*
Find a authProxy image in various places.
Image can be found from:
* specified by user from .Values.graylog.authProxy.image
* default value
*/}}
{{- define "authProxy.image" -}}
  {{- if .Values.graylog.authProxy.image -}}
    {{- printf "%s" .Values.graylog.authProxy.image -}}
  {{- else -}}
    {{- print "ghcr.io/netcracker/qubership-graylog-auth-proxy:main" -}}
  {{- end -}}
{{- end -}}

{{/*
Find a init_setup image in various places.
Image can be found from:
* specified by user from .Values.graylog.initSetupImage
* default value
*/}}
{{- define "init-setup.image" -}}
  {{- if .Values.graylog.initSetupImage -}}
    {{- printf "%s" .Values.graylog.initSetupImage -}}
  {{- else -}}
    {{- print "docker.io/alpine:3.21.3" -}}
  {{- end -}}
{{- end -}}

{{/*
MongoDB images for sequential upgrades.
Upgrade path:
3.6.23 -> 4.0.28 -> 4.2.22 -> 4.4.17 -> 5.0.19
*/}}

{{/*
MongoDB 4.0 image.
*/}}
{{- define "mongodb40.image" -}}
  {{- if .Values.graylog.mongodb40Image -}}
    {{- printf "%s" .Values.graylog.mongodb40Image -}}
  {{- else -}}
    {{- print "docker.io/mongo:4.0.28" -}}
  {{- end -}}
{{- end -}}

{{/*
MongoDB 4.2 image.
*/}}
{{- define "mongodb42.image" -}}
  {{- if .Values.graylog.mongodb42Image -}}
    {{- printf "%s" .Values.graylog.mongodb42Image -}}
  {{- else -}}
    {{- print "docker.io/mongo:4.2.22" -}}
  {{- end -}}
{{- end -}}

{{/*
MongoDB 4.4 image.
*/}}
{{- define "mongodb44.image" -}}
  {{- if .Values.graylog.mongodb44Image -}}
    {{- printf "%s" .Values.graylog.mongodb44Image -}}
  {{- else -}}
    {{- print "docker.io/mongo:4.4.17" -}}
  {{- end -}}
{{- end -}}

{{/*
Return securityContext section for operator pods
*/}}
{{- define "operator.securityContext" -}}
  {{- if .Values.securityContext }}
    {{- toYaml .Values.securityContext | nindent 8 }}
    {{- if not (.Capabilities.APIVersions.Has "apps.openshift.io/v1") }}
      {{- if not .Values.securityContext.runAsUser }}
        runAsUser: 10001
      {{- end }}
    {{- end }}
    {{- if (eq (.Values.securityContext.runAsNonRoot | toString) "false") }}
      runAsNonRoot: false
    {{- else }}
      runAsNonRoot: true
    {{- end }}
    {{- if and (ge .Capabilities.KubeVersion.Minor "25") (not .Values.securityContext.seccompProfile) }}
      seccompProfile:
        type: "RuntimeDefault"
    {{- end }}
  {{- else }}
       runAsUser: 10001
       runAsNonRoot: true
    {{- if ge .Capabilities.KubeVersion.Minor "25" }}
       seccompProfile:
         type: "RuntimeDefault"
    {{- end }}
  {{- end }}
{{- end -}}

{{/*
Return container securityContext section for operator pods
*/}}
{{- define "operator.containerSecurityContext" -}}
  {{- if ge .Capabilities.KubeVersion.Minor "25" }}
    {{- if .Values.containerSecurityContext }}
      {{- toYaml .Values.containerSecurityContext | nindent 12 }}
    {{- else }}
           allowPrivilegeEscalation: false
           readOnlyRootFilesystem: true
           capabilities:
             drop:
               - ALL
    {{- end }}
  {{- else }}
    {{- if .Values.containerSecurityContext }}
      {{- toYaml .Values.containerSecurityContext | nindent 12 }}
    {{- else }}
      {}
    {{- end }}
  {{- end }}
{{- end -}}

{{- define "operator.readinessProbe" -}}
readinessProbe:
  tcpSocket:
    port: 8383
  initialDelaySeconds: 10
  periodSeconds: 5
  failureThreshold: 3
  timeoutSeconds: 1
{{- end -}}

{{- define "operator.livenessProbe" -}}
livenessProbe:
  tcpSocket:
    port: 8383
  initialDelaySeconds: 10
  periodSeconds: 5
  failureThreshold: 3
  timeoutSeconds: 3
{{- end -}}
