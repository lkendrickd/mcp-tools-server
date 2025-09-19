package tools

import (
	"log/slog"

	"github.com/google/uuid"
)

// UUIDGen provides UUID generation functionality and implements Tool
type UUIDGen struct {
	logger *slog.Logger
}

// NewUUIDGen creates a new UUID generator
func NewUUIDGen(logger *slog.Logger) *UUIDGen {
	return &UUIDGen{
		logger: logger,
	}
}

// GenerateUUID generates a random UUID v4 string
func (g *UUIDGen) GenerateUUID() (string, error) {
	u, err := uuid.NewRandom()
	if err != nil {
		g.logger.Error(
			"Failed to generate UUID in GenerateUUID() method", "error",
			err,
		)
		return "", err
	}
	return u.String(), nil
}

// Name returns the tool's name
func (g *UUIDGen) Name() string {
	return "generate_uuid"
}

// Description returns the tool's description
func (g *UUIDGen) Description() string {
	return "Generates a random UUID v4 string"
}

// Execute runs the tool with the given arguments
func (g *UUIDGen) Execute(args map[string]interface{}) (map[string]interface{}, error) {
	uuid, err := g.GenerateUUID()
	if err != nil {
		g.logger.Error("Failed to generate UUID", "error", err)
		return map[string]interface{}{"error": err.Error()}, err
	}
	g.logger.Info("Generated UUID", "uuid", uuid)
	return map[string]interface{}{"uuid": uuid}, nil
}
