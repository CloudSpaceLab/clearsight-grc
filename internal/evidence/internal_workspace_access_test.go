package evidence

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestInternalWorkspaceIdentityAndRecipientAreExact(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "person", Kind: "PERSON", IssuedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour)}
	distribution := FormDistribution{ID: "distribution", TenantID: "bank", LegalEntityID: "entity", SubjectType: "PROGRAM", SubjectID: "program", FormTemplateID: "form", FormTemplateVersion: 1, Status: DistributionOpen, Deadline: now.Add(time.Hour)}
	workspace := ResponseWorkspace{ID: "workspace", TenantID: "bank", LegalEntityID: "entity", DistributionID: "distribution", Status: ResponseWorkspaceOpen}
	recipient := DistributionRecipient{ID: "recipient", TenantID: "bank", LegalEntityID: "entity", DistributionID: "distribution", RequestID: "request", Role: RecipientTo, Type: RecipientInternalPrincipal, PrincipalID: "person", State: DistributionRecipientPending}
	request := Request{ID: "request", TenantID: "bank", LegalEntityID: "entity", SubjectType: "PROGRAM", SubjectID: "program", FormTemplateID: "form", FormTemplateVersion: 1, Status: RequestReady, Deadline: now.Add(time.Hour), Recipient: Recipient{Type: RecipientInternalPrincipal, PrincipalID: "person", State: RecipientStateAssigned}}
	check := func(a identity.Actor, d FormDistribution, w ResponseWorkspace, r DistributionRecipient, q Request, eligible bool) bool {
		return internalWorkspaceAccessAllowed(identity.WithActor(context.Background(), a), d, w, r, q, eligible, now)
	}
	if !check(actor, distribution, workspace, recipient, request, true) {
		t.Fatal("assigned active person denied")
	}
	if internalWorkspaceAccessAllowed(context.Background(), distribution, workspace, recipient, request, true, now) {
		t.Fatal("missing identity permitted")
	}
	tests := map[string]func(*identity.Actor, *FormDistribution, *ResponseWorkspace, *DistributionRecipient, *Request, *bool){
		"other principal": func(a *identity.Actor, _ *FormDistribution, _ *ResponseWorkspace, _ *DistributionRecipient, _ *Request, _ *bool) {
			a.PrincipalID = "other"
		},
		"service": func(a *identity.Actor, _ *FormDistribution, _ *ResponseWorkspace, _ *DistributionRecipient, _ *Request, _ *bool) {
			a.Kind = "SERVICE"
		},
		"unknown actor kind": func(a *identity.Actor, _ *FormDistribution, _ *ResponseWorkspace, _ *DistributionRecipient, _ *Request, _ *bool) {
			a.Kind = "AUTOMATION"
		},
		"expired identity": func(a *identity.Actor, _ *FormDistribution, _ *ResponseWorkspace, _ *DistributionRecipient, _ *Request, _ *bool) {
			a.ExpiresAt = now
		},
		"wildcard entity": func(a *identity.Actor, _ *FormDistribution, _ *ResponseWorkspace, _ *DistributionRecipient, _ *Request, _ *bool) {
			a.LegalEntityID = "*"
		},
		"wrong tenant": func(_ *identity.Actor, d *FormDistribution, _ *ResponseWorkspace, _ *DistributionRecipient, _ *Request, _ *bool) {
			d.TenantID = "other"
		},
		"wrong entity": func(_ *identity.Actor, _ *FormDistribution, w *ResponseWorkspace, _ *DistributionRecipient, _ *Request, _ *bool) {
			w.LegalEntityID = "other"
		},
		"cc recipient": func(_ *identity.Actor, _ *FormDistribution, _ *ResponseWorkspace, r *DistributionRecipient, _ *Request, _ *bool) {
			r.Role = RecipientCC
		},
		"revoked recipient": func(_ *identity.Actor, _ *FormDistribution, _ *ResponseWorkspace, r *DistributionRecipient, _ *Request, _ *bool) {
			r.State = DistributionRecipientRevoked
		},
		"reassigned request": func(_ *identity.Actor, _ *FormDistribution, _ *ResponseWorkspace, _ *DistributionRecipient, q *Request, _ *bool) {
			q.Recipient.State = RecipientStateReassignmentRequired
		},
		"different request": func(_ *identity.Actor, _ *FormDistribution, _ *ResponseWorkspace, r *DistributionRecipient, _ *Request, _ *bool) {
			r.RequestID = "other"
		},
		"different form revision": func(_ *identity.Actor, _ *FormDistribution, _ *ResponseWorkspace, _ *DistributionRecipient, q *Request, _ *bool) {
			q.FormTemplateVersion++
		},
		"different subject": func(_ *identity.Actor, _ *FormDistribution, _ *ResponseWorkspace, _ *DistributionRecipient, q *Request, _ *bool) {
			q.SubjectID = "other-program"
		},
		"closed workspace": func(_ *identity.Actor, _ *FormDistribution, w *ResponseWorkspace, _ *DistributionRecipient, _ *Request, _ *bool) {
			w.Status = ResponseWorkspaceCompleted
		},
		"expired request": func(_ *identity.Actor, _ *FormDistribution, _ *ResponseWorkspace, _ *DistributionRecipient, q *Request, _ *bool) {
			q.Deadline = now
		},
		"revoked subject access": func(_ *identity.Actor, _ *FormDistribution, _ *ResponseWorkspace, _ *DistributionRecipient, _ *Request, e *bool) {
			*e = false
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			a, d, w, r, q, e := actor, distribution, workspace, recipient, request, true
			mutate(&a, &d, &w, &r, &q, &e)
			if check(a, d, w, r, q, e) {
				t.Fatal("unauthorized workspace access permitted")
			}
		})
	}
}
