//go:build postgres

package main

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

type thirdPartyRequirement struct {
	Key, Service, Finding, VendorResponse, InternalComment, EvidenceLabel, SourceRange string
	Assessor, BusinessOwner, Performer, SourceSeverity, SourceRating                   string
	AssessmentDate, Deadline, SourceStatus, Recommendation, Implication                string
}

type thirdPartySemanticGroup struct {
	Requirements []thirdPartyRequirement
}

func (group thirdPartySemanticGroup) FormLabels() []string {
	labels := make([]string, 0, len(group.Requirements)*2)
	for range group.Requirements {
		labels = append(labels, "Vendor response", "Supporting evidence")
	}
	return labels
}

func thirdPartySemanticCapture(group sourceRecordGroup) (thirdPartySemanticGroup, bool, error) {
	if !strings.HasPrefix(group.Key, "third-party-risk-register") {
		return thirdPartySemanticGroup{}, false, nil
	}
	result := thirdPartySemanticGroup{Requirements: make([]thirdPartyRequirement, 0, len(group.Records))}
	var service, assessor, businessOwner string
	for _, record := range group.Records {
		values := map[string]string{}
		for _, field := range record.Fields {
			label := normalizeThirdPartyLabel(field.Label)
			if _, allowed := thirdPartyRegisterLabels[label]; !allowed && strings.TrimSpace(field.Value) != "" {
				return result, true, fmt.Errorf("third-party register column %q is not classified", field.Label)
			}
			values[label] = strings.TrimSpace(field.Value)
		}
		if values["SERVICES OFFERED"] != "" {
			service = values["SERVICES OFFERED"]
		}
		if values["ASSESSOR"] != "" {
			assessor = values["ASSESSOR"]
		}
		if values["BUSINESS OWNER"] != "" {
			businessOwner = values["BUSINESS OWNER"]
		}
		if strings.TrimSpace(record.Assessor) != "" {
			assessor = strings.TrimSpace(record.Assessor)
		}
		finding := values["FINDINGS"]
		if finding == "" {
			continue
		}
		performer := strings.TrimSpace(record.Owner)
		if performer == "" {
			performer = values["RESPONSIBILITY"]
		}
		comment := values["RISK OWNER COMMENT"]
		vendorResponse, internalComment := comment, ""
		if strings.Contains(strings.ToUpper(comment), "BUSINESS TEAM COMMENT") {
			vendorResponse, internalComment = "", comment
		}
		result.Requirements = append(result.Requirements, thirdPartyRequirement{
			Key: record.Key, Service: service, Finding: finding, VendorResponse: vendorResponse, InternalComment: internalComment, EvidenceLabel: thirdPartyEvidenceLabel(finding), SourceRange: record.SourceRange,
			Assessor: assessor, BusinessOwner: businessOwner, Performer: performer, SourceSeverity: values["SEVERITY"], SourceRating: values["OVERALL RATING"], AssessmentDate: values["DATE OF ASSESSMENT"], Deadline: values["TIMELINE"], SourceStatus: values["STATUS"], Recommendation: values["RECOMMENDATIONS"], Implication: values["RISK/ IMPLICATIONS"],
		})
	}
	if len(result.Requirements) == 0 {
		return result, true, fmt.Errorf("third-party register has no findings")
	}
	return result, true, nil
}

var thirdPartyRegisterLabels = map[string]bool{
	"S/N": true, "ASSESSOR": true, "BUSINESS OWNER": true, "SERVICE PROVIDER": true, "SERVICES OFFERED": true,
	"FINDINGS": true, "RISK/ IMPLICATIONS": true, "SEVERITY": true, "OVERALL RATING": true, "RECOMMENDATIONS": true,
	"DATE OF ASSESSMENT": true, "RESPONSIBILITY": true, "TIMELINE": true, "STATUS": true, "RISK OWNER COMMENT": true, "IT RISK": true,
}

func normalizeThirdPartyLabel(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	return strings.Join(strings.FieldsFunc(value, func(r rune) bool { return unicode.IsSpace(r) }), " ")
}

func thirdPartyEvidenceLabel(finding string) string {
	lower := strings.ToLower(finding)
	switch {
	case strings.Contains(lower, "vapt"):
		return "Independent vulnerability assessment report"
	case strings.Contains(lower, "right to audit"):
		return "Contract or addendum containing audit rights"
	case strings.Contains(lower, "pci-dss") || strings.Contains(lower, "pci dss"):
		return "Current PCI-DSS certificate"
	case strings.Contains(lower, "22301"):
		return "Current ISO 27001 and ISO 22301 certificates"
	default:
		return "Current information security certification"
	}
}

func buildThirdPartySemanticForm(group sourceRecordGroup) (monitoring.CreateFormInput, map[string]formcontract.AnswerValue, error) {
	semantic, ok, err := thirdPartySemanticCapture(group)
	if err != nil || !ok {
		return monitoring.CreateFormInput{}, nil, err
	}
	input := monitoring.CreateFormInput{Name: "Third-party security and continuity review", Purpose: "Review current vendor responses and supporting evidence for the identified service requirements. Sample data; bank assessment and source history remain separate.", ScoringMode: formcontract.ScoringNone, Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard, AllowModeSwitch: true}}
	answers := map[string]formcontract.AnswerValue{}
	sections := map[string]string{}
	for index, requirement := range semantic.Requirements {
		sectionID := sections[requirement.Service]
		if sectionID == "" {
			sectionID = fmt.Sprintf("service_%d", len(sections)+1)
			sections[requirement.Service] = sectionID
			input.Sections = append(input.Sections, formcontract.Section{ID: sectionID, Title: sourceShort(requirement.Service, 200)})
		}
		responseID := fmt.Sprintf("requirement_%d_response", index+1)
		evidenceID := fmt.Sprintf("requirement_%d_evidence", index+1)
		input.Fields = append(input.Fields,
			formcontract.Field{ID: responseID, SectionID: sectionID, Label: sourceShort(requirement.Finding, 200), Type: formcontract.TypeLongText, Description: "Vendor response · Service: " + requirement.Service, Assessment: &formcontract.FieldAssessment{Mode: formcontract.AssessmentManual, Required: true, Weight: 20, ReviewerRole: "REVIEWER", Rubric: thirdPartyReviewRubric()}},
			formcontract.Field{ID: evidenceID, SectionID: sectionID, Label: requirement.EvidenceLabel, Type: formcontract.TypeVendorDocument, Description: "Supporting evidence · Service: " + requirement.Service},
		)
		if requirement.VendorResponse != "" {
			answers[responseID] = formcontract.TextAnswer(requirement.VendorResponse)
		}
	}
	return input, answers, nil
}

func thirdPartyReviewRubric() []formcontract.AssessmentOutcome {
	return []formcontract.AssessmentOutcome{{ID: "SATISFACTORY", Label: "Satisfactory", Points: 0}, {ID: "FOLLOW_UP", Label: "Needs follow-up", Points: 50}, {ID: "MATERIAL_CONCERN", Label: "Material concern", Points: 100}}
}
