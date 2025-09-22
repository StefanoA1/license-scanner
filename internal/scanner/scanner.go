package scanner

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/stefano/license-scanner/internal/detector"
	"github.com/stefano/license-scanner/internal/parser"
)

type Scanner struct {
	rootPath        string
	licenseDetector *detector.Detector
	fs              parser.FileSystem
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

	fmt.Fprintf(os.Stderr, "Found %s lock file: %s\n", packageManager, lockFilePath)

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
	nodeModulesPath := filepath.Join(s.rootPath, "node_modules")

	var enrichedDeps []EnrichedDependency
	for _, dep := range dependencies {
		packagePath := filepath.Join(nodeModulesPath, dep.Name)
		licenseInfo, err := s.licenseDetector.DetectLicense(packagePath)
		if err != nil {
			// If detection fails, use default values
			licenseInfo = &detector.LicenseInfo{
				License:    "Unknown",
				Confidence: 0.0,
				Source:     "detection failed",
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
