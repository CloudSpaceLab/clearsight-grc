package evidence

import (
	"context"
	"fmt"
	"math"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

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

var telephonePattern = regexp.MustCompile(`^[+0-9][0-9 ()-]{6,29}$`)

func (s *Service) validateAnswers(ctx context.Context, request Request, answers map[string]formcontract.AnswerValue) error {
	contract, err := formContract(request.Presentation, request.Sections, request.Fields)
	if err != nil {
		return err
	}
	visible, err := formcontract.VisibleFields(contract, answers)
	if err != nil {
		return err
	}
	visibleByID := make(map[string]formcontract.Field, len(visible))
	requestByID := make(map[string]Field, len(request.Fields))
	for _, field := range visible {
		visibleByID[field.ID] = field
	}
	for _, field := range request.Fields {
		requestByID[field.ID] = field
	}
	for fieldID := range answers {
		if _, requested := requestByID[fieldID]; !requested {
			return fmt.Errorf("response contains an unrequested field")
		}
		if _, shown := visibleByID[fieldID]; !shown {
			return fmt.Errorf("%s was not requested for the current answers", requestByID[fieldID].Label)
		}
	}

	for _, field := range visible {
		answer, exists := answers[field.ID]
		if field.Required && (!exists || !answer.Answered()) {
			return fmt.Errorf("%s is required", field.Label)
		}
		if !exists || !answer.Answered() {
			continue
		}
		if err := s.validateTypedAnswer(ctx, request, requestByID[field.ID], field, answer); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) validateTypedAnswer(ctx context.Context, request Request, requestField Field, field formcontract.Field, answer formcontract.AnswerValue) error {
	fieldType := field.Type
	switch fieldType {
	case formcontract.TypeShortText, formcontract.TypeLongText, formcontract.TypeEmail, formcontract.TypeTelephone, formcontract.TypeURL:
		value, err := requiredScalar(field, answer)
		if err != nil {
			return err
		}
		minimum, maximum := 0, maxShortAnswerBytes
		if fieldType == formcontract.TypeLongText {
			maximum = maxLongAnswerBytes
		}
		if field.Constraints.MinLength != nil {
			minimum = *field.Constraints.MinLength
		}
		if field.Constraints.MaxLength != nil {
			maximum = *field.Constraints.MaxLength
		}
		length := utf8.RuneCountInString(value)
		if length < minimum || length > maximum {
			return fmt.Errorf("%s must contain %d-%d characters", field.Label, minimum, maximum)
		}
		switch fieldType {
		case formcontract.TypeEmail:
			address, parseErr := mail.ParseAddress(value)
			if parseErr != nil || address.Address != value || !strings.Contains(value, "@") {
				return fmt.Errorf("%s must be a valid email address", field.Label)
			}
		case formcontract.TypeTelephone:
			if !telephonePattern.MatchString(value) {
				return fmt.Errorf("%s must be a valid telephone number", field.Label)
			}
		case formcontract.TypeURL:
			parsed, parseErr := url.ParseRequestURI(value)
			if parseErr != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
				return fmt.Errorf("%s must be a valid URL beginning with https:// or http://", field.Label)
			}
		}
	case formcontract.TypeInteger, formcontract.TypeDecimal, formcontract.TypePercentage, formcontract.TypeCurrency:
		value, err := requiredScalar(field, answer)
		if err != nil {
			return err
		}
		number, parseErr := strconv.ParseFloat(value, 64)
		if parseErr != nil || math.IsNaN(number) || math.IsInf(number, 0) {
			return fmt.Errorf("%s must be a valid number", field.Label)
		}
		if fieldType == formcontract.TypeInteger && number != math.Trunc(number) {
			return fmt.Errorf("%s must be a whole number", field.Label)
		}
		if field.Constraints.Minimum != nil && number < *field.Constraints.Minimum {
			return fmt.Errorf("%s must be at least %v", field.Label, *field.Constraints.Minimum)
		}
		if field.Constraints.Maximum != nil && number > *field.Constraints.Maximum {
			return fmt.Errorf("%s must be at most %v", field.Label, *field.Constraints.Maximum)
		}
		if field.Constraints.DecimalPrecision != nil && decimalPlaces(value) > *field.Constraints.DecimalPrecision {
			return fmt.Errorf("%s permits at most %d decimal places", field.Label, *field.Constraints.DecimalPrecision)
		}
		if field.Constraints.Step != nil && field.Constraints.Minimum != nil {
			steps := (number - *field.Constraints.Minimum) / *field.Constraints.Step
			if math.Abs(steps-math.Round(steps)) > 1e-9 {
				return fmt.Errorf("%s must use increments of %v", field.Label, *field.Constraints.Step)
			}
		}
	case formcontract.TypeDate:
		value, err := requiredScalar(field, answer)
		if err != nil {
			return err
		}
		date, parseErr := time.Parse("2006-01-02", value)
		if parseErr != nil {
			return fmt.Errorf("%s must be a valid date", field.Label)
		}
		if field.Constraints.MinDate != "" {
			minimum, _ := time.Parse("2006-01-02", field.Constraints.MinDate)
			if date.Before(minimum) {
				return fmt.Errorf("%s must be on or after %s", field.Label, field.Constraints.MinDate)
			}
		}
		if field.Constraints.MaxDate != "" {
			maximum, _ := time.Parse("2006-01-02", field.Constraints.MaxDate)
			if date.After(maximum) {
				return fmt.Errorf("%s must be on or before %s", field.Label, field.Constraints.MaxDate)
			}
		}
	case formcontract.TypeYesNo, formcontract.TypeSingleSelect:
		value, err := requiredScalar(field, answer)
		if err != nil {
			return err
		}
		if !containsOption(field.Options, value) {
			return fmt.Errorf("%s contains an invalid selection", field.Label)
		}
	case formcontract.TypeMultiSelect:
		if answer.Text != nil || answer.Document != nil || len(answer.ArtifactIDs) != 0 {
			return fmt.Errorf("%s must contain selected values", field.Label)
		}
		minimum, maximum := 0, len(field.Options)
		if field.Constraints.MinSelections != nil {
			minimum = *field.Constraints.MinSelections
		}
		if field.Constraints.MaxSelections != nil {
			maximum = *field.Constraints.MaxSelections
		}
		if len(answer.Values) < minimum {
			return fmt.Errorf("%s requires at least %d selections", field.Label, minimum)
		}
		if len(answer.Values) > maximum {
			return fmt.Errorf("%s permits at most %d selections", field.Label, maximum)
		}
		seen := map[string]struct{}{}
		for _, value := range answer.Values {
			value = strings.TrimSpace(value)
			if !containsOption(field.Options, value) {
				return fmt.Errorf("%s contains an invalid selection", field.Label)
			}
			if _, duplicate := seen[value]; duplicate {
				return fmt.Errorf("%s contains a duplicate selection", field.Label)
			}
			seen[value] = struct{}{}
		}
	case formcontract.TypeCheckbox, formcontract.TypeAttestation:
		value, err := requiredScalar(field, answer)
		if err != nil {
			return err
		}
		if !strings.EqualFold(value, "yes") && !strings.EqualFold(value, "true") {
			return fmt.Errorf("%s must be confirmed", field.Label)
		}
	case formcontract.TypeFile, formcontract.TypePhoto, formcontract.TypeSignature, formcontract.TypeVendorDocument:
		artifactIDs, err := answerArtifactIDs(field, answer)
		if err != nil {
			return err
		}
		maximum := 10
		if fieldType == formcontract.TypePhoto || fieldType == formcontract.TypeSignature || fieldType == formcontract.TypeVendorDocument {
			maximum = 1
		}
		if field.Constraints.MaxFiles != nil {
			maximum = *field.Constraints.MaxFiles
		}
		if len(artifactIDs) > maximum {
			return fmt.Errorf("%s permits at most %d files", field.Label, maximum)
		}
		for _, artifactID := range artifactIDs {
			artifact, loadErr := s.repo.GetArtifact(ctx, request.TenantID, request.ID, artifactID)
			if loadErr != nil {
				return fmt.Errorf("%s must reference a file uploaded for this request", field.Label)
			}
			if err := validateArtifactForField(requestField, artifact); err != nil {
				return err
			}
			if field.Constraints.MaxFileBytes != nil && artifact.SizeBytes > *field.Constraints.MaxFileBytes {
				return fmt.Errorf("%s contains a file larger than permitted", field.Label)
			}
		}
	default:
		return fmt.Errorf("%s uses an unsupported response type", field.Label)
	}
	return nil
}

func requiredScalar(field formcontract.Field, answer formcontract.AnswerValue) (string, error) {
	value, exists := answer.ScalarText()
	if !exists || len(answer.Values) != 0 || len(answer.ArtifactIDs) != 0 || answer.Document != nil {
		return "", fmt.Errorf("%s must contain one value", field.Label)
	}
	return value, nil
}

func answerArtifactIDs(field formcontract.Field, answer formcontract.AnswerValue) ([]string, error) {
	if field.Type == formcontract.TypeVendorDocument {
		if answer.Document == nil || strings.TrimSpace(answer.Document.ArtifactID) == "" || strings.TrimSpace(answer.Document.DocumentType) == "" || answer.Text != nil || len(answer.Values) != 0 || len(answer.ArtifactIDs) != 0 {
			return nil, fmt.Errorf("%s requires one uploaded document and its type", field.Label)
		}
		for _, value := range []string{answer.Document.IssuedOn, answer.Document.ExpiresOn} {
			if value != "" {
				if _, err := time.Parse("2006-01-02", value); err != nil {
					return nil, fmt.Errorf("%s contains an invalid document date", field.Label)
				}
			}
		}
		return []string{strings.TrimSpace(answer.Document.ArtifactID)}, nil
	}
	if answer.Document != nil || len(answer.Values) != 0 {
		return nil, fmt.Errorf("%s must contain uploaded file references", field.Label)
	}
	if len(answer.ArtifactIDs) > 0 {
		return answer.ArtifactIDs, nil
	}
	if legacy, exists := answer.ScalarText(); exists && legacy != "" {
		return []string{legacy}, nil
	}
	return nil, fmt.Errorf("%s must contain an uploaded file", field.Label)
}

func decimalPlaces(value string) int {
	value = strings.TrimSpace(value)
	if exponent := strings.IndexAny(value, "eE"); exponent >= 0 {
		value = value[:exponent]
	}
	if decimal := strings.IndexByte(value, '.'); decimal >= 0 {
		return len(strings.TrimRight(value[decimal+1:], "0"))
	}
	return 0
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
