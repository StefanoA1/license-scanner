package detector

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/StefanoA1/license-scanner/internal/constants"
)

type LicenseInfo struct {
	License    string  `json:"license"`
	Confidence float64 `json:"confidence"`
	Source     string  `json:"source"`
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

type Detector struct {
	fs FileSystem
}

func New() *Detector {
	return &Detector{
		fs: &RealFileSystem{},
	}
}

func NewWithFileSystem(fs FileSystem) *Detector {
	return &Detector{
		fs: fs,
	}
}

func (d *Detector) DetectLicense(packagePath string) (*LicenseInfo, error) {
	// Try to get license from package.json first
	if info := d.detectFromPackageJSON(packagePath); info != nil {
		return info, nil
	}

	// Then try LICENSE files
	if info := d.detectFromLicenseFile(packagePath); info != nil {
		return info, nil
	}

	// Default to unknown
	return &LicenseInfo{
		License:    constants.UnknownLicense,
		Confidence: 0.0,
		Source:     constants.NotFoundSource,
	}, nil
}

func (d *Detector) detectFromPackageJSON(packagePath string) *LicenseInfo {
	packageJSONPath := d.fs.Join(packagePath, constants.PackageJSONFile)

	file, err := d.fs.Open(packageJSONPath)
	if err != nil {
		return nil
	}
	defer func() {
		_ = file.Close() // Ignore close error as we already read the file
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil
	}

	var pkg struct {
		License interface{} `json:"license"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	license := extractLicenseFromField(pkg.License)
	if license != "" {
		return &LicenseInfo{
			License:    license,
			Confidence: 1.0,
			Source:     constants.PackageJSONSource,
		}
	}

	return nil
}

func (d *Detector) detectFromLicenseFile(packagePath string) *LicenseInfo {
	for _, filename := range constants.LicenseFileVariants {
		licensePath := d.fs.Join(packagePath, filename)
		if info, err := d.fs.Stat(licensePath); err == nil && !info.IsDir() {
			license, confidence := d.analyzeLicenseFile(licensePath)
			return &LicenseInfo{
				License:    license,
				Confidence: confidence,
				Source:     constants.LicenseFileSource,
			}
		}
	}

	return nil
}

func (d *Detector) analyzeLicenseFile(licensePath string) (string, float64) {
	file, err := d.fs.Open(licensePath)
	if err != nil {
		return constants.UnknownLicense, 0.2
	}
	defer func() {
		_ = file.Close() // Ignore close error as we already read the file
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		return constants.UnknownLicense, 0.2
	}

	content := string(data)
	content = strings.ToLower(content)

	// License patterns with confidence scores
	patterns := map[string]struct {
		pattern    *regexp.Regexp
		confidence float64
	}{
		"MIT": {
			pattern:    regexp.MustCompile(`mit\s+license|permission\s+is\s+hereby\s+granted.*free\s+of\s+charge`),
			confidence: 0.9,
		},
		"Apache-2.0": {
			pattern:    regexp.MustCompile(`apache\s+license.*version\s+2\.0|licensed\s+under\s+the\s+apache\s+license|apache\s+license.*version\s+2.*january.*2004`),
			confidence: 0.9,
		},
		"GPL-3.0": {
			pattern:    regexp.MustCompile(`gnu\s+general\s+public\s+license.*version\s+3|gplv3|version\s+3.*june\s+2007`),
			confidence: 0.9,
		},
		"GPL-2.0": {
			pattern:    regexp.MustCompile(`gnu\s+general\s+public\s+license.*version\s+2|gplv2`),
			confidence: 0.9,
		},
		"BSD-3-Clause": {
			pattern:    regexp.MustCompile(`bsd.*3.*clause|redistribution\s+and\s+use.*binary\s+forms.*conditions`),
			confidence: 0.8,
		},
		"BSD-2-Clause": {
			pattern:    regexp.MustCompile(`bsd.*2.*clause`),
			confidence: 0.8,
		},
		"ISC": {
			pattern:    regexp.MustCompile(`isc\s+license|permission\s+to\s+use.*copy.*modify.*distribute`),
			confidence: 0.8,
		},
	}

	// Check for license patterns
	for license, info := range patterns {
		if info.pattern.MatchString(content) {
			return license, info.confidence
		}
	}

	return constants.UnknownLicense, 0.2
}

func extractLicenseFromField(licenseField interface{}) string {
	switch v := licenseField.(type) {
	case string:
		return normalizedLicense(v)
	case map[string]interface{}:
		if typeVal, ok := v["type"].(string); ok {
			return normalizedLicense(typeVal)
		}
	case []interface{}:
		if len(v) > 0 {
			if firstLicense := extractLicenseFromField(v[0]); firstLicense != "" {
				return firstLicense
			}
		}
	}
	return ""
}

func normalizedLicense(license string) string {
	license = strings.TrimSpace(license)
	if license == "" {
		return ""
	}

	// Common license normalizations
	license = strings.ReplaceAll(license, " ", "-")

	switch strings.ToLower(license) {
	case "mit":
		return "MIT"
	case "apache-2.0", "apache2", "apache-v2":
		return "Apache-2.0"
	case "gpl-3.0", "gplv3", "gpl3":
		return "GPL-3.0"
	case "gpl-2.0", "gplv2", "gpl2":
		return "GPL-2.0"
	case "bsd-3-clause", "bsd3":
		return "BSD-3-Clause"
	case "bsd-2-clause", "bsd2":
		return "BSD-2-Clause"
	case "isc":
		return "ISC"
	default:
		return license
	}
}
