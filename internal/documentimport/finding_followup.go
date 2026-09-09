package documentimport

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

const findingFollowUpVersion = "FINDING_FOLLOW_UP_V2"

// Keeps optional choice metadata comfortably below the durable provenance budget.
const maxFindingAssessments = 200

// FindingAssessment identifies source rows, never a matched bank vendor or recipient.
type FindingAssessment struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Sheet        string `json:"sheet"`
	RowStart     int    `json:"row_start"`
	RowEnd       int    `json:"row_end"`
	FindingCount int    `json:"finding_count"`
}

type findingAssessmentRows struct {
	metadata FindingAssessment
	context  [3]string
	headers  []string
	columns  map[string]int
	rows     []ExtractedElement
}

// FindingFollowUpAssessments requires complete structured extraction. Callers offering
// ordinary row proposals may ignore its error; ambiguity must never trigger grouping.
func FindingFollowUpAssessments(document Document) ([]FindingAssessment, error) {
	groups, err := findingAssessmentGroups(document)
	if err != nil {
		return nil, err
	}
	result := make([]FindingAssessment, len(groups))
	for i := range groups {
		result[i] = groups[i].metadata
	}
	return result, nil
}

func findingHeaders(values []string) (map[string]int, bool, error) {
	columns := map[string]int{}
	duplicate := false
	for i, value := range values {
		key := spreadsheetHeader(value)
		if _, exists := columns[key]; exists && key != "" {
			duplicate = true
		}
		columns[key] = i
	}
	for _, key := range []string{"s n", "service provider", "services offered", "date of assessment", "findings", "recommendations"} {
		if _, exists := columns[key]; !exists {
			return nil, false, nil
		}
	}
	if duplicate {
		return nil, false, errors.New("The findings register has repeated column names. Give each source column a distinct heading, then upload the file again.")
	}
	return columns, true, nil
}

func findingAssessmentGroups(document Document) ([]findingAssessmentRows, error) {
	if document.ExtractionStatus != ExtractionExtracted || document.ContentTruncated || document.SectionsOmitted > 0 || document.SectionsTotal > len(document.Sections) || len(document.Degradations) > 0 {
		return nil, errors.New("Finding follow-up needs all source rows. Complete extraction or upload the complete source file again.")
	}
	if document.Tabular != nil && (document.Tabular.FatalError != "" || document.Tabular.RowsRejected > 0 || len(document.Tabular.RowErrors) > 0) {
		return nil, errors.New("Some spreadsheet rows could not be read. Correct the source rows and upload the file again before preparing finding follow-up.")
	}
	sheets := map[string][]ExtractedElement{}
	order := []string{}
	for _, element := range document.Elements {
		if element.Kind != ElementTable || element.Anchor.Sheet == "" {
			continue
		}
		if _, exists := sheets[element.Anchor.Sheet]; !exists {
			order = append(order, element.Anchor.Sheet)
		}
		sheets[element.Anchor.Sheet] = append(sheets[element.Anchor.Sheet], element)
	}
	groups := []findingAssessmentRows{}
	recognizedSheets := map[string]bool{}
	for _, sheet := range order {
		rows := sheets[sheet]
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].Anchor.RowStart < rows[j].Anchor.RowStart })
		if len(rows[0].Values) != 1 {
			return nil, findingRowError(rows[0].Anchor, "has no retained cells; upload the source file again")
		}
		columns, recognized, err := findingHeaders(rows[0].Values[0])
		if err != nil {
			return nil, err
		}
		if !recognized {
			continue
		}
		recognizedSheets[sheet] = true
		retained := map[int]bool{}
		quotes := map[int]bool{}
		for _, section := range document.Sections {
			if section.Sheet == sheet && section.RowStart == section.RowEnd && strings.TrimSpace(section.Text) != "" {
				quotes[section.RowStart] = true
			}
		}
		for _, row := range rows {
			if row.Anchor.RowStart < 1 || row.Anchor.RowStart != row.Anchor.RowEnd || retained[row.Anchor.RowStart] || len(row.Values) != 1 || !quotes[row.Anchor.RowStart] {
				return nil, findingRowError(row.Anchor, "has missing or conflicting retained source rows; upload the complete source file again")
			}
			retained[row.Anchor.RowStart] = true
		}
		for row := range quotes {
			if !retained[row] {
				return nil, findingRowError(SourceAnchor{Sheet: sheet, RowStart: row}, "has no retained cells; upload the source file again")
			}
		}
		if document.Tabular != nil {
			// Tabular rows include formatted/explicitly empty cells; extraction
			// SectionsTotal counts nonempty content before retention limits. A
			// complete section receipt plus the exact row checks above therefore
			// proves retention without treating blank separators as missing findings.
			completeSections := document.SectionsTotal > 0 && document.SectionsTotal == len(document.Sections)
			for _, resource := range document.Tabular.Resources {
				if resource.Name == sheet && (resource.RowsRejected > 0 || (!completeSections && resource.RowsTotal > len(rows)-1)) {
					return nil, errors.New("The findings register has rows missing from extraction. Upload the complete source file again.")
				}
			}
		}
		bySerial := map[string]int{}
		for _, row := range rows[1:] {
			values := row.Values[0]
			if _, repeated, err := findingHeaders(values); repeated || err != nil {
				return nil, findingRowError(row.Anchor, "repeats the assessment headings; separate the registers before preparing follow-up")
			}
			serial, finding := findingCell(values, columns, "s n"), findingCell(values, columns, "findings")
			if serial == "" {
				if finding != "" {
					return nil, findingRowError(row.Anchor, "has a finding without S/N; record its assessment number or merge only the intended assessment rows, then upload again")
				}
				continue
			}
			index, exists := bySerial[serial]
			if !exists {
				if len(groups) >= maxFindingAssessments {
					return nil, errors.New("This register contains more than 200 source assessments. Split it into smaller complete registers before preparing finding follow-up.")
				}
				index = len(groups)
				bySerial[serial] = index
				groups = append(groups, findingAssessmentRows{metadata: FindingAssessment{ID: stableFormProposalID("assessment", document.SHA256, sheet, serial, anchorIdentity(row.Anchor)), Sheet: sheet, RowStart: row.Anchor.RowStart}, headers: rows[0].Values[0], columns: columns})
			}
			group := &groups[index]
			group.metadata.RowEnd = row.Anchor.RowEnd
			for i, key := range []string{"service provider", "services offered", "date of assessment"} {
				value := findingCell(values, columns, key)
				if value != "" && group.context[i] != "" && value != group.context[i] {
					return nil, findingRowError(row.Anchor, "conflicts with the vendor, service or assessment date recorded for its S/N; correct the assessment rows before preparing follow-up")
				}
				if value != "" {
					group.context[i] = value
				}
			}
			// A numbered header contributes context even when it has no finding.
			if finding != "" {
				group.rows = append(group.rows, row)
				group.metadata.FindingCount++
			}
		}
	}
	if document.Tabular != nil {
		for _, resource := range document.Tabular.Resources {
			headers := make([]string, len(resource.Fields))
			for i, field := range resource.Fields {
				headers[i] = field.Name
			}
			_, recognized, headerErr := findingHeaders(headers)
			if headerErr != nil {
				return nil, headerErr
			}
			if recognized && !recognizedSheets[resource.Name] {
				return nil, errors.New("The findings register needs its source rows extracted again. Upload the complete source file before preparing finding follow-up.")
			}
		}
	}
	result := make([]findingAssessmentRows, 0, len(groups))
	for _, group := range groups {
		if len(group.rows) == 0 {
			continue
		}
		if group.context[0] == "" || group.context[1] == "" || group.context[2] == "" {
			return nil, findingRowError(group.rows[0].Anchor, "needs a service provider, service and assessment date for its S/N before finding follow-up can be prepared")
		}
		group.metadata.Label = truncateProposalText(strings.Join(group.context[:], " · "), 200)
		result = append(result, group)
	}
	return result, nil
}

func hasFindingFollowUpHeaders(document Document) bool {
	for _, element := range document.Elements {
		if element.Kind == ElementTable && len(element.Values) == 1 {
			if _, recognized, err := findingHeaders(element.Values[0]); recognized || err != nil {
				return true
			}
		}
	}
	if document.Tabular != nil {
		for _, resource := range document.Tabular.Resources {
			headers := make([]string, len(resource.Fields))
			for i, field := range resource.Fields {
				headers[i] = field.Name
			}
			if _, recognized, err := findingHeaders(headers); recognized || err != nil {
				return true
			}
		}
	}
	return false
}

func findingCell(values []string, columns map[string]int, key string) string {
	index, exists := columns[key]
	if !exists || index >= len(values) {
		return ""
	}
	return strings.TrimSpace(values[index])
}

func findingRowError(anchor SourceAnchor, message string) error {
	return fmt.Errorf("%s row %d %s.", anchor.Sheet, anchor.RowStart, message)
}

// ProposeFindingFollowUp prepares one complete, explicitly selected source assessment.
func ProposeFindingFollowUp(document Document, assessmentID string, policy ProposalPolicy) (FormTemplateProposal, error) {
	groups, err := findingAssessmentGroups(document)
	if err != nil {
		return FormTemplateProposal{}, err
	}
	var selected *findingAssessmentRows
	for i := range groups {
		if groups[i].metadata.ID == assessmentID {
			selected = &groups[i]
			break
		}
	}
	if selected == nil {
		return FormTemplateProposal{}, errors.New("Choose an available source assessment before preparing finding follow-up.")
	}
	policy = policy.normalized()
	if len(selected.rows) > policy.MaxSections || len(selected.rows) > policy.MaxFields/5 {
		return FormTemplateProposal{}, errors.New("This assessment exceeds the form's question or section limit. Split the source assessment into complete smaller assessments, then upload it again.")
	}
	p := FormTemplateProposal{Provenance: FormProposalProvenance{ProposalVersion: findingFollowUpVersion, SourceDocumentID: document.ID, SourceSHA256: document.SHA256, SourceVersion: document.Version, ParserVersion: parserVersionFor(document), AdapterVersion: document.AdapterVersion, ExtractionStatus: string(document.ExtractionStatus), FindingAssessments: []FindingAssessment{selected.metadata}}}
	if document.Tabular != nil {
		p.Provenance.TabularParser = document.Tabular.ParserVersion
	}
	p.Contract.Presentation = formcontract.Presentation{DefaultMode: formcontract.PresentationAutomatic, AllowModeSwitch: true}
	p.Contract.ScoringMode = formcontract.ScoringNone
	p.UnresolvedItems = []ProposalUnresolvedItem{{Code: "HISTORICAL_FINDING_REVIEW", Message: "Confirm the selected vendor, service, current finding status and recipient before approving this form. Source dates and ratings are historical. A response does not prove the finding is resolved."}}
	for index, row := range selected.rows {
		sectionID := stableFormProposalID("section", selected.metadata.ID, anchorIdentity(row.Anchor))
		title := fmt.Sprintf("Finding %d: %s", index+1, findingCell(row.Values[0], selected.columns, "findings"))
		context := selected.rowContext(row)
		shortened := len(title) > 200 || len(context) > 1000 || len(strings.Join(selected.context[:], " · ")) > 200
		p.Contract.Sections = append(p.Contract.Sections, formcontract.Section{ID: sectionID, Title: truncateProposalText(title, 200), Help: truncateProposalText(context, 1000)})
		owner := findingCell(row.Values[0], selected.columns, "responsibility")
		if owner == "" {
			owner = "not recorded"
		}
		for fieldIndex, definition := range []struct {
			label string
			kind  formcontract.Type
			help  string
		}{
			{"Response to finding", formcontract.TypeLongText, "Explain the current position and any correction to this finding."},
			{"Action or explanation", formcontract.TypeLongText, "Describe the action taken or proposed, or explain why no action is needed."},
			{"Remediation owner", formcontract.TypeShortText, "Recorded responsibility: " + owner + ". Confirm who will complete the proposed action."},
			{"Proposed completion date", formcontract.TypeDate, "Enter a proposed date when further action is needed. The source deadline remains historical."},
			{"Supporting evidence", formcontract.TypeFile, "Attach a PDF, Word document or spreadsheet that supports this response. Uploading a file does not confirm the finding is resolved."},
		} {
			fieldID := stableFormProposalID("field", findingFollowUpVersion, selected.metadata.ID, anchorIdentity(row.Anchor), definition.label)
			field := formcontract.Field{ID: fieldID, SectionID: sectionID, Label: definition.label, Type: definition.kind, Required: fieldIndex < 3, Description: truncateProposalText(definition.help, 1000), CollectionIntent: formcontract.IntentCapture, BrowserCachePolicy: formcontract.BrowserCacheAllowed}
			if fieldIndex == 4 {
				field.AcceptedFormats = []string{"application/pdf", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"}
				field.BrowserCachePolicy = formcontract.BrowserCacheDenied
			}
			change := FormFieldChange{ID: stableFormProposalID("change", fieldID), Kind: "ADD_FIELD", Field: field, Anchor: row.Anchor, Confidence: 0.70, Unresolved: []string{"HISTORICAL_FINDING_REVIEW"}}
			if shortened || len(definition.help) > 1000 {
				change.Unresolved = append(change.Unresolved, "ROW_CONTEXT_TRUNCATED")
				p.UnresolvedItems = append(p.UnresolvedItems, ProposalUnresolvedItem{Code: "ROW_CONTEXT_TRUNCATED", Message: "The question preview shortens this source row. Review the complete source row before approving the form.", FieldChangeID: change.ID, Anchor: cloneSourceAnchor(row.Anchor)})
			}
			p.FieldChanges = append(p.FieldChanges, change)
			p.Contract.Fields = append(p.Contract.Fields, field)
		}
	}
	if len(p.UnresolvedItems) > policy.MaxUnresolved {
		// Every affected field retains its warning code. Bound the summary without
		// dropping findings or rejecting a complete assessment for long source text.
		p.UnresolvedItems = p.UnresolvedItems[:policy.MaxUnresolved]
		p.UnresolvedItems[len(p.UnresolvedItems)-1] = ProposalUnresolvedItem{Code: "ROW_CONTEXT_TRUNCATED", Message: "Some question previews shorten the source rows. Review each complete source row and confirm the vendor, service, current finding status and recipient before approval. Source dates and ratings are historical; a response does not prove a finding is resolved."}
	}
	p.Contract, err = formcontract.Normalize(p.Contract)
	if err != nil {
		return FormTemplateProposal{}, fmt.Errorf("finding follow-up form needs correction: %w", err)
	}
	for i := range p.FieldChanges {
		p.FieldChanges[i].Field = p.Contract.Fields[i]
	}
	return p, nil
}

func (g findingAssessmentRows) rowContext(row ExtractedElement) string {
	parts := []string{"Historical source record. Confirm its current status before sending.", "Service provider: " + g.context[0], "Services offered: " + g.context[1], "Date of assessment: " + g.context[2]}
	order := make([]int, len(g.headers))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		return historicalContextPriority(g.headers[order[i]]) < historicalContextPriority(g.headers[order[j]])
	})
	for _, i := range order {
		key := spreadsheetHeader(g.headers[i])
		if key == "service provider" || key == "services offered" || key == "date of assessment" {
			continue
		}
		if value := findingCell(row.Values[0], g.columns, key); value != "" {
			parts = append(parts, strings.TrimSpace(g.headers[i])+": "+value)
		}
	}
	return strings.Join(parts, "\n")
}
