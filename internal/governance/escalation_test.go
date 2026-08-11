package governance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestParseEscalationSequencesSupportsDepartmentHierarchy(t *testing.T) {
	definition := json.RawMessage(`{
		"rules":[{"id":"r1","responsibility":"ESCALATION_OWNER","selector":{"kind":"ROLE","ref":"RISK_MANAGER"}}],
		"escalations":[{
			"id":"overdue-review",
			"trigger":"OVERDUE",
			"steps":[
				{"after":"0s","responsibility":"ACCOUNTABLE_OWNER","department_levels_up":0},
				{"after":"4h","responsibility":"ESCALATION_OWNER","department_levels_up":1},
				{"after":"24h","responsibility":"ESCALATION_OWNER"}
			]
		}]
	}`)

	sequences, err := ParseEscalationSequences(definition)
	if err != nil {
		t.Fatal(err)
	}
	if len(sequences) != 1 || len(sequences[0].Steps) != 3 {
		t.Fatalf("unexpected escalation parse: %#v", sequences)
	}
	if sequences[0].Steps[1].After != 4*time.Hour || sequences[0].Steps[1].DepartmentLevelsUp == nil || *sequences[0].Steps[1].DepartmentLevelsUp != 1 {
		t.Fatalf("unexpected parent-department step: %#v", sequences[0].Steps[1])
	}
	if sequences[0].Steps[2].DepartmentLevelsUp != nil {
		t.Fatalf("final step should use legal-entity routing scope: %#v", sequences[0].Steps[2])
	}
}

func TestParseEscalationSequencesSupportsSourceRoleAndTargetRoleOrGroup(t *testing.T) {
	definition := json.RawMessage(`{
		"escalations":[{
			"id":"compliance-overdue",
			"trigger":"OVERDUE",
			"steps":[{
				"after":"0s",
				"responsibility":"ESCALATION_OWNER",
				"source_roles":["Compliance Officer"],
				"targets":{
					"roles":["Supervisor"],
					"groups":["019fede5-67de-733a-95ae-97f4db546c1e"]
				}
			}]
		}]
	}`)

	sequences, err := ParseEscalationSequences(definition)
	if err != nil {
		t.Fatal(err)
	}
	step := sequences[0].Steps[0]
	if len(step.SourceRoles) != 1 || step.SourceRoles[0] != "COMPLIANCE_OFFICER" {
		t.Fatalf("unexpected source roles: %#v", step.SourceRoles)
	}
	if len(step.TargetRoles) != 1 || step.TargetRoles[0] != "SUPERVISOR" {
		t.Fatalf("unexpected target roles: %#v", step.TargetRoles)
	}
	if len(step.TargetGroupIDs) != 1 || step.TargetGroupIDs[0] != "019fede5-67de-733a-95ae-97f4db546c1e" {
		t.Fatalf("unexpected target groups: %#v", step.TargetGroupIDs)
	}
}

func TestParseEscalationSequencesRejectsEmptyOrInvalidTargetConstraints(t *testing.T) {
	cases := []string{
		`{"escalations":[{"id":"bad","trigger":"OVERDUE","steps":[{"after":"0s","responsibility":"ESCALATION_OWNER","source_roles":[]}]}]}`,
		`{"escalations":[{"id":"bad","trigger":"OVERDUE","steps":[{"after":"0s","responsibility":"ESCALATION_OWNER","targets":{}}]}]}`,
		`{"escalations":[{"id":"bad","trigger":"OVERDUE","steps":[{"after":"0s","responsibility":"ESCALATION_OWNER","targets":{"groups":["not-a-uuid"]}}]}]}`,
		`{"escalations":[{"id":"bad","trigger":"OVERDUE","steps":[{"after":"0s","responsibility":"ESCALATION_OWNER","targets":{"roles":["SUPERVISOR","supervisor"]}}]}]}`,
	}
	for _, raw := range cases {
		if _, err := ParseEscalationSequences(json.RawMessage(raw)); err == nil {
			t.Fatalf("expected target constraint rejection for %s", raw)
		}
	}
}

func TestParseEscalationSequencesRejectsDuplicateTrigger(t *testing.T) {
	definition := json.RawMessage(`{
		"escalations":[
			{"id":"overdue-owner","trigger":"OVERDUE","steps":[{"after":"0s","responsibility":"ACCOUNTABLE_OWNER"}]},
			{"id":"overdue-risk","trigger":"OVERDUE","steps":[{"after":"1h","responsibility":"ESCALATION_OWNER"}]}
		]
	}`)
	_, err := ParseEscalationSequences(definition)
	if err == nil || !strings.Contains(err.Error(), "already defined") {
		t.Fatalf("expected duplicate trigger rejection, got %v", err)
	}
}

func TestParseEscalationSequencesRejectsNonIncreasingThresholds(t *testing.T) {
	definition := json.RawMessage(`{
		"escalations":[{
			"id":"overdue-review",
			"trigger":"OVERDUE",
			"steps":[
				{"after":"4h","responsibility":"ACCOUNTABLE_OWNER","department_levels_up":0},
				{"after":"4h","responsibility":"ESCALATION_OWNER","department_levels_up":1}
			]
		}]
	}`)
	_, err := ParseEscalationSequences(definition)
	if err == nil || !strings.Contains(err.Error(), "must increase") {
		t.Fatalf("expected monotonic escalation error, got %v", err)
	}
}

func TestParseEscalationSequencesRejectsUnknownFieldsAndDeepTraversal(t *testing.T) {
	unknown := json.RawMessage(`{
		"escalations":[{
			"id":"bad",
			"trigger":"OVERDUE",
			"selector":{"kind":"ROLE","ref":"ADMIN"},
			"steps":[{"after":"0s","responsibility":"ESCALATION_OWNER"}]
		}]
	}`)
	if _, err := ParseEscalationSequences(unknown); err == nil {
		t.Fatal("expected escalation actor selector to be rejected")
	}

	tooDeep := json.RawMessage(`{
		"escalations":[{
			"id":"deep",
			"trigger":"OVERDUE",
			"steps":[{"after":"0s","responsibility":"ESCALATION_OWNER","department_levels_up":9}]
		}]
	}`)
	if _, err := ParseEscalationSequences(tooDeep); err == nil || !strings.Contains(err.Error(), "department_levels_up") {
		t.Fatalf("expected department traversal bound error, got %v", err)
	}
}

func TestParseEscalationSequencesRejectsTrailingJSON(t *testing.T) {
	definition := json.RawMessage(`{
		"escalations":[{
			"id":"overdue-review",
			"trigger":"OVERDUE",
			"steps":[{"after":"0s","responsibility":"ESCALATION_OWNER"}]
		}]
	} {"unexpected":true}`)
	if _, err := ParseEscalationSequences(definition); err == nil {
		t.Fatal("expected trailing top-level JSON to be rejected")
	}
}
