package constants

// Directory and file names
const (
	NodeModulesDir  = "node_modules"
	PnpmStoreDir    = ".pnpm"
	PackageJSONFile = "package.json"
)

// License-related constants
const (
	UnknownLicense        = "Unknown"
	LicenseFileSource     = "LICENSE file"
	PackageJSONSource     = "package.json"
	NotFoundSource        = "not found"
	DetectionFailedSource = "detection failed"
)

// Lock file names
const (
	PackageLockJSON = "package-lock.json"
	YarnLock        = "yarn.lock"
	PnpmLockYAML    = "pnpm-lock.yaml"
)

// LicenseFileVariants contains all possible LICENSE file name variations
var LicenseFileVariants = []string{
	"LICENSE",
	"LICENSE.txt",
	"LICENSE.md",
	"LICENCE",
	"LICENCE.txt",
	"LICENCE.md",
}

// Package manager names
const (
	PackageManagerNPM  = "npm"
	PackageManagerYarn = "yarn"
	PackageManagerPnpm = "pnpm"
)
