{{- define "tradeops.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "tradeops.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name (include "tradeops.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "tradeops.namespace" -}}
{{- default .Release.Namespace .Values.namespace.name -}}
{{- end -}}

{{- define "tradeops.labels" -}}
app.kubernetes.io/name: {{ include "tradeops.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/part-of: tradeops
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
{{- end -}}

{{- define "tradeops.selectorLabels" -}}
app.kubernetes.io/name: {{ include "tradeops.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "tradeops.componentName" -}}
{{- printf "%s-%s" (include "tradeops.fullname" .root) .component.name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "tradeops.componentLabels" -}}
{{ include "tradeops.labels" .root }}
app.kubernetes.io/component: {{ .component.name }}
{{- end -}}

{{- define "tradeops.componentSelectorLabels" -}}
{{ include "tradeops.selectorLabels" .root }}
app.kubernetes.io/component: {{ .component.name }}
{{- end -}}

{{- define "tradeops.serviceAccountName" -}}
{{- if .component.serviceAccountName -}}
{{- .component.serviceAccountName -}}
{{- else if .root.Values.serviceAccount.create -}}
{{- printf "%s-%s" (include "tradeops.fullname" .root) .component.name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- default "default" .root.Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "tradeops.secretName" -}}
{{- if .Values.secrets.existingSecretName -}}
{{- .Values.secrets.existingSecretName -}}
{{- else -}}
{{- printf "%s-secrets" (include "tradeops.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "tradeops.configName" -}}
{{- printf "%s-config" (include "tradeops.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "tradeops.runtimeConfigName" -}}
{{- printf "%s-runtime-config" (include "tradeops.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "tradeops.postgresqlHost" -}}
{{- if eq .Values.postgresql.mode "local" -}}
{{- printf "%s-postgresql" (include "tradeops.fullname" .) -}}
{{- else -}}
{{- required "postgresql.external.host is required when postgresql.mode=external" .Values.postgresql.external.host -}}
{{- end -}}
{{- end -}}

{{- define "tradeops.redisHost" -}}
{{- if eq .Values.redis.mode "local" -}}
{{- printf "%s-redis" (include "tradeops.fullname" .) -}}
{{- else -}}
{{- required "redis.external.host is required when redis.mode=external" .Values.redis.external.host -}}
{{- end -}}
{{- end -}}

{{- define "tradeops.kafkaBrokers" -}}
{{- if eq .Values.kafka.mode "local" -}}
{{- printf "%s-redpanda:9092" (include "tradeops.fullname" .) -}}
{{- else -}}
{{- required "kafka.external.brokers is required when kafka.mode=external" .Values.kafka.external.brokers -}}
{{- end -}}
{{- end -}}

{{- define "tradeops.mqttBroker" -}}
{{- if eq .Values.mqtt.mode "local" -}}
{{- printf "tcp://%s-mosquitto:1883" (include "tradeops.fullname" .) -}}
{{- else -}}
{{- required "mqtt.external.brokerUrl is required when mqtt.mode=external" .Values.mqtt.external.brokerUrl -}}
{{- end -}}
{{- end -}}

{{- define "tradeops.databaseUrl" -}}
{{- if .Values.postgresql.databaseUrlOverride -}}
{{- .Values.postgresql.databaseUrlOverride -}}
{{- else -}}
{{- printf "postgres://%s:$(POSTGRES_PASSWORD)@%s:%v/%s?sslmode=%s" .Values.postgresql.auth.username (include "tradeops.postgresqlHost" .) (.Values.postgresql.port | int) .Values.postgresql.auth.database .Values.postgresql.sslMode -}}
{{- end -}}
{{- end -}}

{{- define "tradeops.pythonDatabaseUrl" -}}
{{- if .Values.postgresql.pythonDatabaseUrlOverride -}}
{{- .Values.postgresql.pythonDatabaseUrlOverride -}}
{{- else -}}
{{- printf "postgresql+psycopg://%s:$(POSTGRES_PASSWORD)@%s:%v/%s" .Values.postgresql.auth.username (include "tradeops.postgresqlHost" .) (.Values.postgresql.port | int) .Values.postgresql.auth.database -}}
{{- end -}}
{{- end -}}

