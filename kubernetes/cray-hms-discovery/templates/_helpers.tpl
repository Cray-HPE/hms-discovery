{{/*
Helper function to get the proper image prefix
*/}}
{{- define "cray-hms-discovery.image-prefix" -}}
    {{- printf "%s/" .Values.imagesHost -}}
{{- end -}}

{{/*
Helper function to get the proper image tag
*/}}
{{- define "cray-hms-discovery.imageTag" -}}
{{- default "latest" .Chart.AppVersion -}}
{{- end -}}
