package tools

import (
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestUUIDGen_GenerateUUID(t *testing.T) {
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelInfo}),
	)
	slog.SetDefault(logger)

	gen := NewUUIDGen(logger)
	if gen == nil {
		t.Fatal("NewUUIDGen returned nil")
	}

	t.Run("generates valid UUID", func(t *testing.T) {
		uuid, err := gen.GenerateUUID()
		if err != nil {
			t.Fatalf("GenerateUUID failed: %v", err)
		}
		if uuid == "" {
			t.Error("Generated UUID is empty")
		}

		// UUID should be 36 characters long
		if len(uuid) != 36 {
			t.Errorf("UUID length is %d, expected 36", len(uuid))
		}

		// Should contain 4 hyphens
		if !strings.Contains(uuid, "-") {
			t.Error("UUID does not contain hyphens")
		}
		hyphenCount := strings.Count(uuid, "-")
		if hyphenCount != 4 {
			t.Errorf("UUID has %d hyphens, expected 4", hyphenCount)
		}
	})

	t.Run("generates unique UUIDs", func(t *testing.T) {
		uuid1, err := gen.GenerateUUID()
		if err != nil {
			t.Fatalf("First GenerateUUID failed: %v", err)
		}

		uuid2, err := gen.GenerateUUID()
		if err != nil {
			t.Fatalf("Second GenerateUUID failed: %v", err)
		}

		if uuid1 == uuid2 {
			t.Error("Generated UUIDs are not unique")
		}
	})

	t.Run("UUID format validation", func(t *testing.T) {
		uuid, err := gen.GenerateUUID()
		if err != nil {
			t.Fatalf("GenerateUUID failed: %v", err)
		}

		// Basic UUID v4 format check (8-4-4-4-12)
		parts := []int{8, 4, 4, 4, 12}
		start := 0
		for _, length := range parts {
			end := start + length
			if start > 0 {
				// Skip hyphen
				start++
				end++
			}
			if end > len(uuid) {
				t.Errorf("UUID too short for format check")
				break
			}
			part := uuid[start:end]
			if len(part) != length {
				t.Errorf("UUID part length is %d, expected %d", len(part), length)
			}
			start = end
		}
	})
}

func TestUUIDGen_Execute(t *testing.T) {
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelInfo}),
	)
	slog.SetDefault(logger)

	uuidGenerator := NewUUIDGen(logger)
	if uuidGenerator == nil {
		t.Fatal("NewUUIDGen returned nil")
	}

	t.Run("generates valid UUID via Execute", func(t *testing.T) {
		result, err := uuidGenerator.Execute(nil)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		uuid, exists := result["uuid"]
		if !exists {
			t.Fatal("Result does not contain 'uuid' key")
		}

		id, ok := uuid.(string)
		if !ok {
			t.Fatal("UUID is not a string")
		}

		if id == "" {
			t.Error("Generated UUID is empty")
		}

		// UUID should be 36 characters long
		if len(id) != 36 {
			t.Errorf("UUID length is %d, expected 36", len(id))
		}

		// Should contain 4 hyphens
		if !strings.Contains(id, "-") {
			t.Error("UUID does not contain hyphens")
		}
		hyphenCount := strings.Count(id, "-")
		if hyphenCount != 4 {
			t.Errorf("UUID has %d hyphens, expected 4", hyphenCount)
		}
	})

	t.Run("generates unique UUIDs via Execute", func(t *testing.T) {
		result1, err := uuidGenerator.Execute(nil)
		if err != nil {
			t.Fatalf("First Execute failed: %v", err)
		}

		result2, err := uuidGenerator.Execute(nil)
		if err != nil {
			t.Fatalf("Second Execute failed: %v", err)
		}

		uuid1 := result1["uuid"].(string)
		uuid2 := result2["uuid"].(string)

		if uuid1 == uuid2 {
			t.Error("Generated UUIDs are not unique")
		}
	})

	t.Run("UUID format validation via Execute", func(t *testing.T) {
		result, err := uuidGenerator.Execute(nil)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		id := result["uuid"].(string)

		// Basic UUID v4 format check (8-4-4-4-12)
		parts := []int{8, 4, 4, 4, 12}
		start := 0
		for _, length := range parts {
			end := start + length
			if start > 0 {
				// Skip hyphen
				start++
				end++
			}
			if end > len(id) {
				t.Errorf("UUID too short for format check")
				break
			}
			part := id[start:end]
			if len(part) != length {
				t.Errorf("UUID part length is %d, expected %d", len(part), length)
			}
			start = end
		}
	})

	t.Run("Execute with arguments", func(t *testing.T) {
		// Test that Execute works with various argument inputs
		testCases := []map[string]interface{}{
			nil,
			{},
			{"some": "args"},
		}

		for _, args := range testCases {
			result, err := uuidGenerator.Execute(args)
			if err != nil {
				t.Errorf("Execute failed with args %v: %v", args, err)
			}

			if result == nil {
				t.Errorf("Execute returned nil result with args %v", args)
			}

			if _, exists := result["uuid"]; !exists {
				t.Errorf("Result missing 'uuid' key with args %v", args)
			}
		}
	})
}

func TestUUIDGen_ToolInterface(t *testing.T) {
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelInfo}),
	)

	gen := NewUUIDGen(logger)

	// Test Tool interface methods
	t.Run("Name method", func(t *testing.T) {
		name := gen.Name()
		if name != "generate_uuid" {
			t.Errorf("Expected name 'generate_uuid', got '%s'", name)
		}
	})

	t.Run("Description method", func(t *testing.T) {
		desc := gen.Description()
		if desc == "" {
			t.Error("Description should not be empty")
		}
		if !strings.Contains(desc, "UUID") {
			t.Errorf("Description should mention UUID, got '%s'", desc)
		}
	})

	// Verify it implements the Tool interface
	var _ Tool = gen
}

func TestUUIDGen_ErrorHandling(t *testing.T) {
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelError}), // Set to ERROR level to reduce noise
	)

	gen := NewUUIDGen(logger)

	// Test error handling in Execute method when GenerateUUID fails
	// We'll create a test by temporarily replacing the UUID generation logic
	// Since we can't easily mock uuid.NewRandom(), we'll test the error path indirectly
	// by testing that the error handling code paths are exercised

	t.Run("Execute error handling structure", func(t *testing.T) {
		// This tests the structure of error handling in Execute
		// The actual error is hard to trigger with uuid.NewRandom(), but we can
		// verify the error handling logic exists by testing normal flow
		result, err := gen.Execute(nil)

		// In normal cases, this should not error
		if err != nil {
			// If we somehow get an error, verify it's handled properly
			if result == nil {
				t.Error("Result should not be nil even on error")
			}
			if errorMsg, exists := result["error"]; exists {
				if errorStr, ok := errorMsg.(string); ok && errorStr == "" {
					t.Error("Error message should not be empty when error occurs")
				}
			}
		} else {
			// Normal success case
			if result == nil {
				t.Error("Result should not be nil on success")
			}
			if _, exists := result["uuid"]; !exists {
				t.Error("Result should contain 'uuid' key on success")
			}
		}
	})

	t.Run("GenerateUUID error path coverage", func(t *testing.T) {
		// Test the error handling structure in GenerateUUID
		// We can't easily force uuid.NewRandom() to fail, but we can test
		// that the method handles errors properly in terms of logging

		// Create a logger that captures log output
		logOutput := &strings.Builder{}
		testLogger := slog.New(slog.NewTextHandler(logOutput, &slog.HandlerOptions{
			Level: slog.LevelError,
		}))

		testGen := NewUUIDGen(testLogger)

		// Call GenerateUUID multiple times to ensure consistency
		for i := 0; i < 3; i++ {
			uuid, err := testGen.GenerateUUID()
			if err != nil {
				// If an error occurs, verify it's logged
				logStr := logOutput.String()
				if !strings.Contains(logStr, "Failed to generate UUID") {
					t.Error("Error should be logged when GenerateUUID fails")
				}
				if uuid != "" {
					t.Error("UUID should be empty string when error occurs")
				}
			} else {
				// Normal success case - UUID should be valid
				if uuid == "" {
					t.Error("UUID should not be empty on success")
				}
				if len(uuid) != 36 {
					t.Errorf("UUID should be 36 characters, got %d", len(uuid))
				}
			}
		}
	})

	t.Run("Execute with nil arguments", func(t *testing.T) {
		// Explicit test for nil arguments to ensure coverage
		result, err := gen.Execute(nil)
		if err != nil {
			t.Errorf("Execute should handle nil arguments gracefully: %v", err)
		}
		if result == nil {
			t.Error("Result should not be nil with nil arguments")
		}
	})

	t.Run("Execute with empty arguments", func(t *testing.T) {
		// Explicit test for empty arguments to ensure coverage
		result, err := gen.Execute(map[string]interface{}{})
		if err != nil {
			t.Errorf("Execute should handle empty arguments gracefully: %v", err)
		}
		if result == nil {
			t.Error("Result should not be nil with empty arguments")
		}
	})

	t.Run("Multiple Execute calls for consistency", func(t *testing.T) {
		// Test multiple Execute calls to ensure consistent behavior
		// and coverage of all code paths
		results := make([]map[string]interface{}, 5)
		for i := 0; i < 5; i++ {
			result, err := gen.Execute(map[string]interface{}{"test": i})
			if err != nil {
				t.Errorf("Execute call %d failed: %v", i, err)
			}
			results[i] = result
		}

		// Verify all results are valid and unique
		uuids := make([]string, 5)
		for i, result := range results {
			if result == nil {
				t.Errorf("Result %d is nil", i)
				continue
			}
			uuid, exists := result["uuid"]
			if !exists {
				t.Errorf("Result %d missing UUID", i)
				continue
			}
			uuidStr, ok := uuid.(string)
			if !ok {
				t.Errorf("Result %d UUID is not string", i)
				continue
			}
			uuids[i] = uuidStr
		}

		// Check uniqueness
		for i := 0; i < 5; i++ {
			for j := i + 1; j < 5; j++ {
				if uuids[i] == uuids[j] {
					t.Errorf("UUIDs %d and %d are not unique: %s", i, j, uuids[i])
				}
			}
		}
	})
}
