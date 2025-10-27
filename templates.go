package cliutil

import (
	_ "embed"
	"text/template"
)

//go:embed templates/usage.gotmpl
var UsageTemplateText string

var UsageTemplate = template.Must(template.New("usage").Parse(UsageTemplateText))
