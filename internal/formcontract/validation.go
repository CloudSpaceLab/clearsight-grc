package formcontract

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
)

func Normalize(input Contract) (Contract, error) {
	input.Presentation.DefaultMode = PresentationMode(strings.ToUpper(strings.TrimSpace(string(input.Presentation.DefaultMode))))
	if input.Presentation.DefaultMode == "" {
		input.Presentation.DefaultMode = PresentationAutomatic
	}
	if !slices.Contains([]PresentationMode{PresentationClassic, PresentationWizard, PresentationAutomatic}, input.Presentation.DefaultMode) {
		return Contract{}, invalid("presentation mode is not supported")
	}
	input.ScoringMode = ScoringMode(strings.ToUpper(strings.TrimSpace(string(input.ScoringMode))))
	if input.ScoringMode == "" {
		input.ScoringMode = ScoringNone
		for _, field := range input.Fields {
			if field.Scoring != nil {
				input.ScoringMode = ScoringRisk
				break
			}
		}
	}
	if !slices.Contains([]ScoringMode{ScoringNone, ScoringRisk, ScoringCompliance}, input.ScoringMode) {
		return Contract{}, invalid("scoring mode is not supported")
	}
	if len(input.Sections) == 0 {
		input.Sections = []Section{{ID: DefaultSectionID, Title: "General"}}
	}
	if len(input.Sections) > MaxSections {
		return Contract{}, invalid("a form may contain at most %d sections", MaxSections)
	}
	if len(input.Fields) == 0 || len(input.Fields) > MaxFields {
		return Contract{}, invalid("a form must contain 1-%d fields", MaxFields)
	}

	sectionIDs := make(map[string]struct{}, len(input.Sections))
	for index := range input.Sections {
		section := &input.Sections[index]
		section.ID = strings.TrimSpace(section.ID)
		section.Title = strings.TrimSpace(section.Title)
		section.Help = strings.TrimSpace(section.Help)
		if section.ID == "" || section.Title == "" || len(section.ID) > maxFieldIDLength || len(section.Title) > maxSectionTitle || len(section.Help) > maxSectionHelp || section.Weight < 0 || section.Weight > 100 {
			return Contract{}, invalid("every section requires a bounded id and title")
		}
		if _, exists := sectionIDs[section.ID]; exists {
			return Contract{}, invalid("section ids must be unique")
		}
		sectionIDs[section.ID] = struct{}{}
	}

	fieldIDs := make(map[string]struct{}, len(input.Fields))
	for index := range input.Fields {
		field := &input.Fields[index]
		field.ID = strings.TrimSpace(field.ID)
		field.SectionID = strings.TrimSpace(field.SectionID)
		field.Label = strings.TrimSpace(field.Label)
		field.Description = strings.TrimSpace(field.Description)
		field.Attestation = strings.TrimSpace(field.Attestation)
		field.Type = normalizedType(field.Type)
		if field.SectionID == "" && len(input.Sections) == 1 {
			field.SectionID = input.Sections[0].ID
		}
		if field.ID == "" || field.Label == "" || len(field.ID) > maxFieldIDLength || len(field.Label) > maxFieldLabel || len(field.Description) > maxFieldHelp {
			return Contract{}, invalid("every field requires a bounded id and label")
		}
		if _, exists := fieldIDs[field.ID]; exists {
			return Contract{}, invalid("field ids must be unique")
		}
		fieldIDs[field.ID] = struct{}{}
		if _, exists := sectionIDs[field.SectionID]; !exists {
			return Contract{}, invalid("%s references an unknown section", field.Label)
		}
		if err := normalizeField(field); err != nil {
			return Contract{}, err
		}
	}
	if err := validateVisibility(&input); err != nil {
		return Contract{}, err
	}
	if err := validateScoringContract(input); err != nil {
		return Contract{}, err
	}
	if err := normalizeScoreProfile(&input); err != nil {
		return Contract{}, err
	}
	return input, nil
}

func normalizedType(value Type) Type {
	switch strings.ToLower(strings.TrimSpace(string(value))) {
	case "text":
		return TypeShortText
	case "number":
		return TypeDecimal
	default:
		return Type(strings.ToLower(strings.TrimSpace(string(value))))
	}
}

func normalizeField(field *Field) error {
	if !slices.Contains([]Type{
		TypeShortText, TypeLongText, TypeEmail, TypeTelephone, TypeURL,
		TypeInteger, TypeDecimal, TypePercentage, TypeCurrency, TypeDate,
		TypeYesNo, TypeSingleSelect, TypeMultiSelect, TypeCheckbox,
		TypeAttestation, TypeFile, TypePhoto, TypeSignature, TypeVendorDocument,
	}, field.Type) {
		return invalid("%s uses unsupported field type %q", field.Label, field.Type)
	}
	field.CollectionIntent = CollectionIntent(strings.ToUpper(strings.TrimSpace(string(field.CollectionIntent))))
	if field.CollectionIntent == "" {
		field.CollectionIntent = IntentCapture
	}
	if !slices.Contains([]CollectionIntent{IntentCapture, IntentConfirmOrCorrect, IntentReplaceHeldDocument}, field.CollectionIntent) {
		return invalid("%s uses an unsupported collection intent", field.Label)
	}
	field.BrowserCachePolicy = BrowserCachePolicy(strings.ToUpper(strings.TrimSpace(string(field.BrowserCachePolicy))))
	if field.BrowserCachePolicy == "" {
		field.BrowserCachePolicy = BrowserCacheAllowed
	}
	if !slices.Contains([]BrowserCachePolicy{BrowserCacheAllowed, BrowserCacheDenied}, field.BrowserCachePolicy) {
		return invalid("%s uses an unsupported browser cache policy", field.Label)
	}
	if field.RecordTarget != nil {
		target := *field.RecordTarget
		target.Key = strings.ToUpper(strings.TrimSpace(target.Key))
		target.RequiredSubjectType = strings.ToUpper(strings.TrimSpace(target.RequiredSubjectType))
		if !validRecordTargetIdentifier(target.Key, 200, true) || !validRecordTargetIdentifier(target.RequiredSubjectType, 80, false) {
			return invalid("%s requires a bounded record target", field.Label)
		}
		field.RecordTarget = &target
	}
	if field.CollectionIntent != IntentCapture && field.RecordTarget == nil {
		return invalid("%s requires a record target for its collection intent", field.Label)
	}
	if field.CollectionIntent == IntentReplaceHeldDocument && field.Type != TypeFile && field.Type != TypeVendorDocument {
		return invalid("%s must use a document field for replacement", field.Label)
	}
	field.Constraints.Currency = strings.ToUpper(strings.TrimSpace(field.Constraints.Currency))
	for index := range field.Options {
		field.Options[index] = strings.TrimSpace(field.Options[index])
	}
	for index := range field.AcceptedFormats {
		field.AcceptedFormats[index] = strings.ToLower(strings.TrimSpace(strings.Split(field.AcceptedFormats[index], ";")[0]))
	}
	if field.Type == TypeYesNo {
		if len(field.Options) == 0 {
			field.Options = []string{"Yes", "No"}
		}
		if !slices.Equal(field.Options, []string{"Yes", "No"}) {
			return invalid("%s must use Yes and No choices", field.Label)
		}
	}
	selection := field.Type == TypeSingleSelect || field.Type == TypeMultiSelect || field.Type == TypeYesNo
	if selection {
		if len(field.Options) < 2 || len(field.Options) > MaxChoices {
			return invalid("%s must define 2-%d choices", field.Label, MaxChoices)
		}
		seen := map[string]struct{}{}
		for _, option := range field.Options {
			if option == "" {
				return invalid("%s contains an empty choice", field.Label)
			}
			if _, exists := seen[option]; exists {
				return invalid("%s contains duplicate choices", field.Label)
			}
			seen[option] = struct{}{}
		}
	} else if len(field.Options) != 0 {
		return invalid("%s cannot define choices", field.Label)
	}
	fileType := field.Type == TypeFile || field.Type == TypePhoto || field.Type == TypeSignature || field.Type == TypeVendorDocument
	if !fileType && len(field.AcceptedFormats) != 0 {
		return invalid("%s cannot define accepted file formats", field.Label)
	}
	for _, format := range field.AcceptedFormats {
		if !slices.Contains([]string{
			"application/pdf", "image/png", "image/jpeg", "text/plain", "text/csv",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		}, format) {
			return invalid("%s contains unsupported file format %q", field.Label, format)
		}
		if field.Type == TypePhoto && format != "image/jpeg" && format != "image/png" {
			return invalid("%s accepts a non-image photo format", field.Label)
		}
		if field.Type == TypeSignature && format != "image/png" {
			return invalid("%s signatures must use image/png", field.Label)
		}
	}
	if field.Type == TypeAttestation {
		if field.Attestation == "" || len(field.Attestation) > maxAttestation {
			return invalid("%s requires bounded attestation text", field.Label)
		}
	} else if field.Attestation != "" {
		return invalid("%s cannot define attestation text", field.Label)
	}
	if err := validateConstraints(field); err != nil {
		return err
	}
	if field.Scoring != nil {
		field.Scoring.ID, field.Scoring.Required = field.ID, field.Required
		if !selection || field.Scoring.Weight < 1 || field.Scoring.Weight > 100 || len(field.Scoring.AnswerScores) == 0 {
			return invalid("%s has invalid scoring", field.Label)
		}
		for answer, points := range field.Scoring.AnswerScores {
			if !slices.Contains(field.Options, answer) || points < 0 || points > 100 {
				return invalid("%s has an invalid answer score", field.Label)
			}
		}
		for _, answer := range field.Scoring.CriticalAnswers {
			if !slices.Contains(field.Options, answer) {
				return invalid("%s has an invalid critical answer", field.Label)
			}
		}
	}
	return nil
}

func validateScoringContract(contract Contract) error {
	sectionFieldWeights := make(map[string]int, len(contract.Sections))
	sectionsWithScoring := make(map[string]struct{}, len(contract.Sections))
	for _, field := range contract.Fields {
		if field.Scoring == nil {
			continue
		}
		sectionsWithScoring[field.SectionID] = struct{}{}
		sectionFieldWeights[field.SectionID] += field.Scoring.Weight
	}

	switch contract.ScoringMode {
	case ScoringNone:
		if len(sectionsWithScoring) != 0 || contract.ScoreProfile != nil {
			return invalid("NONE scoring mode cannot contain scored fields")
		}
	case ScoringRisk:
		for _, section := range contract.Sections {
			if section.Weight != 0 {
				return invalid("section weights require COMPLIANCE scoring mode")
			}
		}
	case ScoringCompliance:
		if len(sectionsWithScoring) == 0 && contract.ScoreProfile == nil {
			return invalid("COMPLIANCE scoring mode requires scored fields")
		}
		if len(sectionsWithScoring) == 0 {
			for _, section := range contract.Sections {
				if section.Weight != 0 {
					return invalid("section weights require legacy compliance field scoring")
				}
			}
			break
		}
		sectionWeightTotal := 0
		for _, section := range contract.Sections {
			_, scored := sectionsWithScoring[section.ID]
			if !scored {
				if section.Weight != 0 {
					return invalid("unscored section %s cannot have compliance weight", section.Title)
				}
				continue
			}
			if section.Weight < 1 || sectionFieldWeights[section.ID] != 100 {
				return invalid("scored section %s and its fields must each allocate 100 percent", section.Title)
			}
			sectionWeightTotal += section.Weight
		}
		if sectionWeightTotal != 100 {
			return invalid("compliance section weights must total 100 percent")
		}
	}
	return nil
}

func validRecordTargetIdentifier(value string, maximum int, allowDot bool) bool {
	if value == "" || len(value) > maximum || strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") || strings.Contains(value, "..") {
		return false
	}
	for _, character := range value {
		if character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || character == '_' || allowDot && character == '.' {
			continue
		}
		return false
	}
	return true
}

func validateConstraints(field *Field) error {
	c := &field.Constraints
	textType := slices.Contains([]Type{TypeShortText, TypeLongText, TypeEmail, TypeTelephone, TypeURL}, field.Type)
	numericType := slices.Contains([]Type{TypeInteger, TypeDecimal, TypePercentage, TypeCurrency}, field.Type)
	selectionType := slices.Contains([]Type{TypeYesNo, TypeSingleSelect, TypeMultiSelect}, field.Type)
	fileType := slices.Contains([]Type{TypeFile, TypePhoto, TypeSignature, TypeVendorDocument}, field.Type)

	if (c.MinLength != nil || c.MaxLength != nil) && !textType {
		return invalid("%s cannot define text length limits", field.Label)
	}
	if textType && !validIntRange(c.MinLength, c.MaxLength, 0, 5000) {
		return invalid("%s has invalid text length limits", field.Label)
	}
	if (c.Minimum != nil || c.Maximum != nil || c.Step != nil || c.DecimalPrecision != nil || c.Currency != "") && !numericType {
		return invalid("%s cannot define numeric limits", field.Label)
	}
	if numericType {
		if !validFloatRange(c.Minimum, c.Maximum) || (c.Step != nil && (*c.Step <= 0 || math.IsNaN(*c.Step) || math.IsInf(*c.Step, 0))) {
			return invalid("%s has invalid numeric limits", field.Label)
		}
		if c.DecimalPrecision != nil && (*c.DecimalPrecision < 0 || *c.DecimalPrecision > 6) {
			return invalid("%s has invalid decimal precision", field.Label)
		}
		if field.Type == TypePercentage {
			zero, hundred := 0.0, 100.0
			if c.Minimum == nil {
				c.Minimum = &zero
			}
			if c.Maximum == nil {
				c.Maximum = &hundred
			}
		}
		if field.Type == TypeCurrency {
			if !slices.Contains([]string{"NGN", "USD", "EUR", "GBP"}, c.Currency) {
				return invalid("%s requires an approved currency", field.Label)
			}
		} else if c.Currency != "" {
			return invalid("%s cannot define a currency", field.Label)
		}
	}
	if (c.MinDate != "" || c.MaxDate != "") && field.Type != TypeDate {
		return invalid("%s cannot define date limits", field.Label)
	}
	if field.Type == TypeDate && !validDateRange(c.MinDate, c.MaxDate) {
		return invalid("%s has invalid date limits", field.Label)
	}
	if (c.MinSelections != nil || c.MaxSelections != nil) && !selectionType {
		return invalid("%s cannot define selection limits", field.Label)
	}
	if selectionType {
		maximum := len(field.Options)
		if field.Type != TypeMultiSelect {
			maximum = 1
		}
		if !validIntRange(c.MinSelections, c.MaxSelections, 0, maximum) {
			return invalid("%s has invalid selection limits", field.Label)
		}
	}
	if (c.MinFiles != nil || c.MaxFiles != nil || c.MaxFileBytes != nil || c.MaxTotalFileBytes != nil) && !fileType {
		return invalid("%s cannot define file limits", field.Label)
	}
	if fileType {
		if !validIntRange(c.MinFiles, c.MaxFiles, 0, 10) || (c.MaxFiles != nil && *c.MaxFiles < 1) {
			return invalid("%s has an invalid file count", field.Label)
		}
		if c.MaxFileBytes != nil && (*c.MaxFileBytes < 1 || *c.MaxFileBytes > 100<<20) {
			return invalid("%s has an invalid file size", field.Label)
		}
		if c.MaxTotalFileBytes != nil && (*c.MaxTotalFileBytes < 1 || *c.MaxTotalFileBytes > 500<<20) {
			return invalid("%s has an invalid total file size", field.Label)
		}
		if c.MaxFileBytes != nil && c.MaxTotalFileBytes != nil && *c.MaxTotalFileBytes < *c.MaxFileBytes {
			return invalid("%s total file size must not be smaller than its per-file limit", field.Label)
		}
		if (field.Type == TypePhoto || field.Type == TypeSignature || field.Type == TypeVendorDocument) && c.MaxFiles != nil && *c.MaxFiles != 1 {
			return invalid("%s accepts one file", field.Label)
		}
		if (field.Type == TypePhoto || field.Type == TypeSignature || field.Type == TypeVendorDocument) && c.MinFiles != nil && *c.MinFiles > 1 {
			return invalid("%s accepts one file", field.Label)
		}
	}
	return nil
}

func validIntRange(minimum, maximum *int, lower, upper int) bool {
	if minimum != nil && (*minimum < lower || *minimum > upper) {
		return false
	}
	if maximum != nil && (*maximum < lower || *maximum > upper) {
		return false
	}
	return minimum == nil || maximum == nil || *minimum <= *maximum
}

func validFloatRange(minimum, maximum *float64) bool {
	for _, value := range []*float64{minimum, maximum} {
		if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0)) {
			return false
		}
	}
	return minimum == nil || maximum == nil || *minimum <= *maximum
}

func validDateRange(minimum, maximum string) bool {
	var minTime, maxTime time.Time
	var err error
	if minimum != "" {
		minTime, err = time.Parse("2006-01-02", minimum)
		if err != nil {
			return false
		}
	}
	if maximum != "" {
		maxTime, err = time.Parse("2006-01-02", maximum)
		if err != nil {
			return false
		}
	}
	return minimum == "" || maximum == "" || !minTime.After(maxTime)
}

func invalid(format string, args ...any) error {
	return errors.Join(ErrInvalid, fmt.Errorf(format, args...))
}
