{{- define "nico-dpfhcp-provider.namespace" -}}
{{- default .Release.Namespace .Values.namespaceOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "nico-dpfhcp-provider.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "nico-dpfhcp-provider.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "nico-dpfhcp-provider.labels" -}}
helm.sh/chart: {{ include "nico-dpfhcp-provider.chart" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: site-controller
app.kubernetes.io/name: {{ include "nico-dpfhcp-provider.name" . }}
app.kubernetes.io/component: dpfhcp-provider
{{- end }}

{{- define "nico-dpfhcp-provider.selectorLabels" -}}
app: nico-dpfhcp-provider
app.kubernetes.io/name: {{ include "nico-dpfhcp-provider.name" . }}
app.kubernetes.io/component: dpfhcp-provider
{{- end }}

{{/*
Resolve the container image. Prefers image.override when set, otherwise
composes global.image.repository/image.name:global.image.tag. Falls back
to bare image.name:tag for local Kind dev (when repository is empty).
*/}}
{{- define "nico-dpfhcp-provider.image" -}}
{{- if .Values.image.override -}}
{{ .Values.image.override }}
{{- else if .Values.global.image.repository -}}
{{ .Values.global.image.repository }}/{{ .Values.image.name }}:{{ .Values.global.image.tag }}
{{- else -}}
{{ .Values.image.name }}:{{ .Values.global.image.tag }}
{{- end -}}
{{- end }}
