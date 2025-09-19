package version

import (
	"os"
	"strings"
)

// These variables will be set by LDFLAGS during build
var (
	ldflagsVersion   = "dev"
	ldflagsBuildTime = "unknown"
	ldflagsGitCommit = "unknown"
)

func getVersion() string {
	// Try to read from version file first
	if data, err := os.ReadFile("version"); err == nil {
		fileVersion := strings.TrimSpace(string(data))
		if fileVersion != "" {
			return fileVersion
		}
	}
	// Fall back to LDFLAGS version
	return ldflagsVersion
}

// GetVersion returns the current version
func GetVersion() string {
	return getVersion()
}

// GetBuildTime returns the build time
func GetBuildTime() string {
	return ldflagsBuildTime
}

// GetGitCommit returns the git commit hash
func GetGitCommit() string {
	return ldflagsGitCommit
}
