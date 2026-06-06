{{/*
Expand the name of the chart.
*/}}
{{- define "kloud-csi.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "kloud-csi.fullname" -}}
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

{{- define "kloud-csi.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "kloud-csi.labels" -}}
helm.sh/chart: {{ include "kloud-csi.chart" . }}
{{ include "kloud-csi.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "kloud-csi.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kloud-csi.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "kloud-csi.controllerName" -}}
{{- printf "%s-controller" (include "kloud-csi.fullname" .) }}
{{- end }}

{{- define "kloud-csi.nodeName" -}}
{{- printf "%s-node" (include "kloud-csi.fullname" .) }}
{{- end }}

{{- define "kloud-csi.controllerServiceAccountName" -}}
{{- if .Values.serviceAccount.controller.create }}
{{- default (printf "%s-controller-sa" (include "kloud-csi.fullname" .)) .Values.serviceAccount.controller.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.controller.name }}
{{- end }}
{{- end }}

{{- define "kloud-csi.nodeServiceAccountName" -}}
{{- if .Values.serviceAccount.node.create }}
{{- default (printf "%s-node-sa" (include "kloud-csi.fullname" .)) .Values.serviceAccount.node.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.node.name }}
{{- end }}
{{- end }}

{{- define "kloud-csi.driverImage" -}}
{{- printf "%s:%s" .Values.image.repository (default .Chart.AppVersion .Values.image.tag) }}
{{- end }}

{{- define "kloud-csi.validateRequired" -}}
{{- if and (not .Values.existingSecret) (not .Values.accessToken) }}
{{- fail "accessToken or existingSecret is required" }}
{{- end }}
{{- if not .Values.serverURL }}
{{- fail "serverURL is required" }}
{{- end }}
{{- end }}
