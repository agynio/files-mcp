{{- define "files-mcp.configureEnv" -}}
{{- $env := list -}}

{{- $mcpPort := int (default 8100 .Values.filesMcp.mcpPort) -}}
{{- $env = append $env (dict "name" "MCP_PORT" "value" (printf "%d" $mcpPort)) -}}

{{- $gatewayAddress := trimAll " \n\t" (default "gateway.ziti" .Values.filesMcp.gatewayAddress) -}}
{{- $env = append $env (dict "name" "GATEWAY_ADDRESS" "value" $gatewayAddress) -}}

{{- $tokenSecret := trim (default "" .Values.filesMcp.apiToken.existingSecret) -}}
{{- $tokenVar := dict "name" "AGYN_API_TOKEN" -}}
{{- if $tokenSecret }}
  {{- $tokenKey := default "agyn-api-token" .Values.filesMcp.apiToken.existingSecretKey -}}
  {{- $_ := set $tokenVar "valueFrom" (dict "secretKeyRef" (dict "name" $tokenSecret "key" $tokenKey)) -}}
  {{- $env = append $env $tokenVar -}}
{{- else }}
  {{- $tokenValue := trimAll " \n\t" (default "" .Values.filesMcp.apiToken.value) -}}
  {{- if $tokenValue }}
    {{- $_ := set $tokenVar "value" $tokenValue -}}
    {{- $env = append $env $tokenVar -}}
  {{- end }}
{{- end }}

{{- $maxSize := int (default 20971520 .Values.filesMcp.maxFileSize) -}}
{{- $env = append $env (dict "name" "MAX_FILE_SIZE" "value" (printf "%d" $maxSize)) -}}

{{- $userEnv := .Values.env | default (list) -}}
{{- $_ := set .Values "env" (concat $env $userEnv) -}}
{{- end -}}
