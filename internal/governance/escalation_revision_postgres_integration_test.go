//go:build postgres && postgresintegration

package governance

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestEscalationGuardRevisionKeepsApprovedPolicyLiveUntilCheckerActivation(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	const (
		tenantID       = "98999999-9999-7999-8999-999999999901"
		entityID       = "98999999-9999-7999-8999-999999999912"
		makerID        = "98999999-9999-7999-8999-999999999902"
		checkerID      = "98999999-9999-7999-8999-999999999903"
		auditorID      = "98999999-9999-7999-8999-999999999904"
		auditorRoleID  = "98999999-9999-7999-8999-999999999905"
		supervisorRole = "98999999-9999-7999-8999-999999999906"
		complianceRole = "98999999-9999-7999-8999-999999999907"
		auditorPosID   = "98999999-9999-7999-8999-999999999908"
		auditorBindID  = "98999999-9999-7999-8999-999999999909"
		sourceID       = "98999999-9999-7999-8999-999999999910"
		groupID        = "98999999-9999-7999-8999-999999999911"
	)
	const tenantSlug = "eia5-policy-revision"
	now := time.Date(2026, 8, 11, 5, 0, 0, 0, time.UTC)

	cleanupGovernanceRevisionFixture(ctx, pool, tenantID)
	t.Cleanup(func() { cleanupGovernanceRevisionFixture(context.Background(), pool, tenantID) })

	mustExecGovernanceRevision(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'EIA5 policy revision')`, tenantID, tenantSlug)
	mustExecGovernanceRevision(t, ctx, pool, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1::uuid,$2::uuid,'EIA5-NG','EIA5 Nigeria','NG')`, entityID, tenantID)
	mustExecGovernanceRevision(t, ctx, pool, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
		($1::uuid,$4::uuid,'PERSON','Guard maker','ACTIVE',$5),
		($2::uuid,$4::uuid,'PERSON','Guard checker','ACTIVE',$5),
		($3::uuid,$4::uuid,'PERSON','Auditor','ACTIVE',$5)`, makerID, checkerID, auditorID, tenantID, now.Add(-time.Hour))
	mustExecGovernanceRevision(t, ctx, pool, `INSERT INTO role_templates(id,tenant_id,code,name,responsibilities,valid_from) VALUES
		($1::uuid,$4::uuid,'AUDITOR','Auditor',ARRAY['ESCALATION_OWNER'],$5),
		($2::uuid,$4::uuid,'SUPERVISOR','Supervisor',ARRAY['ESCALATION_OWNER'],$5),
		($3::uuid,$4::uuid,'COMPLIANCE_OFFICER','Compliance officer',ARRAY['ACCOUNTABLE_OWNER'],$5)`, auditorRoleID, supervisorRole, complianceRole, tenantID, now.Add(-time.Hour))
	mustExecGovernanceRevision(t, ctx, pool, `INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id,valid_from) VALUES($1::uuid,$2::uuid,$3::uuid,'AUDITOR','Auditor',$4::uuid,$5)`, auditorPosID, tenantID, entityID, auditorID, now.Add(-time.Hour))
	mustExecGovernanceRevision(t, ctx, pool, `INSERT INTO position_role_bindings(id,tenant_id,position_id,role_template_id,valid_from) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5)`, auditorBindID, tenantID, auditorPosID, auditorRoleID, now.Add(-time.Hour))
	mustExecGovernanceRevision(t, ctx, pool, `INSERT INTO scim_sources(id,tenant_id,code,token_hash,status) VALUES($1::uuid,$2::uuid,'ENTRA',decode(repeat('ac',32),'hex'),'ACTIVE')`, sourceID, tenantID)
	mustExecGovernanceRevision(t, ctx, pool, `INSERT INTO directory_groups(id,tenant_id,source_id,external_id,display_name) VALUES($1::uuid,$2::uuid,$3::uuid,'network-auditors','Network Auditors')`, groupID, tenantID, sourceID)

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	currentTime := now
	svc.now = func() time.Time { return currentTime }
	definition := json.RawMessage(`{
		"rules":[{"id":"auditor-route","legal_entity_id":"` + entityID + `","responsibility":"ESCALATION_OWNER","selector":{"kind":"ROLE","ref":"AUDITOR"}}],
		"escalations":[{"id":"compliance-overdue","trigger":"OVERDUE","steps":[{"after":"0s","responsibility":"ESCALATION_OWNER"}]}]
	}`)
	policy, err := svc.CreatePolicy(ctx, CreatePolicyInput{TenantID: tenantSlug, LegalEntityID: entityID, Code: "EIA5", Name: "EIA5", MakerID: makerID, Definition: definition})
	if err != nil {
		t.Fatal(err)
	}
	policy, err = svc.SubmitPolicy(ctx, TransitionInput{TenantID: tenantSlug, LegalEntityID: entityID, ID: policy.ID, ActorID: makerID, ExpectedVersion: policy.Version})
	if err != nil {
		t.Fatal(err)
	}
	currentTime = now.Add(time.Minute)
	policy, err = svc.ApprovePolicy(ctx, TransitionInput{TenantID: tenantSlug, LegalEntityID: entityID, ID: policy.ID, ActorID: checkerID, ExpectedVersion: policy.Version, Rationale: "Initial route reviewed"})
	if err != nil {
		t.Fatal(err)
	}
	if policy.Status != PolicyActive || policy.CurrentVersion != 1 {
		t.Fatalf("unexpected initial active policy: %#v", policy)
	}

	currentTime = now.Add(2 * time.Minute)
	revision, err := svc.ProposeEscalationGuardRevision(ctx, EscalationGuardRevisionInput{
		TenantID: tenantSlug, LegalEntityID: entityID, PolicyID: policy.ID, SequenceID: "compliance-overdue", StepIndex: 0,
		SourceRoles: []string{"COMPLIANCE_OFFICER"}, TargetRoles: []string{"SUPERVISOR"}, TargetGroupIDs: []string{groupID},
		ActorID: makerID, ExpectedPolicyVersion: policy.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if revision.Version != 2 || revision.BaseVersion != 1 {
		t.Fatalf("unexpected pending revision: %#v", revision)
	}

	policyAfterProposal, err := repo.GetPolicy(ctx, tenantSlug, policy.ID)
	if err != nil {
		t.Fatal(err)
	}
	if policyAfterProposal.CurrentVersion != 1 {
		t.Fatalf("pending guard revision displaced approved current version: %d", policyAfterProposal.CurrentVersion)
	}
	var pendingApproved *time.Time
	if err := pool.QueryRow(ctx, `SELECT approved_at FROM routing_policy_versions WHERE policy_id=$1::uuid AND version=2`, policy.ID).Scan(&pendingApproved); err != nil {
		t.Fatal(err)
	}
	if pendingApproved != nil {
		t.Fatalf("pending revision was prematurely approved at %v", pendingApproved)
	}
	var activeRouteVersion string
	if err := pool.QueryRow(ctx, `SELECT policy_version FROM effective_authority_routes WHERE source_policy_id=$1::uuid AND valid_until IS NULL ORDER BY priority DESC LIMIT 1`, policy.ID).Scan(&activeRouteVersion); err != nil {
		t.Fatal(err)
	}
	if activeRouteVersion != "EIA5:v1" {
		t.Fatalf("pending guard revision changed effective authority route version: %s", activeRouteVersion)
	}

	if _, err := svc.ApprovePolicyRevision(ctx, ApprovePolicyRevisionInput{
		TenantID: tenantSlug, LegalEntityID: entityID, PolicyID: policy.ID, RevisionVersion: revision.Version, ActorID: makerID,
		ExpectedPolicyVersion: policyAfterProposal.Version, Rationale: "self approval",
	}); !errors.Is(err, ErrMakerChecker) {
		t.Fatalf("expected maker-checker rejection, got %v", err)
	}

	currentTime = now.Add(3 * time.Minute)
	approved, err := svc.ApprovePolicyRevision(ctx, ApprovePolicyRevisionInput{
		TenantID: tenantSlug, LegalEntityID: entityID, PolicyID: policy.ID, RevisionVersion: revision.Version, ActorID: checkerID,
		ExpectedPolicyVersion: policyAfterProposal.Version, Rationale: "Approved role and group target boundary",
	})
	if err != nil {
		t.Fatal(err)
	}
	if approved.CurrentVersion != 2 {
		t.Fatalf("approved revision did not become current: %#v", approved)
	}
	var oldUntil, newApproved *time.Time
	if err := pool.QueryRow(ctx, `SELECT effective_until FROM routing_policy_versions WHERE policy_id=$1::uuid AND version=1`, policy.ID).Scan(&oldUntil); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT approved_at FROM routing_policy_versions WHERE policy_id=$1::uuid AND version=2`, policy.ID).Scan(&newApproved); err != nil {
		t.Fatal(err)
	}
	if oldUntil == nil || newApproved == nil {
		t.Fatalf("policy version handoff was not effective-dated: old_until=%v new_approved=%v", oldUntil, newApproved)
	}
	if err := pool.QueryRow(ctx, `SELECT policy_version FROM effective_authority_routes WHERE source_policy_id=$1::uuid AND valid_until IS NULL ORDER BY priority DESC LIMIT 1`, policy.ID).Scan(&activeRouteVersion); err != nil {
		t.Fatal(err)
	}
	if activeRouteVersion != "EIA5:v2" {
		t.Fatalf("approved revision did not refresh effective authority route: %s", activeRouteVersion)
	}

	// A stale/nonexistent group may be proposed, but activation must validate
	// current directory truth and leave version 2 active when validation fails.
	approvedPolicy, _ := repo.GetPolicy(ctx, tenantSlug, policy.ID)
	currentTime = now.Add(4 * time.Minute)
	badRevision, err := svc.ProposeEscalationGuardRevision(ctx, EscalationGuardRevisionInput{
		TenantID: tenantSlug, LegalEntityID: entityID, PolicyID: policy.ID, SequenceID: "compliance-overdue", StepIndex: 0,
		SourceRoles: []string{"COMPLIANCE_OFFICER"}, TargetGroupIDs: []string{"98999999-9999-7999-8999-999999999999"},
		ActorID: makerID, ExpectedPolicyVersion: approvedPolicy.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	policyWithBadProposal, _ := repo.GetPolicy(ctx, tenantSlug, policy.ID)
	_, err = svc.ApprovePolicyRevision(ctx, ApprovePolicyRevisionInput{
		TenantID: tenantSlug, LegalEntityID: entityID, PolicyID: policy.ID, RevisionVersion: badRevision.Version, ActorID: checkerID,
		ExpectedPolicyVersion: policyWithBadProposal.Version, Rationale: "Should fail reference validation",
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected inactive/missing group reference conflict, got %v", err)
	}
	stillActive, _ := repo.GetPolicy(ctx, tenantSlug, policy.ID)
	if stillActive.CurrentVersion != 2 {
		t.Fatalf("failed revision activation changed current policy: %#v", stillActive)
	}
}

func mustExecGovernanceRevision(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, sql, args...); err != nil {
		t.Fatal(err)
	}
}

func cleanupGovernanceRevisionFixture(ctx context.Context, pool *pgxpool.Pool, tenantID string) {
	_, _ = pool.Exec(ctx, `DELETE FROM outbox_events WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM governance_decisions WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM effective_authority_routes WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM routing_policy_versions WHERE policy_id IN (SELECT id FROM routing_policies WHERE tenant_id=$1::uuid)`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM routing_policies WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM directory_group_members WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM directory_group_role_bindings WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM directory_groups WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM scim_users WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM scim_sources WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM position_role_bindings WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM org_positions WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM role_templates WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
}
