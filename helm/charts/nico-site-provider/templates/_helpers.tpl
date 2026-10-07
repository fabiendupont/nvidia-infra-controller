{{- define "nico-site-provider.namespace" -}}
{{- default .Release.Namespace .Values.namespaceOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "nico-site-provider.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "nico-site-provider.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "nico-site-provider.labels" -}}
helm.sh/chart: {{ include "nico-site-provider.chart" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: site-controller
app.kubernetes.io/name: {{ include "nico-site-provider.name" . }}
app.kubernetes.io/component: site-provider
{{- end }}

{{- define "nico-site-provider.selectorLabels" -}}
app: nico-site-provider
app.kubernetes.io/name: {{ include "nico-site-provider.name" . }}
app.kubernetes.io/component: site-provider
{{- end }}

{{/*
Resolve the container image. Prefers image.override when set, otherwise
composes global.image.repository/image.name:global.image.tag. Falls back
to bare image.name:tag for local Kind dev (when repository is empty).
*/}}
{{- define "nico-site-provider.image" -}}
{{- if .Values.image.override -}}
{{ .Values.image.override }}
{{- else if .Values.global.image.repository -}}
{{ .Values.global.image.repository }}/{{ .Values.image.name }}:{{ .Values.global.image.tag }}
{{- else -}}
{{ .Values.image.name }}:{{ .Values.global.image.tag }}
{{- end -}}
{{- end }}
