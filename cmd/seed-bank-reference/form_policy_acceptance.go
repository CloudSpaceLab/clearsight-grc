//go:build postgres

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formpolicy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

const responsePolicyAcceptanceCode = "reference-poor-control-response"

type formPolicyAcceptanceResult struct {
	PolicyID                  string `json:"policy_id"`
	PolicyVersion             int64  `json:"policy_version"`
	GoodResponseID            string `json:"good_response_id"`
	PoorResponseID            string `json:"poor_response_id"`
	SameEpisodeResponseID     string `json:"same_episode_response_id"`
	GoodExecutionState        string `json:"good_execution_state"`
	PoorExecutionState        string `json:"poor_execution_state"`
	SameEpisodeExecutionState string `json:"same_episode_execution_state"`
	MatterID                  string `json:"matter_id"`
	CreatedMatter             bool   `json:"created_matter"`
	ReplayExecutionID         string `json:"replay_execution_id"`
	ExactlyOnceMatter         bool   `json:"exactly_once_matter"`
	SameEpisodeReusedMatter   bool   `json:"same_episode_reused_matter"`
	AlreadySeeded             bool   `json:"already_seeded"`
}

type seedFormReader struct {
	repo *monitoring.PostgresRepository
}

func (r seedFormReader) GetDistributionFormRevision(ctx context.Context, tenantID, legalEntityID, formID string, version int64) (evidence.DistributionFormRevision, error) {
	form, err := r.repo.ReusableFormRevision(ctx, tenantID, legalEntityID, formID, version)
	if err != nil {
		return evidence.DistributionFormRevision{}, err
	}
	return evidence.DistributionFormRevision{
		ID:            form.ID,
		TenantID:      form.TenantID,
		LegalEntityID: form.LegalEntityID,
		Version:       form.Version,
		Sensitivity:   form.Sensitivity,
		Presentation:  form.Presentation,
		ScoringMode:   form.ScoringMode,
		ScoreProfile:  form.ScoreProfile,
		Sections:      append([]formcontract.Section(nil), form.Sections...),
		Fields:        append([]formcontract.Field(nil), form.Fields...),
		Active:        form.Status == monitoring.LifecycleActive && form.IsCurrent,
	}, nil
}

func seedFormPolicyAcceptance(
	ctx context.Context,
	cfg config.Config,
	pool *pgxpool.Pool,
	seed bankverticals.SeedConfig,
	monitoringRepo *monitoring.PostgresRepository,
	evidenceRepo *evidence.PostgresRepository,
	scoring scoringAcceptanceResult,
) (formPolicyAcceptanceResult, error) {
	if existing, ok, err := existingAcceptanceExecution(ctx, pool, seed); err != nil {
		return formPolicyAcceptanceResult{}, err
	} else if ok {
		existing.AlreadySeeded = true
		return existing, nil
	}
	if scoring.FormID == "" || scoring.FormVersion < 1 || scoring.SubjectID == "" {
		return formPolicyAcceptanceResult{}, fmt.Errorf("scoring acceptance population is incomplete")
	}
	if seed.ActorID == "" || seed.ReviewerPrincipalID == "" || seed.SignatoryPrincipalID == "" {
		return formPolicyAcceptanceResult{}, fmt.Errorf("response-policy acceptance requires maker, reviewer and authorizer principals")
	}

	keyring, err := evidence.NewRecipientKeyring(cfg.RecipientSecurity.ActiveKeyID, cfg.RecipientSecurity.Keyring)
	if err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("configure response-policy acceptance keyring: %w", err)
	}
	distributionStore := evidence.NewPostgresDistributionStore(evidenceRepo, keyring)
	distributions := evidence.NewDistributionService(distributionStore)
	access, err := evidence.NewDistributionAccessService(distributionStore, keyring, nil, cfg.RecipientSecurity.AccessHMACKey, cfg.CaptureSessionTTL)
	if err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("configure response-policy acceptance access: %w", err)
	}
	form, err := monitoringRepo.ReusableFormRevision(ctx, seed.TenantID, seed.LegalEntityID, scoring.FormID, scoring.FormVersion)
	if err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("load scoring acceptance form: %w", err)
	}

	policyRepo := formpolicy.NewPostgresRepository(pool)
	automation := autonomy.NewService(autonomy.NewPostgresRepository(pool))
	authorityService := authority.NewEffectivePostgresService(pool)
	policyService := formpolicy.NewService(policyRepo, seedFormReader{repo: monitoringRepo}, distributions)
	policyService.ConfigureActivationAuthority(formpolicy.ActivationAuthorityResolver{
		Automation: automation,
		Authority:  authorityService,
		Subjects:   evidence.CanonicalSubjectTypeRegistry{},
	})

	maker := formpolicy.Actor{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, PrincipalID: seed.ActorID}
	checker := formpolicy.Actor{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, PrincipalID: seed.ReviewerPrincipalID}
	authorizer := formpolicy.Actor{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, PrincipalID: seed.SignatoryPrincipalID}
	base := policyAcceptanceInput(scoring)

	shadowAutomationID, err := ensureAcceptanceAutomationPolicy(ctx, pool, seed, base, formpolicy.RolloutShadow, 1)
	if err != nil {
		return formPolicyAcceptanceResult{}, err
	}
	shadowInput := base
	shadowInput.Rollout = formpolicy.RolloutShadow
	shadowInput.AutomationPolicyID = shadowAutomationID
	shadowInput.AutomationPolicyVersion = 1
	shadow, err := ensureActivatedAcceptancePolicy(ctx, policyService, policyRepo, maker, checker, authorizer, shadowInput, formpolicy.RolloutShadow)
	if err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("activate shadow response policy: %w", err)
	}
	if shadow.Status == formpolicy.PolicyActive {
		if _, err := policyService.Suspend(ctx, authorizer, shadow.ID, shadow.RecordVersion); err != nil {
			return formPolicyAcceptanceResult{}, fmt.Errorf("suspend shadow response policy: %w", err)
		}
	}

	enforceAutomationID, err := ensureAcceptanceAutomationPolicy(ctx, pool, seed, base, formpolicy.RolloutEnforce, 1)
	if err != nil {
		return formPolicyAcceptanceResult{}, err
	}
	enforceInput := base
	enforceInput.Rollout = formpolicy.RolloutEnforce
	enforceInput.AutomationPolicyID = enforceAutomationID
	enforceInput.AutomationPolicyVersion = 1
	enforce, err := ensureActivatedAcceptancePolicy(ctx, policyService, policyRepo, maker, checker, authorizer, enforceInput, formpolicy.RolloutEnforce)
	if err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("activate enforce response policy: %w", err)
	}

	now := time.Now().UTC()
	good, err := submitScoringAcceptanceResponse(ctx, distributions, access, seed, form, scoring.SubjectID, scoringResponseFixture{label: "post-policy-good", answers: scoringResponseFixtures[0].answers}, now)
	if err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("submit post-policy good response: %w", err)
	}
	poor, err := submitScoringAcceptanceResponse(ctx, distributions, access, seed, form, scoring.SubjectID, scoringResponseFixture{label: "post-policy-poor", answers: scoringResponseFixtures[2].answers}, now.Add(time.Second))
	if err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("submit post-policy poor response: %w", err)
	}
	sameEpisodePoor, err := submitScoringAcceptanceResponse(ctx, distributions, access, seed, form, scoring.SubjectID, scoringResponseFixture{label: "post-policy-poor-same-episode", answers: scoringResponseFixtures[2].answers}, now.Add(2*time.Second))
	if err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("submit same-episode poor response: %w", err)
	}

	executor := formpolicy.NewExecutor(
		policyRepo,
		distributions,
		formpolicy.ExecutionAuthorityResolver{Automation: automation, Authority: authorityService, Subjects: evidenceRepo},
	)
	publisher := formpolicy.ScoredResponsePublisher{Handler: executor}
	goodEvent, err := scoredOutboxEvent(ctx, pool, seed.TenantID, good.ID)
	if err != nil {
		return formPolicyAcceptanceResult{}, err
	}
	poorEvent, err := scoredOutboxEvent(ctx, pool, seed.TenantID, poor.ID)
	if err != nil {
		return formPolicyAcceptanceResult{}, err
	}
	sameEpisodeEvent, err := scoredOutboxEvent(ctx, pool, seed.TenantID, sameEpisodePoor.ID)
	if err != nil {
		return formPolicyAcceptanceResult{}, err
	}
	if err := publisher.Publish(ctx, goodEvent); err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("execute good scored response: %w", err)
	}
	if err := publisher.Publish(ctx, poorEvent); err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("execute poor scored response: %w", err)
	}

	goodExecution, err := executionForResponse(ctx, pool, enforce.ID, good.ID)
	if err != nil {
		return formPolicyAcceptanceResult{}, err
	}
	poorExecution, err := executionForResponse(ctx, pool, enforce.ID, poor.ID)
	if err != nil {
		return formPolicyAcceptanceResult{}, err
	}
	if goodExecution.State != string(formpolicy.ExecutionNotMatched) || goodExecution.MatterID != "" || goodExecution.CreatedMatter {
		return formPolicyAcceptanceResult{}, fmt.Errorf("good response created or matched a Matter: %#v", goodExecution)
	}
	if poorExecution.State != string(formpolicy.ExecutionApplied) || poorExecution.MatterID == "" || !poorExecution.CreatedMatter {
		return formPolicyAcceptanceResult{}, fmt.Errorf("poor response did not create a governed Matter: %#v", poorExecution)
	}
	if err := publisher.Publish(ctx, poorEvent); err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("replay poor scored response: %w", err)
	}
	replayed, err := executionForResponse(ctx, pool, enforce.ID, poor.ID)
	if err != nil {
		return formPolicyAcceptanceResult{}, err
	}
	if err := publisher.Publish(ctx, sameEpisodeEvent); err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("execute same-episode poor scored response: %w", err)
	}
	sameEpisodeExecution, err := executionForResponse(ctx, pool, enforce.ID, sameEpisodePoor.ID)
	if err != nil {
		return formPolicyAcceptanceResult{}, err
	}
	if sameEpisodeExecution.State != string(formpolicy.ExecutionReused) || sameEpisodeExecution.MatterID != poorExecution.MatterID || sameEpisodeExecution.CreatedMatter {
		return formPolicyAcceptanceResult{}, fmt.Errorf("same-episode poor response did not reuse the governed Matter: %#v", sameEpisodeExecution)
	}
	exactlyOnce, err := verifyExactlyOnceMatter(ctx, pool, seed.TenantID, seed.LegalEntityID, enforce.ID, poor.ID, poorExecution)
	if err != nil {
		return formPolicyAcceptanceResult{}, err
	}
	sameEpisodeReused, err := verifySameEpisodeMatterReuse(ctx, pool, seed.TenantID, seed.LegalEntityID, enforce.ID, poor.ID, sameEpisodePoor.ID, poorExecution.MatterID)
	if err != nil {
		return formPolicyAcceptanceResult{}, err
	}
	return formPolicyAcceptanceResult{
		PolicyID:                  enforce.ID,
		PolicyVersion:             enforce.Version,
		GoodResponseID:            good.ID,
		PoorResponseID:            poor.ID,
		SameEpisodeResponseID:     sameEpisodePoor.ID,
		GoodExecutionState:        goodExecution.State,
		PoorExecutionState:        poorExecution.State,
		SameEpisodeExecutionState: sameEpisodeExecution.State,
		MatterID:                  poorExecution.MatterID,
		CreatedMatter:             poorExecution.CreatedMatter,
		ReplayExecutionID:         replayed.ID,
		ExactlyOnceMatter:         exactlyOnce && replayed.ID == poorExecution.ID && replayed.MatterID == poorExecution.MatterID,
		SameEpisodeReusedMatter:   sameEpisodeReused,
	}, nil
}

func policyAcceptanceInput(scoring scoringAcceptanceResult) formpolicy.CreateInput {
	return formpolicy.CreateInput{
		Code:    responsePolicyAcceptanceCode,
		Name:    "Create a Matter for poor control responses",
		Purpose: "Reference acceptance policy proving that a poor scored response creates one governed Matter while a good response does not.",
		Eligibility: formpolicy.Eligibility{
			FormTemplateID:      scoring.FormID,
			FormTemplateVersion: scoring.FormVersion,
			SubjectTypes:        []string{"PROGRAM"},
			CurrentOnly:         true,
			MinimumCoverage:     1,
			Bands:               []formcontract.ConcernBand{formcontract.ConcernCritical, formcontract.ConcernHigh},
		},
		Action: formpolicy.MatterAction{
			Type:              "CONTROL_GAP",
			Priority:          4,
			TitleTemplate:     "Review {{form_title}}",
			SummaryTemplate:   "A completed control response crossed the approved concern threshold.",
			RequestedHandling: "Review the adverse response, record treatment and independently verify the outcome.",
		},
		BlastRadius: formpolicy.BlastRadius{PerRun: 10, PerDay: 25},
		Outcome: formpolicy.OutcomeContract{
			ExpectedOutcome:   "The control concern is remediated or an approved treatment is recorded.",
			CheckAfterMinutes: 60,
			FailureResponse:   "ESCALATE",
		},
	}
}
