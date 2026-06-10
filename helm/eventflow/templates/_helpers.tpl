{{- define "eventflow.name" -}}
eventflow
{{- end }}

{{- define "eventflow.labels" -}}
app.kubernetes.io/name: {{ include "eventflow.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "eventflow.databaseUrl" -}}
postgres://{{ .Values.postgres.user }}:{{ .Values.postgres.password }}@{{ include "eventflow.name" . }}-postgres:5432/{{ .Values.postgres.database }}?sslmode=disable
{{- end }}

{{- define "eventflow.redisUrl" -}}
redis://{{ include "eventflow.name" . }}-redis:6379/0
{{- end }}
