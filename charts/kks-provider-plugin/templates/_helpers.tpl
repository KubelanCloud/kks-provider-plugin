{{- define "kks-provider-plugin.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "kks-provider-plugin.fullname" -}}
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

{{- define "kks-provider-plugin.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "kks-provider-plugin.labels" -}}
helm.sh/chart: {{ include "kks-provider-plugin.chart" . }}
{{ include "kks-provider-plugin.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "kks-provider-plugin.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kks-provider-plugin.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "kks-provider-plugin.image" -}}
{{- printf "%s:%s" .Values.image.repository (default .Chart.AppVersion .Values.image.tag) }}
{{- end }}

{{- define "kks-provider-plugin.daemonSetName" -}}
{{- printf "%s-provider" (include "kks-provider-plugin.fullname" .) }}
{{- end }}

{{- define "kks-provider-plugin.lbServiceAccountName" -}}
{{- if .Values.lb.serviceAccount.create }}
{{- default (printf "%s-lb-sa" (include "kks-provider-plugin.fullname" .)) .Values.lb.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.lb.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "kks-provider-plugin.lbSecretName" -}}
{{- default (printf "%s-lb-config" (include "kks-provider-plugin.fullname" .)) .Values.lb.existingSecret }}
{{- end }}

{{- define "kks-provider-plugin.csiSecretName" -}}
{{- default (printf "%s-csi-config" (include "kks-provider-plugin.fullname" .)) .Values.csi.existingSecret }}
{{- end }}

{{- define "kks-provider-plugin.validateRequired" -}}
{{- if and (not .Values.lb.existingSecret) (not .Values.lb.accessToken) }}
{{- fail "lb.accessToken or lb.existingSecret is required" }}
{{- end }}
{{- if not .Values.lb.serverURL }}
{{- fail "lb.serverURL is required" }}
{{- end }}
{{- if and (not .Values.csi.existingSecret) (not .Values.csi.accessToken) }}
{{- fail "csi.accessToken or csi.existingSecret is required" }}
{{- end }}
{{- if not .Values.csi.serverURL }}
{{- fail "csi.serverURL is required" }}
{{- end }}
{{- end }}
