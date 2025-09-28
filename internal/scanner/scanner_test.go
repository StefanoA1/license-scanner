package scanner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stefano/license-scanner/internal/detector"
)

// MockFileSystem implements detector.FileSystem for testing
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
	// Normalize path to use filepath.Clean for cross-platform compatibility
	normalizedPath := filepath.Clean(path)
	fs.files[normalizedPath] = content
}

func (fs *MockFileSystem) AddDir(path string) {
	// Normalize path to use filepath.Clean for cross-platform compatibility
	normalizedPath := filepath.Clean(path)
	fs.dirs[normalizedPath] = true
}

func (fs *MockFileSystem) Open(path string) (io.ReadCloser, error) {
	// Normalize path to handle cross-platform path separators
	normalizedPath := filepath.Clean(path)
	content, exists := fs.files[normalizedPath]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

func (fs *MockFileSystem) Stat(path string) (os.FileInfo, error) {
	// Normalize path to handle cross-platform path separators
	normalizedPath := filepath.Clean(path)
	if _, exists := fs.files[normalizedPath]; exists {
		return &mockFileInfo{name: normalizedPath, isDir: false}, nil
	}
	if _, exists := fs.dirs[normalizedPath]; exists {
		return &mockFileInfo{name: normalizedPath, isDir: true}, nil
	}
	return nil, os.ErrNotExist
}

// Join implements the FileSystem interface for cross-platform path joining
func (fs *MockFileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
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

func TestScanner_Scan_NPM(t *testing.T) {
	fs := NewMockFileSystem()

	// Add package-lock.json
	lockContent := `{
		"name": "test-project",
		"version": "1.0.0",
		"packages": {
			"": {
				"name": "test-project",
				"version": "1.0.0"
			},
			"node_modules/lodash": {
				"version": "4.17.21",
				"license": "MIT"
			},
			"node_modules/express": {
				"version": "4.18.0"
			}
		}
	}`
	// Use cross-platform test root path
	testRoot := filepath.Join("test")

	fs.AddFile(filepath.Join(testRoot, "package-lock.json"), lockContent)

	// Add package.json files for dependencies
	fs.AddFile(filepath.Join(testRoot, "node_modules", "lodash", "package.json"), `{"license": "MIT"}`)
	fs.AddFile(filepath.Join(testRoot, "node_modules", "express", "package.json"), `{"license": "MIT"}`)

	// Create mock detector with file system
	mockDetector := detector.NewWithFileSystem(fs)

	// Create scanner with mock detector and file system
	scanner := NewWithDependencies(testRoot, mockDetector, fs)

	result, err := scanner.Scan()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(result.Dependencies) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(result.Dependencies))
		return
	}

	// Check that we have both dependencies
	depNames := make(map[string]bool)
	for _, dep := range result.Dependencies {
		depNames[dep.Name] = true

		// Both should have MIT license
		if dep.License != "MIT" {
			t.Errorf("dependency %s: expected license MIT, got %s", dep.Name, dep.License)
		}
	}

	if !depNames["lodash"] {
		t.Error("expected lodash dependency")
	}
	if !depNames["express"] {
		t.Error("expected express dependency")
	}
}

func TestScanner_Scan_Yarn(t *testing.T) {
	fs := NewMockFileSystem()

	// Add yarn.lock
	lockContent := `# yarn lockfile v1

lodash@^4.17.21:
  version "4.17.21"
  resolved "https://registry.yarnpkg.com/lodash/-/lodash-4.17.21.tgz"

express@4.18.0:
  version "4.18.0"
  resolved "https://registry.yarnpkg.com/express/-/express-4.18.0.tgz"
`
	// Use cross-platform test root path
	testRoot := filepath.Join("test")

	fs.AddFile(filepath.Join(testRoot, "yarn.lock"), lockContent)

	// Add LICENSE files for dependencies
	fs.AddFile(filepath.Join(testRoot, "node_modules", "lodash", "LICENSE"), "MIT License\n\nPermission is hereby granted, free of charge")
	fs.AddFile(filepath.Join(testRoot, "node_modules", "express", "LICENSE"), "MIT License\n\nPermission is hereby granted, free of charge")

	// Create mock detector with file system
	mockDetector := detector.NewWithFileSystem(fs)

	// Create scanner with mock detector and file system
	scanner := NewWithDependencies(testRoot, mockDetector, fs)

	result, err := scanner.Scan()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(result.Dependencies) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(result.Dependencies))
		return
	}

	// Check that both dependencies have MIT license from LICENSE files
	for _, dep := range result.Dependencies {
		if dep.License != "MIT" {
			t.Errorf("dependency %s: expected license MIT, got %s", dep.Name, dep.License)
		}
		if dep.Source != "LICENSE file" {
			t.Errorf("dependency %s: expected source 'LICENSE file', got %s", dep.Name, dep.Source)
		}
		if dep.Confidence != 0.9 {
			t.Errorf("dependency %s: expected confidence 0.9, got %f", dep.Name, dep.Confidence)
		}
	}
}

func TestScanner_Scan_Pnpm(t *testing.T) {
	fs := NewMockFileSystem()

	// Add pnpm-lock.yaml
	lockContent := `lockfileVersion: 5.4

dependencies:
  lodash: 4.17.21

packages:
  /lodash@4.17.21:
    resolution: {integrity: sha512-v2kDEe57lecTulaDIuNTPy3Ry4gLGJ6Z1O3vE1krgXZNrsQ+LFTGHVxVjcXPs+cA6SoVHLIkD1k6qPy5f8d9cw==}
    dev: false
`
	// Use cross-platform test root path
	testRoot := filepath.Join("test")

	fs.AddFile(filepath.Join(testRoot, "pnpm-lock.yaml"), lockContent)

	// Add package.json for dependency in pnpm structure
	fs.AddFile(filepath.Join(testRoot, "node_modules", ".pnpm", "lodash@4.17.21", "node_modules", "lodash", "package.json"), `{"license": "MIT"}`)

	// Also add fallback in standard location for hoisted packages
	fs.AddFile(filepath.Join(testRoot, "node_modules", "lodash", "package.json"), `{"license": "MIT"}`)

	// Create mock detector with file system
	mockDetector := detector.NewWithFileSystem(fs)

	// Create scanner with mock detector and file system
	scanner := NewWithDependencies(testRoot, mockDetector, fs)

	result, err := scanner.Scan()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(result.Dependencies) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(result.Dependencies))
		return
	}

	dep := result.Dependencies[0]
	if dep.Name != "lodash" {
		t.Errorf("expected dependency name 'lodash', got %s", dep.Name)
	}
	if dep.Version != "4.17.21" {
		t.Errorf("expected version '4.17.21', got %s", dep.Version)
	}
	if dep.License != "MIT" {
		t.Errorf("expected license 'MIT', got %s", dep.License)
	}
}

func TestScanner_Scan_NoLockFile(t *testing.T) {
	fs := NewMockFileSystem()
	// Don't add any lock files

	mockDetector := detector.NewWithFileSystem(fs)
	scanner := NewWithDependencies("/test", mockDetector, fs)

	_, err := scanner.Scan()
	if err == nil {
		t.Error("expected error when no lock file found")
		return
	}

	if !strings.Contains(err.Error(), "no lock file found") {
		t.Errorf("expected 'no lock file found' error, got: %v", err)
	}
}

func TestScanner_Scan_LicenseDetectionFallback(t *testing.T) {
	fs := NewMockFileSystem()

	// Add package-lock.json with a dependency
	lockContent := `{
		"name": "test-project",
		"version": "1.0.0",
		"packages": {
			"": {
				"name": "test-project",
				"version": "1.0.0"
			},
			"node_modules/some-package": {
				"version": "1.0.0"
			}
		}
	}`
	// Use cross-platform test root path
	testRoot := filepath.Join("test")

	fs.AddFile(filepath.Join(testRoot, "package-lock.json"), lockContent)

	// Don't add any license information for the dependency
	// This should trigger the fallback to "Unknown" license

	mockDetector := detector.NewWithFileSystem(fs)
	scanner := NewWithDependencies(testRoot, mockDetector, fs)

	result, err := scanner.Scan()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(result.Dependencies) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(result.Dependencies))
		return
	}

	dep := result.Dependencies[0]
	if dep.Name != "some-package" {
		t.Errorf("expected dependency name 'some-package', got %s", dep.Name)
	}
	if dep.License != "Unknown" {
		t.Errorf("expected license 'Unknown', got %s", dep.License)
	}
	if dep.Confidence != 0.0 {
		t.Errorf("expected confidence 0.0, got %f", dep.Confidence)
	}
	if dep.Source != "not found" {
		t.Errorf("expected source 'not found', got %s", dep.Source)
	}
}

func TestScanner_Scan_MixedLicenseSources(t *testing.T) {
	fs := NewMockFileSystem()

	// Add package-lock.json with multiple dependencies
	lockContent := `{
		"name": "test-project",
		"version": "1.0.0",
		"packages": {
			"": {
				"name": "test-project",
				"version": "1.0.0"
			},
			"node_modules/package-json-license": {
				"version": "1.0.0"
			},
			"node_modules/license-file-license": {
				"version": "1.0.0"
			},
			"node_modules/no-license": {
				"version": "1.0.0"
			}
		}
	}`
	// Use cross-platform test root path
	testRoot := filepath.Join("test")

	fs.AddFile(filepath.Join(testRoot, "package-lock.json"), lockContent)

	// Add license via package.json for first dependency
	fs.AddFile(filepath.Join(testRoot, "node_modules", "package-json-license", "package.json"), `{"license": "Apache-2.0"}`)

	// Add license via LICENSE file for second dependency
	fs.AddFile(filepath.Join(testRoot, "node_modules", "license-file-license", "LICENSE"), "Apache License\nVersion 2.0, January 2004\n\nLicensed under the Apache License, Version 2.0")

	// No license information for third dependency

	mockDetector := detector.NewWithFileSystem(fs)
	scanner := NewWithDependencies(testRoot, mockDetector, fs)

	result, err := scanner.Scan()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(result.Dependencies) != 3 {
		t.Errorf("expected 3 dependencies, got %d", len(result.Dependencies))
		return
	}

	// Convert to map for easier checking
	depMap := make(map[string]EnrichedDependency)
	for _, dep := range result.Dependencies {
		depMap[dep.Name] = dep
	}

	// Check package.json source
	if dep, exists := depMap["package-json-license"]; exists {
		if dep.License != "Apache-2.0" {
			t.Errorf("package-json-license: expected license 'Apache-2.0', got %s", dep.License)
		}
		if dep.Source != "package.json" {
			t.Errorf("package-json-license: expected source 'package.json', got %s", dep.Source)
		}
		if dep.Confidence != 1.0 {
			t.Errorf("package-json-license: expected confidence 1.0, got %f", dep.Confidence)
		}
	} else {
		t.Error("package-json-license dependency not found")
	}

	// Check LICENSE file source
	if dep, exists := depMap["license-file-license"]; exists {
		if dep.License != "Apache-2.0" {
			t.Errorf("license-file-license: expected license 'Apache-2.0', got %s", dep.License)
		}
		if dep.Source != "LICENSE file" {
			t.Errorf("license-file-license: expected source 'LICENSE file', got %s", dep.Source)
		}
		if dep.Confidence != 0.9 {
			t.Errorf("license-file-license: expected confidence 0.9, got %f", dep.Confidence)
		}
	} else {
		t.Error("license-file-license dependency not found")
	}

	// Check no license fallback
	if dep, exists := depMap["no-license"]; exists {
		if dep.License != "Unknown" {
			t.Errorf("no-license: expected license 'Unknown', got %s", dep.License)
		}
		if dep.Source != "not found" {
			t.Errorf("no-license: expected source 'not found', got %s", dep.Source)
		}
		if dep.Confidence != 0.0 {
			t.Errorf("no-license: expected confidence 0.0, got %f", dep.Confidence)
		}
	} else {
		t.Error("no-license dependency not found")
	}
}
