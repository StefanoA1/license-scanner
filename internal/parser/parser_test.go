package parser

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

func TestDetectLockFile(t *testing.T) {
	tests := []struct {
		name            string
		files           map[string]string
		expectedPath    string
		expectedManager string
		expectedError   bool
	}{
		{
			name: "npm lock file",
			files: map[string]string{
				"/test/package-lock.json": "{}",
			},
			expectedPath:    "/test/package-lock.json",
			expectedManager: "npm",
		},
		{
			name: "yarn lock file",
			files: map[string]string{
				"/test/yarn.lock": "# yarn lockfile",
			},
			expectedPath:    "/test/yarn.lock",
			expectedManager: "yarn",
		},
		{
			name: "pnpm lock file",
			files: map[string]string{
				"/test/pnpm-lock.yaml": "lockfileVersion: 5.4",
			},
			expectedPath:    "/test/pnpm-lock.yaml",
			expectedManager: "pnpm",
		},
		{
			name: "multiple lock files - npm takes precedence",
			files: map[string]string{
				"/test/package-lock.json": "{}",
				"/test/yarn.lock":         "# yarn lockfile",
			},
			expectedPath:    "/test/package-lock.json",
			expectedManager: "npm",
		},
		{
			name:          "no lock files",
			files:         map[string]string{},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewMockFileSystem()
			for path, content := range tt.files {
				fs.AddFile(path, content)
			}

			lockPath, manager, err := DetectLockFile(fs, "/test")

			if tt.expectedError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if lockPath != tt.expectedPath {
				t.Errorf("expected path %q, got %q", tt.expectedPath, lockPath)
			}
			if manager != tt.expectedManager {
				t.Errorf("expected manager %q, got %q", tt.expectedManager, manager)
			}
		})
	}
}

func TestNPMParser_Parse(t *testing.T) {
	tests := []struct {
		name         string
		lockContent  string
		expectedDeps []Dependency
		expectedErr  bool
	}{
		{
			name: "npm v7+ format with packages",
			lockContent: `{
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
					"node_modules/@types/node": {
						"version": "18.0.0",
						"license": "MIT"
					}
				}
			}`,
			expectedDeps: []Dependency{
				{Name: "lodash", Version: "4.17.21", License: "MIT"},
				{Name: "@types/node", Version: "18.0.0", License: "MIT"},
			},
		},
		{
			name: "npm legacy format with dependencies",
			lockContent: `{
				"name": "test-project",
				"version": "1.0.0",
				"packages": {},
				"dependencies": {
					"lodash": {
						"version": "4.17.21"
					},
					"express": {
						"version": "4.18.0",
						"dependencies": {
							"accepts": {
								"version": "1.3.8"
							}
						}
					}
				}
			}`,
			expectedDeps: []Dependency{
				{Name: "lodash", Version: "4.17.21"},
				{Name: "express", Version: "4.18.0"},
				{Name: "accepts", Version: "1.3.8"},
			},
		},
		{
			name:        "invalid JSON",
			lockContent: `{invalid json`,
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewMockFileSystem()
			fs.AddFile("/test/package-lock.json", tt.lockContent)

			parser := NewNPMParserWithFS(fs)
			deps, err := parser.Parse("/test/package-lock.json")

			if tt.expectedErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(deps) != len(tt.expectedDeps) {
				t.Errorf("expected %d dependencies, got %d", len(tt.expectedDeps), len(deps))
				return
			}

			// Convert to map for easier comparison
			depsMap := make(map[string]Dependency)
			for _, dep := range deps {
				depsMap[dep.Name] = dep
			}

			for _, expected := range tt.expectedDeps {
				actual, exists := depsMap[expected.Name]
				if !exists {
					t.Errorf("expected dependency %q not found", expected.Name)
					continue
				}
				if actual.Version != expected.Version {
					t.Errorf("dependency %q: expected version %q, got %q", expected.Name, expected.Version, actual.Version)
				}
				if actual.License != expected.License {
					t.Errorf("dependency %q: expected license %q, got %q", expected.Name, expected.License, actual.License)
				}
			}
		})
	}
}

func TestPnpmParser_Parse(t *testing.T) {
	lockContent := `lockfileVersion: 5.4

dependencies:
  lodash: 4.17.21
  express: 4.18.0

devDependencies:
  typescript: 4.9.0

packages:
  /lodash@4.17.21:
    resolution: {integrity: sha512-v2kDEe57lecTulaDIuNTPy3Ry4gLGJ6Z1O3vE1krgXZNrsQ+LFTGHVxVjcXPs+cA6SoVHLIkD1k6qPy5f8d9cw==}
    dev: false

  /@types/node@18.0.0:
    resolution: {integrity: sha512-cHlGmko4gWLVI27cGJntjs/Sj8th9aYwplmZFwmmgYQQvL5NUsgVJG7OddLvNfLqYS31KFN0s3qlaD9qCaxACA==}
    dev: true

  /express@4.18.0:
    resolution: {integrity: sha512-EJEiyBqAT9RIcpWPzUIakLMZPOTU8C1MH+bugNP7KbFApXTDv/nJDfWQB7HRVXKOylGpfquwUlTDlqRlnCVJng==}
    dev: false
`

	expectedDeps := []Dependency{
		{Name: "lodash", Version: "4.17.21"},
		{Name: "@types/node", Version: "18.0.0"},
		{Name: "express", Version: "4.18.0"},
	}

	fs := NewMockFileSystem()
	fs.AddFile("/test/pnpm-lock.yaml", lockContent)

	parser := NewPnpmParserWithFS(fs)
	deps, err := parser.Parse("/test/pnpm-lock.yaml")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(deps) != len(expectedDeps) {
		t.Errorf("expected %d dependencies, got %d", len(expectedDeps), len(deps))
		return
	}

	// Convert to map for easier comparison
	depsMap := make(map[string]Dependency)
	for _, dep := range deps {
		depsMap[dep.Name] = dep
	}

	for _, expected := range expectedDeps {
		actual, exists := depsMap[expected.Name]
		if !exists {
			t.Errorf("expected dependency %q not found", expected.Name)
			continue
		}
		if actual.Version != expected.Version {
			t.Errorf("dependency %q: expected version %q, got %q", expected.Name, expected.Version, actual.Version)
		}
	}
}

func TestYarnParser_Parse(t *testing.T) {
	lockContent := `# THIS IS AN AUTOGENERATED FILE. DO NOT EDIT THIS FILE DIRECTLY.
# yarn lockfile v1

lodash@^4.17.21:
  version "4.17.21"
  resolved "https://registry.yarnpkg.com/lodash/-/lodash-4.17.21.tgz"
  integrity sha512-v2kDEe57lecTulaDIuNTPy3Ry4gLGJ6Z1O3vE1krgXZNrsQ+LFTGHVxVjcXPs+cA6SoVHLIkD1k6qPy5f8d9cw==

"@types/node@^18.0.0":
  version "18.0.0"
  resolved "https://registry.yarnpkg.com/@types/node/-/node-18.0.0.tgz"
  integrity sha512-cHlGmko4gWLVI27cGJntjs/Sj8th9aYwplmZFwmmgYQQvL5NUsgVJG7OddLvNfLqYS31KFN0s3qlaD9qCaxACA==

express@4.18.0:
  version "4.18.0"
  resolved "https://registry.yarnpkg.com/express/-/express-4.18.0.tgz"
  integrity sha512-EJEiyBqAT9RIcpWPzUIakLMZPOTU8C1MH+bugNP7KbFApXTDv/nJDfWQB7HRVXKOylGpfquwUlTDlqRlnCVJng==
`

	expectedDeps := []Dependency{
		{Name: "lodash", Version: "4.17.21"},
		{Name: "@types/node", Version: "18.0.0"},
		{Name: "express", Version: "4.18.0"},
	}

	fs := NewMockFileSystem()
	fs.AddFile("/test/yarn.lock", lockContent)

	parser := NewYarnParserWithFS(fs)
	deps, err := parser.Parse("/test/yarn.lock")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(deps) != len(expectedDeps) {
		t.Errorf("expected %d dependencies, got %d", len(expectedDeps), len(deps))
		return
	}

	// Convert to map for easier comparison
	depsMap := make(map[string]Dependency)
	for _, dep := range deps {
		depsMap[dep.Name] = dep
	}

	for _, expected := range expectedDeps {
		actual, exists := depsMap[expected.Name]
		if !exists {
			t.Errorf("expected dependency %q not found", expected.Name)
			continue
		}
		if actual.Version != expected.Version {
			t.Errorf("dependency %q: expected version %q, got %q", expected.Name, expected.Version, actual.Version)
		}
	}
}

func TestExtractPackageName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"node_modules/lodash", "lodash"},
		{"node_modules/@types/node", "@types/node"},
		{"node_modules/@babel/core", "@babel/core"},
		{"node_modules/lodash/lib/index.js", "lodash"},
		{"node_modules/@types/node/lib/index.d.ts", "@types/node"},
		{"invalid/path", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractPackageName(tt.input)
			if result != tt.expected {
				t.Errorf("extractPackageName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractPnpmPackageInfo(t *testing.T) {
	tests := []struct {
		input           string
		expectedName    string
		expectedVersion string
	}{
		{"/lodash@4.17.21", "lodash", "4.17.21"},
		{"/@types/node@18.0.0", "@types/node", "18.0.0"},
		{"/@babel/core@7.20.0", "@babel/core", "7.20.0"},
		{"/express@4.18.0", "express", "4.18.0"},
		{"invalid-format", "", ""},
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			name, version := extractPnpmPackageInfo(tt.input)
			if name != tt.expectedName {
				t.Errorf("extractPnpmPackageInfo(%q) name = %q, want %q", tt.input, name, tt.expectedName)
			}
			if version != tt.expectedVersion {
				t.Errorf("extractPnpmPackageInfo(%q) version = %q, want %q", tt.input, version, tt.expectedVersion)
			}
		})
	}
}
