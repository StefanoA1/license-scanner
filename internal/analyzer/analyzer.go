package analyzer

import (
	"fmt"
	"strings"
)

// LicenseCategory represents the type of license
type LicenseCategory int

const (
	Permissive LicenseCategory = iota
	WeakCopyleft
	StrongCopyleft
	Proprietary
	Unknown
)

// LicenseInfo contains metadata about a license type
type LicenseInfo struct {
	Name      string
	Category  LicenseCategory
	RiskLevel string
}

// KnownLicenses maps license identifiers to their metadata
var KnownLicenses = map[string]LicenseInfo{
	"MIT":          {Name: "MIT", Category: Permissive, RiskLevel: "low"},
	"ISC":          {Name: "ISC", Category: Permissive, RiskLevel: "low"},
	"BSD-2-Clause": {Name: "BSD-2-Clause", Category: Permissive, RiskLevel: "low"},
	"BSD-3-Clause": {Name: "BSD-3-Clause", Category: Permissive, RiskLevel: "low"},
	"Apache-2.0":   {Name: "Apache-2.0", Category: Permissive, RiskLevel: "low"},
	"Apache 2.0":   {Name: "Apache-2.0", Category: Permissive, RiskLevel: "low"},
	"MPL-2.0":      {Name: "MPL-2.0", Category: WeakCopyleft, RiskLevel: "medium"},
	"LGPL-2.1":     {Name: "LGPL-2.1", Category: WeakCopyleft, RiskLevel: "medium"},
	"LGPL-3.0":     {Name: "LGPL-3.0", Category: WeakCopyleft, RiskLevel: "medium"},
	"GPL-2.0":      {Name: "GPL-2.0", Category: StrongCopyleft, RiskLevel: "high"},
	"GPL-3.0":      {Name: "GPL-3.0", Category: StrongCopyleft, RiskLevel: "high"},
	"AGPL-3.0":     {Name: "AGPL-3.0", Category: StrongCopyleft, RiskLevel: "high"},
	"UNLICENSED":   {Name: "UNLICENSED", Category: Proprietary, RiskLevel: "high"},
}

// AnalysisResult contains the results of license analysis
type AnalysisResult struct {
	RiskLevel       string
	Conflicts       []string
	Recommendations []string
	LicenseCounts   map[string]int
}

// Dependency represents a dependency with license information
type Dependency struct {
	Name       string
	Version    string
	License    string
	Confidence float64
}

// Analyzer performs license compatibility and risk analysis
type Analyzer struct{}

// New creates a new Analyzer
func New() *Analyzer {
	return &Analyzer{}
}

// Analyze performs comprehensive license analysis
func (a *Analyzer) Analyze(dependencies []Dependency) *AnalysisResult {
	result := &AnalysisResult{
		Conflicts:       []string{},
		Recommendations: []string{},
		LicenseCounts:   make(map[string]int),
	}

	// Count licenses by category
	permissiveCount := 0
	weakCopyleftCount := 0
	strongCopyleftCount := 0
	unknownCount := 0
	lowConfidenceCount := 0
	hasLGPL := false
	hasMPL := false

	for _, dep := range dependencies {
		license := normalizeLicense(dep.License)
		result.LicenseCounts[license]++

		info, known := KnownLicenses[license]
		if !known {
			if license != "Unknown" {
				unknownCount++
			}
			continue
		}

		// Track low confidence detections
		if dep.Confidence < 0.5 {
			lowConfidenceCount++
		}

		switch info.Category {
		case Permissive:
			permissiveCount++
		case WeakCopyleft:
			weakCopyleftCount++
			if license == "LGPL-2.1" || license == "LGPL-3.0" {
				hasLGPL = true
			}
			if license == "MPL-2.0" {
				hasMPL = true
			}
		case StrongCopyleft:
			strongCopyleftCount++
		}
	}

	// Calculate unknown count from license counts
	if count, exists := result.LicenseCounts["Unknown"]; exists {
		unknownCount = count
	}

	// Determine overall risk level
	result.RiskLevel = a.calculateRiskLevel(strongCopyleftCount, weakCopyleftCount, unknownCount, lowConfidenceCount)

	// Check for GPL conflicts
	result.Conflicts = a.detectConflicts(result.LicenseCounts)

	// Generate recommendations
	result.Recommendations = a.generateRecommendations(
		permissiveCount,
		weakCopyleftCount,
		strongCopyleftCount,
		unknownCount,
		lowConfidenceCount,
		len(result.Conflicts) > 0,
		hasLGPL,
		hasMPL,
	)

	return result
}

// calculateRiskLevel determines the overall risk based on license types
func (a *Analyzer) calculateRiskLevel(strongCopyleft, weakCopyleft, unknown, lowConfidence int) string {
	if strongCopyleft > 0 || unknown > 5 {
		return "high"
	}
	if weakCopyleft > 0 || unknown > 0 || lowConfidence > 3 {
		return "medium"
	}
	return "low"
}

// detectConflicts identifies incompatible license combinations
func (a *Analyzer) detectConflicts(licenseCounts map[string]int) []string {
	conflicts := []string{}

	hasGPL2 := licenseCounts["GPL-2.0"] > 0
	hasGPL3 := licenseCounts["GPL-3.0"] > 0
	hasAGPL := licenseCounts["AGPL-3.0"] > 0
	hasApache := licenseCounts["Apache-2.0"] > 0 || licenseCounts["Apache 2.0"] > 0

	// AGPL is the most restrictive - report first
	if hasAGPL {
		conflicts = append(conflicts, "AGPL-3.0 requires source disclosure for network use - ensure compliance")
	}

	// GPL-2.0 and Apache-2.0 are incompatible
	if hasGPL2 && hasApache {
		conflicts = append(conflicts, "GPL-2.0 and Apache-2.0 licenses are incompatible")
	}

	// GPL-3.0 with GPL-2.0 (without "or later" clause) can be problematic
	if hasGPL2 && hasGPL3 {
		conflicts = append(conflicts, "GPL-2.0 and GPL-3.0 detected - verify 'or later' clauses for compatibility")
	}

	return conflicts
}

// generateRecommendations creates actionable guidance based on analysis
func (a *Analyzer) generateRecommendations(
	permissive, weakCopyleft, strongCopyleft, unknown, lowConfidence int,
	hasConflicts, hasLGPL, hasMPL bool,
) []string {
	recommendations := []string{}

	// Conflict-based recommendations
	if hasConflicts {
		recommendations = append(recommendations, "⚠️  License conflicts detected - review dependencies for compatibility issues")
	}

	// Strong copyleft recommendations
	if strongCopyleft > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("⚠️  Found %d GPL/AGPL dependencies - ensure compliance with copyleft requirements", strongCopyleft))
		recommendations = append(recommendations, "📋 Consider legal review if distributing proprietary software")
	}

	// Weak copyleft recommendations
	if weakCopyleft > 0 && (hasLGPL || hasMPL) {
		recommendations = append(recommendations,
			fmt.Sprintf("ℹ️  Found %d LGPL/MPL dependencies - these allow proprietary use with conditions", weakCopyleft))
	}

	// Unknown license recommendations
	if unknown > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("⚠️  %d dependencies have unknown licenses - manual review required", unknown))
		recommendations = append(recommendations, "🔍 Check package repositories or contact maintainers for license clarification")
	}

	// Low confidence recommendations
	if lowConfidence > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("⚠️  %d dependencies have low-confidence license detection - verify manually", lowConfidence))
	}

	// All clear
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "✓ All licenses are permissive and compatible - no compliance issues detected")
	}

	return recommendations
}

// normalizeLicense normalizes license strings for consistent comparison
func normalizeLicense(license string) string {
	normalized := strings.TrimSpace(license)

	// Handle common variations
	lower := strings.ToLower(normalized)
	if strings.Contains(lower, "apache") {
		return "Apache-2.0"
	}
	// Check AGPL before LGPL/GPL (AGPL contains "gpl")
	if strings.Contains(lower, "agpl") {
		return "AGPL-3.0"
	}
	// Check LGPL before GPL (LGPL contains "gpl")
	if strings.Contains(lower, "lgpl") && strings.Contains(lower, "3") {
		return "LGPL-3.0"
	}
	if strings.Contains(lower, "lgpl") && strings.Contains(lower, "2") {
		return "LGPL-2.1"
	}
	// Now check GPL patterns
	if strings.Contains(lower, "gpl") && strings.Contains(lower, "3") {
		return "GPL-3.0"
	}
	if strings.Contains(lower, "gpl") && strings.Contains(lower, "2") {
		return "GPL-2.0"
	}

	return normalized
}
