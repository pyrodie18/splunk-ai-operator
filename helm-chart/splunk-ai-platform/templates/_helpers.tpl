{{/*
Expand the name of the chart.
*/}}
{{- define "splunk-ai-platform.name" -}}
{{- default .Chart.Name .Values.nameOverride -}}
{{- end -}}

{{- define "splunk-ai-platform.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" (include "splunk-ai-platform.name" .) .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "splunk-ai-platform.labels" -}}
app.kubernetes.io/name: {{ include "splunk-ai-platform.name" . }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Define namespace of release and allow for namespace override
*/}}
{{- define "splunk-ai-platform.namespace" -}}
{{- default .Release.Namespace .Values.namespaceOverride }}
{{- end }}
