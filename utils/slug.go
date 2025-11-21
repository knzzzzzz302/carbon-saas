package utils

import (
	"strings"

	"github.com/google/uuid"
)

func GenerateSlug(seed string) string {
	base := strings.ToLower(strings.TrimSpace(seed))
	base = strings.ReplaceAll(base, " ", "-")
	base = strings.ReplaceAll(base, "_", "-")
	if base == "" {
		base = "org"
	}
	return base + "-" + uuid.NewString()[:8]
}
