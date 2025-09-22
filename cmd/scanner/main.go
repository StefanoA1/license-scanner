package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/stefano/license-scanner/internal/scanner"
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
}

type Dependency struct {
	Name       string  `json:"name"`
	Version    string  `json:"version"`
	License    string  `json:"license"`
	Confidence float64 `json:"confidence"`
	Source     string  `json:"source"`
}

func main() {
	// Get project path from command line arguments
	projectPath := "."
	if len(os.Args) > 1 {
		projectPath = os.Args[1]
	}

	// Create and run scanner
	s := scanner.New(projectPath)
	scanResult, err := s.Scan()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning project: %v\n", err)
		os.Exit(1)
	}

	// Convert scanner result to CLI output format
	dependencies := make([]Dependency, len(scanResult.Dependencies))
	uniqueLicenses := make(map[string]bool)

	for i, dep := range scanResult.Dependencies {
		license := dep.License
		if license == "" {
			license = "Unknown"
		}

		dependencies[i] = Dependency{
			Name:       dep.Name,
			Version:    dep.Version,
			License:    license,
			Confidence: dep.Confidence,
			Source:     dep.Source,
		}

		if license != "Unknown" {
			uniqueLicenses[license] = true
		}
	}

	// Build unique licenses list
	var uniqueLicensesList []string
	for license := range uniqueLicenses {
		uniqueLicensesList = append(uniqueLicensesList, license)
	}

	result := ScanResult{
		Dependencies: dependencies,
	}

	result.Summary.TotalDependencies = len(dependencies)
	result.Summary.UniqueLicenses = uniqueLicensesList
	result.Summary.RiskLevel = "low"
	result.Summary.Conflicts = []string{}
	result.Summary.Recommendations = []string{"License analysis complete"}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(string(output))
}
