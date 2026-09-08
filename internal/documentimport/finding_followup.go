package documentimport

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

const FindingFollowUpVersion = "FINDING_FOLLOW_UP_V1"

// Retained XLSX sections label columns explicitly. Multiline values end only at
// the next column marker; they must not be split into independent findings.
var retainedColumn = regexp.MustCompile(`(?m)^Column ([0-9]+): `)

func retainedColumns(text string) map[int]string {
	result := map[int]string{}
	matches := retainedColumn.FindAllStringSubmatchIndex(text, -1)
	for i, match := range matches {
		column, _ := strconv.Atoi(text[match[2]:match[3]])
		end := len(text)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		result[column] = strings.TrimSpace(text[match[1]:end])
	}
	return result
}

func proposeFindingFollowUp(document Document, policy ProposalPolicy) (FormTemplateProposal, bool, error) {
	var result FormTemplateProposal
	if document.Tabular == nil || document.Tabular.Format != TabularXLSX {
		return result, false, nil
	}
	matched := false
	for _, resource := range document.Tabular.Resources {
		columns := map[string]int{}
		for i, field := range resource.Fields {
			columns[strings.ToUpper(strings.TrimSpace(field.Name))] = i + 1
		}
		if columns["FINDINGS"] == 0 || columns["RECOMMENDATIONS"] == 0 || columns["SERVICE PROVIDER"] == 0 || columns["SERVICES OFFERED"] == 0 || columns["S/N"] == 0 {
			continue
		}
		matched = true
		if document.ExtractionStatus != ExtractionExtracted || document.ContentTruncated || document.SectionsOmitted > 0 {
			return result, true, fmt.Errorf("The findings register has missing source content. Correct the extraction before creating a follow-up form.")
		}
		var groupID, groupLabel, serial string
		contextValues := map[string]string{}
		count := 0
		for _, row := range document.Sections {
			if row.Sheet != resource.Name || row.RowStart <= 1 {
				continue
			}
			values := retainedColumns(row.Text)
			get := func(name string) string { return values[columns[name]] }
			if get("FINDINGS") == "" {
				continue
			}
			if next := get("S/N"); next != "" && (next != serial || groupID == "") {
				serial = next
				contextValues = map[string]string{}
				for _, key := range []string{"SERVICE PROVIDER", "SERVICES OFFERED", "ASSESSOR", "DATE OF ASSESSMENT", "BUSINESS OWNER", "OVERALL RATING", "RESPONSIBILITY"} {
					contextValues[key] = get(key)
				}
				if contextValues["SERVICE PROVIDER"] == "" || contextValues["SERVICES OFFERED"] == "" || contextValues["DATE OF ASSESSMENT"] == "" {
					return result, true, fmt.Errorf("Assessment %s is missing its vendor, service or assessment date. Correct the source before creating a follow-up form.", serial)
				}
				groupID = stableFormProposalID("assessment", document.SHA256, resource.Name, strconv.Itoa(row.RowStart))
				groupLabel = fmt.Sprintf("Assessment %s: %s — %s — %s", serial, contextValues["SERVICE PROVIDER"], contextValues["SERVICES OFFERED"], contextValues["DATE OF ASSESSMENT"])
				count = 0
			}
			if groupID == "" {
				return result, true, fmt.Errorf("Finding at %s row %d has no assessment group. Correct the source before creating a follow-up form.", row.Sheet, row.RowStart)
			}
			// A populated continuation value must agree with the group's baseline.
			for _, key := range []string{"SERVICE PROVIDER", "SERVICES OFFERED", "ASSESSOR", "DATE OF ASSESSMENT", "BUSINESS OWNER", "OVERALL RATING"} {
				if value := get(key); value != "" && value != contextValues[key] {
					return result, true, fmt.Errorf("Finding at %s row %d has conflicting assessment details. Confirm its group in the source.", row.Sheet, row.RowStart)
				}
			}
			if get("RECOMMENDATIONS") == "" {
				return result, true, fmt.Errorf("Finding at %s row %d has no recommendation. Complete the source before creating a follow-up form.", row.Sheet, row.RowStart)
			}
			if len(result.Contract.Sections)+1 > policy.MaxSections || len(result.FieldChanges)+5 > policy.MaxFields {
				return result, true, fmt.Errorf("This register exceeds the follow-up form limit. Import one assessment at a time.")
			}
			count++
			sectionID := stableFormProposalID("finding", groupID, strconv.Itoa(row.RowStart))
			context := fmt.Sprintf("%s. Assessor: %s. Business owner: %s. Overall rating at assessment: %s. Finding: %s. Severity at assessment: %s. Original deadline: %s. Recorded status: %s.", groupLabel, contextValues["ASSESSOR"], contextValues["BUSINESS OWNER"], contextValues["OVERALL RATING"], get("FINDINGS"), get("SEVERITY"), get("TIMELINE"), get("STATUS"))
			if len(context) > 1000 {
				return result, true, fmt.Errorf("Finding at %s row %d is too long for the form context. Shorten the source or prepare the form manually without omitting evidence.", row.Sheet, row.RowStart)
			}
			result.Contract.Sections = append(result.Contract.Sections, formcontract.Section{ID: sectionID, Title: fmt.Sprintf("Assessment %s · Finding %d", serial, count), Help: context})
			questions := []struct {
				key, label, help string
				kind             formcontract.Type
				required         bool
			}{
				{"response", "Response to this finding", "Recorded risk or implication: " + get("RISK/ IMPLICATIONS"), formcontract.TypeLongText, true},
				{"action", "Remediation action or explanation", "Bank recommendation: " + get("RECOMMENDATIONS"), formcontract.TypeLongText, true},
				{"owner", "Person responsible for the remediation", "Bank-recorded responsibility: " + contextValues["RESPONSIBILITY"], formcontract.TypeShortText, true},
				{"date", "Proposed completion date", "Enter a proposed date if remediation is planned. Explain any unavailable date in your response.", formcontract.TypeDate, false},
				{"evidence", "Supporting evidence", "Attach current supporting documents, if available. Explain missing evidence in your response. Historical comment: " + get("RISK OWNER COMMENT"), formcontract.TypeFile, false},
			}
			for _, q := range questions {
				if len(q.help) > 1000 {
					return result, true, fmt.Errorf("Finding at %s row %d has context exceeding the form limit. Prepare this finding manually without omitting source text.", row.Sheet, row.RowStart)
				}
				fieldID := stableFormProposalID("field", sectionID, q.key)
				field := formcontract.Field{ID: fieldID, SectionID: sectionID, Label: q.label, Description: q.help, Type: q.kind, Required: q.required, CollectionIntent: formcontract.IntentCapture, BrowserCachePolicy: formcontract.BrowserCacheAllowed}
				if q.kind == formcontract.TypeFile {
					field.AcceptedFormats = []string{
						"application/pdf",
						"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
						"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
					}
				}
				result.FieldChanges = append(result.FieldChanges, FormFieldChange{ID: stableFormProposalID("change", fieldID), Kind: "ADD_FIELD", Field: field, Anchor: SourceAnchor{Sheet: row.Sheet, RowStart: row.RowStart, RowEnd: row.RowEnd}, GroupID: groupID, GroupLabel: groupLabel, Confidence: 1})
				result.Contract.Fields = append(result.Contract.Fields, field)
			}
		}
	}
	if !matched {
		return result, false, nil
	}
	if len(result.FieldChanges) == 0 {
		return result, true, fmt.Errorf("No complete findings were retained. Review the source before creating a follow-up form.")
	}
	result.Contract.ScoringMode = formcontract.ScoringNone
	result.Contract.Presentation = formcontract.Presentation{DefaultMode: formcontract.PresentationClassic, AllowModeSwitch: true}
	contract, err := formcontract.Normalize(result.Contract)
	if err != nil {
		return result, true, err
	}
	result.Contract = contract
	for i := range result.FieldChanges {
		result.FieldChanges[i].Field = contract.Fields[i]
	}
	result.UnresolvedItems = []ProposalUnresolvedItem{{Code: "CONFIRM_ASSESSMENT_GROUP", Message: "Confirm the selected assessment against the original register. Blank continuation cells use the preceding numbered assessment; vendor names in the source do not establish an existing vendor match."}}
	result.Provenance = FormProposalProvenance{ProposalVersion: FindingFollowUpVersion, SourceDocumentID: document.ID, SourceSHA256: document.SHA256, SourceVersion: document.Version, ParserVersion: parserVersionFor(document), ExtractionStatus: string(document.ExtractionStatus), TabularParser: document.Tabular.ParserVersion}
	return result, true, nil
}
