package detect

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
)

// PackageManager holds the detected package manager and its install command.
type PackageManager struct {
	Name           string // "pnpm", "yarn", "bun", "npm"
	InstallCommand string
	LockFile       string
}

// Lock files checked in priority order.
var lockFileOrder = []struct {
	File    string
	Manager PackageManager
}{
	{
		File: "pnpm-lock.yaml",
		Manager: PackageManager{
			Name:           "pnpm",
			InstallCommand: "pnpm install --frozen-lockfile",
			LockFile:       "pnpm-lock.yaml",
		},
	},
	{
		File: "yarn.lock",
		Manager: PackageManager{
			Name:           "yarn",
			InstallCommand: "yarn install --frozen-lockfile",
			LockFile:       "yarn.lock",
		},
	},
	{
		File: "bun.lockb",
		Manager: PackageManager{
			Name:           "bun",
			InstallCommand: "bun install --frozen-lockfile",
			LockFile:       "bun.lockb",
		},
	},
	{
		File: "package-lock.json",
		Manager: PackageManager{
			Name:           "npm",
			InstallCommand: "npm ci",
			LockFile:       "package-lock.json",
		},
	},
}

// DetectPackageManager examines lock files in sourceDir to determine the package manager.
func DetectPackageManager(sourceDir string) PackageManager {
	for _, lf := range lockFileOrder {
		if _, err := os.Stat(filepath.Join(sourceDir, lf.File)); err == nil {
			return lf.Manager
		}
	}

	return PackageManager{
		Name:           "npm",
		InstallCommand: "npm install",
		LockFile:       "",
	}
}

// HashLockFile reads the lock file and returns its SHA-256 hex digest.
func HashLockFile(sourceDir string, lockFileName string) string {
	if lockFileName == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(sourceDir, lockFileName))
	if err != nil {
		return ""
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
