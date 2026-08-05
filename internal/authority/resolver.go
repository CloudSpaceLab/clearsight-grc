package authority

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var ErrNoRoute = errors.New("no eligible authority route")

type Resolver struct {
	version string
	rules   []Rule
}

func NewResolver(version string, rules []Rule) *Resolver {
	copied := append([]Rule(nil), rules...)
	sort.SliceStable(copied, func(i, j int) bool { return copied[i].Priority > copied[j].Priority })
	return &Resolver{version: version, rules: copied}
}

func (r *Resolver) Resolve(input ResolveInput) (Resolution, error) {
	at := input.At
	if at.IsZero() {
		at = time.Now().UTC()
	}
	for _, rule := range r.rules {
		if !matches(rule, input, at) {
			continue
		}
		return Resolution{
			Principal:     rule.Principal,
			RuleID:        rule.ID,
			PolicyVersion: r.version,
			Explanation: fmt.Sprintf("%s selected for %s on %s:%s within legal entity %s", rule.Principal.DisplayName, input.Responsibility, input.ObjectType, input.ObjectID, input.LegalEntityID),
		}, nil
	}
	return Resolution{}, ErrNoRoute
}

func matches(rule Rule, input ResolveInput, at time.Time) bool {
	if rule.TenantID != input.TenantID || rule.Responsibility != input.Responsibility {
		return false
	}
	if rule.LegalEntityID != "*" && rule.LegalEntityID != input.LegalEntityID {
		return false
	}
	if rule.ObjectType != "*" && !strings.EqualFold(rule.ObjectType, input.ObjectType) {
		return false
	}
	if rule.ObjectID != "*" && rule.ObjectID != input.ObjectID {
		return false
	}
	if input.Materiality < rule.MinMateriality {
		return false
	}
	if !rule.ValidFrom.IsZero() && at.Before(rule.ValidFrom) {
		return false
	}
	if !rule.ValidUntil.IsZero() && !at.Before(rule.ValidUntil) {
		return false
	}
	return true
}

func DemoPolicySet() (string, []Rule) {
	return "demo-2026-08-05", []Rule{
		{ID: "route-ndpa-dpo", TenantID: "bank-demo", LegalEntityID: "bank-ng", ObjectType: "PROGRAM", ObjectID: "ndpa", Responsibility: ResponsibilityAuthorizer, Principal: Principal{ID: "role-dpo", DisplayName: "Data Protection Officer", Kind: "ROLE", Role: "DPO"}, Priority: 100},
		{ID: "route-control-assurance", TenantID: "bank-demo", LegalEntityID: "bank-ng", ObjectType: "MATTER", ObjectID: "*", Responsibility: ResponsibilityReviewer, Principal: Principal{ID: "team-control-assurance", DisplayName: "Control Assurance", Kind: "TEAM", Role: "Second-line reviewer"}, Priority: 80},
		{ID: "route-cro-material", TenantID: "bank-demo", LegalEntityID: "bank-ng", ObjectType: "MATTER", ObjectID: "*", Responsibility: ResponsibilityAuthorizer, MinMateriality: 4, Principal: Principal{ID: "role-cro", DisplayName: "Chief Risk Officer", Kind: "ROLE", Role: "CRO"}, Priority: 70},
		{ID: "route-risk-owner", TenantID: "bank-demo", LegalEntityID: "bank-ng", ObjectType: "MATTER", ObjectID: "*", Responsibility: ResponsibilityOwner, Principal: Principal{ID: "queue-risk-owners", DisplayName: "Scoped Risk Owners", Kind: "QUEUE", Role: "Accountable owner queue"}, Priority: 50},
	}
}
