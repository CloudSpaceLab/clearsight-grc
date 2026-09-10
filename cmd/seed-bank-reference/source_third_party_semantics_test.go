//go:build postgres

package main

import (
	"slices"
	"strings"
	"testing"
)

func TestThirdPartySemanticCaptureSeparatesAnswersAssessmentsAndAssignments(t *testing.T) {
	group := sourceRecordGroup{Key: "third-party-risk-register", Title: "Third-party risk register", SourceFile: "Sample Third-Party Risk Register.xlsx", SourceSheet: "2026 Register", Records: []sourceRecord{
		{Key: "finding-1", SourceRange: "A2:P2", Owner: "Hakeem", Assessor: "Blessing", Fields: thirdPartyFixtureFields(map[string]string{
			"S/N": "1", "ASSESSOR": "Blessing", "BUSINESS OWNER": "POS Business", "SERVICE PROVIDER": "Cloudspace OEM", "SERVICES OFFERED": "Moneytor GetPaid application", "FINDINGS": "Information security certification missing", "RISK/ IMPLICATIONS": "Information security risk", "Severity": "Medium", "OVERALL RATING": "Medium", "RECOMMENDATIONS": "Provide ISO 27001 certification", "DATE OF ASSESSMENT": "13th February 2026", "RESPONSIBILITY": "Hakeem", "TIMELINE": "March 31st 2026", "STATUS": "Open", "RISK OWNER COMMENT": "Surveillance audit is in progress.",
		})},
		{Key: "finding-2", SourceRange: "A5:P5", Owner: "Hakeem", Assessor: "Joel", Fields: thirdPartyFixtureFields(map[string]string{
			"S/N": "2", "ASSESSOR": "Joel", "BUSINESS OWNER": "POS Business", "SERVICE PROVIDER": "Cloudspace OEM", "SERVICES OFFERED": "Payment Terminal Service Provider (PTSP)", "FINDINGS": "PCI-DSS certificate expired", "RISK/ IMPLICATIONS": "Card-data risk", "Severity": "Medium", "OVERALL RATING": "Medium", "RECOMMENDATIONS": "Provide a current PCI-DSS certificate", "DATE OF ASSESSMENT": "6th February 2026", "RESPONSIBILITY": "Hakeem", "TIMELINE": "March 31st 2026", "STATUS": "Open", "RISK OWNER COMMENT": "Business Team Comment: contract addendum is being prepared.",
		})},
	}}

	capture, ok, err := thirdPartySemanticCapture(group)
	if err != nil || !ok {
		t.Fatalf("capture=%+v ok=%v err=%v", capture, ok, err)
	}
	if len(capture.Requirements) != 2 || capture.Requirements[0].Assessor != "Blessing" || capture.Requirements[1].Performer != "Hakeem" {
		t.Fatalf("semantic assignments were not retained: %+v", capture.Requirements)
	}
	if capture.Requirements[1].VendorResponse != "" || capture.Requirements[1].InternalComment == "" {
		t.Fatalf("business-owner comment was presented as a vendor answer: %+v", capture.Requirements[1])
	}
	for _, forbidden := range []string{"S/N", "ASSESSOR", "BUSINESS OWNER", "SERVICE PROVIDER", "RESPONSIBILITY", "TIMELINE", "STATUS"} {
		if slices.Contains(capture.FormLabels(), forbidden) {
			t.Fatalf("internal column became a vendor field: %s", forbidden)
		}
	}
	form, answers, err := buildThirdPartySemanticForm(group)
	if err != nil {
		t.Fatal(err)
	}
	if len(form.Fields) != 4 || len(form.Sections) != 2 {
		t.Fatalf("fields=%d sections=%d", len(form.Fields), len(form.Sections))
	}
	if !strings.Contains(form.Fields[0].Description, "Service: Moneytor GetPaid application") || !strings.Contains(form.Fields[2].Description, "Service: Payment Terminal Service Provider (PTSP)") {
		t.Fatalf("service context missing from semantic fields: %+v", form.Fields)
	}
	if len(answers) != 1 {
		t.Fatalf("answers=%d, want only the recorded vendor response", len(answers))
	}
	encoded := sourceJSON(map[string]any{"fields": form.Fields, "answers": answers})
	for _, name := range []string{"Blessing", "Joel", "Hakeem"} {
		if strings.Contains(string(encoded), name) {
			t.Fatalf("assignment %s leaked into the vendor answer contract", name)
		}
	}
}

func TestThirdPartySemanticCaptureUsesReplacementIdentities(t *testing.T) {
	group := sourceRecordGroup{Key: "third-party-risk-register"}
	code, idempotency := sourceFormIdentity(group, 0)
	if !strings.HasPrefix(code, "SOURCE-TPR-V2-") || !strings.HasPrefix(idempotency, "fidelity-source-records-v2:") {
		t.Fatalf("code=%q idempotency=%q", code, idempotency)
	}
}

func thirdPartyFixtureFields(values map[string]string) []sourceRecordField {
	labels := []string{"S/N", "ASSESSOR", "BUSINESS OWNER", "SERVICE PROVIDER", "SERVICES OFFERED", "FINDINGS", "RISK/ IMPLICATIONS", "Severity", "OVERALL RATING", "RECOMMENDATIONS", "DATE OF ASSESSMENT", "RESPONSIBILITY", "TIMELINE", "STATUS", "RISK OWNER COMMENT", "IT RISK"}
	fields := make([]sourceRecordField, 0, len(labels))
	for index, label := range labels {
		fields = append(fields, sourceRecordField{Label: label, Value: values[label], SourceCell: string(rune('A'+index)) + "2"})
	}
	return fields
}
