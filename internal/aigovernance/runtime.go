package aigovernance

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

type RuntimeProvider struct {
	repo    Repository
	sources *sourceaccess.CatalogService
	ready   atomic.Bool

	baselineCache sync.Map
	baselineTTL   time.Duration
	now           func() time.Time
}

func NewRuntimeProvider(repo Repository, sources *sourceaccess.CatalogService) *RuntimeProvider {
	p := &RuntimeProvider{repo: repo, sources: sources, baselineTTL: 5 * time.Second, now: time.Now}
	p.ready.Store(repo != nil)
	return p
}

func (p *RuntimeProvider) Ready() bool { return p != nil && p.ready.Load() }

func (p *RuntimeProvider) Authenticate(ctx context.Context, header string) (*aigateway.Workload, error) {
	if p == nil || p.repo == nil {
		return nil, aigateway.ErrPolicyUnavailable
	}
	parts := strings.Fields(strings.TrimSpace(header))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || len(parts[1]) < 8 || len(parts[1]) > 4096 {
		return nil, aigateway.ErrUnauthorized
	}
	digest := sha256.Sum256([]byte(parts[1]))
	workload, policy, err := p.repo.WorkloadByCredential(ctx, digest)
	if err != nil {
		return nil, aigateway.ErrUnauthorized
	}
	allowed := make(map[string]struct{}, len(workload.AllowedModels))
	for _, model := range workload.AllowedModels {
		allowed[model] = struct{}{}
	}
	policySnapshot := aigateway.PolicySnapshot{ID: policy.ID, Code: policy.Code, Version: policy.Version, RolloutMode: policy.RolloutMode, Definition: policy.Definition}
	baseline, found, err := p.activeGatewayBaseline(ctx, workload.TenantID)
	if err != nil {
		return nil, aigateway.ErrPolicyUnavailable
	}
	if found && baseline.ID != policy.ID {
		policySnapshot.Baseline = &baseline
	}
	result := &aigateway.Workload{
		ID: workload.WorkloadID, TenantID: workload.TenantID, Purpose: workload.Purpose, Environment: workload.Environment,
		VerifiedMetadata: cloneMap(workload.VerifiedMetadata), AllowedModels: allowed,
		RequestsPerMinute: workload.RequestsPerMinute, TokensPerMinute: workload.TokensPerMinute,
		CostMicroUSDPerMinute: workload.CostMicroUSDPerMinute, MaxConcurrent: workload.MaxConcurrent,
		Policy: policySnapshot,
	}
	return result, nil
}

func (p *RuntimeProvider) ResolveFacts(ctx context.Context, workload aigateway.Workload, request aigateway.Request, requirements []aigateway.BindingRequirement) ([]aigateway.Fact, error) {
	facts := make([]aigateway.Fact, 0, len(requirements))
	for _, requirement := range requirements {
		fact := aigateway.Fact{Key: requirement.FactKey, State: aigateway.FactUnknown}
		switch requirement.Mode {
		case aigateway.ResolutionMetadata:
			if value, ok := workload.VerifiedMetadata[requirement.MetadataKey]; ok {
				fact.Value, fact.State, fact.Source, fact.ObservedAt = value, aigateway.FactKnown, "WORKLOAD_METADATA", time.Now().UTC()
			}
		case aigateway.ResolutionLiveLookup:
			if p.sources == nil {
				fact.State = aigateway.FactUnavailable
				break
			}
			lookupValue, ok := request.Metadata[requirement.MetadataKey]
			if !ok || strings.TrimSpace(lookupValue) == "" {
				fact.State = aigateway.FactUnknown
				break
			}
			result, err := p.sources.LookupBinding(ctx, workload.TenantID, requirement.BindingID, requirement.BindingVersion, sourceaccess.LookupRequest{Values: []sourceaccess.Scalar{sourceaccess.StringValue(lookupValue)}})
			if err != nil {
				if errors.Is(err, sourceaccess.ErrCatalogNotFound) || errors.Is(err, sourceaccess.ErrCatalogInvalid) || errors.Is(err, sourceaccess.ErrCapabilityUnavailable) {
					fact.State = aigateway.FactUnavailable
					break
				}
				return nil, err
			}
			fact.Source = "LIVE_LOOKUP:" + requirement.BindingID
			fact.ObservedAt = result.Receipt.ObservedAt
			if requirement.MaxAgeSeconds > 0 && !fact.ObservedAt.IsZero() && time.Since(fact.ObservedAt) > time.Duration(requirement.MaxAgeSeconds)*time.Second {
				fact.State = aigateway.FactStale
				break
			}
			if len(result.Records) == 0 {
				fact.State = aigateway.FactUnknown
				break
			}
			field := requirement.LookupField
			if field == "" {
				field = requirement.FactKey
			}
			scalar, ok := result.Records[0][field]
			if !ok || scalar.Kind == sourceaccess.ScalarNull {
				fact.State = aigateway.FactUnknown
				break
			}
			fact.Value, fact.State = scalar.Text, aigateway.FactKnown
		case aigateway.ResolutionAdapterCache:
			if value, ok := workload.VerifiedMetadata["cache:"+requirement.FactKey]; ok {
				fact.Value = value
				fact.State = aigateway.FactKnown
				fact.Source = "ADAPTER_CACHE"
				fact.ObservedAt = time.Now().UTC()
			}
		case aigateway.ResolutionExternalControl:
			if value, ok := workload.VerifiedMetadata["external:"+requirement.FactKey]; ok {
				fact.Value = value
				fact.State = aigateway.FactKnown
				fact.Source = "EXTERNAL_CONTROL"
				fact.ObservedAt = time.Now().UTC()
			}
		case aigateway.ResolutionAsync:
			fact.State = aigateway.FactUnknown
		default:
			fact.State = aigateway.FactUnavailable
		}
		facts = append(facts, fact)
	}
	return facts, nil
}

func constantDigestEqual(a, b [sha256.Size]byte) bool {
	return subtle.ConstantTimeCompare(a[:], b[:]) == 1
}
func parseVersion(value string) int64 { v, _ := strconv.ParseInt(value, 10, 64); return v }

func (p *RuntimeProvider) RecordReceipt(ctx context.Context, record aigateway.ReceiptRecord) error {
	if p == nil || p.repo == nil || strings.TrimSpace(record.RequestID) == "" || strings.TrimSpace(record.TenantID) == "" {
		return nil
	}
	// Keep all material/non-allow outcomes. Ordinary successful ALLOW traffic is
	// deterministically sampled at 1% so retries produce the same decision. A
	// material SHADOW proposal in the organization baseline must never be sampled
	// away even if the workload policy itself allows the request.
	baselineMaterial := record.Decision.BaselineAction == aigateway.DecisionShadow && record.Decision.BaselineProposedAction != ""
	if record.Decision.Action == aigateway.DecisionAllow && record.Outcome == "AUTHORIZED" && !baselineMaterial {
		digest := sha256.Sum256([]byte(record.TenantID + "|" + record.WorkloadID + "|" + record.RequestID))
		if int(digest[0]) >= 3 { // ~1.17%, deterministic and stateless.
			return nil
		}
	}
	obligations := make([]string, 0, len(record.Decision.Obligations))
	for _, obligation := range record.Decision.Obligations {
		if obligation.Code != "" {
			obligations = append(obligations, obligation.Code)
		}
	}
	receipt := DecisionReceipt{
		ReceiptID: record.RequestID + ":governance", TenantID: record.TenantID, RequestID: record.RequestID,
		WorkloadID: record.WorkloadID, PolicyID: record.Decision.PolicyID, PolicyCode: record.Decision.PolicyCode,
		PolicyVersion: record.Decision.PolicyVersion, Decision: record.Decision.Action, ProposedAction: record.Decision.ProposedAction,
		ReasonCodes: uniqueSorted(record.Decision.ReasonCodes), Obligations: uniqueSorted(obligations), ModelAlias: record.ModelAlias,
		RouteID: record.RouteID, Outcome: record.Outcome, ErrorCode: record.ErrorCode, ObservedAt: record.ObservedAt,
		ExpiresAt: record.ObservedAt.Add(90 * 24 * time.Hour),
		BaselinePolicyID: record.Decision.BaselinePolicyID, BaselinePolicyCode: record.Decision.BaselinePolicyCode,
		BaselinePolicyVersion: record.Decision.BaselinePolicyVersion, BaselineRolloutMode: record.Decision.BaselineRolloutMode,
		BaselineDecision: record.Decision.BaselineAction, BaselineProposedAction: record.Decision.BaselineProposedAction,
		BaselineReasonCodes: uniqueSorted(record.Decision.BaselineReasonCodes),
	}
	receipt.Fingerprint = checksumReceipt(receipt)
	_, err := p.repo.IngestReceipt(ctx, receipt)
	if errors.Is(err, ErrConflict) {
		return err
	}
	return err
}
