package cliutil

import (
	_ "embed"
	"text/template"
)

//go:embed templates/usage.gotmpl
var UsageTemplateText string

var UsageTemplate = template.Must(template.New("usage").Parse(UsageTemplateText))

//go:embed templates/cmd_usage.gotmpl
var CmdUsageTemplateText string

var CmdUsageTemplate = template.Must(template.New("cmd_usage").Parse(CmdUsageTemplateText))
