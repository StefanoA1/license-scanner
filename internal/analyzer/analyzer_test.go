package analyzer

import (
	"testing"
)

func TestAnalyze_AllPermissive(t *testing.T) {
	analyzer := New()
	deps := []Dependency{
		{Name: "react", Version: "18.2.0", License: "MIT", Confidence: 1.0},
		{Name: "lodash", Version: "4.17.21", License: "MIT", Confidence: 1.0},
		{Name: "express", Version: "4.18.0", License: "MIT", Confidence: 1.0},
	}

	result := analyzer.Analyze(deps)

	if result.RiskLevel != "low" {
		t.Errorf("Expected risk level 'low', got '%s'", result.RiskLevel)
	}

	if len(result.Conflicts) != 0 {
		t.Errorf("Expected no conflicts, got %d", len(result.Conflicts))
	}

	if len(result.Recommendations) != 1 {
		t.Errorf("Expected 1 recommendation, got %d", len(result.Recommendations))
	}

	if result.Recommendations[0] != "✓ All licenses are permissive and compatible - no compliance issues detected" {
		t.Errorf("Unexpected recommendation: %s", result.Recommendations[0])
	}
}

func TestAnalyze_WithGPL(t *testing.T) {
	analyzer := New()
	deps := []Dependency{
		{Name: "react", Version: "18.2.0", License: "MIT", Confidence: 1.0},
		{Name: "gpl-package", Version: "1.0.0", License: "GPL-3.0", Confidence: 1.0},
	}

	result := analyzer.Analyze(deps)

	if result.RiskLevel != "high" {
		t.Errorf("Expected risk level 'high', got '%s'", result.RiskLevel)
	}

	foundGPLWarning := false
	for _, rec := range result.Recommendations {
		if containsString(rec, "GPL/AGPL dependencies") {
			foundGPLWarning = true
			break
		}
	}

	if !foundGPLWarning {
		t.Errorf("Expected GPL warning in recommendations, got: %v", result.Recommendations)
	}
}

func TestAnalyze_GPLApacheConflict(t *testing.T) {
	analyzer := New()
	deps := []Dependency{
		{Name: "gpl-package", Version: "1.0.0", License: "GPL-2.0", Confidence: 1.0},
		{Name: "apache-package", Version: "1.0.0", License: "Apache-2.0", Confidence: 1.0},
	}

	result := analyzer.Analyze(deps)

	if result.RiskLevel != "high" {
		t.Errorf("Expected risk level 'high', got '%s'", result.RiskLevel)
	}

	if len(result.Conflicts) == 0 {
		t.Error("Expected GPL-2.0 and Apache-2.0 conflict to be detected")
	}

	foundConflict := false
	for _, conflict := range result.Conflicts {
		if containsString(conflict, "GPL-2.0 and Apache-2.0") {
			foundConflict = true
			break
		}
	}

	if !foundConflict {
		t.Errorf("Expected GPL/Apache conflict, got: %v", result.Conflicts)
	}
}

func TestAnalyze_UnknownLicenses(t *testing.T) {
	analyzer := New()
	deps := []Dependency{
		{Name: "react", Version: "18.2.0", License: "MIT", Confidence: 1.0},
		{Name: "unknown1", Version: "1.0.0", License: "Unknown", Confidence: 0.0},
		{Name: "unknown2", Version: "1.0.0", License: "Unknown", Confidence: 0.0},
	}

	result := analyzer.Analyze(deps)

	if result.RiskLevel == "low" {
		t.Errorf("Expected risk level to be elevated, got '%s'", result.RiskLevel)
	}

	foundUnknownWarning := false
	for _, rec := range result.Recommendations {
		if containsString(rec, "unknown licenses") {
			foundUnknownWarning = true
			break
		}
	}

	if !foundUnknownWarning {
		t.Errorf("Expected unknown license warning, got: %v", result.Recommendations)
	}
}

func TestAnalyze_LowConfidence(t *testing.T) {
	analyzer := New()
	deps := []Dependency{
		{Name: "pkg1", Version: "1.0.0", License: "MIT", Confidence: 0.3},
		{Name: "pkg2", Version: "1.0.0", License: "MIT", Confidence: 0.2},
		{Name: "pkg3", Version: "1.0.0", License: "MIT", Confidence: 0.4},
		{Name: "pkg4", Version: "1.0.0", License: "MIT", Confidence: 0.1},
	}

	result := analyzer.Analyze(deps)

	if result.RiskLevel == "low" {
		t.Errorf("Expected elevated risk level due to low confidence, got '%s'", result.RiskLevel)
	}

	foundLowConfidenceWarning := false
	for _, rec := range result.Recommendations {
		if containsString(rec, "low-confidence") {
			foundLowConfidenceWarning = true
			break
		}
	}

	if !foundLowConfidenceWarning {
		t.Errorf("Expected low confidence warning, got: %v", result.Recommendations)
	}
}

func TestAnalyze_WeakCopyleft(t *testing.T) {
	analyzer := New()
	deps := []Dependency{
		{Name: "react", Version: "18.2.0", License: "MIT", Confidence: 1.0},
		{Name: "lgpl-lib", Version: "1.0.0", License: "LGPL-2.1", Confidence: 1.0},
	}

	result := analyzer.Analyze(deps)

	if result.RiskLevel != "medium" {
		t.Errorf("Expected risk level 'medium', got '%s'", result.RiskLevel)
	}

	foundLGPLInfo := false
	for _, rec := range result.Recommendations {
		if containsString(rec, "LGPL/MPL") {
			foundLGPLInfo = true
			break
		}
	}

	if !foundLGPLInfo {
		t.Errorf("Expected LGPL info in recommendations, got: %v", result.Recommendations)
	}
}

func TestAnalyze_AGPL(t *testing.T) {
	analyzer := New()
	deps := []Dependency{
		{Name: "agpl-package", Version: "1.0.0", License: "AGPL-3.0", Confidence: 1.0},
	}

	result := analyzer.Analyze(deps)

	if result.RiskLevel != "high" {
		t.Errorf("Expected risk level 'high', got '%s'", result.RiskLevel)
	}

	foundAGPLConflict := false
	for _, conflict := range result.Conflicts {
		if containsString(conflict, "AGPL-3.0") && containsString(conflict, "network use") {
			foundAGPLConflict = true
			break
		}
	}

	if !foundAGPLConflict {
		t.Errorf("Expected AGPL network use warning, got: %v", result.Conflicts)
	}
}

func TestAnalyze_LicenseCounts(t *testing.T) {
	analyzer := New()
	deps := []Dependency{
		{Name: "pkg1", Version: "1.0.0", License: "MIT", Confidence: 1.0},
		{Name: "pkg2", Version: "1.0.0", License: "MIT", Confidence: 1.0},
		{Name: "pkg3", Version: "1.0.0", License: "Apache-2.0", Confidence: 1.0},
		{Name: "pkg4", Version: "1.0.0", License: "ISC", Confidence: 1.0},
	}

	result := analyzer.Analyze(deps)

	if result.LicenseCounts["MIT"] != 2 {
		t.Errorf("Expected 2 MIT licenses, got %d", result.LicenseCounts["MIT"])
	}

	if result.LicenseCounts["Apache-2.0"] != 1 {
		t.Errorf("Expected 1 Apache-2.0 license, got %d", result.LicenseCounts["Apache-2.0"])
	}

	if result.LicenseCounts["ISC"] != 1 {
		t.Errorf("Expected 1 ISC license, got %d", result.LicenseCounts["ISC"])
	}
}

func TestNormalizeLicense(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Apache-2.0", "Apache-2.0"},
		{"Apache 2.0", "Apache-2.0"},
		{"apache-2.0", "Apache-2.0"},
		{"GPL-3.0", "GPL-3.0"},
		{"GPL-2.0", "GPL-2.0"},
		{"gpl-3.0", "GPL-3.0"},
		{"MIT", "MIT"},
		{"  MIT  ", "MIT"},
	}

	for _, tt := range tests {
		result := normalizeLicense(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeLicense(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestAnalyze_GPL2AndGPL3Conflict(t *testing.T) {
	analyzer := New()
	deps := []Dependency{
		{Name: "gpl2-package", Version: "1.0.0", License: "GPL-2.0", Confidence: 1.0},
		{Name: "gpl3-package", Version: "1.0.0", License: "GPL-3.0", Confidence: 1.0},
	}

	result := analyzer.Analyze(deps)

	foundGPLVersionConflict := false
	for _, conflict := range result.Conflicts {
		if containsString(conflict, "GPL-2.0 and GPL-3.0") {
			foundGPLVersionConflict = true
			break
		}
	}

	if !foundGPLVersionConflict {
		t.Errorf("Expected GPL version conflict warning, got: %v", result.Conflicts)
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
