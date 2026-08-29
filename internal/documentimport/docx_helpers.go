package documentimport

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
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

func skipXMLElement(ctx context.Context, decoder *xml.Decoder) error {
	depth := 1
	for depth > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		token, err := decoder.Token()
		if err == io.EOF {
			return fmt.Errorf("DOCX element ended unexpectedly")
		}
		if err != nil {
			return err
		}
		switch token.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
		}
	}
	return nil
}
