package documentimport

import (
	"strings"
	"unicode/utf8"
)

func appendBounded(builder *strings.Builder, value string, maximum int, truncated *bool) {
	if maximum <= 0 || builder.Len() >= maximum {
		*truncated = true
		return
	}
	remaining := maximum - builder.Len()
	if len(value) > remaining {
		value = truncateUTF8(value, remaining)
		*truncated = true
	}
	builder.WriteString(value)
}

func normalizeText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.TrimSpace(value)
}

func looksLikeHeading(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 140 {
		return false
	}
	if strings.HasPrefix(value, "#") {
		return true
	}
	words := strings.Fields(value)
	if len(words) > 12 {
		return false
	}
	return !strings.ContainsAny(value, ".!?;")
}

func truncateUTF8(value string, maximum int) string {
	if maximum <= 0 {
		return ""
	}
	if len(value) <= maximum {
		return value
	}
	value = value[:maximum]
	for !utf8.ValidString(value) && len(value) > 0 {
		value = value[:len(value)-1]
	}
	return value
}
