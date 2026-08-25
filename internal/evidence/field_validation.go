package evidence

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

const (
	maxShortAnswerBytes = 500
	maxLongAnswerBytes  = 5000
	maxSignatureBytes   = 2 << 20
)

func validateFieldContracts(fields []Field) error {
	_, err := normalizeFieldContracts(formcontract.Presentation{}, nil, fields)
	return err
}

func normalizeFieldContracts(presentation formcontract.Presentation, sections []formcontract.Section, fields []Field) ([]Field, error) {
	contract, err := formContract(presentation, sections, fields)
	if err != nil {
		return nil, err
	}
	normalized := append([]Field(nil), fields...)
	for index := range normalized {
		applyContractField(&normalized[index], contract.Fields[index])
	}
	return normalized, nil
}

func formContract(presentation formcontract.Presentation, sections []formcontract.Section, fields []Field) (formcontract.Contract, error) {
	contractFields := make([]formcontract.Field, len(fields))
	for index, field := range fields {
		contractFields[index] = formcontract.Field{
			ID: field.ID, SectionID: field.SectionID, Label: field.Label, Type: formcontract.Type(field.Type), Required: field.Required,
			Description: field.Description, Options: append([]string(nil), field.Options...), AcceptedFormats: append([]string(nil), field.AcceptedFormats...),
			Attestation: field.Attestation, Constraints: field.Constraints, Condition: field.Condition, Scoring: field.Scoring,
		}
	}
	return formcontract.Normalize(formcontract.Contract{Presentation: presentation, Sections: sections, Fields: contractFields})
}

func applyContractField(target *Field, source formcontract.Field) {
	target.ID = source.ID
	target.SectionID = source.SectionID
	target.Label = source.Label
	target.Type = string(source.Type)
	target.Required = source.Required
	target.Description = source.Description
	target.Options = append([]string(nil), source.Options...)
	target.AcceptedFormats = append([]string(nil), source.AcceptedFormats...)
	target.Attestation = source.Attestation
	target.Constraints = source.Constraints
	target.Condition = source.Condition
	target.Scoring = source.Scoring
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
		case "number", "decimal", "integer", "percentage", "currency":
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
