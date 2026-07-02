{{- define "gateway.name" -}}
web3-gateway
{{- end }}

{{- define "gateway.fullname" -}}
{{ include "gateway.name" . }}
{{- end }}
