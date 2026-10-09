{{- define "nico-provisioner-metal3.namespace" -}}
{{- default .Release.Namespace .Values.namespaceOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "nico-provisioner-metal3.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "nico-provisioner-metal3.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "nico-provisioner-metal3.labels" -}}
helm.sh/chart: {{ include "nico-provisioner-metal3.chart" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: site-controller
app.kubernetes.io/name: {{ include "nico-provisioner-metal3.name" . }}
app.kubernetes.io/component: provisioner-metal3
nico.nvidia.com/provider: "true"
{{- end }}

{{- define "nico-provisioner-metal3.selectorLabels" -}}
app: nico-provisioner-metal3
app.kubernetes.io/name: {{ include "nico-provisioner-metal3.name" . }}
app.kubernetes.io/component: provisioner-metal3
{{- end }}

{{- define "nico-provisioner-metal3.image" -}}
{{- if .Values.image.override -}}
{{ .Values.image.override }}
{{- else if .Values.global.image.repository -}}
{{ .Values.global.image.repository }}/{{ .Values.image.name }}:{{ .Values.global.image.tag }}
{{- else -}}
{{ .Values.image.name }}:{{ .Values.global.image.tag }}
{{- end -}}
{{- end }}
