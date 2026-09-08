package main

import (
	"context"
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"testing"
)

type bankAssessmentRouteStub struct {
	formPolicyAuthorityStub
	route authority.Resolution
	err   error
}

func (s *bankAssessmentRouteStub) Resolve(_ context.Context, in authority.ResolveInput) (authority.Resolution, error) {
	s.inputs = append(s.inputs, in)
	return s.route, s.err
}
func TestAssessmentAuthorityUsesCurrentRoleScopeAndDelegateRoute(t *testing.T) {
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "delegate"}
	response := evidence.CompletedResponseSummary{ID: "response", TenantID: "bank", LegalEntityID: "entity"}
	field := formcontract.Field{ID: "certificate", Assessment: &formcontract.FieldAssessment{Mode: formcontract.AssessmentManual, ReviewerRole: "CISO"}}
	stub := &bankAssessmentRouteStub{route: authority.Resolution{RuleID: "review-route", PolicyVersion: "version-2", Principal: authority.Principal{ID: "delegate", Role: "CISO"}, EffectiveOrigins: []authority.EffectiveOrigin{{PrincipalID: "delegate", OriginPrincipalID: "absent-reviewer"}}}}
	authorize := formAssessmentAuthorizer(stub)
	if receipt, err := authorize(context.Background(), actor, response, field); err != nil || receipt != "version-2:review-route" {
		t.Fatalf("delegate receipt=%q err=%v", receipt, err)
	}
	if stub.inputs[0].ObjectID != "response" || stub.inputs[0].Responsibility != authority.ResponsibilityReviewer {
		t.Fatal("wrong route")
	}
	actor.LegalEntityID = "*"
	if _, err := authorize(context.Background(), actor, response, field); err != nil {
		t.Fatalf("verified bank-wide actor rejected: %v", err)
	}
	if stub.inputs[len(stub.inputs)-1].LegalEntityID != response.LegalEntityID {
		t.Fatal("wildcard scope entered material authority route")
	}
	actor.LegalEntityID = "entity"
	stub.route.Principal.Role = "OWNER"
	if _, err := authorize(context.Background(), actor, response, field); err == nil {
		t.Fatal("wrong bank role authorized")
	}
	stub.route.Principal.Role = "CISO"
	stub.route.Principal.ID = "replacement"
	if _, err := authorize(context.Background(), actor, response, field); err == nil {
		t.Fatal("revoked delegate authorized")
	}
	stub.err = errors.New("authority unavailable")
	if _, err := authorize(context.Background(), actor, response, field); err == nil {
		t.Fatal("outage authorized")
	}
	actor.TenantID = "other-bank"
	if _, err := authorize(context.Background(), actor, response, field); err == nil {
		t.Fatal("tenant mismatch authorized")
	}
}
