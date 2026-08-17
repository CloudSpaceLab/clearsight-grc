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
			Title: "Configuration checklist", Description: "Check identity, approval routing, sources and service health before granting wider access.", Illustration: "guided-orbit",
			Steps: []Step{
				{ID: "configure", Title: "Check configuration health", Description: "Review routing errors, inactive policies and tasks that still need an owner.", Action: "Open Configure", View: "configure", Target: "configure-workspace"},
				{ID: "routing", Title: "Test an approval route", Description: "Confirm that the selected approver is active, eligible and supported by the current policy.", Action: "Check approval route", View: "today", Target: "authority-action", Intent: "open-routing"},
				{ID: "imports", Title: "Review source imports", Description: "Imports retain the original file, extraction results and reviewer decisions.", Action: "Open Imports", View: "imports", Target: "document-import-form"},
				{ID: "finish", Title: "Complete remaining setup", Description: "Finish directory, policy and access changes through the configured approval process.", Action: "Done", View: "configure", Target: "configure-workspace"},
			},
		},
		{
			Code: "authorizer-first-run", Profile: "authorizer", Role: "Authorizer or signatory",
			RoleCodes: []string{"AUTHORIZER", "SIGNATORY"}, Priority: 90, Version: 1,
			Title: "Decision review", Description: "Review assigned decisions, supporting evidence and required follow-up checks.", Illustration: "guided-orbit",
			Steps: []Step{
				{ID: "today", Title: "Review assigned decisions", Description: "Today shows the reason, due date and next action for each decision assigned to you.", Action: "Open Today", View: "today", Target: "attention-list"},
				{ID: "routing", Title: "Check your authority", Description: "Review the legal entity, decision category, approval threshold and policy version.", Action: "Check approval route", View: "today", Target: "authority-action", Intent: "open-routing"},
				{ID: "matter", Title: "Review the decision record", Description: "Check the facts, open questions, options, actions and closure blockers before deciding.", Action: "Open decision record", View: "work", Target: "matters-workspace", Intent: "open-first-matter"},
				{ID: "finish", Title: "Confirm follow-up checks", Description: "Approval does not close the issue. Review the required outcome check and closure evidence.", Action: "Done", View: "work", Target: "matters-workspace"},
			},
		},
		{
			Code: "executive-first-run", Profile: "executive", Role: "Executive risk or compliance leader",
			RoleCodes: []string{"CRO", "CCO", "CISO", "DPO", "GENERAL_COUNSEL", "EXECUTIVE"}, Priority: 80, Version: 1,
			Title: "Executive review", Description: "Review priority work, Program status and supporting evidence.", Illustration: "guided-orbit",
			Steps: []Step{
				{ID: "brief", Title: "Review priority work", Description: "Today shows work assigned to you, due dates and data freshness.", Action: "Open Today", View: "today", Target: "today-brief"},
				{ID: "attention", Title: "Review a priority item", Description: "Open the first Program, issue or evidence request in the queue.", Action: "Review first item", View: "today", Target: "attention-list", Intent: "open-first-attention"},
				{ID: "programs", Title: "Check Program status", Description: "Programs show status, requirements, controls, evidence and open issues.", Action: "Open Programs", View: "programs", Target: "programs-workspace"},
				{ID: "finish", Title: "Review status details", Description: "Check the status reason, source, owner and next action.", Action: "Done", View: "programs", Target: "programs-workspace"},
			},
		},
		{
			Code: "reviewer-first-run", Profile: "reviewer", Role: "Reviewer or independent challenger",
			RoleCodes: []string{"REVIEWER", "CHALLENGER", "CONTROL_ASSURANCE_LEAD"}, Priority: 70, Version: 1,
			Title: "Review queue", Description: "Review assigned issues, evidence and independent outcome checks.", Illustration: "guided-orbit",
			Steps: []Step{
				{ID: "today", Title: "Review assigned work", Description: "Today shows reviews, approvals and evidence requests assigned to your role.", Action: "Open Today", View: "today", Target: "attention-list"},
				{ID: "matter", Title: "Check facts and open questions", Description: "Review known facts, missing information, conflicts, actions and outcome checks.", Action: "Open first issue", View: "work", Target: "matters-workspace", Intent: "open-first-matter"},
				{ID: "evidence", Title: "Request missing evidence", Description: "Check information already held by the bank before requesting a response.", Action: "Open evidence requests", View: "work", Target: "evidence-workspace", Intent: "switch-evidence"},
				{ID: "finish", Title: "Record an independent result", Description: "Keep the outcome check separate from the action it verifies.", Action: "Done", View: "work", Target: "matters-workspace"},
			},
		},
		{
			Code: "program-owner-first-run", Profile: "program-owner", Role: "Program or control owner",
			RoleCodes: []string{"PROGRAM_OWNER", "CONTROL_OWNER", "RISK_OWNER"}, Priority: 60, Version: 1,
			Title: "Program review", Description: "Check Program status, open work and supporting evidence.", Illustration: "guided-orbit",
			Steps: []Step{
				{ID: "programs", Title: "Review your Programs", Description: "Programs show requirements, controls, evidence checks and open issues.", Action: "Open Programs", View: "programs", Target: "programs-workspace"},
				{ID: "detail", Title: "Check status details", Description: "Open the first Program to review its status reasons, requirements and evidence checks.", Action: "Open first Program", View: "programs", Target: "programs-workspace", Intent: "open-first-program"},
				{ID: "evidence", Title: "Address missing evidence", Description: "Review what is missing, who is responding and when the response is due.", Action: "Open evidence requests", View: "work", Target: "evidence-workspace", Intent: "switch-evidence"},
				{ID: "finish", Title: "Confirm outcomes", Description: "Program status updates from approved requirements, controls, evidence and open issues.", Action: "Done", View: "programs", Target: "programs-workspace"},
			},
		},
		{
			Code: "evidence-respondent-first-run", Profile: "evidence-respondent", Role: "Evidence respondent",
			RoleCodes: []string{"EVIDENCE_RESPONDENT", "RECORDS_CUSTODIAN", "BUSINESS_OWNER"}, Priority: 50, Version: 1,
			Title: "Evidence requests", Description: "Review assigned requests and provide the requested information.", Illustration: "guided-orbit",
			Steps: []Step{
				{ID: "today", Title: "Review assigned requests", Description: "Today shows why each request is due and links to the request details.", Action: "Open Today", View: "today", Target: "attention-list"},
				{ID: "request", Title: "Check the request", Description: "Review the purpose, deadline and information already held by the bank.", Action: "Open evidence request", View: "work", Target: "evidence-workspace", Intent: "open-first-evidence"},
				{ID: "capture", Title: "Submit requested information", Description: "Your submission receives a receipt and remains subject to evidence review.", Action: "Open response form", View: "today", Target: "capture-action", Intent: "open-capture"},
				{ID: "finish", Title: "Report an incorrect assignment", Description: "Redirect the request or report that it was assigned to the wrong person or team.", Action: "Done", View: "work", Target: "evidence-workspace"},
			},
		},
		{
			Code: "auditor-first-run", Profile: "auditor", Role: "Auditor or read-only reviewer",
			RoleCodes: []string{"AUDITOR", "INTERNAL_AUDIT", "LEGAL_REVIEWER", "READ_ONLY"}, Priority: 40, Version: 1,
			Title: "Audit review", Description: "Review source evidence, decisions and outcome history without changing records.", Illustration: "guided-orbit",
			Steps: []Step{
				{ID: "programs", Title: "Review current Program status", Description: "Programs show the latest status, reasons and supporting records.", Action: "Open Programs", View: "programs", Target: "programs-workspace"},
				{ID: "matters", Title: "Review decisions and closure evidence", Description: "Issues and changes retain facts, decisions, actions, responses and outcome checks.", Action: "Open issues", View: "work", Target: "matters-workspace", Intent: "open-first-matter"},
				{ID: "imports", Title: "Trace imported source documents", Description: "Review the original file hash, extraction method, source locations and reviewer decisions.", Action: "Open Imports", View: "imports", Target: "document-import-workspace"},
				{ID: "finish", Title: "Check access limits", Description: "Search and counts exclude records outside your access scope.", Action: "Done", View: "imports", Target: "document-import-workspace"},
			},
		},
		{
			Code: "general-first-run", Profile: "general", Role: "ClearSight user", Priority: 0, Version: 1,
			Title: "Workspace guide", Description: "Review assigned work, Programs, issues and evidence.", Illustration: "guided-orbit",
			Steps: []Step{
				{ID: "today", Title: "Review assigned work", Description: "Today lists work assigned to you and explains why it needs attention.", Action: "Open Today", View: "today", Target: "today-brief"},
				{ID: "programs", Title: "Check Programs", Description: "Programs show requirements, controls, evidence checks and open issues.", Action: "Open Programs", View: "programs", Target: "programs-workspace"},
				{ID: "work", Title: "Review issues and evidence", Description: "Work shows issues, actions, responses and evidence requests with their source records.", Action: "Open Work", View: "work", Target: "matters-workspace"},
				{ID: "finish", Title: "Check completion evidence", Description: "A completed task or uploaded file still requires the applicable outcome check.", Action: "Done", View: "today", Target: "today-brief"},
			},
		},
	}
}
