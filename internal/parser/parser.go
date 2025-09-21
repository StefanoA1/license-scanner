package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Dependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	License string `json:"license,omitempty"`
}

type FileSystem interface {
	Open(path string) (io.ReadCloser, error)
	Stat(path string) (os.FileInfo, error)
	Join(elem ...string) string
}

type RealFileSystem struct{}

func (fs *RealFileSystem) Open(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

func (fs *RealFileSystem) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (fs *RealFileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

type LockFileParser interface {
	Parse(lockFilePath string) ([]Dependency, error)
}

func DetectLockFile(fs FileSystem, rootPath string) (string, string, error) {
	lockFiles := map[string]string{
		"package-lock.json": "npm",
		"yarn.lock":         "yarn",
		"pnpm-lock.yaml":    "pnpm",
	}

	for filename, packageManager := range lockFiles {
		lockFilePath := fs.Join(rootPath, filename)
		if _, err := fs.Stat(lockFilePath); err == nil {
			return lockFilePath, packageManager, nil
		}
	}

	return "", "", os.ErrNotExist
}

func DetectLockFileDefault(rootPath string) (string, string, error) {
	return DetectLockFile(&RealFileSystem{}, rootPath)
}

// NPMParser implements parsing for package-lock.json files
type NPMParser struct {
	fs FileSystem
}

func NewNPMParser() *NPMParser {
	return &NPMParser{fs: &RealFileSystem{}}
}

func NewNPMParserWithFS(fs FileSystem) *NPMParser {
	return &NPMParser{fs: fs}
}

func (p *NPMParser) Parse(lockFilePath string) ([]Dependency, error) {
	file, err := p.fs.Open(lockFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open package-lock.json: %w", err)
	}
	defer func() {
		_ = file.Close() // Ignore close error as we already read the file
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read package-lock.json: %w", err)
	}

	var lockFile NPMLockFile
	if err := json.Unmarshal(data, &lockFile); err != nil {
		return nil, fmt.Errorf("failed to parse package-lock.json: %w", err)
	}

	var dependencies []Dependency

	// Parse dependencies from the packages section (npm v2+ format)
	for packagePath, pkg := range lockFile.Packages {
		// Skip the root package (empty path)
		if packagePath == "" {
			continue
		}

		// Extract package name from path (remove node_modules/ prefix)
		name := extractPackageName(packagePath)
		if name == "" {
			continue
		}

		dependencies = append(dependencies, Dependency{
			Name:    name,
			Version: pkg.Version,
			License: pkg.License,
		})
	}

	// Fallback to legacy dependencies format if packages section is empty
	if len(dependencies) == 0 && lockFile.Dependencies != nil {
		dependencies = parseLegacyDependencies(lockFile.Dependencies)
	}

	return dependencies, nil
}

// NPMLockFile represents the structure of package-lock.json
type NPMLockFile struct {
	Name         string                   `json:"name"`
	Version      string                   `json:"version"`
	Packages     map[string]NPMPackage    `json:"packages"`
	Dependencies map[string]NPMDependency `json:"dependencies"` // Legacy format
}

type NPMPackage struct {
	Version string `json:"version"`
	License string `json:"license"`
}

type NPMDependency struct {
	Version      string                   `json:"version"`
	Dependencies map[string]NPMDependency `json:"dependencies"`
}

func extractPackageName(packagePath string) string {
	// Remove "node_modules/" prefix and get the package name
	if !strings.HasPrefix(packagePath, "node_modules/") {
		return ""
	}

	name := packagePath[len("node_modules/"):]

	// Handle scoped packages (@scope/package)
	if len(name) > 0 && name[0] == '@' {
		parts := strings.Split(name, "/")
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	}

	// Handle regular packages - just take the first directory after node_modules/
	parts := strings.Split(name, "/")
	if len(parts) > 0 {
		return parts[0]
	}

	return name
}

func parseLegacyDependencies(deps map[string]NPMDependency) []Dependency {
	var dependencies []Dependency

	for name, dep := range deps {
		dependencies = append(dependencies, Dependency{
			Name:    name,
			Version: dep.Version,
		})

		// Recursively parse nested dependencies
		if dep.Dependencies != nil {
			nested := parseLegacyDependencies(dep.Dependencies)
			dependencies = append(dependencies, nested...)
		}
	}

	return dependencies
}

// PnpmParser implements parsing for pnpm-lock.yaml files
type PnpmParser struct {
	fs FileSystem
}

func NewPnpmParser() *PnpmParser {
	return &PnpmParser{fs: &RealFileSystem{}}
}

func NewPnpmParserWithFS(fs FileSystem) *PnpmParser {
	return &PnpmParser{fs: fs}
}

func (p *PnpmParser) Parse(lockFilePath string) ([]Dependency, error) {
	file, err := p.fs.Open(lockFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open pnpm-lock.yaml: %w", err)
	}
	defer func() {
		_ = file.Close() // Ignore close error as we already read the file
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read pnpm-lock.yaml: %w", err)
	}

	var lockFile PnpmLockFile
	if err := yaml.Unmarshal(data, &lockFile); err != nil {
		return nil, fmt.Errorf("failed to parse pnpm-lock.yaml: %w", err)
	}

	var dependencies []Dependency

	// Parse packages from the packages section
	for packageKey := range lockFile.Packages {
		name, version := extractPnpmPackageInfo(packageKey)
		if name == "" {
			continue
		}

		dependencies = append(dependencies, Dependency{
			Name:    name,
			Version: version,
			License: "", // License info not typically in pnpm lock file
		})
	}

	return dependencies, nil
}

// PnpmLockFile represents the structure of pnpm-lock.yaml
type PnpmLockFile struct {
	LockfileVersion string                 `yaml:"lockfileVersion"`
	Dependencies    map[string]string      `yaml:"dependencies"`
	DevDependencies map[string]string      `yaml:"devDependencies"`
	Packages        map[string]PnpmPackage `yaml:"packages"`
}

type PnpmPackage struct {
	Resolution   PnpmResolution    `yaml:"resolution"`
	Dependencies map[string]string `yaml:"dependencies"`
	Dev          bool              `yaml:"dev"`
}

type PnpmResolution struct {
	Integrity string `yaml:"integrity"`
	Tarball   string `yaml:"tarball"`
}

func extractPnpmPackageInfo(packageKey string) (name, version string) {
	// pnpm package keys are in format like "/package-name@1.0.0" or "/@scope/package@1.0.0"
	// Remove leading slash if present
	key := strings.TrimPrefix(packageKey, "/")

	// Handle scoped packages first
	if strings.HasPrefix(key, "@") {
		re := regexp.MustCompile(`^(@[^/]+/[^@]+)@(.+)$`)
		matches := re.FindStringSubmatch(key)
		if len(matches) == 3 {
			return matches[1], matches[2]
		}
	}

	// Handle regular packages
	re := regexp.MustCompile(`^([^@]+)@(.+)$`)
	matches := re.FindStringSubmatch(key)
	if len(matches) == 3 {
		return matches[1], matches[2]
	}

	return "", ""
}

// YarnParser implements parsing for yarn.lock files
type YarnParser struct {
	fs FileSystem
}

func NewYarnParser() *YarnParser {
	return &YarnParser{fs: &RealFileSystem{}}
}

func NewYarnParserWithFS(fs FileSystem) *YarnParser {
	return &YarnParser{fs: fs}
}

func (p *YarnParser) Parse(lockFilePath string) ([]Dependency, error) {
	file, err := p.fs.Open(lockFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open yarn.lock: %w", err)
	}
	defer func() {
		_ = file.Close() // Ignore close error as we already read the file
	}()

	var dependencies []Dependency
	scanner := bufio.NewScanner(file)

	// Regular expressions for parsing yarn.lock format
	packageRe := regexp.MustCompile(`^"?([^@\s"]+|@[^/]+/[^@\s"]+)@([^"]*)"?:$`)
	versionRe := regexp.MustCompile(`^\s+version\s+"([^"]+)"$`)

	var currentPackage *Dependency

	for scanner.Scan() {
		line := scanner.Text()

		// Check for package declaration line
		if matches := packageRe.FindStringSubmatch(line); matches != nil {
			// Save previous package if exists
			if currentPackage != nil {
				dependencies = append(dependencies, *currentPackage)
			}

			// Start new package
			currentPackage = &Dependency{
				Name:    matches[1],
				License: "", // License info not typically in yarn.lock
			}
		} else if currentPackage != nil {
			// Check for version line
			if matches := versionRe.FindStringSubmatch(line); matches != nil {
				currentPackage.Version = matches[1]
			}
		}
	}

	// Don't forget the last package
	if currentPackage != nil {
		dependencies = append(dependencies, *currentPackage)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading yarn.lock: %w", err)
	}

	return dependencies, nil
}
