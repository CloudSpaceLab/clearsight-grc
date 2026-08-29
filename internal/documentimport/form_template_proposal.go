package documentimport

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

const formProposalVersion = "FORM_TEMPLATE_PROPOSAL_V1"

type ProposalPolicy struct {
	MaxFields     int
	MaxSections   int
	MaxUnresolved int
}

func DefaultProposalPolicy() ProposalPolicy {
	return ProposalPolicy{MaxFields: formcontract.MaxFields, MaxSections: formcontract.MaxSections, MaxUnresolved: 500}
}

func (p ProposalPolicy) normalized() ProposalPolicy {
	defaults := DefaultProposalPolicy()
	if p.MaxFields < 1 || p.MaxFields > formcontract.MaxFields {
		p.MaxFields = defaults.MaxFields
	}
	if p.MaxSections < 1 || p.MaxSections > formcontract.MaxSections {
		p.MaxSections = defaults.MaxSections
	}
	if p.MaxUnresolved < 1 || p.MaxUnresolved > 1000 {
		p.MaxUnresolved = defaults.MaxUnresolved
	}
	return p
}

type FormFieldChange struct {
	ID         string             `json:"id"`
	Kind       string             `json:"kind"`
	Field      formcontract.Field `json:"field"`
	Anchor     SourceAnchor       `json:"anchor"`
	Confidence float64            `json:"confidence"`
	Unresolved []string           `json:"unresolved,omitempty"`
}

type ProposalUnresolvedItem struct {
	Code          string        `json:"code"`
	Message       string        `json:"message"`
	FieldChangeID string        `json:"field_change_id,omitempty"`
	Anchor        *SourceAnchor `json:"anchor,omitempty"`
}

type FormProposalProvenance struct {
	ProposalVersion  string `json:"proposal_version"`
	SourceDocumentID string `json:"source_document_id"`
	SourceSHA256     string `json:"source_sha256"`
	SourceVersion    int64  `json:"source_version"`
	ParserVersion    string `json:"parser_version,omitempty"`
	AdapterVersion   string `json:"adapter_version,omitempty"`
	ExtractionStatus string `json:"extraction_status"`
	TabularParser    string `json:"tabular_parser_version,omitempty"`
}

type FormTemplateProposal struct {
	Contract        formcontract.Contract    `json:"contract"`
	FieldChanges    []FormFieldChange        `json:"field_changes"`
	UnresolvedItems []ProposalUnresolvedItem `json:"unresolved_items"`
	Provenance      FormProposalProvenance   `json:"provenance"`
	Truncated       bool                     `json:"truncated"`
}

func ProposeFormTemplate(document Document, policy ProposalPolicy) (FormTemplateProposal, error) {
	policy = policy.normalized()
	if document.ExtractionStatus != ExtractionExtracted && document.ExtractionStatus != ExtractionPartial && document.ExtractionStatus != ExtractionTruncated {
		return FormTemplateProposal{}, errors.New("form proposal requires extracted, partially extracted, or explicitly truncated source content")
	}
	if len(document.Elements) == 0 && (document.Tabular == nil || len(document.Tabular.Resources) == 0) {
		return FormTemplateProposal{}, errors.New("form proposal source contains no usable structured fields")
	}

	builder := formProposalBuilder{
		policy:           policy,
		document:         document,
		sections:         []formcontract.Section{{ID: formcontract.DefaultSectionID, Title: "General"}},
		sectionIDs:       map[string]struct{}{formcontract.DefaultSectionID: {}},
		currentSectionID: formcontract.DefaultSectionID,
		changes:          make([]FormFieldChange, 0, min(policy.MaxFields, 32)),
		unresolved:       make([]ProposalUnresolvedItem, 0, min(policy.MaxUnresolved, 32)),
	}
	if document.ExtractionStatus == ExtractionPartial {
		builder.truncated = true
		builder.addUnresolved(ProposalUnresolvedItem{Code: "SOURCE_PARTIAL", Message: unresolvedMessage("SOURCE_PARTIAL")})
		for _, degradation := range document.Degradations {
			builder.addUnresolved(ProposalUnresolvedItem{Code: degradation.Code, Message: degradation.Message, Anchor: cloneSourceAnchorPointer(degradation.Anchor)})
		}
	}
	builder.consumeElements(document.Elements)
	builder.consumeTabular(document.Tabular)
	if len(builder.changes) == 0 {
		return FormTemplateProposal{}, errors.New("form proposal source contains no supported field candidates")
	}

	fields := make([]formcontract.Field, len(builder.changes))
	for index := range builder.changes {
		fields[index] = builder.changes[index].Field
	}
	contract, err := formcontract.Normalize(formcontract.Contract{
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationAutomatic, AllowModeSwitch: true},
		ScoringMode:  formcontract.ScoringNone,
		Sections:     builder.sections,
		Fields:       fields,
	})
	if err != nil {
		return FormTemplateProposal{}, fmt.Errorf("proposed form contract is invalid: %w", err)
	}
	for index := range builder.changes {
		builder.changes[index].Field = contract.Fields[index]
	}

	provenance := FormProposalProvenance{
		ProposalVersion:  formProposalVersion,
		SourceDocumentID: strings.TrimSpace(document.ID),
		SourceSHA256:     strings.TrimSpace(document.SHA256),
		SourceVersion:    document.Version,
		ParserVersion:    parserVersionFor(document),
		AdapterVersion:   strings.TrimSpace(document.AdapterVersion),
		ExtractionStatus: string(document.ExtractionStatus),
	}
	if document.Tabular != nil {
		provenance.TabularParser = strings.TrimSpace(document.Tabular.ParserVersion)
	}
	return FormTemplateProposal{
		Contract:        contract,
		FieldChanges:    builder.changes,
		UnresolvedItems: builder.unresolved,
		Provenance:      provenance,
		Truncated:       builder.truncated || document.ContentTruncated || document.ExtractionStatus == ExtractionPartial || document.ExtractionStatus == ExtractionTruncated,
	}, nil
}

type formProposalBuilder struct {
	policy           ProposalPolicy
	document         Document
	sections         []formcontract.Section
	sectionIDs       map[string]struct{}
	currentSectionID string
	changes          []FormFieldChange
	unresolved       []ProposalUnresolvedItem
	truncated        bool
}

func (b *formProposalBuilder) consumeElements(elements []ExtractedElement) {
	for _, element := range elements {
		switch element.Kind {
		case ElementHeading:
			b.useHeading(element.Text)
		case ElementFormControl:
			b.addControl(element)
		}
		if b.truncated && len(b.changes) >= b.policy.MaxFields {
			return
		}
	}
}

func (b *formProposalBuilder) useHeading(value string) {
	title := strings.TrimSpace(value)
	if title == "" {
		return
	}
	sectionID := stableFormProposalID("section", b.document.SHA256, title)
	if _, exists := b.sectionIDs[sectionID]; exists {
		b.currentSectionID = sectionID
		return
	}
	if len(b.sections) >= b.policy.MaxSections {
		b.truncated = true
		b.addUnresolved(ProposalUnresolvedItem{Code: "SECTION_LIMIT_REACHED", Message: unresolvedMessage("SECTION_LIMIT_REACHED")})
		return
	}
	b.sections = append(b.sections, formcontract.Section{ID: sectionID, Title: truncateProposalText(title, 200)})
	b.sectionIDs[sectionID] = struct{}{}
	b.currentSectionID = sectionID
}

func (b *formProposalBuilder) addControl(element ExtractedElement) {
	if element.Control == nil {
		return
	}
	if len(b.changes) >= b.policy.MaxFields {
		b.markFieldLimit()
		return
	}

	label := strings.TrimSpace(element.Control.Label)
	unresolved := []string{"REQUIREDNESS_UNKNOWN"}
	if label == "" {
		label = strings.TrimSpace(element.Text)
	}
	if label == "" {
		label = fmt.Sprintf("Imported field %d", len(b.changes)+1)
		unresolved = append(unresolved, "LABEL_MISSING")
	}
	fieldType, options, typeUnresolved := formTypeForControl(*element.Control)
	unresolved = append(unresolved, typeUnresolved...)
	anchor := element.Anchor
	fieldID := stableFormProposalID("field", b.document.SHA256, anchorIdentity(anchor), label)
	changeID := stableFormProposalID("change", fieldID)
	field := formcontract.Field{
		ID:                 fieldID,
		SectionID:          b.currentSectionID,
		Label:              truncateProposalText(label, 200),
		Type:               fieldType,
		Description:        truncateProposalText(strings.TrimSpace(element.Control.Help), 1000),
		Options:            append([]string(nil), options...),
		CollectionIntent:   formcontract.IntentCapture,
		BrowserCachePolicy: formcontract.BrowserCacheAllowed,
	}
	confidence := 0.95
	if len(typeUnresolved) > 0 || slicesContains(unresolved, "LABEL_MISSING") {
		confidence = 0.75
	}
	b.changes = append(b.changes, FormFieldChange{
		ID: changeID, Kind: "ADD_FIELD", Field: field, Anchor: anchor,
		Confidence: confidence, Unresolved: append([]string(nil), unresolved...),
	})
	for _, code := range unresolved {
		b.addUnresolved(ProposalUnresolvedItem{
			Code: code, Message: unresolvedMessage(code), FieldChangeID: changeID,
			Anchor: cloneSourceAnchor(anchor),
		})
	}
}

func formTypeForControl(control FormControl) (formcontract.Type, []string, []string) {
	switch strings.ToUpper(strings.TrimSpace(control.Kind)) {
	case "DROPDOWN":
		options := boundedUniqueOptions(control.Options)
		if len(options) >= 2 {
			return formcontract.TypeSingleSelect, options, nil
		}
		return formcontract.TypeShortText, nil, []string{"OPTIONS_MISSING"}
	case "CHECKBOX":
		return formcontract.TypeYesNo, []string{"Yes", "No"}, nil
	case "DATE":
		return formcontract.TypeDate, nil, nil
	case "TEXT":
		return formcontract.TypeShortText, nil, nil
	default:
		return formcontract.TypeShortText, nil, []string{"TYPE_UNKNOWN"}
	}
}

func (b *formProposalBuilder) consumeTabular(metadata *TabularMetadata) {
	if metadata == nil {
		return
	}
	for _, resource := range metadata.Resources {
		if len(resource.Fields) == 0 {
			continue
		}
		sectionID := stableFormProposalID("section", b.document.SHA256, "tabular", resource.Name)
		if _, exists := b.sectionIDs[sectionID]; !exists {
			if len(b.sections) < b.policy.MaxSections {
				title := strings.TrimSpace(resource.Name)
				if title == "" {
					title = "Imported data"
				}
				b.sections = append(b.sections, formcontract.Section{ID: sectionID, Title: truncateProposalText(title, 200)})
				b.sectionIDs[sectionID] = struct{}{}
			} else {
				sectionID = formcontract.DefaultSectionID
				b.truncated = true
				b.addUnresolved(ProposalUnresolvedItem{Code: "SECTION_LIMIT_REACHED", Message: unresolvedMessage("SECTION_LIMIT_REACHED")})
			}
		}

		for fieldIndex, sourceField := range resource.Fields {
			if len(b.changes) >= b.policy.MaxFields {
				b.markFieldLimit()
				return
			}
			label := strings.TrimSpace(sourceField.Name)
			if label == "" {
				label = fmt.Sprintf("Column %d", fieldIndex+1)
			}
			anchor := SourceAnchor{Sheet: strings.TrimSpace(resource.Name), RowStart: 1, RowEnd: max(1, resource.RowsTotal)}
			fieldID := stableFormProposalID("field", b.document.SHA256, "tabular", resource.Name, sourceField.Name)
			changeID := stableFormProposalID("change", fieldID)
			field := formcontract.Field{
				ID:                 fieldID,
				SectionID:          sectionID,
				Label:              truncateProposalText(label, 200),
				Type:               formTypeForNativeType(sourceField.NativeType),
				CollectionIntent:   formcontract.IntentCapture,
				BrowserCachePolicy: formcontract.BrowserCacheAllowed,
			}
			unresolved := []string{"REQUIREDNESS_UNKNOWN", "OPTIONS_NOT_INFERRED"}
			b.changes = append(b.changes, FormFieldChange{
				ID: changeID, Kind: "ADD_FIELD", Field: field, Anchor: anchor,
				Confidence: 0.70, Unresolved: append([]string(nil), unresolved...),
			})
			for _, code := range unresolved {
				b.addUnresolved(ProposalUnresolvedItem{
					Code: code, Message: unresolvedMessage(code), FieldChangeID: changeID,
					Anchor: cloneSourceAnchor(anchor),
				})
			}
		}
	}
}

func formTypeForNativeType(value string) formcontract.Type {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "integer", "int", "int64", "int32":
		return formcontract.TypeInteger
	case "number", "decimal", "float", "float64", "float32":
		return formcontract.TypeDecimal
	case "boolean", "bool":
		return formcontract.TypeYesNo
	case "date", "datetime", "timestamp":
		return formcontract.TypeDate
	default:
		return formcontract.TypeShortText
	}
}

func (b *formProposalBuilder) markFieldLimit() {
	if b.truncated && containsProposalCode(b.unresolved, "FIELD_LIMIT_REACHED") {
		return
	}
	b.truncated = true
	b.addUnresolved(ProposalUnresolvedItem{Code: "FIELD_LIMIT_REACHED", Message: unresolvedMessage("FIELD_LIMIT_REACHED")})
}

func (b *formProposalBuilder) addUnresolved(item ProposalUnresolvedItem) {
	if len(b.unresolved) >= b.policy.MaxUnresolved {
		b.truncated = true
		return
	}
	b.unresolved = append(b.unresolved, item)
}

func stableFormProposalID(prefix string, parts ...string) string {
	normalized := make([]string, 0, len(parts)+1)
	normalized = append(normalized, strings.ToLower(strings.TrimSpace(prefix)))
	for _, part := range parts {
		normalized = append(normalized, strings.ToLower(strings.TrimSpace(part)))
	}
	digest := sha256.Sum256([]byte(strings.Join(normalized, "\x1f")))
	return prefix + "_" + hex.EncodeToString(digest[:12])
}

func anchorIdentity(anchor SourceAnchor) string {
	return fmt.Sprintf("p=%d|s=%s|r=%d-%d|para=%s|table=%s|cell=%s", anchor.Page, anchor.Sheet, anchor.RowStart, anchor.RowEnd, anchor.Paragraph, anchor.Table, anchor.Cell)
}

func boundedUniqueOptions(values []string) []string {
	result := make([]string, 0, min(len(values), formcontract.MaxChoices))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = truncateProposalText(strings.TrimSpace(value), 200)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
		if len(result) >= formcontract.MaxChoices {
			break
		}
	}
	return result
}

func truncateProposalText(value string, maximum int) string {
	value = strings.TrimSpace(value)
	if len(value) <= maximum {
		return value
	}
	return truncateUTF8(value, maximum)
}

func unresolvedMessage(code string) string {
	switch code {
	case "SOURCE_PARTIAL":
		return "Only the successfully extracted portions of this source were used; review the retained source gaps before approving the draft."
	case "REQUIREDNESS_UNKNOWN":
		return "The source does not establish whether this field is mandatory; an author must decide."
	case "OPTIONS_NOT_INFERRED":
		return "Choice options were not inferred because retained tabular metadata does not contain bounded categorical samples."
	case "OPTIONS_MISSING":
		return "The source identifies a selection control but does not provide at least two usable choices."
	case "TYPE_UNKNOWN":
		return "The source control type is not directly supported and was proposed as short text for review."
	case "LABEL_MISSING":
		return "The source control has no reliable label; a placeholder label requires author review."
	case "SECTION_LIMIT_REACHED":
		return "The proposal section limit was reached; additional source structure requires author review."
	case "FIELD_LIMIT_REACHED":
		return "The proposal field limit was reached; additional source fields were omitted from this proposal."
	default:
		return "The source contains an unresolved form-authoring ambiguity."
	}
}

func cloneSourceAnchorPointer(value *SourceAnchor) *SourceAnchor {
	if value == nil {
		return nil
	}
	return cloneSourceAnchor(*value)
}

func containsProposalCode(items []ProposalUnresolvedItem, code string) bool {
	for _, item := range items {
		if item.Code == code {
			return true
		}
	}
	return false
}

func slicesContains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
