package evidence

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	maxShortAnswerBytes = 500
	maxLongAnswerBytes  = 5000
	maxSignatureBytes   = 2 << 20
)

func validateFieldContracts(fields []Field) error {
	for _, field := range fields {
		fieldType := strings.ToLower(strings.TrimSpace(field.Type))
		switch fieldType {
		case "text", "short_text", "long_text", "date", "number", "photo", "file", "signature":
			if len(field.Options) != 0 {
				return fmt.Errorf("%s must not define selection options", field.Label)
			}
		case "single_select":
			if len(field.Options) < 2 || len(field.Options) > 50 {
				return fmt.Errorf("%s must define 2-50 choices", field.Label)
			}
			seen := map[string]struct{}{}
			for _, option := range field.Options {
				option = strings.TrimSpace(option)
				if option == "" {
					return fmt.Errorf("%s contains an empty choice", field.Label)
				}
				if _, exists := seen[option]; exists {
					return fmt.Errorf("%s contains duplicate choices", field.Label)
				}
				seen[option] = struct{}{}
			}
		default:
			return fmt.Errorf("%s uses unsupported field type %q", field.Label, field.Type)
		}

		if len(field.ID) > 80 || len(field.Label) > 200 || len(field.Description) > 1000 {
			return fmt.Errorf("%s exceeds field metadata limits", field.Label)
		}
		if len(field.AcceptedFormats) > 0 && fieldType != "photo" && fieldType != "file" && fieldType != "signature" {
			return fmt.Errorf("%s cannot define accepted file formats", field.Label)
		}
		for _, format := range field.AcceptedFormats {
			mediaType := normalizeMediaType(format)
			if !allowedMediaType(mediaType) {
				return fmt.Errorf("%s contains unsupported file format %q", field.Label, format)
			}
			if fieldType == "photo" && mediaType != "image/jpeg" && mediaType != "image/png" {
				return fmt.Errorf("%s accepts a non-image photo format", field.Label)
			}
			if fieldType == "signature" && mediaType != "image/png" {
				return fmt.Errorf("%s signatures must use image/png", field.Label)
			}
		}
	}
	return nil
}

func (s *Service) validateAnswers(ctx context.Context, request Request, answers map[string]string) error {
	fields := make(map[string]Field, len(request.Fields))
	for _, field := range request.Fields {
		fields[field.ID] = field
	}
	for fieldID := range answers {
		if _, ok := fields[fieldID]; !ok {
			return fmt.Errorf("response contains an unrequested field")
		}
	}

	for _, field := range request.Fields {
		value := strings.TrimSpace(answers[field.ID])
		if field.Required && value == "" {
			return fmt.Errorf("%s is required", field.Label)
		}
		if value == "" {
			continue
		}
		fieldType := strings.ToLower(strings.TrimSpace(field.Type))
		switch fieldType {
		case "text", "short_text":
			if len(value) > maxShortAnswerBytes {
				return fmt.Errorf("%s is too long", field.Label)
			}
		case "long_text":
			if len(value) > maxLongAnswerBytes {
				return fmt.Errorf("%s is too long", field.Label)
			}
		case "single_select":
			if !containsOption(field.Options, value) {
				return fmt.Errorf("%s contains an invalid selection", field.Label)
			}
		case "date":
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return fmt.Errorf("%s must be a valid date", field.Label)
			}
		case "number":
			number, err := strconv.ParseFloat(value, 64)
			if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
				return fmt.Errorf("%s must be a valid number", field.Label)
			}
		case "photo", "file", "signature":
			artifact, err := s.repo.GetArtifact(ctx, request.TenantID, request.ID, value)
			if err != nil {
				return fmt.Errorf("%s must reference a file uploaded for this request", field.Label)
			}
			if err := validateArtifactForField(field, artifact); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s uses an unsupported response type", field.Label)
		}
	}
	return nil
}

func validateArtifactForField(field Field, artifact Artifact) error {
	switch artifact.Status {
	case ArtifactStoredUnscanned, ArtifactAvailable:
	default:
		return fmt.Errorf("%s references an unavailable file", field.Label)
	}
	if artifact.SizeBytes <= 0 {
		return fmt.Errorf("%s references an empty file", field.Label)
	}
	mediaType := normalizeMediaType(artifact.MediaType)
	fieldType := strings.ToLower(strings.TrimSpace(field.Type))
	if fieldType == "photo" && mediaType != "image/jpeg" && mediaType != "image/png" {
		return fmt.Errorf("%s must reference a JPEG or PNG image", field.Label)
	}
	if fieldType == "signature" {
		if mediaType != "image/png" {
			return fmt.Errorf("%s must reference a PNG signature", field.Label)
		}
		if artifact.SizeBytes > maxSignatureBytes {
			return fmt.Errorf("%s signature is too large", field.Label)
		}
	}
	if len(field.AcceptedFormats) > 0 && !containsMediaType(field.AcceptedFormats, mediaType) {
		return fmt.Errorf("%s file format is not permitted", field.Label)
	}
	return nil
}

func containsOption(values []string, expected string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == expected {
			return true
		}
	}
	return false
}

func containsMediaType(values []string, expected string) bool {
	for _, value := range values {
		if normalizeMediaType(value) == expected {
			return true
		}
	}
	return false
}

func normalizeMediaType(value string) string {
	return strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
}
