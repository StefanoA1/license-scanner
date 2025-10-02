package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/StefanoA1/license-scanner/internal/constants"
	"github.com/StefanoA1/license-scanner/internal/detector"
	"github.com/StefanoA1/license-scanner/internal/parser"
)

type Scanner struct {
	rootPath        string
	licenseDetector *detector.Detector
	fs              parser.FileSystem
	verbose         bool
}

type ScanResult struct {
	Dependencies []EnrichedDependency `json:"dependencies"`
}

type EnrichedDependency struct {
	Name       string  `json:"name"`
	Version    string  `json:"version"`
	License    string  `json:"license"`
	Confidence float64 `json:"confidence"`
	Source     string  `json:"source"`
}

func New(rootPath string) *Scanner {
	return &Scanner{
		rootPath:        rootPath,
		licenseDetector: detector.New(),
		fs:              &parser.RealFileSystem{},
		verbose:         false,
	}
}

func NewWithVerbose(rootPath string, verbose bool) *Scanner {
	return &Scanner{
		rootPath:        rootPath,
		licenseDetector: detector.New(),
		fs:              &parser.RealFileSystem{},
		verbose:         verbose,
	}
}

func NewWithDetector(rootPath string, licenseDetector *detector.Detector) *Scanner {
	return &Scanner{
		rootPath:        rootPath,
		licenseDetector: licenseDetector,
		fs:              &parser.RealFileSystem{},
	}
}

func NewWithDependencies(rootPath string, licenseDetector *detector.Detector, fs parser.FileSystem) *Scanner {
	return &Scanner{
		rootPath:        rootPath,
		licenseDetector: licenseDetector,
		fs:              fs,
	}
}

func (s *Scanner) Scan() (*ScanResult, error) {
	// Detect which lock file exists
	lockFilePath, packageManager, err := parser.DetectLockFile(s.fs, s.rootPath)
	if err != nil {
		return nil, fmt.Errorf("no lock file found in %s", s.rootPath)
	}

	if s.verbose {
		fmt.Fprintf(os.Stderr, "Found %s lock file: %s\n", packageManager, lockFilePath)
	}

	// Parse the lock file based on package manager
	var lockParser parser.LockFileParser
	switch packageManager {
	case "npm":
		lockParser = parser.NewNPMParserWithFS(s.fs)
	case "pnpm":
		lockParser = parser.NewPnpmParserWithFS(s.fs)
	case "yarn":
		lockParser = parser.NewYarnParserWithFS(s.fs)
	default:
		return nil, fmt.Errorf("unsupported package manager: %s", packageManager)
	}

	dependencies, err := lockParser.Parse(lockFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lock file: %w", err)
	}

	// Enrich dependencies with license information
	nodeModulesPath := filepath.Join(s.rootPath, constants.NodeModulesDir)

	var enrichedDeps []EnrichedDependency
	for _, dep := range dependencies {
		packagePath := s.resolvePackagePath(nodeModulesPath, packageManager, dep)
		licenseInfo, err := s.licenseDetector.DetectLicense(packagePath)
		if err != nil {
			// If detection fails, use default values
			licenseInfo = &detector.LicenseInfo{
				License:    constants.UnknownLicense,
				Confidence: 0.0,
				Source:     constants.DetectionFailedSource,
			}
		}

		enrichedDeps = append(enrichedDeps, EnrichedDependency{
			Name:       dep.Name,
			Version:    dep.Version,
			License:    licenseInfo.License,
			Confidence: licenseInfo.Confidence,
			Source:     licenseInfo.Source,
		})
	}

	return &ScanResult{
		Dependencies: enrichedDeps,
	}, nil
}

// resolvePackagePath resolves the actual file system path for a package based on the package manager
func (s *Scanner) resolvePackagePath(nodeModulesPath, packageManager string, dep parser.Dependency) string {
	switch packageManager {
	case constants.PackageManagerPnpm:
		// For pnpm, try multiple possible paths since the structure can vary
		// Pattern: node_modules/.pnpm/<package>@<version>/node_modules/<package>
		pnpmStorePath := filepath.Join(nodeModulesPath, constants.PnpmStoreDir)

		// For scoped packages, pnpm may encode the @ symbol
		encodedName := strings.ReplaceAll(dep.Name, "@", "%40")

		// Try with exact version match (both encoded and non-encoded names)
		candidates := []string{
			dep.Name + "@" + dep.Version,
			encodedName + "@" + dep.Version,
		}

		for _, candidate := range candidates {
			pnpmPackagePath := filepath.Join(pnpmStorePath, candidate, constants.NodeModulesDir, dep.Name)
			if s.pathExists(pnpmPackagePath) {
				return pnpmPackagePath
			}
		}

		// Try to find any version of the package in the .pnpm store
		// This handles cases where the version might have additional qualifiers
		if entries, err := os.ReadDir(pnpmStorePath); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					entryName := entry.Name()
					// Check for both regular and encoded package names
					if strings.HasPrefix(entryName, dep.Name+"@") ||
						strings.HasPrefix(entryName, encodedName+"@") {
						candidatePath := filepath.Join(pnpmStorePath, entryName, constants.NodeModulesDir, dep.Name)
						if s.pathExists(candidatePath) {
							return candidatePath
						}
					}
				}
			}
		}

		// Fallback to standard node_modules path (for hoisted packages)
		fallbackPath := filepath.Join(nodeModulesPath, dep.Name)
		if s.pathExists(fallbackPath) {
			return fallbackPath
		}

		// Return the expected pnpm path even if it doesn't exist (for error handling)
		return filepath.Join(pnpmStorePath, dep.Name+"@"+dep.Version, constants.NodeModulesDir, dep.Name)

	case constants.PackageManagerNPM, constants.PackageManagerYarn:
		// Standard node_modules structure
		return filepath.Join(nodeModulesPath, dep.Name)

	default:
		// Default to standard structure
		return filepath.Join(nodeModulesPath, dep.Name)
	}
}

// pathExists checks if a path exists on the file system
func (s *Scanner) pathExists(path string) bool {
	_, err := s.fs.Stat(path)
	return err == nil
}
