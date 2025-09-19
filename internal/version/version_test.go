package version

import (
	"os"
	"testing"
)

func TestGetVersion(t *testing.T) {
	// Save original values to restore after tests
	originalLdflagsVersion := ldflagsVersion
	defer func() {
		ldflagsVersion = originalLdflagsVersion
	}()

	t.Run("returns version from file when available", func(t *testing.T) {
		// Create a temporary directory
		tempDir, err := os.MkdirTemp("", "version_test_dir")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create version file in temp directory
		versionFile := tempDir + "/version"
		testVersion := "1.2.3-test"
		if err := os.WriteFile(versionFile, []byte(testVersion), 0644); err != nil {
			t.Fatalf("Failed to write version file: %v", err)
		}

		// Change working directory to temp directory temporarily
		originalWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get working directory: %v", err)
		}
		defer os.Chdir(originalWd)

		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("Failed to change to temp dir: %v", err)
		}

		version := getVersion()
		if version != testVersion {
			t.Errorf("Expected version %s, got %s", testVersion, version)
		}
	})

	t.Run("returns LDFLAGS version when file not available", func(t *testing.T) {
		// Set LDFLAGS version
		ldflagsVersion = "2.0.0-ldflags"

		// Change to a directory without version file
		tempDir, err := os.MkdirTemp("", "no_version_test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		originalWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get working directory: %v", err)
		}
		defer os.Chdir(originalWd)

		os.Chdir(tempDir)

		version := getVersion()
		if version != "2.0.0-ldflags" {
			t.Errorf("Expected version 2.0.0-ldflags, got %s", version)
		}
	})

	t.Run("handles empty version file", func(t *testing.T) {
		// Create a temporary directory
		tempDir, err := os.MkdirTemp("", "empty_version_test_dir")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create empty version file
		versionFile := tempDir + "/version"
		if err := os.WriteFile(versionFile, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to write empty version file: %v", err)
		}

		// Set LDFLAGS fallback
		ldflagsVersion = "fallback-version"

		originalWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get working directory: %v", err)
		}
		defer os.Chdir(originalWd)

		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("Failed to change to temp dir: %v", err)
		}

		version := getVersion()
		if version != "fallback-version" {
			t.Errorf("Expected fallback-version, got %s", version)
		}
	})
}

func TestGetVersion_PublicAPI(t *testing.T) {
	version := GetVersion()
	if version == "" {
		t.Error("GetVersion should not return empty string")
	}
}

func TestGetBuildTime(t *testing.T) {
	buildTime := GetBuildTime()
	// Should at least return the default value
	if buildTime == "" {
		t.Error("GetBuildTime should not return empty string")
	}
}

func TestGetGitCommit(t *testing.T) {
	gitCommit := GetGitCommit()
	// Should at least return the default value
	if gitCommit == "" {
		t.Error("GetGitCommit should not return empty string")
	}
}
