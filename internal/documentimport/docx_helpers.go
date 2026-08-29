package documentimport

import (
	"strconv"
	"strings"
)

func parseNonNegativeInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}
