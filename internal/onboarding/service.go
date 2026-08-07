package onboarding

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

type Service struct {
	repo   Repository
	guides []Guide
	byCode map[string]Guide
}

func NewService(repo Repository) *Service {
	guides := DemoGuides()
	sort.SliceStable(guides, func(i, j int) bool { return guides[i].Priority > guides[j].Priority })
	indexed := make(map[string]Guide, len(guides))
	for _, guide := range guides {
		indexed[guide.Code] = guide
	}
	return &Service{repo: repo, guides: guides, byCode: indexed}
}

func (s *Service) Guide(role, code string) (Guide, error) {
	if code != "" {
		value, ok := s.byCode[code]
		if !ok {
			return Guide{}, fmt.Errorf("guide not found")
		}
		return value, nil
	}
	return s.ResolveRoles(splitRoles(role))
}

func (s *Service) ResolveRoles(roleCodes []string) (Guide, error) {
	roles := normalizeRoles(roleCodes)
	for _, guide := range s.guides {
		if len(guide.RoleCodes) == 0 {
			continue
		}
		for _, candidate := range guide.RoleCodes {
			if _, ok := roles[normalizeRole(candidate)]; ok {
				return guide, nil
			}
		}
	}
	for _, guide := range s.guides {
		if len(guide.RoleCodes) == 0 {
			return guide, nil
		}
	}
	return Guide{}, fmt.Errorf("guide not found")
}

func (s *Service) State(ctx context.Context, tenant, principal, guideCode string) (State, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(principal) == "" || strings.TrimSpace(guideCode) == "" {
		return State{}, fmt.Errorf("tenant_id, principal_id and guide_code are required")
	}
	value, err := s.repo.Get(ctx, tenant, principal, guideCode)
	if err == ErrStateNotFound {
		guide, guideErr := s.Guide("", guideCode)
		if guideErr != nil {
			return State{}, guideErr
		}
		return State{TenantID: tenant, PrincipalID: principal, GuideCode: guideCode, GuideVersion: guide.Version, Version: 0}, nil
	}
	return value, err
}

func (s *Service) Update(ctx context.Context, tenant, principal, guideCode string, input UpdateInput) (State, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(principal) == "" || strings.TrimSpace(guideCode) == "" {
		return State{}, fmt.Errorf("tenant_id, principal_id and guide_code are required")
	}
	if input.ExpectedVersion < 0 {
		return State{}, fmt.Errorf("expected_version cannot be negative")
	}
	if input.Completed && input.Dismissed {
		return State{}, fmt.Errorf("a guide cannot be completed and dismissed at the same time")
	}
	guide, err := s.Guide("", guideCode)
	if err != nil {
		return State{}, err
	}
	if input.CurrentStep < 0 || input.CurrentStep > len(guide.Steps) {
		return State{}, fmt.Errorf("current_step is outside the guide")
	}
	if input.Completed && input.CurrentStep < len(guide.Steps) {
		return State{}, fmt.Errorf("completed guides must be positioned after the final step")
	}
	return s.repo.Upsert(ctx, State{
		TenantID: tenant, PrincipalID: principal, GuideCode: guideCode, GuideVersion: guide.Version,
		CurrentStep: input.CurrentStep, Completed: input.Completed, Dismissed: input.Dismissed,
	}, input.ExpectedVersion)
}

func splitRoles(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' || r == '|' })
}

func normalizeRoles(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		if normalized := normalizeRole(value); normalized != "" {
			result[normalized] = struct{}{}
		}
	}
	return result
}

func normalizeRole(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.NewReplacer("-", "_", " ", "_").Replace(value)
	for strings.Contains(value, "__") {
		value = strings.ReplaceAll(value, "__", "_")
	}
	return strings.Trim(value, "_")
}

func DemoGuides() []Guide {
	return []Guide{
		{
			Code: "configure-admin-first-run", Profile: "configure-admin", Role: "Configure administrator",
			RoleCodes: []string{"GRC_ADMIN", "PLATFORM_ADMIN", "CONFIGURE_ADMIN"}, Priority: 100, Version: 1,
			Title: "Set up governed operations", Description: "Review identity, routing, source intake and operational health before enabling wider use.", Illustration: "guided-orbit",
			Steps: []Step{
				{ID: "configure", Title: "Review configuration health", Description: "Start with routing integrity, active policies and the tasks that still need an owner.", Action: "Open Configure", View: "configure", Target: "configure-workspace"},
				{ID: "routing", Title: "Inspect an approval route", Description: "Confirm that a material decision resolves to an active, eligible authorizer and an explainable policy version.", Action: "View approval route", View: "today", Target: "authority-action", Intent: "open-routing"},
				{ID: "imports", Title: "Bring in a controlled source", Description: "Imports preserve the original digest, extraction limits and human review before any governed follow-up.", Action: "Open Imports", View: "imports", Target: "document-import-form"},
				{ID: "finish", Title: "Continue with maker-checker setup", Description: "Directory synchronization, policy editing and activation remain governed configuration work rather than hidden administrator shortcuts.", Action: "Finish introduction", View: "configure", Target: "configure-workspace"},
			},
		},
		{
			Code: "authorizer-first-run", Profile: "authorizer", Role: "Authorizer or signatory",
			RoleCodes: []string{"AUTHORIZER", "SIGNATORY"}, Priority: 90, Version: 1,
			Title: "Understand your decision authority", Description: "See why work reached you, what evidence supports it and what must be verified after a decision.", Illustration: "guided-orbit",
			Steps: []Step{
				{ID: "today", Title: "Review decisions assigned to you", Description: "Today prioritizes items that require your authority, including the reason, due date and next valid action.", Action: "Open Today", View: "today", Target: "attention-list"},
				{ID: "routing", Title: "Confirm your route and limits", Description: "The route explains the legal entity, responsibility, materiality and selected policy version.", Action: "View approval route", View: "today", Target: "authority-action", Intent: "open-routing"},
				{ID: "matter", Title: "Inspect the complete decision record", Description: "Review facts, contradictions, options, actions, responses and closure blockers before deciding.", Action: "Open decision record", View: "work", Target: "matters-workspace", Intent: "open-first-matter"},
				{ID: "finish", Title: "Decisions remain evidence-backed", Description: "Approval is not the end state. ClearSight keeps the required outcome check and closure evidence visible.", Action: "Finish introduction", View: "work", Target: "matters-workspace"},
			},
		},
		{
			Code: "executive-first-run", Profile: "executive", Role: "Executive risk or compliance leader",
			RoleCodes: []string{"CRO", "CCO", "CISO", "DPO", "GENERAL_COUNSEL", "EXECUTIVE"}, Priority: 80, Version: 1,
			Title: "Read the operating brief", Description: "Distinguish current status from unknown, stale, at-risk and overdue work without opening several dashboards.", Illustration: "guided-orbit",
			Steps: []Step{
				{ID: "brief", Title: "Start with what needs attention", Description: "Today separates assigned work, due items and readiness so an empty count is never mistaken for a complete population.", Action: "Review Today", View: "today", Target: "today-brief"},
				{ID: "attention", Title: "Open one material record", Description: "Move from the brief to the exact Program, issue or evidence request instead of a generic dashboard.", Action: "Open first item", View: "today", Target: "attention-list", Intent: "open-first-attention"},
				{ID: "programs", Title: "Understand ongoing exposure", Description: "Programs explain current status from requirements, safeguards, evidence and open issues.", Action: "Open Programs", View: "programs", Target: "programs-workspace"},
				{ID: "finish", Title: "Use the reason, not only the colour", Description: "Every material status should expose the source, reason, owner and next valid action.", Action: "Finish introduction", View: "programs", Target: "programs-workspace"},
			},
		},
		{
			Code: "reviewer-first-run", Profile: "reviewer", Role: "Reviewer or independent challenger",
			RoleCodes: []string{"REVIEWER", "CHALLENGER", "CONTROL_ASSURANCE_LEAD"}, Priority: 70, Version: 1,
			Title: "Review evidence and challenge safely", Description: "Focus on changed, uncertain, contradictory or material items while preserving independent conclusions.", Illustration: "guided-orbit",
			Steps: []Step{
				{ID: "today", Title: "Review your assigned queue", Description: "Today shows reviews, approvals and evidence requests assigned to your verified role.", Action: "Open Today", View: "today", Target: "attention-list"},
				{ID: "matter", Title: "Inspect facts and contradictions", Description: "The issue workspace keeps known facts, missing facts, contradictions, actions and outcome checks together.", Action: "Open first issue", View: "work", Target: "matters-workspace", Intent: "open-first-matter"},
				{ID: "evidence", Title: "Request only what is unresolved", Description: "Known context is shown first so respondents are not asked to recreate information the bank already holds.", Action: "Open evidence", View: "work", Target: "evidence-workspace", Intent: "switch-evidence"},
				{ID: "finish", Title: "Keep independence visible", Description: "A completed action is not a verified outcome; the independent result remains a separate record.", Action: "Finish introduction", View: "work", Target: "matters-workspace"},
			},
		},
		{
			Code: "program-owner-first-run", Profile: "program-owner", Role: "Program or control owner",
			RoleCodes: []string{"PROGRAM_OWNER", "CONTROL_OWNER", "RISK_OWNER"}, Priority: 60, Version: 1,
			Title: "Keep your Program current", Description: "See why the Program has its current status and what evidence or action will make it current again.", Illustration: "guided-orbit",
			Steps: []Step{
				{ID: "programs", Title: "Open your ongoing responsibility", Description: "Programs connect requirements, safeguards, evidence checks and open issues in one current view.", Action: "Open Programs", View: "programs", Target: "programs-workspace"},
				{ID: "detail", Title: "Inspect the reasons", Description: "Open the first Program to see all recorded status reasons and the underlying requirements and evidence checks.", Action: "Open first Program", View: "programs", Target: "programs-workspace", Intent: "open-first-program"},
				{ID: "evidence", Title: "Resolve the smallest evidence gap", Description: "Use the evidence queue to see what is already known, who is responding and when the response is due.", Action: "Open evidence", View: "work", Target: "evidence-workspace", Intent: "switch-evidence"},
				{ID: "finish", Title: "Status follows evidence and outcomes", Description: "ClearSight recalculates status from current records rather than asking owners to choose a colour.", Action: "Finish introduction", View: "programs", Target: "programs-workspace"},
			},
		},
		{
			Code: "evidence-respondent-first-run", Profile: "evidence-respondent", Role: "Evidence respondent",
			RoleCodes: []string{"EVIDENCE_RESPONDENT", "RECORDS_CUSTODIAN", "BUSINESS_OWNER"}, Priority: 50, Version: 1,
			Title: "Respond with the minimum required effort", Description: "Understand why you were selected, review known facts and answer only the unresolved questions.", Illustration: "guided-orbit",
			Steps: []Step{
				{ID: "today", Title: "Find the request assigned to you", Description: "The work brief shows why the request is due and links to the exact request record.", Action: "Open Today", View: "today", Target: "attention-list"},
				{ID: "request", Title: "Review purpose and known facts", Description: "The evidence workspace explains the request, audience, deadline and information the bank already has.", Action: "Open evidence request", View: "work", Target: "evidence-workspace", Intent: "open-first-evidence"},
				{ID: "capture", Title: "Submit only unresolved information", Description: "The response form preserves a clear receipt and still separates submission from evidence sufficiency.", Action: "Open response form", View: "today", Target: "capture-action", Intent: "open-capture"},
				{ID: "finish", Title: "Redirect incorrect scope", Description: "A finished product must support redirect, delegation and wrong-scope reporting rather than forcing a false answer.", Action: "Finish introduction", View: "work", Target: "evidence-workspace"},
			},
		},
		{
			Code: "auditor-first-run", Profile: "auditor", Role: "Auditor or read-only reviewer",
			RoleCodes: []string{"AUDITOR", "INTERNAL_AUDIT", "LEGAL_REVIEWER", "READ_ONLY"}, Priority: 40, Version: 1,
			Title: "Inspect lineage without changing state", Description: "Review source evidence, decisions and outcome history while keeping read-only and restricted boundaries clear.", Illustration: "guided-orbit",
			Steps: []Step{
				{ID: "programs", Title: "Review the current institutional record", Description: "Programs show the latest state and reasons without implying that projections are the authoritative event history.", Action: "Open Programs", View: "programs", Target: "programs-workspace"},
				{ID: "matters", Title: "Inspect decisions and closure evidence", Description: "Issues and changes retain facts, decisions, actions, responses and independent outcome checks.", Action: "Open issues", View: "work", Target: "matters-workspace", Intent: "open-first-matter"},
				{ID: "imports", Title: "Trace imported source material", Description: "The import record preserves the original hash, extraction method, source anchors and human review limitations.", Action: "Open Imports", View: "imports", Target: "document-import-workspace"},
				{ID: "finish", Title: "Restricted existence remains protected", Description: "Search, counts and unavailable states must not reveal records outside the verified actor scope.", Action: "Finish introduction", View: "imports", Target: "document-import-workspace"},
			},
		},
		{
			Code: "general-first-run", Profile: "general", Role: "ClearSight user", Priority: 0, Version: 1,
			Title: "Understand your ClearSight workspace", Description: "Start with assigned work, inspect ongoing Programs and use evidence-backed records to understand what happens next.", Illustration: "guided-orbit",
			Steps: []Step{
				{ID: "today", Title: "Review assigned work", Description: "Today lists the work routed to your verified identity and explains why it needs attention.", Action: "Open Today", View: "today", Target: "today-brief"},
				{ID: "programs", Title: "Check ongoing Programs", Description: "Programs show requirements, safeguards, evidence checks and open issues in one current view.", Action: "Open Programs", View: "programs", Target: "programs-workspace"},
				{ID: "work", Title: "Inspect issues and evidence", Description: "Work keeps specific issues, actions, responses and evidence requests connected to their source context.", Action: "Open Work", View: "work", Target: "matters-workspace"},
				{ID: "finish", Title: "Use exact records and next actions", Description: "A completed task or uploaded file is not automatically a verified outcome.", Action: "Finish introduction", View: "today", Target: "today-brief"},
			},
		},
	}
}
