{{/*
Expand the name of the chart.
*/}}
{{- define "automq-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "automq-operator.fullname" -}}
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
{{- define "automq-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "automq-operator.labels" -}}
helm.sh/chart: {{ include "automq-operator.chart" . }}
{{ include "automq-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "automq-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "automq-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}


{{/*
Extract the tag from the image repository string.
Assumes the tag is the part after the last colon `:`.
If no tag is present, returns an empty string.
*/}}
{{- define "automq-operator.getTag" -}}
  {{- $image := .Values.image -}}
  {{- $splitImage := splitList ":" $image -}}
  {{- if eq (len $splitImage) 2 -}}
    {{- index $splitImage 1 -}}
  {{- else -}}
    latest
  {{- end -}}
{{- end -}}

{{/*
Determine the image pull policy.
If the image tag is 'latest', set the pull policy to 'Always'.
Otherwise, use the default pull policy from values.yaml.
*/}}
{{- define "automq-operator.pullPolicy" -}}
{{- if eq (include "automq-operator.getTag" .) "latest" -}}
Always
{{- else -}}
IfNotPresent
{{- end -}}
{{- end -}}

{{- define "automq-operator.revision" -}}
{{- if eq (include "automq-operator.getTag" .) "latest" -}}
{{.Release.Revision}}
{{- else -}}
{{ include "automq-operator.getTag" .}}
{{- end -}}
{{- end -}}
