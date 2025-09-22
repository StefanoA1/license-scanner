package detector

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// MockFileSystem implements FileSystem for testing
type MockFileSystem struct {
	files map[string]string
	dirs  map[string]bool
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files: make(map[string]string),
		dirs:  make(map[string]bool),
	}
}

func (fs *MockFileSystem) AddFile(path, content string) {
	fs.files[path] = content
}

func (fs *MockFileSystem) AddDir(path string) {
	fs.dirs[path] = true
}

func (fs *MockFileSystem) Open(path string) (io.ReadCloser, error) {
	content, exists := fs.files[path]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

func (fs *MockFileSystem) Stat(path string) (os.FileInfo, error) {
	if _, exists := fs.files[path]; exists {
		return &mockFileInfo{name: path, isDir: false}, nil
	}
	if _, exists := fs.dirs[path]; exists {
		return &mockFileInfo{name: path, isDir: true}, nil
	}
	return nil, os.ErrNotExist
}

func (fs *MockFileSystem) Join(elem ...string) string {
	return strings.Join(elem, "/")
}

type mockFileInfo struct {
	name  string
	isDir bool
}

func (fi *mockFileInfo) Name() string       { return fi.name }
func (fi *mockFileInfo) Size() int64        { return 0 }
func (fi *mockFileInfo) Mode() os.FileMode  { return 0 }
func (fi *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (fi *mockFileInfo) IsDir() bool        { return fi.isDir }
func (fi *mockFileInfo) Sys() interface{}   { return nil }

func TestDetector_DetectLicense_FromPackageJSON(t *testing.T) {
	tests := []struct {
		name          string
		packageJSON   string
		expectedInfo  *LicenseInfo
		expectedError bool
	}{
		{
			name:        "MIT license string",
			packageJSON: `{"license": "MIT"}`,
			expectedInfo: &LicenseInfo{
				License:    "MIT",
				Confidence: 1.0,
				Source:     "package.json",
			},
		},
		{
			name:        "Apache license object",
			packageJSON: `{"license": {"type": "Apache-2.0"}}`,
			expectedInfo: &LicenseInfo{
				License:    "Apache-2.0",
				Confidence: 1.0,
				Source:     "package.json",
			},
		},
		{
			name:        "License array",
			packageJSON: `{"license": ["MIT", "Apache-2.0"]}`,
			expectedInfo: &LicenseInfo{
				License:    "MIT",
				Confidence: 1.0,
				Source:     "package.json",
			},
		},
		{
			name:        "No license field",
			packageJSON: `{"name": "test-package"}`,
			expectedInfo: &LicenseInfo{
				License:    "Unknown",
				Confidence: 0.0,
				Source:     "not found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewMockFileSystem()
			fs.AddFile("/test/package/package.json", tt.packageJSON)

			detector := NewWithFileSystem(fs)
			result, err := detector.DetectLicense("/test/package")

			if tt.expectedError && err == nil {
				t.Errorf("expected error but got none")
				return
			}
			if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.License != tt.expectedInfo.License {
				t.Errorf("expected license %q, got %q", tt.expectedInfo.License, result.License)
			}
			if result.Confidence != tt.expectedInfo.Confidence {
				t.Errorf("expected confidence %f, got %f", tt.expectedInfo.Confidence, result.Confidence)
			}
			if result.Source != tt.expectedInfo.Source {
				t.Errorf("expected source %q, got %q", tt.expectedInfo.Source, result.Source)
			}
		})
	}
}

func TestDetector_DetectLicense_FromLicenseFile(t *testing.T) {
	tests := []struct {
		name            string
		licenseContent  string
		filename        string
		expectedLicense string
		expectedConf    float64
	}{
		{
			name:            "MIT license file",
			filename:        "LICENSE",
			licenseContent:  "MIT License\n\nPermission is hereby granted, free of charge, to any person obtaining a copy",
			expectedLicense: "MIT",
			expectedConf:    0.9,
		},
		{
			name:            "Apache license file",
			filename:        "LICENSE.txt",
			licenseContent:  "Apache License\nVersion 2.0, January 2004\n\nLicensed under the Apache License",
			expectedLicense: "Apache-2.0",
			expectedConf:    0.9,
		},
		{
			name:            "GPL-3.0 license file",
			filename:        "LICENSE.md",
			licenseContent:  "GNU GENERAL PUBLIC LICENSE\nVersion 3, 29 June 2007",
			expectedLicense: "GPL-3.0",
			expectedConf:    0.9,
		},
		{
			name:            "Unknown license content",
			filename:        "LICENSE",
			licenseContent:  "Some custom license text that doesn't match patterns",
			expectedLicense: "Unknown",
			expectedConf:    0.2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewMockFileSystem()
			licensePath := "/test/package/" + tt.filename
			fs.AddFile(licensePath, tt.licenseContent)

			detector := NewWithFileSystem(fs)
			result, err := detector.DetectLicense("/test/package")

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.License != tt.expectedLicense {
				t.Errorf("expected license %q, got %q", tt.expectedLicense, result.License)
			}
			if result.Confidence != tt.expectedConf {
				t.Errorf("expected confidence %f, got %f", tt.expectedConf, result.Confidence)
			}
			if result.Source != "LICENSE file" {
				t.Errorf("expected source %q, got %q", "LICENSE file", result.Source)
			}
		})
	}
}

func TestDetector_DetectLicense_PackageJSONOverridesLicenseFile(t *testing.T) {
	fs := NewMockFileSystem()
	fs.AddFile("/test/package/package.json", `{"license": "MIT"}`)
	fs.AddFile("/test/package/LICENSE", "Apache License\nVersion 2.0")

	detector := NewWithFileSystem(fs)
	result, err := detector.DetectLicense("/test/package")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// package.json should take precedence
	if result.License != "MIT" {
		t.Errorf("expected license %q, got %q", "MIT", result.License)
	}
	if result.Source != "package.json" {
		t.Errorf("expected source %q, got %q", "package.json", result.Source)
	}
}

func TestNormalizedLicense(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"MIT", "MIT"},
		{"mit", "MIT"},
		{"Apache-2.0", "Apache-2.0"},
		{"apache2", "Apache-2.0"},
		{"GPL-3.0", "GPL-3.0"},
		{"gplv3", "GPL-3.0"},
		{"BSD-3-Clause", "BSD-3-Clause"},
		{"bsd3", "BSD-3-Clause"},
		{"ISC", "ISC"},
		{"isc", "ISC"},
		{"Custom License", "Custom-License"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizedLicense(tt.input)
			if result != tt.expected {
				t.Errorf("normalizedLicense(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractLicenseFromField(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "string license",
			input:    "MIT",
			expected: "MIT",
		},
		{
			name:     "license object with type",
			input:    map[string]interface{}{"type": "Apache-2.0"},
			expected: "Apache-2.0",
		},
		{
			name:     "license array",
			input:    []interface{}{"MIT", "Apache-2.0"},
			expected: "MIT",
		},
		{
			name:     "empty array",
			input:    []interface{}{},
			expected: "",
		},
		{
			name:     "nil input",
			input:    nil,
			expected: "",
		},
		{
			name:     "invalid object",
			input:    map[string]interface{}{"name": "test"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractLicenseFromField(tt.input)
			if result != tt.expected {
				t.Errorf("extractLicenseFromField(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
