//go:build postgres

package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

const scoringAcceptanceFormCode = "RESPONSE-POLICY-ACCEPTANCE"

type scoringAcceptanceResult struct {
	FormID        string                              `json:"form_id"`
	FormVersion   int64                               `json:"form_version"`
	SubjectID     string                              `json:"subject_id"`
	Responses     []evidence.CompletedResponseSummary `json:"responses"`
	AlreadySeeded bool                                `json:"already_seeded"`
}

type scoringResponseFixture struct {
	label   string
	answers map[string]string
}

var scoringResponseFixtures = []scoringResponseFixture{
	{label: "good", answers: map[string]string{"control_designed": "Yes", "control_operating": "Yes", "exceptions_resolved": "Yes", "critical_gap": "No"}},
	{label: "borderline", answers: map[string]string{"control_designed": "Yes", "control_operating": "No", "exceptions_resolved": "Yes", "critical_gap": "No"}},
	{label: "poor", answers: map[string]string{"control_designed": "No", "control_operating": "No", "exceptions_resolved": "No", "critical_gap": "Yes"}},
}

func seedScoringAcceptanceResponses(
	ctx context.Context,
	cfg config.Config,
	pool *pgxpool.Pool,
	seed bankverticals.SeedConfig,
	journeys []bankverticals.Journey,
	monitoringRepo *monitoring.PostgresRepository,
	evidenceRepo *evidence.PostgresRepository,
) (scoringAcceptanceResult, error) {
	if len(cfg.RecipientSecurity.Keyring) == 0 || strings.TrimSpace(cfg.RecipientSecurity.ActiveKeyID) == "" || cfg.RecipientSecurity.AccessHMACKey == ([32]byte{}) {
		return scoringAcceptanceResult{}, fmt.Errorf("scoring acceptance requires CLEARSIGHT_RECIPIENT_KEYRING, CLEARSIGHT_RECIPIENT_ACTIVE_KEY_ID and CLEARSIGHT_ACCESS_HMAC_KEY")
	}
	keyring, err := evidence.NewRecipientKeyring(cfg.RecipientSecurity.ActiveKeyID, cfg.RecipientSecurity.Keyring)
	if err != nil {
		return scoringAcceptanceResult{}, fmt.Errorf("configure scoring acceptance recipient keyring: %w", err)
	}
	forms, err := monitoringRepo.ListReusableFormRevisions(ctx, seed.TenantID, seed.LegalEntityID, 100)
	if err != nil {
		return scoringAcceptanceResult{}, fmt.Errorf("list reference forms: %w", err)
	}
	var form monitoring.FormTemplate
	for _, candidate := range forms {
		if candidate.Code == scoringAcceptanceFormCode && candidate.Status == monitoring.LifecycleActive && candidate.IsCurrent {
			form = candidate
			break
		}
	}
	if form.ID == "" {
		return scoringAcceptanceResult{}, fmt.Errorf("active %s form revision was not installed", scoringAcceptanceFormCode)
	}
	subjectID := referenceProgramID(journeys)
	if subjectID == "" {
		return scoringAcceptanceResult{}, fmt.Errorf("reference scoring acceptance requires an installed Program subject")
	}
	result := scoringAcceptanceResult{FormID: form.ID, FormVersion: form.Version, SubjectID: subjectID}
	seeded, err := scoringAcceptanceAlreadySeeded(ctx, pool, seed, form.ID, form.Version, subjectID)
	if err != nil {
		return scoringAcceptanceResult{}, err
	}
	if seeded {
		result.AlreadySeeded = true
		return result, nil
	}

	distributionStore := evidence.NewPostgresDistributionStore(evidenceRepo, keyring)
	distributions := evidence.NewDistributionService(distributionStore)
	access, err := evidence.NewDistributionAccessService(distributionStore, keyring, nil, cfg.RecipientSecurity.AccessHMACKey, cfg.CaptureSessionTTL)
	if err != nil {
		return scoringAcceptanceResult{}, fmt.Errorf("configure scoring acceptance access: %w", err)
	}
	now := time.Now().UTC()
	for index, fixture := range scoringResponseFixtures {
		completed, submitErr := submitScoringAcceptanceResponse(ctx, distributions, access, seed, form, subjectID, fixture, now.Add(time.Duration(index)*time.Minute))
		if submitErr != nil {
			return scoringAcceptanceResult{}, fmt.Errorf("submit %s scoring acceptance response: %w", fixture.label, submitErr)
		}
		result.Responses = append(result.Responses, completed)
	}
	return result, nil
}

func submitScoringAcceptanceResponse(
	ctx context.Context,
	distributions *evidence.DistributionService,
	access *evidence.DistributionAccessService,
	seed bankverticals.SeedConfig,
	form monitoring.FormTemplate,
	subjectID string,
	fixture scoringResponseFixture,
	now time.Time,
) (evidence.CompletedResponseSummary, error) {
	title := "Scoring acceptance — " + fixture.label
	bundle, err := distributions.Create(ctx, evidence.CreateDistributionInput{
		TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID,
		FormTemplateID: form.ID, FormTemplateVersion: form.Version,
		SubjectType: "PROGRAM", SubjectID: subjectID,
		Title: title, Purpose: "Persist a governed scored response through the same respondent path used by external evidence collection.",
		AccessPolicy: evidence.AccessDirectMagicLink, EstimatedMinutes: 2,
		Deadline: now.Add(30 * 24 * time.Hour), RouteExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedBy:  seed.ActorID,
		Recipients: []evidence.DistributionRecipientInput{{Role: evidence.RecipientTo, Type: evidence.RecipientExternalAudience, Address: fixture.label + "@scoring.demo.invalid", AudienceHint: fixture.label + " scoring respondent", ContactLabel: "Scoring acceptance " + fixture.label}},
	})
	if err != nil {
		return evidence.CompletedResponseSummary{}, err
	}
	routes, err := access.IssueDistributionAccessRoutes(ctx, seed.TenantID, seed.LegalEntityID, bundle.Distribution.ID, seed.ActorID)
	if err != nil || len(routes) != 1 {
		return evidence.CompletedResponseSummary{}, fmt.Errorf("issue direct route: routes=%d err=%w", len(routes), err)
	}
	session, err := access.RedeemDirectRoute(ctx, routes[0].Selector)
	if err != nil {
		return evidence.CompletedResponseSummary{}, err
	}
	workspace, err := access.GetResponseWorkspace(ctx, session.SessionToken)
	if err != nil {
		return evidence.CompletedResponseSummary{}, err
	}
	edits := make([]evidence.FieldEdit, 0, len(fixture.answers))
	for _, fieldID := range []string{"control_designed", "control_operating", "exceptions_resolved", "critical_gap"} {
		edits = append(edits, evidence.FieldEdit{FieldID: fieldID, Value: formcontract.TextAnswer(fixture.answers[fieldID]), BaseSequence: workspace.FieldSequences[fieldID]})
	}
	workspace, err = access.SaveResponseWorkspace(ctx, session.SessionToken, evidence.SaveWorkspaceInput{ExpectedVersion: workspace.Workspace.Version, Edits: edits})
	if err != nil {
		return evidence.CompletedResponseSummary{}, err
	}
	submitted, err := access.SubmitResponseWorkspace(ctx, session.SessionToken, evidence.SubmitWorkspaceInput{ExpectedVersion: workspace.Workspace.Version})
	if err != nil {
		return evidence.CompletedResponseSummary{}, err
	}
	if submitted.Revision.Score == nil || submitted.Revision.Score.State != evidence.ResponseScoreFinal || !submitted.Revision.Score.Final {
		return evidence.CompletedResponseSummary{}, fmt.Errorf("%s response did not produce a final score: %#v", fixture.label, submitted.Revision.Score)
	}
	return distributions.GetCompletedResponseForExecution(ctx, seed.TenantID, submitted.Revision.ID)
}

func scoringAcceptanceAlreadySeeded(ctx context.Context, pool *pgxpool.Pool, seed bankverticals.SeedConfig, formID string, version int64, subjectID string) (bool, error) {
	var count int
	err := pool.QueryRow(ctx, `
		SELECT count(DISTINCT d.title)
		FROM capture_response_revisions rr
		JOIN capture_form_distributions d ON d.id=rr.distribution_id AND d.tenant_id=rr.tenant_id AND d.legal_entity_id=rr.legal_entity_id
		WHERE rr.tenant_id=$1::uuid AND rr.legal_entity_id=$2::uuid AND rr.is_current
		  AND d.form_template_id=$3::uuid AND d.form_template_version=$4 AND d.subject_type='PROGRAM' AND d.subject_id=$5::uuid
		  AND d.title IN ('Scoring acceptance — good','Scoring acceptance — borderline','Scoring acceptance — poor')`, seed.TenantID, seed.LegalEntityID, formID, version, subjectID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check scoring acceptance population: %w", err)
	}
	return count == len(scoringResponseFixtures), nil
}

func referenceProgramID(journeys []bankverticals.Journey) string {
	for _, journey := range journeys {
		if strings.TrimSpace(journey.ProgramID) != "" {
			return journey.ProgramID
		}
	}
	return ""
}
