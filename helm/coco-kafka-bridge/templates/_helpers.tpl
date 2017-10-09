{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 24 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 24 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-%s" .Release.Name $name | trunc 24 | trimSuffix "-" -}}
{{- end -}}

{{- define "target_env" -}}
{{- required "The target_env value is required for this app. Use helm upgrade ... --set target_env=... when installing." .Values.target_env -}}
{{- end -}}

{{- define "env-full-name" -}}
{{- $region := .Values.region -}}
{{- $env := required "The target_env value is required for this app. Use helm upgrade ... --set target_env=... when installing." .Values.target_env -}}
{{- if eq $region "null" -}}
{{- printf "%s" $env -}}
{{- else -}}
{{- printf "%s-%s" $env $region -}}
{{- end -}}
{{- end -}}
