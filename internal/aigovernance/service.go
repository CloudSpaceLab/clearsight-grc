package aigovernance

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

type Service struct {
	repo    Repository
	auto    *autonomy.Service
	sources *sourceaccess.CatalogService
	matters *continuity.Service
	now     func() time.Time
}

func NewService(repo Repository, auto *autonomy.Service, sources *sourceaccess.CatalogService, matters *continuity.Service) *Service {
	return &Service{repo: repo, auto: auto, sources: sources, matters: matters, now: time.Now}
}

func (s *Service) CreatePolicy(ctx context.Context, input CreatePolicyInput) (Policy, error) {
	if s == nil || s.repo == nil || strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.Code) == "" || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.MakerID) == "" {
		return Policy{}, ErrInvalid
	}
	if input.RolloutMode == "" {
		input.RolloutMode = aigateway.RolloutShadow
	}
	if input.RolloutMode != aigateway.RolloutShadow && input.RolloutMode != aigateway.RolloutEnforce {
		return Policy{}, ErrInvalid
	}
	if err := validatePolicyDefinition(input.Definition); err != nil {
		return Policy{}, errors.Join(ErrInvalid, err)
	}
	idValue, err := id.NewUUIDv7()
	if err != nil {
		return Policy{}, err
	}
	tenantID := strings.TrimSpace(input.TenantID)
	code := strings.TrimSpace(input.Code)
	version, err := s.repo.NextPolicyVersion(ctx, tenantID, code)
	if err != nil {
		return Policy{}, err
	}
	value := Policy{ID: idValue, TenantID: tenantID, Code: code, Name: strings.TrimSpace(input.Name), ActionClass: strings.TrimSpace(input.ActionClass), Eligibility: normalizedJSON(input.Eligibility, `{}`), BlastRadiusLimit: normalizedJSON(input.BlastRadiusLimit, `{}`), VerificationContract: normalizedJSON(input.VerificationContract, `{}`), Definition: input.Definition, Status: "DRAFT", RolloutMode: input.RolloutMode, MakerID: strings.TrimSpace(input.MakerID), EffectiveFrom: input.EffectiveFrom, EffectiveUntil: input.EffectiveUntil, Version: version, RecordVersion: 1}
	value.Checksum = checksumPolicy(value)
	return s.repo.CreatePolicy(ctx, value)
}

func (s *Service) GetPolicy(ctx context.Context, tenantID, id string) (Policy, error) {
	return s.repo.Policy(ctx, tenantID, id)
}
func (s *Service) ListPolicies(ctx context.Context, tenantID string, limit int) ([]Policy, error) {
	return s.repo.ListPolicies(ctx, tenantID, boundedLimit(limit))
}

func (s *Service) TransitionPolicy(ctx context.Context, action string, input TransitionInput) (Policy, error) {
	value, err := s.repo.Policy(ctx, input.TenantID, input.ID)
	if err != nil {
		return Policy{}, err
	}
	if input.ExpectedVersion > 0 && value.RecordVersion != input.ExpectedVersion {
		return Policy{}, ErrConflict
	}
	actor := strings.TrimSpace(input.ActorID)
	if actor == "" {
		return Policy{}, ErrInvalid
	}
	now := s.now().UTC()
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "submit":
		if value.Status != "DRAFT" {
			return Policy{}, ErrInvalidTransition
		}
		value.Status = "PENDING_APPROVAL"
		value.SubmittedAt = &now
	case "approve":
		if value.Status != "PENDING_APPROVAL" {
			return Policy{}, ErrInvalidTransition
		}
		if actor == value.MakerID {
			return Policy{}, ErrMakerChecker
		}
		value.Status = "APPROVED"
		value.CheckerID = actor
		value.ApprovedAt = &now
	case "activate":
		if value.Status != "APPROVED" && value.Status != "SUSPENDED" {
			return Policy{}, ErrInvalidTransition
		}
		if actor == value.MakerID {
			return Policy{}, ErrMakerChecker
		}
		if value.RolloutMode == aigateway.RolloutEnforce {
			hasShadow, historyErr := s.repo.HasShadowHistory(ctx, value.TenantID, value.Code, value.Version)
			if historyErr != nil {
				return Policy{}, historyErr
			}
			if !hasShadow {
				return Policy{}, errors.Join(ErrInvalidTransition, fmt.Errorf("enforcement requires a prior shadow revision"))
			}
		}
		value.Status = "ACTIVE"
		value.CheckerID = actor
		value.ActivatedAt = &now
	case "suspend":
		if value.Status != "ACTIVE" {
			return Policy{}, ErrInvalidTransition
		}
		value.Status = "SUSPENDED"
		value.SuspendedAt = &now
	case "retire":
		if value.Status == "RETIRED" {
			return Policy{}, ErrInvalidTransition
		}
		value.Status = "RETIRED"
		value.RetiredAt = &now
	default:
		return Policy{}, ErrInvalidTransition
	}
	value.RecordVersion++
	value.Checksum = checksumPolicy(value)
	return s.repo.UpdatePolicy(ctx, value, value.RecordVersion-1)
}

func (s *Service) CreateWorkload(ctx context.Context, input CreateWorkloadInput) (Workload, error) {
	if s == nil || s.repo == nil || strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.WorkloadID) == "" || strings.TrimSpace(input.Code) == "" || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Purpose) == "" || strings.TrimSpace(input.OwnerPrincipalID) == "" || strings.TrimSpace(input.PolicyID) == "" || input.PolicyVersion < 1 || strings.TrimSpace(input.MakerID) == "" {
		return Workload{}, ErrInvalid
	}
	if len(input.AllowedModels) == 0 || len(input.AllowedModels) > 256 || input.RequestsPerMinute < 1 || input.TokensPerMinute < 1 || input.CostMicroUSDPerMinute < 1 || input.MaxConcurrent < 1 {
		return Workload{}, ErrInvalid
	}
	policy, err := s.repo.Policy(ctx, input.TenantID, input.PolicyID)
	if err != nil || policy.Version != input.PolicyVersion {
		return Workload{}, ErrInvalid
	}
	keyDigest, err := parseSHA256(input.KeySHA256)
	if err != nil {
		return Workload{}, ErrInvalid
	}
	idValue, err := id.NewUUIDv7()
	if err != nil {
		return Workload{}, err
	}
	tenantID := strings.TrimSpace(input.TenantID)
	workloadID := strings.TrimSpace(input.WorkloadID)
	version, err := s.repo.NextWorkloadVersion(ctx, tenantID, workloadID)
	if err != nil {
		return Workload{}, err
	}
	now := s.now().UTC()
	value := Workload{ID: idValue, WorkloadID: workloadID, TenantID: tenantID, Code: strings.TrimSpace(input.Code), Name: strings.TrimSpace(input.Name), Purpose: strings.TrimSpace(input.Purpose), Environment: strings.ToUpper(strings.TrimSpace(input.Environment)), OwnerPrincipalID: strings.TrimSpace(input.OwnerPrincipalID), ServicePrincipalID: strings.TrimSpace(input.ServicePrincipalID), AllowedModels: uniqueSorted(input.AllowedModels), RequestsPerMinute: input.RequestsPerMinute, TokensPerMinute: input.TokensPerMinute, CostMicroUSDPerMinute: input.CostMicroUSDPerMinute, MaxConcurrent: input.MaxConcurrent, VerifiedMetadata: cloneMap(input.VerifiedMetadata), ApprovedResources: uniqueSorted(input.ApprovedResources), PolicyID: policy.ID, PolicyVersion: policy.Version, State: "DRAFT", MakerID: strings.TrimSpace(input.MakerID), CreatedAt: now, UpdatedAt: now, Version: version, RecordVersion: 1, KeySHA256: hex.EncodeToString(keyDigest[:])}
	value.Checksum = checksumWorkload(value)
	return s.repo.CreateWorkload(ctx, value)
}

func (s *Service) GetWorkload(ctx context.Context, tenantID, id string) (Workload, error) {
	return s.repo.Workload(ctx, tenantID, id)
}
func (s *Service) ListWorkloads(ctx context.Context, tenantID string, limit int) ([]Workload, error) {
	return s.repo.ListWorkloads(ctx, tenantID, boundedLimit(limit))
}

func (s *Service) TransitionWorkload(ctx context.Context, action string, input TransitionInput) (Workload, error) {
	value, err := s.repo.Workload(ctx, input.TenantID, input.ID)
	if err != nil {
		return Workload{}, err
	}
	if input.ExpectedVersion > 0 && value.RecordVersion != input.ExpectedVersion {
		return Workload{}, ErrConflict
	}
	actor := strings.TrimSpace(input.ActorID)
	if actor == "" {
		return Workload{}, ErrInvalid
	}
	now := s.now().UTC()
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "submit":
		if value.State != "DRAFT" {
			return Workload{}, ErrInvalidTransition
		}
		value.State = "PENDING_APPROVAL"
		value.SubmittedAt = &now
	case "approve":
		if value.State != "PENDING_APPROVAL" {
			return Workload{}, ErrInvalidTransition
		}
		if actor == value.MakerID {
			return Workload{}, ErrMakerChecker
		}
		value.State = "APPROVED"
		value.CheckerID = actor
		value.ApprovedAt = &now
	case "activate":
		if value.State != "APPROVED" && value.State != "SUSPENDED" {
			return Workload{}, ErrInvalidTransition
		}
		if actor == value.MakerID {
			return Workload{}, ErrMakerChecker
		}
		policy, policyErr := s.repo.Policy(ctx, value.TenantID, value.PolicyID)
		if policyErr != nil || policy.Version != value.PolicyVersion || policy.Status != "ACTIVE" {
			return Workload{}, errors.Join(ErrInvalidTransition, fmt.Errorf("workload policy revision is not active"))
		}
		value.State = "ACTIVE"
		value.CheckerID = actor
		value.ActivatedAt = &now
	case "suspend":
		if value.State != "ACTIVE" {
			return Workload{}, ErrInvalidTransition
		}
		value.State = "SUSPENDED"
		value.SuspendedAt = &now
	case "retire":
		if value.State == "RETIRED" {
			return Workload{}, ErrInvalidTransition
		}
		value.State = "RETIRED"
		value.RetiredAt = &now
	default:
		return Workload{}, ErrInvalidTransition
	}
	value.RecordVersion++
	value.UpdatedAt = now
	value.Checksum = checksumWorkload(value)
	return s.repo.UpdateWorkload(ctx, value, value.RecordVersion-1)
}

func (s *Service) IngestReceipt(ctx context.Context, receipt DecisionReceipt) (bool, error) {
	if s == nil || s.repo == nil || strings.TrimSpace(receipt.ReceiptID) == "" || strings.TrimSpace(receipt.TenantID) == "" || strings.TrimSpace(receipt.RequestID) == "" || strings.TrimSpace(receipt.WorkloadID) == "" || strings.TrimSpace(receipt.PolicyID) == "" || receipt.PolicyVersion < 1 || receipt.ObservedAt.IsZero() {
		return false, ErrInvalid
	}
	if !validReceiptAction(receipt.Decision, false) || !validReceiptAction(receipt.ProposedAction, true) || strings.TrimSpace(receipt.Outcome) == "" || len(receipt.ReceiptID) > 256 || len(receipt.RequestID) > 256 || len(receipt.WorkloadID) > 256 || len(receipt.PolicyCode) > 256 || len(receipt.ModelAlias) > 256 || len(receipt.RouteID) > 256 || len(receipt.Outcome) > 128 || len(receipt.ErrorCode) > 128 || !boundedStrings(receipt.ReasonCodes, 32, 256) || !boundedStrings(receipt.Obligations, 32, 256) {
		return false, ErrInvalid
	}
	policy, err := s.repo.Policy(ctx, receipt.TenantID, receipt.PolicyID)
	if err != nil || policy.Version != receipt.PolicyVersion || policy.Code != receipt.PolicyCode {
		return false, ErrInvalid
	}
	if receipt.ExpiresAt.IsZero() {
		receipt.ExpiresAt = receipt.ObservedAt.Add(90 * 24 * time.Hour)
	}
	if receipt.ExpiresAt.Before(receipt.ObservedAt) || receipt.ExpiresAt.After(receipt.ObservedAt.Add(366*24*time.Hour)) {
		return false, ErrInvalid
	}
	receipt.ReasonCodes = uniqueSorted(receipt.ReasonCodes)
	receipt.Obligations = uniqueSorted(receipt.Obligations)
	receipt.Fingerprint = checksumReceipt(receipt)
	inserted, err := s.repo.IngestReceipt(ctx, receipt)
	if err != nil || !inserted {
		return inserted, err
	}
	if s.auto != nil && receipt.Decision == aigateway.DecisionRequireApproval {
		// Reconcile a material approval condition as one stable episode rather than
		// manufacturing one Signal/Drift per model request. autonomy.Ingest owns
		// dedupe, so repeated receipts for the same governed condition converge.
		episodeKey := approvalEpisodeKey(receipt)
		_, _, _ = s.auto.Ingest(ctx, autonomy.Signal{
			TenantID: receipt.TenantID, Type: autonomy.SignalContextChanged,
			SubjectType: "AI_WORKLOAD", SubjectID: receipt.WorkloadID, Source: "ai-gateway",
			DedupeKey: episodeKey,
			Payload:   map[string]string{"decision": "REQUIRE_APPROVAL", "policy_id": receipt.PolicyID, "policy_version": fmt.Sprint(receipt.PolicyVersion)},
		})
	}
	return true, nil
}

func (s *Service) CreateExecutionGrant(ctx context.Context, input CreateGrantInput) (ExecutionGrant, error) {
	if s.matters == nil || strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.WorkloadID) == "" || strings.TrimSpace(input.MatterID) == "" || strings.TrimSpace(input.DecisionID) == "" || strings.TrimSpace(input.ActorID) == "" || len(input.Action) == 0 || !json.Valid(input.Action) {
		return ExecutionGrant{}, ErrGrantInvalid
	}
	matter, err := s.matters.GetMatter(ctx, input.TenantID, input.MatterID)
	if err != nil {
		return ExecutionGrant{}, errors.Join(ErrGrantInvalid, err)
	}
	actionHash := sha256.Sum256(compactJSON(input.Action))
	actionHex := hex.EncodeToString(actionHash[:])
	var approved *continuity.Decision
	for i := range matter.Decisions {
		if matter.Decisions[i].ID == input.DecisionID {
			approved = &matter.Decisions[i]
			break
		}
	}
	if approved == nil || approved.Type != "AI_EXECUTION_GRANT" || (approved.Status != continuity.DecisionApproved && approved.Status != continuity.DecisionConditionallyApproved) || approved.SelectedOption != actionHex || approved.AuthorityPrincipalID != input.ActorID {
		return ExecutionGrant{}, ErrGrantInvalid
	}
	if approved.ExpiresAt != nil && !approved.ExpiresAt.After(s.now().UTC()) {
		return ExecutionGrant{}, ErrGrantInvalid
	}
	ttl := time.Duration(input.TTLMinutes) * time.Minute
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	if ttl > time.Hour {
		return ExecutionGrant{}, ErrGrantInvalid
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return ExecutionGrant{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	digest := sha256.Sum256([]byte(token))
	grantID, err := id.NewUUIDv7()
	if err != nil {
		return ExecutionGrant{}, err
	}
	now := s.now().UTC()
	grant := ExecutionGrant{ID: grantID, TenantID: input.TenantID, WorkloadID: input.WorkloadID, MatterID: input.MatterID, DecisionID: input.DecisionID, ActionHash: actionHex, ApprovedBy: input.ActorID, State: "ACTIVE", ExpiresAt: now.Add(ttl), CreatedAt: now, RecordVersion: 1, Token: token, TokenSHA256: hex.EncodeToString(digest[:])}
	created, err := s.repo.CreateGrant(ctx, grant, digest)
	if err != nil {
		return ExecutionGrant{}, err
	}
	created.Token = token
	return created, nil
}

func (s *Service) ConsumeExecutionGrant(ctx context.Context, tenantID, workloadID string, action json.RawMessage, token string) (ExecutionGrant, error) {
	if !json.Valid(action) || strings.TrimSpace(token) == "" {
		return ExecutionGrant{}, ErrGrantInvalid
	}
	h := sha256.Sum256(compactJSON(action))
	td := sha256.Sum256([]byte(token))
	return s.repo.ConsumeGrant(ctx, tenantID, workloadID, hex.EncodeToString(h[:]), td, s.now().UTC())
}

func validatePolicyDefinition(def aigateway.PolicyDefinition) error {
	if len(def.Bindings) > 64 || len(def.Rules) > 256 {
		return fmt.Errorf("policy exceeds limits")
	}
	validMode := func(mode aigateway.BindingResolutionMode) bool {
		switch mode {
		case aigateway.ResolutionMetadata, aigateway.ResolutionLiveLookup, aigateway.ResolutionAdapterCache, aigateway.ResolutionAsync, aigateway.ResolutionExternalControl:
			return true
		default:
			return false
		}
	}
	validAction := func(action aigateway.DecisionAction) bool {
		switch action {
		case "", aigateway.DecisionAllow, aigateway.DecisionDeny, aigateway.DecisionModify, aigateway.DecisionRoute, aigateway.DecisionRequireApproval:
			return true
		default:
			return false
		}
	}
	for _, b := range def.Bindings {
		if strings.TrimSpace(b.FactKey) == "" || b.BindingVersion < 0 || !validMode(b.Mode) || b.MaxAgeSeconds < 0 || b.MaxAgeSeconds > int64((30*24*time.Hour)/time.Second) {
			return fmt.Errorf("invalid binding requirement")
		}
		if b.Mode == aigateway.ResolutionLiveLookup && (strings.TrimSpace(b.BindingID) == "" || b.BindingVersion < 1 || strings.TrimSpace(b.MetadataKey) == "") {
			return fmt.Errorf("live lookup requires an exact binding revision and lookup metadata key")
		}
	}
	if !validAction(def.DefaultAction) {
		return fmt.Errorf("invalid default action")
	}
	for _, r := range def.Rules {
		if strings.TrimSpace(r.ID) == "" || len(r.ID) > 256 || r.Priority < 0 || r.Priority > 100000 || !validAction(r.Action) || len(r.Obligations) > 32 || len(r.Redactions) > 32 {
			return fmt.Errorf("invalid policy rule")
		}
		if r.Action == aigateway.DecisionRoute && strings.TrimSpace(r.RouteID) == "" {
			return fmt.Errorf("route action requires a route id")
		}
		for _, redaction := range r.Redactions {
			if len(redaction.Pattern) == 0 || len(redaction.Pattern) > 4096 {
				return fmt.Errorf("invalid request redaction")
			}
			if _, err := regexp.Compile(redaction.Pattern); err != nil {
				return fmt.Errorf("invalid request redaction: %w", err)
			}
		}
	}
	control := def.ResponseControl
	if control.AllowStreaming && (control.MaxBytes > 0 || len(control.DenyPatterns) > 0 || len(control.RedactPatterns) > 0) {
		return fmt.Errorf("streaming cannot enable whole-response controls")
	}
	if control.MaxBytes < 0 || control.MaxBytes > aigateway.MaxTextBytes || len(control.DenyPatterns) > 32 || len(control.RedactPatterns) > 32 {
		return fmt.Errorf("invalid response control")
	}
	for _, pattern := range append(append([]string(nil), control.DenyPatterns...), control.RedactPatterns...) {
		if len(pattern) == 0 || len(pattern) > 4096 {
			return fmt.Errorf("invalid response pattern")
		}
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("invalid response pattern: %w", err)
		}
	}
	return nil
}

func validReceiptAction(action aigateway.DecisionAction, allowEmpty bool) bool {
	if action == "" {
		return allowEmpty
	}
	switch action {
	case aigateway.DecisionAllow, aigateway.DecisionDeny, aigateway.DecisionModify, aigateway.DecisionRoute, aigateway.DecisionRequireApproval, aigateway.DecisionShadow:
		return true
	default:
		return false
	}
}

func boundedStrings(values []string, maxItems, maxLen int) bool {
	if len(values) > maxItems {
		return false
	}
	for _, value := range values {
		if len(strings.TrimSpace(value)) == 0 || len(value) > maxLen {
			return false
		}
	}
	return true
}

func approvalEpisodeKey(receipt DecisionReceipt) string {
	reasons := uniqueSorted(receipt.ReasonCodes)
	basis := receipt.TenantID + "|" + receipt.WorkloadID + "|" + receipt.PolicyID + "|" + fmt.Sprint(receipt.PolicyVersion) + "|" + strings.Join(reasons, ",")
	sum := sha256.Sum256([]byte(basis))
	return "ai-approval:" + hex.EncodeToString(sum[:12])
}

func checksumPolicy(value Policy) string {
	value.Checksum = ""
	value.RecordVersion = 0
	payload, _ := json.Marshal(value)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
func checksumWorkload(value Workload) string {
	value.Checksum = ""
	value.RecordVersion = 0
	value.KeySHA256 = ""
	payload, _ := json.Marshal(value)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
func checksumReceipt(value DecisionReceipt) string {
	value.Fingerprint = ""
	payload, _ := json.Marshal(value)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
func compactJSON(raw json.RawMessage) []byte {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return raw
	}
	b, _ := json.Marshal(v)
	return b
}
func normalizedJSON(raw json.RawMessage, fallback string) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(fallback)
	}
	if !json.Valid(raw) {
		return json.RawMessage(fallback)
	}
	return append(json.RawMessage(nil), raw...)
}
func parseSHA256(value string) ([32]byte, error) {
	var out [32]byte
	b, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil || len(b) != 32 {
		return out, ErrInvalid
	}
	copy(out[:], b)
	return out, nil
}
func boundedLimit(v int) int {
	if v <= 0 {
		return 50
	}
	if v > 200 {
		return 200
	}
	return v
}
func uniqueSorted(v []string) []string {
	m := map[string]struct{}{}
	for _, x := range v {
		x = strings.TrimSpace(x)
		if x != "" {
			m[x] = struct{}{}
		}
	}
	o := make([]string, 0, len(m))
	for x := range m {
		o = append(o, x)
	}
	sort.Strings(o)
	return o
}
func cloneMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	o := make(map[string]string, len(in))
	for k, v := range in {
		o[k] = v
	}
	return o
}
