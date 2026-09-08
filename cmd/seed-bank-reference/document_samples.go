//go:build postgres

package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
	"github.com/jackc/pgx/v5/pgxpool"
)

const documentSampleSource = "fictional_document_samples_v1"
const documentSampleReference = "northstar-infrastructure-documents-v1"
const documentSampleFormCode = "SAMPLE-NORTHSTAR-DOCUMENTS-V1"
const documentSampleAudience = "documents@northstar.example.invalid"

var errDocumentSampleChanged = errors.New("fictional document sample records changed or paused; inspect the existing records before continuing")

type documentSampleReceipt struct {
	RelationshipID      string   `json:"relationship_id"`
	FormTemplateID      string   `json:"form_template_id"`
	AssessmentID        string   `json:"assessment_id"`
	RequestID           string   `json:"request_id"`
	ResponseRevisionIDs []string `json:"response_revision_ids"`
	ArtifactCount       int      `json:"artifact_count"`
	AssessmentStatus    string   `json:"assessment_status"`
	AlreadyInstalled    bool     `json:"already_installed"`
	Limitation          string   `json:"limitation"`
}

type documentSampleInstaller struct {
	pool          *pgxpool.Pool
	cfg           config.Config
	seed          bankverticals.SeedConfig
	guard         *commandauth.Guard
	objects       *evidence.LocalObjectStore
	evidence      *evidence.Service
	forms         *monitoring.Service
	vendors       *thirdparty.Service
	assessments   *thirdparty.AssessmentService
	requests      *thirdparty.AssessmentRequestService
	store         *evidence.PostgresDistributionStore
	distributions *evidence.DistributionService
	access        *evidence.DistributionAccessService
	dispatch      *evidence.WorkflowDistributionDispatcher
}

func installDocumentSamples(ctx context.Context, cfg config.Config, pool *pgxpool.Pool, seed bankverticals.SeedConfig) (documentSampleReceipt, error) {
	if strings.EqualFold(strings.TrimSpace(cfg.Environment), "production") || !cfg.DemoMode {
		return documentSampleReceipt{}, errors.New("document samples require non-production demo mode")
	}
	if strings.TrimSpace(cfg.ArtifactRoot) == "" || cfg.RecipientSecurity.ActiveKeyID == "" || len(cfg.RecipientSecurity.Keyring) == 0 || cfg.RecipientSecurity.AccessHMACKey == ([32]byte{}) || cfg.CapturePublicBaseURL == "" {
		return documentSampleReceipt{}, errors.New("document samples require a durable artifact root, recipient keyring, access HMAC and capture address")
	}
	if pool == nil || seed.ActorID == "" || seed.ActorID != seed.OwnerPrincipalID || seed.ReviewerPrincipalID == "" || seed.ActorID == seed.ReviewerPrincipalID {
		return documentSampleReceipt{}, errors.New("document samples require the current owner as maker and a distinct current checker")
	}
	// Resolve membership before normalizing the exact scope to UUIDs used by distribution reads.
	resolver := access.NewPostgresResolver(pool)
	r, err := resolver.ResolvePrincipal(ctx, seed.TenantID, seed.ActorID, seed.LegalEntityID)
	if err != nil {
		return documentSampleReceipt{}, err
	}
	if _, err = resolver.ResolvePrincipal(ctx, r.TenantID, seed.ReviewerPrincipalID, r.LegalEntityID); err != nil {
		return documentSampleReceipt{}, err
	}
	err = pool.QueryRow(ctx, `SELECT t.id::text,le.id::text FROM tenants t JOIN legal_entities le ON le.tenant_id=t.id WHERE t.slug=$1 AND le.code=$2`, r.TenantID, r.LegalEntityID).Scan(&seed.TenantID, &seed.LegalEntityID)
	if err != nil {
		return documentSampleReceipt{}, err
	}
	// A dedicated connection owns this session lock until all installer reads/writes finish.
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return documentSampleReceipt{}, err
	}
	defer conn.Release()
	lockKey := documentSampleSource + ":" + seed.TenantID + ":" + seed.LegalEntityID
	if _, err = conn.Exec(ctx, `SELECT pg_advisory_lock(hashtextextended($1,0))`, lockKey); err != nil {
		return documentSampleReceipt{}, err
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, unlockErr := conn.Exec(unlockCtx, `SELECT pg_advisory_unlock(hashtextextended($1,0))`, lockKey); unlockErr != nil {
			// Never return a connection retaining a session lock to the shared pool.
			_ = conn.Conn().Close(context.Background())
		}
	}()
	i := &documentSampleInstaller{pool: pool, cfg: cfg, seed: seed}
	if err = i.configure(); err != nil {
		return documentSampleReceipt{}, err
	}
	return i.run(ctx)
}

func (i *documentSampleInstaller) configure() error {
	var err error
	i.guard, err = commandauth.New(authority.NewEffectivePostgresService(i.pool), commandauth.ModeEnforce, nil)
	if err != nil {
		return err
	}
	i.objects, err = evidence.NewLocalObjectStore(i.cfg.ArtifactRoot)
	if err != nil {
		return err
	}
	repo := evidence.NewPostgresRepository(i.pool)
	i.evidence = evidence.NewService(repo, i.objects)
	i.evidence.Configure(i.cfg.CaptureSessionTTL, i.cfg.MaxArtifactBytes)
	i.forms = monitoring.NewService(monitoring.NewPostgresRepository(i.pool), i.evidence)
	i.forms.ConfigureCommandGuard(i.guard)
	thirdRepo := thirdparty.NewPostgresRepository(i.pool)
	i.vendors = thirdparty.NewService(thirdRepo)
	i.assessments = thirdparty.NewAssessmentService(thirdRepo, i.guard)
	keys, err := evidence.NewRecipientKeyring(i.cfg.RecipientSecurity.ActiveKeyID, i.cfg.RecipientSecurity.Keyring)
	if err != nil {
		return err
	}
	i.store = evidence.NewPostgresDistributionStore(repo, keys)
	i.distributions = evidence.NewDistributionService(i.store)
	i.access, err = evidence.NewDistributionAccessService(i.store, keys, nil, i.cfg.RecipientSecurity.AccessHMACKey, i.cfg.CaptureSessionTTL)
	if err != nil {
		return err
	}
	i.dispatch = evidence.NewWorkflowDistributionDispatcher(i.distributions, i.access)
	// Canonical assessment communications are workflow-owned. Nil delivery creates no email.
	i.requests, err = thirdparty.NewAssessmentRequestService(i.assessments, thirdRepo, i.evidence, monitoring.NewPostgresRepository(i.pool), nil, i.cfg.CapturePublicBaseURL, i.cfg.Environment)
	if err != nil {
		return err
	}
	i.requests.ConfigureDistributionDispatcher(i.dispatch)
	return nil
}

func (i *documentSampleInstaller) actorContext(ctx context.Context, principal string) (context.Context, error) {
	r, err := access.NewPostgresResolver(i.pool).ResolvePrincipal(ctx, i.seed.TenantID, principal, i.seed.LegalEntityID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	return identity.WithActor(ctx, identity.Actor{TenantID: i.seed.TenantID, LegalEntityID: i.seed.LegalEntityID, PrincipalID: r.PrincipalID, Kind: r.Kind, IssuedAt: now, ExpiresAt: now.Add(2 * time.Minute), AuthenticationMethod: "non-production-installer"}), nil
}

func (i *documentSampleInstaller) actor() thirdparty.Actor {
	return thirdparty.Actor{TenantID: i.seed.TenantID, LegalEntityID: i.seed.LegalEntityID, PrincipalID: i.seed.ActorID}
}

func (i *documentSampleInstaller) run(ctx context.Context) (documentSampleReceipt, error) {
	var receipt documentSampleReceipt
	ownerCtx, err := i.actorContext(ctx, i.seed.ActorID)
	if err != nil {
		return receipt, err
	}
	vendor, err := i.ensureVendor(ownerCtx)
	if err != nil {
		return receipt, fmt.Errorf("sample vendor: %w", err)
	}
	form, err := i.ensureForm(ctx)
	if err != nil {
		return receipt, fmt.Errorf("sample form: %w", err)
	}
	ownerCtx, err = i.actorContext(ctx, i.seed.ActorID)
	if err != nil {
		return receipt, err
	}
	a, err := i.assessments.StartAssessment(ownerCtx, i.actor(), vendor.Relationship.ID, thirdparty.StartAssessmentInput{RelationshipVersion: vendor.Relationship.Version, ReviewKind: thirdparty.AssessmentReviewOnboarding, FormTemplateID: form.ID, FormTemplateVersion: form.Version, ReviewDueAt: time.Now().UTC().Add(14 * 24 * time.Hour)})
	if err != nil {
		return receipt, err
	}
	for a.Status == thirdparty.AssessmentSetupPending {
		select {
		case <-ctx.Done():
			return receipt, fmt.Errorf("sample assessment setup pending; retry after the worker is ready: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
		a, err = i.assessments.GetAssessment(ctx, i.actor(), a.ID)
		if err != nil {
			return receipt, err
		}
	}
	if a.FormTemplateID != form.ID || a.FormTemplateVersion != form.Version || a.RelationshipID != vendor.Relationship.ID || a.StartedByPrincipalID != i.seed.ActorID || a.ReviewKind != thirdparty.AssessmentReviewOnboarding || a.SourceTrigger != "INITIAL" || a.ScopeVersion != 1 || a.ScopeKind != thirdparty.AssessmentScopeFull || len(a.SelectedFieldIDs) != 0 || (a.Status != thirdparty.AssessmentReadyToSend && a.Status != thirdparty.AssessmentCollecting) || a.ReviewMatterID == "" {
		return receipt, errDocumentSampleChanged
	}
	receipt = documentSampleReceipt{RelationshipID: vendor.Relationship.ID, FormTemplateID: form.ID, AssessmentID: a.ID, AssessmentStatus: string(a.Status), Limitation: "Sample data. No antivirus scan was performed. Submitted responses do not advance the assessment from collecting; bank review and approval remain outstanding."}
	origin := evidence.RequestOrigin{Type: thirdparty.AssessmentRequestOrigin, ID: a.ID, Version: 1}
	request, err := i.evidence.GetRequestByOrigin(ctx, i.seed.TenantID, origin)
	if errors.Is(err, evidence.ErrNotFound) {
		if a.Status != thirdparty.AssessmentReadyToSend {
			return receipt, errDocumentSampleChanged
		}
		ownerCtx, err = i.actorContext(ctx, i.seed.ActorID)
		if err != nil {
			return receipt, err
		}
		out, sendErr := i.requests.SendRequest(ownerCtx, i.actor(), a.ID, thirdparty.SendAssessmentRequestInput{ExpectedVersion: a.Version, Audience: documentSampleAudience, Deadline: a.ReviewDueAt.Add(-24 * time.Hour), InvitationTTLMinutes: 60})
		if sendErr != nil {
			return receipt, sendErr
		}
		request = out.Request
		a = out.Assessment
	} else if err != nil {
		return receipt, err
	}
	if err = i.checkRequest(request, form, a, origin); err != nil {
		return receipt, err
	}
	receipt.RequestID = request.ID
	receipt.AssessmentStatus = string(a.Status)
	result, err := i.completeResponses(ctx, request, a, receipt)
	if err != nil {
		return result, fmt.Errorf("sample responses: %w", err)
	}
	return result, nil
}
