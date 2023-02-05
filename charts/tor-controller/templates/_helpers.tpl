{{/*
Expand the name of the chart.
*/}}
{{- define "tor-controller.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "tor-controller.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "tor-controller.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "tor-controller.labels" -}}
helm.sh/chart: {{ include "tor-controller.chart" . }}
{{ include "tor-controller.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "tor-controller.selectorLabels" -}}
app.kubernetes.io/name: {{ include "tor-controller.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "tor-controller.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "tor-controller.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Helper to dynamically create cluster-wide or namespaced roles & rolebindings
*/}}
{{- define "tor-controller.roleKind" -}}
{{- if .Values.namespaced }}
{{- "Role" }}
{{- else }}
{{- "ClusterRole" }}
{{- end }}
{{- end }}

{{/*
Helper to dynamically create cluster-wide or namespaced roles & rolebindings
*/}}
{{- define "tor-controller.roleBindingKind" -}}
{{- if .Values.namespaced }}
{{- "RoleBinding" }}
{{- else }}
{{- "ClusterRoleBinding" }}
{{- end }}
{{- end }}