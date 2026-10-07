{{- define "nico-health-provider.namespace" -}}
{{- default .Release.Namespace .Values.namespaceOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "nico-health-provider.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "nico-health-provider.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "nico-health-provider.labels" -}}
helm.sh/chart: {{ include "nico-health-provider.chart" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: site-controller
app.kubernetes.io/name: {{ include "nico-health-provider.name" . }}
app.kubernetes.io/component: health-provider
{{- end }}

{{- define "nico-health-provider.selectorLabels" -}}
app: nico-health-provider
app.kubernetes.io/name: {{ include "nico-health-provider.name" . }}
app.kubernetes.io/component: health-provider
{{- end }}

{{/*
Resolve the container image. Prefers image.override when set, otherwise
composes global.image.repository/image.name:global.image.tag. Falls back
to bare image.name:tag for local Kind dev (when repository is empty).
*/}}
{{- define "nico-health-provider.image" -}}
{{- if .Values.image.override -}}
{{ .Values.image.override }}
{{- else if .Values.global.image.repository -}}
{{ .Values.global.image.repository }}/{{ .Values.image.name }}:{{ .Values.global.image.tag }}
{{- else -}}
{{ .Values.image.name }}:{{ .Values.global.image.tag }}
{{- end -}}
{{- end }}
