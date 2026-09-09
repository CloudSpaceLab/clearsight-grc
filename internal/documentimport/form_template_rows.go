package documentimport

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type spreadsheetLayout struct {
	kind    string
	label   int
	section int
	headers []string
}

// Recognition is deliberately narrow: ordinary data tables keep their
// column-oriented proposals. A source row is never an applicability decision.
func classifySpreadsheet(headers []string) spreadsheetLayout {
	result := spreadsheetLayout{label: -1, section: -1, headers: headers}
	hasContext, finding, recommendation, status := false, -1, false, false
	for index, header := range headers {
		switch spreadsheetHeader(header) {
		case "requirement", "requirements", "requirement checklist item", "checklist item":
			result.label = index
		case "control area", "section", "category":
			result.section = index
		case "applicability", "evidence required", "timeline frequency":
			hasContext = true
		case "finding", "findings":
			finding = index
		case "recommendation", "recommendations":
			recommendation = true
		case "status":
			status = true
		}
	}
	if finding >= 0 && recommendation && status {
		result.kind, result.label = "FINDINGS", finding
	} else if result.label >= 0 && hasContext {
		result.kind = "REQUIREMENTS"
	}
	return result
}

func spreadsheetHeader(value string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(value), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }), " ")
}

func (b *formProposalBuilder) consumeSpreadsheetRows() error {
	b.rowSheets = make(map[string]bool)
	layouts := make(map[string]spreadsheetLayout)
	for _, element := range b.document.Elements {
		if element.Kind != ElementTable || element.Anchor.Sheet == "" || element.Anchor.RowStart < 1 || element.Anchor.RowStart != element.Anchor.RowEnd || len(element.Values) != 1 {
			continue
		}
		sheet := element.Anchor.Sheet
		values := element.Values[0]
		layout, found := layouts[sheet]
		if !found {
			layout = classifySpreadsheet(values)
			layouts[sheet] = layout
			if layout.kind != "" {
				b.rowSheets[sheet] = true
			}
			continue
		}
		if layout.kind == "" {
			continue
		}
		if len(b.changes) >= b.policy.MaxFields {
			b.markFieldLimit()
			continue
		}
		b.addSpreadsheetRow(layout, element)
	}
	// Older extracted documents retain only column metadata/display text.
	// Re-extraction is required; treating those headers as questions is unsafe.
	if b.document.Tabular != nil && b.document.Tabular.Format == TabularXLSX {
		for _, resource := range b.document.Tabular.Resources {
			headers := make([]string, len(resource.Fields))
			for index, field := range resource.Fields {
				headers[index] = field.Name
			}
			if classifySpreadsheet(headers).kind != "" && !b.rowSheets[resource.Name] {
				return errors.New("The checklist or findings register needs its source rows extracted again before a form draft can be prepared. Upload the source file again, then review the proposed fields.")
			}
		}
	}
	return nil
}

func (b *formProposalBuilder) addSpreadsheetRow(layout spreadsheetLayout, element ExtractedElement) {
	values := element.Values[0]
	if layout.label >= len(values) || strings.TrimSpace(values[layout.label]) == "" {
		b.addUnresolved(ProposalUnresolvedItem{Code: "ROW_LABEL_MISSING", Message: "A source row has no requirement or finding text. Review the row before adding a question.", Anchor: cloneSourceAnchor(element.Anchor)})
		return
	}
	label := strings.TrimSpace(values[layout.label])
	section := element.Anchor.Sheet
	if layout.section >= 0 && layout.section < len(values) && strings.TrimSpace(values[layout.section]) != "" {
		section = strings.TrimSpace(values[layout.section])
	}
	b.currentSectionID = formcontract.DefaultSectionID
	b.useHeading(section)
	code := "REQUIREMENT_SCOPE_REVIEW"
	message := "Confirm this requirement's source, applicability, timing and responsible organization before using the question. An answer does not prove the requirement is satisfied."
	context := []string{}
	if layout.kind == "FINDINGS" {
		label = "Update on finding: " + label
		code = "HISTORICAL_FINDING_REVIEW"
		message = "Match this historical finding to the correct vendor and service, confirm its current status, and separate internal actions from vendor questions. Source dates and ratings are historical; no vendor has been matched."
		context = append(context, "Historical source record. Confirm the current position before using this question.")
	}
	columnOrder := make([]int, len(layout.headers))
	for index := range columnOrder {
		columnOrder[index] = index
	}
	if layout.kind == "FINDINGS" {
		sort.SliceStable(columnOrder, func(i, j int) bool {
			return historicalContextPriority(layout.headers[columnOrder[i]]) < historicalContextPriority(layout.headers[columnOrder[j]])
		})
	}
	for _, index := range columnOrder {
		header := layout.headers[index]
		if index >= len(values) || strings.TrimSpace(values[index]) == "" {
			continue
		}
		// Retain the full statement in the description when its label is long.
		if index == layout.label && layout.kind != "FINDINGS" && len(label) <= 200 {
			continue
		}
		context = append(context, fmt.Sprintf("%s: %s", strings.TrimSpace(header), strings.TrimSpace(values[index])))
	}
	description := strings.Join(context, "\n")
	fieldID := stableFormProposalID("field", b.document.SHA256, layout.kind, anchorIdentity(element.Anchor))
	changeID := stableFormProposalID("change", fieldID)
	unresolved := []string{code, "REQUIREDNESS_UNKNOWN"}
	if len(label) > 200 || len(description) > 1000 {
		unresolved = append(unresolved, "ROW_CONTEXT_TRUNCATED")
	}
	field := formcontract.Field{ID: fieldID, SectionID: b.currentSectionID, Label: truncateProposalText(label, 200), Description: truncateProposalText(description, 1000), Type: formcontract.TypeLongText, CollectionIntent: formcontract.IntentCapture, BrowserCachePolicy: formcontract.BrowserCacheAllowed}
	b.changes = append(b.changes, FormFieldChange{ID: changeID, Kind: "ADD_FIELD", Field: field, Anchor: element.Anchor, Confidence: 0.70, Unresolved: unresolved})
	for _, itemCode := range unresolved {
		itemMessage := unresolvedMessage(itemCode)
		if itemCode == code {
			itemMessage = message
		}
		if itemCode == "ROW_CONTEXT_TRUNCATED" {
			itemMessage = "The source row is longer than the question preview. Review the complete source row before approving the form."
		}
		b.addUnresolved(ProposalUnresolvedItem{Code: itemCode, Message: itemMessage, FieldChangeID: changeID, Anchor: cloneSourceAnchor(element.Anchor)})
	}
}

func historicalContextPriority(header string) int {
	switch spreadsheetHeader(header) {
	case "status", "timeline", "deadline", "due date", "date of assessment", "severity", "overall rating":
		return 0
	case "service provider", "services offered", "vendor", "service":
		return 1
	case "recommendation", "recommendations":
		return 2
	case "finding", "findings":
		return 3
	default:
		return 4
	}
}
