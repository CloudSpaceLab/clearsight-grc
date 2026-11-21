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

func TestResponseReviewAuthorityUsesSeparateScopedRoute(t *testing.T) {
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "*", PrincipalID: "reviewer"}
	response := evidence.CompletedResponseSummary{ID: "response", TenantID: "bank", LegalEntityID: "entity"}
	stub := &bankAssessmentRouteStub{route: authority.Resolution{RuleID: "rule", PolicyVersion: "v1", Principal: authority.Principal{ID: "reviewer"}}}
	authorize := formResponseReviewAuthorizer(stub)
	if route, err := authorize(context.Background(), actor, response); err != nil || route != "v1:rule" {
		t.Fatalf("route=%q err=%v", route, err)
	}
	in := stub.inputs[0]
	if in.DecisionType != "forms.response.review" || in.ObjectID != response.ID || in.LegalEntityID != response.LegalEntityID || in.Responsibility != authority.ResponsibilityReviewer {
		t.Fatalf("wrong route: %+v", in)
	}
	stub.route.Principal.ID = "someone-else"
	if _, err := authorize(context.Background(), actor, response); err == nil {
		t.Fatal("revoked reviewer authorized")
	}
	stub.err = errors.New("authority unavailable")
	if _, err := authorize(context.Background(), actor, response); err == nil {
		t.Fatal("authority outage authorized")
	}
	actor.TenantID = "other"
	if _, err := authorize(context.Background(), actor, response); err == nil {
		t.Fatal("tenant mismatch authorized")
	}
}

func TestAssessmentAndDiscoveryRequireAnExactResponse(t *testing.T) {
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "reviewer"}
	response := evidence.CompletedResponseSummary{TenantID: "bank", LegalEntityID: "entity"}
	stub := &bankAssessmentRouteStub{route: authority.Resolution{RuleID: "rule", PolicyVersion: "v1", Principal: authority.Principal{ID: "reviewer", Role: "CISO"}}}
	field := formcontract.Field{Assessment: &formcontract.FieldAssessment{Mode: formcontract.AssessmentManual, ReviewerRole: "CISO"}}
	if _, err := formAssessmentAuthorizer(stub)(context.Background(), actor, response, field); err == nil {
		t.Fatal("assessment authorized without an exact response")
	}
	if err := formResponseDiscoveryAuthorizer(stub)(context.Background(), actor, response); err == nil {
		t.Fatal("discovery authorized without an exact response")
	}
	if len(stub.inputs) != 0 {
		t.Fatal("an incomplete response must not be resolved as a wildcard")
	}
	response.ID = "response"
	actor.TenantID, response.TenantID = "", ""
	if err := formResponseDiscoveryAuthorizer(stub)(context.Background(), actor, response); err == nil {
		t.Fatal("discovery authorized without a tenant")
	}
	if len(stub.inputs) != 0 {
		t.Fatal("an absent tenant must not be resolved")
	}
	actor.TenantID, response.TenantID = "bank", "bank"
	response.ID = "*"
	if err := formResponseDiscoveryAuthorizer(stub)(context.Background(), actor, response); err == nil {
		t.Fatal("discovery authorized a wildcard response")
	}
	response.ID = "response"
	if err := formResponseDiscoveryAuthorizer(stub)(context.Background(), actor, response); err != nil {
		t.Fatalf("assessment reviewer cannot discover the response: %v", err)
	}
	if len(stub.inputs) != 1 || stub.inputs[0].DecisionType != "forms.response.assess" || stub.inputs[0].ObjectID != response.ID || stub.inputs[0].LegalEntityID != response.LegalEntityID {
		t.Fatalf("discovery resolved the wrong decision or scope: %+v", stub.inputs)
	}
	stub.route.Principal.ID = "another-reviewer"
	if err := formResponseDiscoveryAuthorizer(stub)(context.Background(), actor, response); err == nil {
		t.Fatal("discovery allowed a different reviewer")
	}
}

func TestInternalRespondentAuthorityBindsExactRequest(t *testing.T) {
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "respondent", Kind: "PERSON"}
	request := evidence.Request{ID: "request", TenantID: "bank", LegalEntityID: "entity"}
	stub := &bankAssessmentRouteStub{route: authority.Resolution{RuleID: "performer", PolicyVersion: "v2", Principal: authority.Principal{ID: "respondent"}}}
	authorize := internalFormResponseAuthorizer(stub)
	if route, err := authorize(context.Background(), actor, request, "forms.response.submit"); err != nil || route != "v2:performer" {
		t.Fatalf("route %q: %v", route, err)
	}
	in := stub.inputs[0]
	if in.ObjectType != "EVIDENCE_REQUEST" || in.ObjectID != request.ID || in.Responsibility != authority.ResponsibilityPerformer || in.DecisionType != "forms.response.submit" {
		t.Fatalf("wrong route: %+v", in)
	}
	stub.route.Principal.ID = "someone-else"
	if _, err := authorize(context.Background(), actor, request, "forms.response.submit"); err == nil {
		t.Fatal("wrong performer accepted")
	}
	stub.err = errors.New("authority unavailable")
	if _, err := authorize(context.Background(), actor, request, "forms.response.submit"); err == nil {
		t.Fatal("authority outage accepted")
	}
}
