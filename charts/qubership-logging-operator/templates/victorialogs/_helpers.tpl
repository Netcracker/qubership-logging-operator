{{/*
Return the VictoriaLogs resource name.
*/}}
{{- define "victorialogs.fullname" -}}
{{- default "victorialogs" .Values.victorialogs.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Return the VMAuth resource name.
*/}}
{{- define "victorialogs.vmauthFullname" -}}
{{- printf "vmauth-%s" (include "victorialogs.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Return true when VMAuth is required for an external VictoriaLogs endpoint.
*/}}
{{- define "victorialogs.vmauthEnabled" -}}
{{- if and .Values.victorialogs.install (or .Values.victorialogs.ingress.install .Values.victorialogs.httpRoute.install) -}}
true
{{- end -}}
{{- end -}}

{{/*
Render the validated VMAuth configuration and add the VictoriaLogs backend when a user does not define routing.
*/}}
{{- define "victorialogs.vmauthConfig" -}}
{{- $config := deepCopy (.Values.victorialogs.vmauth.config | default dict) -}}
{{- $users := $config.users | default list -}}
{{- if not $users -}}
{{- fail "victorialogs.vmauth.config.users must contain at least one authenticated user when Ingress or HTTPRoute is enabled" -}}
{{- end -}}
{{- if hasKey $config "unauthorized_user" -}}
{{- fail "victorialogs.vmauth.config.unauthorized_user is not allowed because it bypasses authentication" -}}
{{- end -}}
{{- $backendURL := printf "http://%s:%v/" (include "victorialogs.fullname" .) .Values.victorialogs.service.port -}}
{{- range $index, $user := $users -}}
{{- $hasUsername := and (hasKey $user "username") (kindIs "string" (get $user "username")) (not (empty (get $user "username"))) -}}
{{- $hasPassword := and (hasKey $user "password") (kindIs "string" (get $user "password")) (not (empty (get $user "password"))) -}}
{{- $hasBearerToken := and (hasKey $user "bearer_token") (kindIs "string" (get $user "bearer_token")) (not (empty (get $user "bearer_token"))) -}}
{{- $hasUnsupportedAuth := or (hasKey $user "auth_token") (hasKey $user "jwt") -}}
{{- $validBasicAuth := and $hasUsername $hasPassword (not (hasKey $user "bearer_token")) (not $hasUnsupportedAuth) -}}
{{- $validBearerAuth := and $hasBearerToken (not (hasKey $user "username")) (not (hasKey $user "password")) (not $hasUnsupportedAuth) -}}
{{- if not (or $validBasicAuth $validBearerAuth) -}}
{{- fail (printf "victorialogs.vmauth.config.users[%d] must define exactly one authentication method: non-empty username and password, or non-empty bearer_token" $index) -}}
{{- end -}}
{{- if not (or (hasKey $user "url_prefix") (hasKey $user "url_map") (hasKey $user "default_url")) -}}
{{- $_ := set $user "url_prefix" $backendURL -}}
{{- end -}}
{{- end -}}
{{- toYaml $config -}}
{{- end -}}

{{/*
Return the default public host used by the VictoriaLogs Ingress and HTTPRoute.
*/}}
{{- define "victorialogs.externalHost" -}}
{{- $cloudPublicHost := required "CLOUD_PUBLIC_HOST must be set when VictoriaLogs Ingress or HTTPRoute hosts are empty" .Values.CLOUD_PUBLIC_HOST -}}
{{- $cloudPublicHost = trimSuffix "/" $cloudPublicHost | trimPrefix "https://" | trimPrefix "http://" | trimPrefix "." -}}
{{- printf "vmauth-%s.%s" .Release.Namespace $cloudPublicHost -}}
{{- end -}}

{{/*
Return labels used to select VMAuth Pods.
*/}}
{{- define "victorialogs.vmauthSelectorLabels" -}}
name: {{ include "victorialogs.vmauthFullname" . }}
app.kubernetes.io/name: {{ include "victorialogs.vmauthFullname" . }}
app.kubernetes.io/component: vmauth
{{- end -}}

{{/*
Return common VMAuth labels.
*/}}
{{- define "victorialogs.vmauthLabels" -}}
{{- $requiredLabels := include "logging.labels" (dict "ctx" . "name" (include "victorialogs.vmauthFullname" .) "component" "vmauth") | fromYaml -}}
{{- $extraLabels := mergeOverwrite (deepCopy (.Values.labels | default dict)) (.Values.victorialogs.labels | default dict) -}}
{{- mergeOverwrite $extraLabels $requiredLabels | toYaml -}}
{{- end -}}

{{/*
Return VMAuth Pod labels with immutable selector labels taking precedence.
*/}}
{{- define "victorialogs.vmauthPodLabels" -}}
{{- $selectorLabels := include "victorialogs.vmauthSelectorLabels" . | fromYaml -}}
{{- $podLabels := deepCopy (.Values.victorialogs.vmauth.podLabels | default dict) -}}
{{- mergeOverwrite $podLabels $selectorLabels | toYaml -}}
{{- end -}}

{{/*
Return the VMAuth image.
*/}}
{{- define "victorialogs.vmauthImage" -}}
{{- if .Values.victorialogs.vmauth.dockerImage -}}
{{- .Values.victorialogs.vmauth.dockerImage -}}
{{- else -}}
{{- /* renovate: datasource=docker depName=victoriametrics/vmauth */ -}}
{{- print "docker.io/victoriametrics/vmauth:v1.147.0" -}}
{{- end -}}
{{- end -}}

{{/*
Return the PVC used by VictoriaLogs.
*/}}
{{- define "victorialogs.claimName" -}}
{{- default (include "victorialogs.fullname" .) .Values.victorialogs.storage.existingClaim -}}
{{- end -}}

{{/*
Return the headless Service name used by the StatefulSet.
*/}}
{{- define "victorialogs.headlessServiceName" -}}
{{- printf "%s-headless" (include "victorialogs.fullname" . | trunc 54 | trimSuffix "-") -}}
{{- end -}}

{{/*
Return labels used to select VictoriaLogs Pods.
*/}}
{{- define "victorialogs.selectorLabels" -}}
name: {{ include "victorialogs.fullname" . }}
app.kubernetes.io/name: {{ include "victorialogs.fullname" . }}
app.kubernetes.io/component: victorialogs
{{- end -}}

{{/*
Return common VictoriaLogs labels.
*/}}
{{- define "victorialogs.labels" -}}
{{- $requiredLabels := include "logging.labels" (dict "ctx" . "name" (include "victorialogs.fullname" .) "component" "victorialogs") | fromYaml -}}
{{- $extraLabels := mergeOverwrite (deepCopy (.Values.labels | default dict)) (.Values.victorialogs.labels | default dict) -}}
{{- mergeOverwrite $extraLabels $requiredLabels | toYaml -}}
{{- end -}}

{{/*
Return VictoriaLogs Pod labels with immutable selector labels taking precedence.
*/}}
{{- define "victorialogs.podLabels" -}}
{{- $selectorLabels := include "victorialogs.selectorLabels" . | fromYaml -}}
{{- $podLabels := deepCopy (.Values.victorialogs.podLabels | default dict) -}}
{{- mergeOverwrite $podLabels $selectorLabels | toYaml -}}
{{- end -}}

{{/*
Return labels for a VictoriaLogs Service. The service role cannot be overridden.
*/}}
{{- define "victorialogs.serviceLabels" -}}
{{- $ctx := .ctx -}}
{{- $requiredLabels := include "victorialogs.labels" $ctx | fromYaml -}}
{{- $_ := set $requiredLabels "app.kubernetes.io/service-role" .role -}}
{{- $serviceLabels := deepCopy ($ctx.Values.victorialogs.service.labels | default dict) -}}
{{- mergeOverwrite $serviceLabels $requiredLabels | toYaml -}}
{{- end -}}

{{/*
Return the VictoriaLogs image.
*/}}
{{- define "victorialogs.image" -}}
{{- if .Values.victorialogs.dockerImage -}}
{{- .Values.victorialogs.dockerImage -}}
{{- else -}}
{{- /* renovate: datasource=docker depName=victoriametrics/victoria-logs */ -}}
{{- print "docker.io/victoriametrics/victoria-logs:v1.51.0" -}}
{{- end -}}
{{- end -}}
