package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func createProgramForCalculationPolicy(t *testing.T, service *Service, ctx context.Context, now time.Time) ProgramAggregate {
	t.Helper()
	value, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "SECURITY",
		Name: "Security assurance", Type: "SECURITY", OwningFunction: "Risk",
		OwnerPrincipalID: "program-owner", AuthorityPrincipalID: "independent-checker",
		Scope: json.RawMessage(`{}`), EffectiveFrom: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func calculationPolicyFixture() ProgramComplianceCalculationPolicy {
	return ProgramComplianceCalculationPolicy{
		Revision: "caller-controlled-not-authoritative",
		MinimumCoverageBps: 9000,
		Categories: []ProgramComplianceCategoryWeight{
			{Code: "REQUIREMENTS", WeightBps: 6000},
			{Code: "VENDORS", WeightBps: 4000},
		},
	}
}

func TestProgramCompliancePolicyGovernedRevisionAndHistory(t *testing.T) {
	ctx := WithTrustedSystemScope(context.Background())
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	program := createProgramForCalculationPolicy(t, service, ctx, now)
	proposedAt := now.Add(time.Minute)
	service.now = func() time.Time { return proposedAt }

	proposal, err := service.ProposeProgramCompliancePolicy(ctx, ProposeProgramCompliancePolicyInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		ActorID: "program-owner", Rationale: "Set quarterly assessment weights.",
		Definition: calculationPolicyFixture(),
	})
	if err != nil {
		t.Fatal(err)
	}
	position := proposal.Program.CalculationPolicy
	if position == nil || position.Pending == nil || position.Active != nil ||
		position.LastRevision != 1 || position.Pending.Number != 1 ||
		position.Pending.Definition.Revision != "program-v1" ||
		position.Pending.MakerID != "program-owner" ||
		len(position.Pending.Checksum) != 64 {
		t.Fatalf("proposal was not a governed immutable revision: %+v", position)
	}
	if _, err := service.ProposeProgramCompliancePolicy(ctx, ProposeProgramCompliancePolicyInput{
		TenantID: "bank", ProgramID: program.Program.ID,
		ExpectedVersion: proposal.Program.Version, ActorID: "program-owner",
		Rationale: "Replace unreviewed proposal.", Definition: calculationPolicyFixture(),
	}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("concurrent proposal replaced awaiting review: %v", err)
	}

	before, err := service.ProgramAt(ctx, "bank", program.Program.ID, proposedAt)
	if err != nil || before.Program.CalculationPolicy == nil ||
		before.Program.CalculationPolicy.Active != nil ||
		before.Program.CalculationPolicy.Pending == nil {
		t.Fatalf("point-in-time proposal was lost: %+v / %v", before.Program.CalculationPolicy, err)
	}
	eventCount, _ := repo.ProgramEvents(ctx, "bank", program.Program.ID, nil)
	if _, err := service.ApproveProgramCompliancePolicy(ctx, DecideProgramCompliancePolicyInput{
		TenantID: "bank", ProgramID: program.Program.ID,
		ExpectedVersion: proposal.Program.Version, ActorID: "program-owner",
		Rationale: "Self approval attempt",
	}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("maker self-approved policy: %v", err)
	}
	afterInvalid, _ := repo.ProgramEvents(ctx, "bank", program.Program.ID, nil)
	if len(afterInvalid) != len(eventCount) {
		t.Fatal("failed approval committed an event")
	}

	approvedAt := proposedAt.Add(time.Minute)
	service.now = func() time.Time { return approvedAt }
	approved, err := service.ApproveProgramCompliancePolicy(ctx, DecideProgramCompliancePolicyInput{
		TenantID: "bank", ProgramID: program.Program.ID,
		ExpectedVersion: proposal.Program.Version, ActorID: "independent-checker",
		Rationale: "Weights and evidence coverage reviewed.",
	})
	if err != nil {
		t.Fatal(err)
	}
	position = approved.Program.CalculationPolicy
	if position == nil || position.Pending != nil || position.Active == nil ||
		position.Active.ApproverID != "independent-checker" ||
		position.Active.ApprovedAt == nil || !position.Active.ApprovedAt.Equal(approvedAt) ||
		position.Active.Checksum != proposal.Program.CalculationPolicy.Pending.Checksum {
		t.Fatalf("approved proposal was not made effective: %+v", position)
	}
	history, err := service.ProgramAt(ctx, "bank", program.Program.ID, proposedAt)
	if err != nil || history.Program.CalculationPolicy == nil ||
		history.Program.CalculationPolicy.Active != nil ||
		history.Program.CalculationPolicy.Pending == nil {
		t.Fatalf("approval rewrote historical pending policy: %+v / %v", history.Program.CalculationPolicy, err)
	}
	events, _ := repo.ProgramEvents(ctx, "bank", program.Program.ID, nil)
	if events[len(events)-1].Type != EventProgramCalculationPolicyApproved ||
		events[len(events)-1].ActorID != "independent-checker" {
		t.Fatalf("approval is not independently audited: %+v", events[len(events)-1])
	}
	// New proposals may not replace an active policy before independent review.
	service.now = func() time.Time { return approvedAt.Add(time.Minute) }
	revisionTwo, err := service.ProposeProgramCompliancePolicy(ctx, ProposeProgramCompliancePolicyInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: approved.Program.Version,
		ActorID: "program-owner", Rationale: "Annual scope adjustment.",
		Definition: calculationPolicyFixture(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if revisionTwo.Program.CalculationPolicy.Pending.Number != 2 ||
		revisionTwo.Program.CalculationPolicy.Active.Number != 1 {
		t.Fatalf("proposal silently displaced active revision: %+v", revisionTwo.Program.CalculationPolicy)
	}
	rejected, err := service.RejectProgramCompliancePolicy(ctx, DecideProgramCompliancePolicyInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: revisionTwo.Program.Version,
		ActorID: "independent-checker", Rationale: "Maintain previous approved weights.",
	})
	if err != nil || rejected.Program.CalculationPolicy.Pending != nil ||
		rejected.Program.CalculationPolicy.Active.Number != 1 ||
		rejected.Program.CalculationPolicy.LastRevision != 2 {
		t.Fatalf("rejection did not retain existing policy/revision sequence: %+v / %v", rejected.Program.CalculationPolicy, err)
	}
}

func TestProgramCompliancePolicyInvalidOrStaleProposalsLeaveNoEvent(t *testing.T) {
	ctx := WithTrustedSystemScope(context.Background())
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	program := createProgramForCalculationPolicy(t, service, ctx, now)
	before, _ := repo.ProgramEvents(ctx, "bank", program.Program.ID, nil)
	invalid := calculationPolicyFixture()
	invalid.Categories[0].WeightBps = 5500
	for _, tc := range []struct {
		name string
		version int64
		actor string
		policy ProgramComplianceCalculationPolicy
	}{
		{name: "invalid weights", version: program.Program.Version, actor: "owner", policy: invalid},
		{name: "no actor", version: program.Program.Version, actor: "", policy: calculationPolicyFixture()},
		{name: "stale program version", version: program.Program.Version + 1, actor: "owner", policy: calculationPolicyFixture()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := service.ProposeProgramCompliancePolicy(ctx, ProposeProgramCompliancePolicyInput{
				TenantID: "bank", ProgramID: program.Program.ID,
				ExpectedVersion: tc.version, ActorID: tc.actor, Rationale: "Review",
				Definition: tc.policy,
			}); err == nil {
				t.Fatalf("invalid proposal committed: %s", tc.name)
			}
		})
	}
	after, _ := repo.ProgramEvents(ctx, "bank", program.Program.ID, nil)
	if len(before) != len(after) {
		t.Fatalf("failed proposal wrote event: %d->%d", len(before), len(after))
	}
}
