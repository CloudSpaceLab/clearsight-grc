package reporting

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

// DemoScope carries the resolved identifiers a PostgreSQL demo install needs.
// The memory composition addresses demo records by slug, so the installer
// cannot assume any identifier representation. The caller resolves the scope
// and the three governance principals; this file only drives the lifecycle.
type DemoScope struct {
	TenantID      string
	LegalEntityID string

	// The three principals are distinct. A definition cannot be reviewed by its
	// proposer or activated by either, so the demo must resolve real,
	// separately-held principals rather than reusing one account.
	MakerPrincipalID      string
	ReviewerPrincipalID   string
	AuthorizerPrincipalID string

	// ScopeRefs maps a seeded definition's scope target onto the identifier that
	// actually exists in this database. A definition whose target is absent is
	// skipped rather than written with a dangling reference.
	ScopeRefs map[string]string
}

// InstallPostgresDemo seeds governed report definitions through the real
// service, so the deployed demo exercises the same propose -> submit -> review
// -> activate path a bank user would. Writing definitions straight into a
// repository, as the memory installer does, would demonstrate a lifecycle the
// product does not actually have.
//
// Each step runs under a verified person identity, because the service refuses
// a service identity for a material command. A stable definition code is the
// idempotency key: a repeat install finds the existing definition and leaves
// its governed history untouched.
func InstallPostgresDemo(ctx context.Context, service *Service, scope DemoScope) error {
	if service == nil || service.repo == nil {
		return ErrInvalid
	}
	if scope.TenantID == "" || scope.LegalEntityID == "" {
		return ErrInvalid
	}
	if scope.MakerPrincipalID == "" || scope.ReviewerPrincipalID == "" || scope.AuthorizerPrincipalID == "" {
		return fmt.Errorf("%w: the demo report scope needs a distinct proposer, reviewer and authorizer", ErrInvalid)
	}
	if scope.MakerPrincipalID == scope.ReviewerPrincipalID ||
		scope.MakerPrincipalID == scope.AuthorizerPrincipalID ||
		scope.ReviewerPrincipalID == scope.AuthorizerPrincipalID {
		return fmt.Errorf("%w: the demo report proposer, reviewer and authorizer must be different principals", ErrInvalid)
	}

	reportScope := ReportScope{TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID}
	existing, err := service.ListDefinitions(demoActorContext(ctx, reportScope, scope.MakerPrincipalID, service.now()), reportScope, false)
	if err != nil {
		return fmt.Errorf("list demo report definitions: %w", err)
	}
	installed := make(map[string]struct{}, len(existing))
	for _, definition := range existing {
		installed[definition.Code] = struct{}{}
	}

	for _, seed := range demoDefinitionSeeds {
		if _, alreadyInstalled := installed[seed.code]; alreadyInstalled {
			// Leaving it alone is what makes a repeated install safe: a second
			// run must not rewrite governed revision history.
			continue
		}
		scopeRef := seed.scopeRef
		if scopeRef != "" {
			resolved, ok := scope.ScopeRefs[seed.code]
			if !ok {
				continue
			}
			scopeRef = resolved
		}
		if err := installPostgresDemoDefinition(ctx, service, reportScope, scope, seed, scopeRef); err != nil {
			return err
		}
	}
	return installPostgresDemoBoundedStopRun(ctx, service, reportScope, scope)
}

// installPostgresDemoBoundedStopRun records a report run that stopped on the
// row ceiling.
//
// This is deliberately a receipt, not a real execution: the seeded estate has
// eight activities and could never reach a 10,000-row ceiling, so a genuine run
// would not stop. A deployed demo that only ever shows successful reports
// teaches an operator that a bounded stop is not something that happens. What
// this run communicates is that it produced no file at all rather than a short
// one, so RowCount stays zero and no artefact keys are set — exactly what the
// service records for a bounded stop.
func installPostgresDemoBoundedStopRun(ctx context.Context, service *Service, reportScope ReportScope, scope DemoScope) error {
	repository, ok := service.repo.(RunRepository)
	if !ok {
		return fmt.Errorf("%w: the report repository does not accept run receipts", ErrInvalid)
	}
	makerContext := demoActorContext(ctx, reportScope, scope.MakerPrincipalID, service.now())
	existing, err := repository.ListRuns(makerContext, reportScope, "", ReportRunPageSize)
	if err != nil {
		return fmt.Errorf("list demo report runs: %w", err)
	}
	for _, run := range existing {
		if run.FailureCode == FailureRowLimitExceeded {
			return nil
		}
	}

	definitions, err := service.ListDefinitions(makerContext, reportScope, false)
	if err != nil {
		return fmt.Errorf("list demo report definitions: %w", err)
	}
	var target *ReportDefinition
	for i := range definitions {
		if definitions[i].Status == DefinitionActive {
			target = &definitions[i]
			break
		}
	}
	if target == nil {
		// With no active definition there is nothing for a run to reference, and
		// inventing a reference would leave a dangling sample record.
		return nil
	}

	now := service.now().UTC()
	runID, err := id.NewUUIDv7()
	if err != nil {
		return err
	}
	// The run is created in its real QUEUED state and then failed, so the
	// receipt reaches FAILED through the same state machine a genuine bounded
	// stop would. Writing a terminal row directly would assert a state the
	// repository deliberately refuses to accept at insert time.
	queued := ReportRun{
		ID: runID, TenantID: reportScope.TenantID, LegalEntityID: reportScope.LegalEntityID,
		DefinitionID: target.ID, DefinitionVersion: target.CurrentVersion,
		DefinitionCode: target.Code, DefinitionChecksum: target.StoredChecksum,
		ScopeKind: target.ScopeKind, ScopeRef: target.ScopeRef, RequestedByRef: scope.MakerPrincipalID,
		AsOf: now.Add(-30 * time.Minute), Filter: cloneReportFilter(target.Filter),
		Dataset: target.Dataset, Format: target.Format, Status: RunQueued,
		CreatedAt: now.Add(-25 * time.Minute), ExpiresAt: now.Add(ReportRunRetention),
		SourceBoundary: SourceBoundary{
			CapturedAt: now.Add(-30 * time.Minute), ProjectionVersion: "sample-report-bound.v1",
			SourceHighWater: map[string]time.Time{"processing_activities": now.Add(-30 * time.Minute)},
			Population:      MaxReportRunRows + 1, PopulationComplete: true,
		},
	}
	if _, err := repository.CreateRun(makerContext, reportScope, queued); err != nil {
		return fmt.Errorf("record the demo bounded-stop report run: %w", err)
	}
	// The repository only fails a RUNNING run, so the receipt walks the same
	// queue -> running -> failed path a real bounded stop would. Claiming with
	// a dedicated worker identifier keeps the sample run separate from any
	// worker the deployment may be running.
	claimed, err := repository.ClaimQueuedRuns(makerContext, reportScope, "demo-install", 1)
	if err != nil {
		return fmt.Errorf("claim the demo bounded-stop report run: %w", err)
	}
	for _, candidate := range claimed {
		if candidate.ID != runID {
			continue
		}
		if _, err := repository.FailRun(makerContext, reportScope, runID, FailureRowLimitExceeded); err != nil {
			return fmt.Errorf("record the demo bounded-stop report run failure: %w", err)
		}
		return nil
	}
	return fmt.Errorf("the demo bounded-stop report run was not claimed")
}

// demoActorContext binds a verified person identity for a demo command. The
// service takes its actor and scope from the request context, so a demo that
// wants the governed path must supply them the same way a request does rather
// than asserting them directly. The identity carries a live session window
// because the service rejects an expired actor, exactly as it would for a real
// request.
func demoActorContext(ctx context.Context, scope ReportScope, principalID string, now time.Time) context.Context {
	return identity.WithActor(ctx, identity.Actor{
		TenantID:             scope.TenantID,
		LegalEntityID:        scope.LegalEntityID,
		PrincipalID:          principalID,
		Kind:                 "PERSON",
		AuthenticationMethod: "demo",
		AssuranceLevel:       "aal2",
		SessionID:            "demo-report-install",
		IssuedAt:             now.Add(-time.Minute),
		ExpiresAt:            now.Add(time.Hour),
	})
}

func installPostgresDemoDefinition(
	ctx context.Context,
	service *Service,
	reportScope ReportScope,
	scope DemoScope,
	seed demoDefinitionSeed,
	scopeRef string,
) error {
	definition, err := service.Propose(demoActorContext(ctx, reportScope, scope.MakerPrincipalID, service.now()), ProposeInput{
		Code:        seed.code,
		Name:        seed.name,
		Description: seed.description,
		Dataset:     seed.dataset,
		ScopeKind:   seed.scopeKind,
		ScopeRef:    scopeRef,
		Format:      seed.format,
		Filter:      seed.filter,
	})
	if err != nil {
		return fmt.Errorf("propose demo report definition %s: %w", seed.code, err)
	}

	// The lifecycle below is the governed path, not a shortcut. A step is only
	// taken when the seeded definition is meant to end in that state, so the
	// demo shows a definition still awaiting review as well as an active one.
	definition, err = service.Submit(demoActorContext(ctx, reportScope, scope.MakerPrincipalID, service.now()), DefinitionTransitionInput{
		Scope:           reportScope,
		DefinitionID:    definition.ID,
		ExpectedVersion: definition.Version,
		ChecksumSeen:    definition.Checksum(),
	})
	if err != nil {
		return fmt.Errorf("submit demo report definition %s: %w", seed.code, err)
	}
	if seed.status == DefinitionDraft || seed.status == DefinitionPendingReview {
		return nil
	}

	definition, err = service.Review(demoActorContext(ctx, reportScope, scope.ReviewerPrincipalID, service.now()), DefinitionTransitionInput{
		Scope:           reportScope,
		DefinitionID:    definition.ID,
		ExpectedVersion: definition.Version,
		ChecksumSeen:    definition.Checksum(),
		Note:            "Sample review: the filter, scope and dataset match the stated reporting purpose.",
	})
	if err != nil {
		return fmt.Errorf("review demo report definition %s: %w", seed.code, err)
	}
	if seed.status != DefinitionActive {
		return nil
	}

	past := service.now().UTC().Add(-time.Hour)
	if _, err = service.Activate(demoActorContext(ctx, reportScope, scope.AuthorizerPrincipalID, service.now()), DefinitionTransitionInput{
		Scope:           reportScope,
		DefinitionID:    definition.ID,
		ExpectedVersion: definition.Version,
		ChecksumSeen:    definition.Checksum(),
		Note:            "Sample activation: approved for the seeded demo estate.",
		EffectiveFrom:   &past,
	}); err != nil {
		return fmt.Errorf("activate demo report definition %s: %w", seed.code, err)
	}
	return nil
}

// DemoScopeRefs resolves the Program and Matter a seeded report definition is
// scoped to, by looking them up in the seeded demo estate. A definition whose
// target cannot be found is reported rather than written with a dangling
// reference.
func DemoScopeRefs(ctx context.Context, lookup func(ctx context.Context, scope ReportScope, code string) (string, error), scope ReportScope) (map[string]string, error) {
	refs := make(map[string]string, len(demoDefinitionSeeds))
	for _, seed := range demoDefinitionSeeds {
		if strings.TrimSpace(seed.scopeRef) == "" || seed.scopeKind == ScopeLegalEntity {
			continue
		}
		resolved, err := lookup(ctx, scope, seed.scopeRef)
		if err != nil || strings.TrimSpace(resolved) == "" {
			continue
		}
		refs[seed.code] = resolved
	}
	return refs, nil
}
