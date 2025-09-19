package config

import (
	"os"
	"testing"
)

func TestNewServerConfig(t *testing.T) {
	t.Run("uses default values when no env vars set", func(t *testing.T) {
		// Clear any existing env vars
		os.Unsetenv("HTTP_PORT")
		os.Unsetenv("SHUTDOWN_TIMEOUT")

		config := NewServerConfig()

		if config.HTTPPort != 8080 {
			t.Errorf("Expected HTTPPort 8080, got %d", config.HTTPPort)
		}
		if config.ShutdownTimeout != 30 {
			t.Errorf("Expected ShutdownTimeout 30, got %d", config.ShutdownTimeout)
		}
	})

	t.Run("uses environment variables when set", func(t *testing.T) {
		// Set environment variables
		os.Setenv("HTTP_PORT", "9090")
		os.Setenv("SHUTDOWN_TIMEOUT", "60")

		defer func() {
			// Clean up
			os.Unsetenv("HTTP_PORT")
			os.Unsetenv("SHUTDOWN_TIMEOUT")
		}()

		config := NewServerConfig()

		if config.HTTPPort != 9090 {
			t.Errorf("Expected HTTPPort 9090, got %d", config.HTTPPort)
		}
		if config.ShutdownTimeout != 60 {
			t.Errorf("Expected ShutdownTimeout 60, got %d", config.ShutdownTimeout)
		}
	})

	t.Run("uses defaults for invalid environment variables", func(t *testing.T) {
		// Set invalid environment variables
		os.Setenv("HTTP_PORT", "invalid")
		os.Setenv("SHUTDOWN_TIMEOUT", "not_a_number")

		defer func() {
			// Clean up
			os.Unsetenv("HTTP_PORT")
			os.Unsetenv("SHUTDOWN_TIMEOUT")
		}()

		config := NewServerConfig()

		if config.HTTPPort != 8080 {
			t.Errorf("Expected HTTPPort 8080 (default), got %d", config.HTTPPort)
		}
		if config.ShutdownTimeout != 30 {
			t.Errorf("Expected ShutdownTimeout 30 (default), got %d", config.ShutdownTimeout)
		}
	})
}

func TestGetEnvInt(t *testing.T) {
	t.Run("returns default when env var not set", func(t *testing.T) {
		os.Unsetenv("TEST_INT")
		result := getEnvInt("TEST_INT", 42)
		if result != 42 {
			t.Errorf("Expected 42, got %d", result)
		}
	})

	t.Run("returns parsed value when env var is valid int", func(t *testing.T) {
		os.Setenv("TEST_INT", "100")
		defer os.Unsetenv("TEST_INT")

		result := getEnvInt("TEST_INT", 42)
		if result != 100 {
			t.Errorf("Expected 100, got %d", result)
		}
	})

	t.Run("returns default when env var is invalid int", func(t *testing.T) {
		os.Setenv("TEST_INT", "not_a_number")
		defer os.Unsetenv("TEST_INT")

		result := getEnvInt("TEST_INT", 42)
		if result != 42 {
			t.Errorf("Expected 42 (default), got %d", result)
		}
	})
}
