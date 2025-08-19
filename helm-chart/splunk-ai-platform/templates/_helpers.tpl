{{/*
Expand the name of the chart.
*/}}
{{- define "splunk-ai-platform.name" -}}
{{- default .Chart.Name .Values.nameOverride -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 38 chars because some Kubernetes name fields are limited to 63 (by the DNS naming spec), and we concatenate this name to create names of other resources.
If release name contains chart name it will be used as a full name.
*/}}
{{- define "splunk-ai-platform.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 38 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 38 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 38 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

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
