package templates

import (
	_ "embed"
	"html/template"
	"strings"
)

//go:embed report.gohtml
var reportHTML string

//go:embed report.css
var reportCSS string

//go:embed report.js
var reportJS string

// TemplateData contains the data and assets for the report template
type TemplateData struct {
	CSS template.CSS
	JS  template.JS
	// Embed the actual report data
	Summary struct {
		TotalDependencies int      `json:"totalDependencies"`
		UniqueLicenses    []string `json:"uniqueLicenses"`
		RiskLevel         string   `json:"riskLevel"`
		Conflicts         []string `json:"conflicts"`
		Recommendations   []string `json:"recommendations"`
	} `json:"summary"`
	Dependencies []Dependency `json:"dependencies"`
	Timestamp    string       `json:"timestamp,omitempty"`
}

type Dependency struct {
	Name       string  `json:"name"`
	Version    string  `json:"version"`
	License    string  `json:"license"`
	Confidence float64 `json:"confidence"`
	Source     string  `json:"source"`
}

// GetReportTemplate returns the parsed HTML report template
func GetReportTemplate() (*template.Template, error) {
	return template.New("report").Funcs(template.FuncMap{
		"title": func(s string) string {
			if len(s) == 0 {
				return s
			}
			return strings.ToUpper(s[:1]) + s[1:]
		},
	}).Parse(reportHTML)
}

// GetTemplateData creates template data with embedded CSS and JS
func GetTemplateData() TemplateData {
	return TemplateData{
		CSS: template.CSS(reportCSS),
		JS:  template.JS(reportJS),
	}
}
