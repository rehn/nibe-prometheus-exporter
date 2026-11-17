{{- define "nibe-exporter.labels" -}}
app: nibe-prometheus-exporter
app.kubernetes.io/name: nibe-prometheus-exporter
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
{{- end -}}

{{- define "nibe-exporter.secretName" -}}
{{- default .Release.Name .Values.nibe.existingSecret -}}
{{- end -}}
