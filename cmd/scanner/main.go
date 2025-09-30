package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/stefano/license-scanner/internal/analyzer"
	"github.com/stefano/license-scanner/internal/constants"
	"github.com/stefano/license-scanner/internal/scanner"
	"github.com/stefano/license-scanner/internal/templates"
)

type ScanResult struct {
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

func main() {
	// Parse command line flags
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	format := flag.String("format", "json", "Output format (json, html)")
	_ = flag.Bool("prod-only", false, "Scan production dependencies only")
	_ = flag.Bool("no-summary", false, "Skip license summary")
	flag.Parse()

	// Get project path from remaining arguments
	projectPath := "."
	if flag.NArg() > 0 {
		projectPath = flag.Arg(0)
	}

	// Create and run scanner
	s := scanner.NewWithVerbose(projectPath, *verbose)
	scanResult, err := s.Scan()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning project: %v\n", err)
		os.Exit(1)
	}

	// Convert scanner result to CLI output format
	dependencies := make([]Dependency, len(scanResult.Dependencies))
	analyzerDeps := make([]analyzer.Dependency, len(scanResult.Dependencies))

	for i, dep := range scanResult.Dependencies {
		license := dep.License
		if license == "" {
			license = constants.UnknownLicense
		}

		dependencies[i] = Dependency{
			Name:       dep.Name,
			Version:    dep.Version,
			License:    license,
			Confidence: dep.Confidence,
			Source:     dep.Source,
		}

		analyzerDeps[i] = analyzer.Dependency{
			Name:       dep.Name,
			Version:    dep.Version,
			License:    license,
			Confidence: dep.Confidence,
		}
	}

	// Perform license analysis
	licenseAnalyzer := analyzer.New()
	analysis := licenseAnalyzer.Analyze(analyzerDeps)

	// Build unique licenses list from analysis
	var uniqueLicensesList []string
	for license := range analysis.LicenseCounts {
		if license != constants.UnknownLicense {
			uniqueLicensesList = append(uniqueLicensesList, license)
		}
	}

	result := ScanResult{
		Dependencies: dependencies,
	}

	result.Summary.TotalDependencies = len(dependencies)
	result.Summary.UniqueLicenses = uniqueLicensesList
	result.Summary.RiskLevel = analysis.RiskLevel
	result.Summary.Conflicts = analysis.Conflicts
	result.Summary.Recommendations = analysis.Recommendations

	// Output based on format
	switch strings.ToLower(*format) {
	case "html":
		result.Timestamp = time.Now().Format("January 2, 2006 at 15:04:05")
		tmpl, err := templates.GetReportTemplate()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating HTML template: %v\n", err)
			os.Exit(1)
		}

		// Create template data with embedded assets
		templateData := templates.GetTemplateData()
		templateData.Summary = result.Summary
		templateData.Dependencies = make([]templates.Dependency, len(result.Dependencies))
		templateData.Timestamp = result.Timestamp

		// Convert dependencies
		for i, dep := range result.Dependencies {
			templateData.Dependencies[i] = templates.Dependency{
				Name:       dep.Name,
				Version:    dep.Version,
				License:    dep.License,
				Confidence: dep.Confidence,
				Source:     dep.Source,
			}
		}

		err = tmpl.Execute(os.Stdout, templateData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error executing HTML template: %v\n", err)
			os.Exit(1)
		}
	case "json":
		fallthrough
	default:
		output, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(string(output))
	}
}
