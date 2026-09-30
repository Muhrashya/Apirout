package registry

import (
	"github.com/google/uuid"
)

// generateID generates a new UUID
func generateID() string {
	return uuid.New().String()
}
