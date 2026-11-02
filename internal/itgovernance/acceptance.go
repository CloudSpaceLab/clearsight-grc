package itgovernance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

const IndicatorSourceAcceptanceSchema = "clearsight.indicator-source-acceptance.v1"

var ErrAcceptanceInvalid = errors.New("indicator source acceptance is invalid")

type IndicatorSourceAcceptanceInput struct {
	Binding       sourceaccess.BindingRevision
	View          sourceaccess.ViewRevision
	Page          sourceaccess.RecordPage
	BranchRef     string
	HeadOfficeRef string
	Check         *monitoring.MonitoringCheck
	Result        *monitoring.MonitoringResult
	GeneratedAt   time.Time
}

type PersistedIndicatorAcceptance struct {
	Proved                   bool      `json:"proved"`
	CheckIdentitySHA256      string    `json:"check_identity_sha256,omitempty"`
	CheckVersion             int64     `json:"check_version,omitempty"`
	ResultIdentitySHA256     string    `json:"result_identity_sha256,omitempty"`
	ResultEvaluatedAt        time.Time `json:"result_evaluated_at,omitempty"`
	SourceReceiptSHA256      string    `json:"source_receipt_sha256,omitempty"`
	NativeMeasurement        bool      `json:"native_measurement"`
	BindingRevisionMatched   bool      `json:"binding_revision_matched"`
	ViewRevisionMatched      bool      `json:"view_revision_matched"`
	SchemaFingerprintMatched bool      `json:"schema_fingerprint_matched"`
}

type IndicatorSourceAcceptanceReceipt struct {
	Schema                    string                       `json:"schema"`
	GeneratedAt               time.Time                    `json:"generated_at"`
	SourceIdentitySHA256      string                       `json:"source_identity_sha256"`
	BindingIdentitySHA256     string                       `json:"binding_identity_sha256"`
	BindingVersion            int64                        `json:"binding_version"`
	ViewIdentitySHA256        string                       `json:"view_identity_sha256"`
	ViewVersion               int64                        `json:"view_version"`
	SchemaFingerprint         string                       `json:"schema_fingerprint"`
	MappingSHA256             string                       `json:"mapping_sha256"`
	OperationReceiptSHA256    string                       `json:"operation_receipt_sha256"`
	ObservedAt                time.Time                    `json:"observed_at"`
	Completeness              sourceaccess.Completeness    `json:"completeness"`
	RecordsRead               int                          `json:"records_read"`
	RecordsMapped             int                          `json:"records_mapped"`
	RecordsUnmapped           int                          `json:"records_unmapped"`
	DistinctOrganizations     int                          `json:"distinct_organizations"`
	MeasurementRoleCounts     map[string]int               `json:"measurement_role_counts"`
	MoreAvailable             bool                         `json:"more_available"`
	BranchWitnessRequired     bool                         `json:"branch_witness_required"`
	BranchWitnessObserved     bool                         `json:"branch_witness_observed"`
	HeadOfficeWitnessRequired bool                         `json:"head_office_witness_required"`
	HeadOfficeWitnessObserved bool                         `json:"head_office_witness_observed"`
	PersistedIndicator        PersistedIndicatorAcceptance `json:"persisted_indicator"`
}

// BuildIndicatorSourceAcceptance validates a bounded live source page through the
// same governed mapping contract used by the runtime. The returned receipt is
// deliberately redacted: it contains no source values, organization labels,
// private identifiers, credentials, or record payloads.
func BuildIndicatorSourceAcceptance(input IndicatorSourceAcceptanceInput) (IndicatorSourceAcceptanceReceipt, error) {
	contract, err := ParseMetricSeriesBinding(input.Binding, input.View)
	if err != nil {
		return IndicatorSourceAcceptanceReceipt{}, errors.Join(ErrAcceptanceInvalid, err)
	}
	if err := validateAcceptancePageIdentity(input.Binding, input.View, input.Page); err != nil {
		return IndicatorSourceAcceptanceReceipt{}, err
	}

	organizations := make(map[string]struct{})
	roleCounts := make(map[string]int)
	branchRef := strings.TrimSpace(input.BranchRef)
	headOfficeRef := strings.TrimSpace(input.HeadOfficeRef)
	branchObserved := branchRef == ""
	headOfficeObserved := headOfficeRef == ""
	mapped, unmapped := 0, 0
	for _, record := range input.Page.Records {
		value, mapErr := contract.MapRecord(record)
		if mapErr != nil {
			unmapped++
			continue
		}
		mapped++
		organizations[value.OrganizationRef] = struct{}{}
		if branchRef != "" && value.OrganizationRef == branchRef {
			branchObserved = true
		}
		if headOfficeRef != "" && value.OrganizationRef == headOfficeRef {
			headOfficeObserved = true
		}
		for role := range value.Measurements {
			roleCounts[role]++
		}
	}
	if mapped == 0 {
		return IndicatorSourceAcceptanceReceipt{}, fmt.Errorf("%w: no source record mapped to a native measurement", ErrAcceptanceInvalid)
	}
	if !branchObserved {
		return IndicatorSourceAcceptanceReceipt{}, fmt.Errorf("%w: required branch witness was not observed", ErrAcceptanceInvalid)
	}
	if !headOfficeObserved {
		return IndicatorSourceAcceptanceReceipt{}, fmt.Errorf("%w: required head-office witness was not observed", ErrAcceptanceInvalid)
	}

	generatedAt := input.GeneratedAt.UTC()
	if generatedAt.IsZero() {
		generatedAt = input.Page.Receipt.ObservedAt.UTC()
	}
	if generatedAt.IsZero() {
		return IndicatorSourceAcceptanceReceipt{}, fmt.Errorf("%w: generated time is required", ErrAcceptanceInvalid)
	}
	receiptBytes, err := json.Marshal(input.Page.Receipt)
	if err != nil {
		return IndicatorSourceAcceptanceReceipt{}, errors.Join(ErrAcceptanceInvalid, err)
	}

	persisted, err := buildPersistedIndicatorAcceptance(input.Binding, input.View, input.Check, input.Result)
	if err != nil {
		return IndicatorSourceAcceptanceReceipt{}, err
	}

	return IndicatorSourceAcceptanceReceipt{
		Schema:                    IndicatorSourceAcceptanceSchema,
		GeneratedAt:               generatedAt,
		SourceIdentitySHA256:      acceptanceIdentityDigest("source", input.Binding.SourceID),
		BindingIdentitySHA256:     acceptanceIdentityDigest("binding", input.Binding.BindingID),
		BindingVersion:            input.Binding.Version,
		ViewIdentitySHA256:        acceptanceIdentityDigest("view", input.View.ViewID),
		ViewVersion:               input.View.Version,
		SchemaFingerprint:         input.View.SchemaFingerprint,
		MappingSHA256:             acceptanceBytesDigest(input.Binding.Mapping),
		OperationReceiptSHA256:    acceptanceBytesDigest(receiptBytes),
		ObservedAt:                input.Page.Receipt.ObservedAt.UTC(),
		Completeness:              input.Page.Receipt.Completeness,
		RecordsRead:               len(input.Page.Records),
		RecordsMapped:             mapped,
		RecordsUnmapped:           unmapped,
		DistinctOrganizations:     len(organizations),
		MeasurementRoleCounts:     roleCounts,
		MoreAvailable:             input.Page.NextCursor != nil,
		BranchWitnessRequired:     branchRef != "",
		BranchWitnessObserved:     branchObserved,
		HeadOfficeWitnessRequired: headOfficeRef != "",
		HeadOfficeWitnessObserved: headOfficeObserved,
		PersistedIndicator:        persisted,
	}, nil
}

func validateAcceptancePageIdentity(binding sourceaccess.BindingRevision, view sourceaccess.ViewRevision, page sourceaccess.RecordPage) error {
	receipt := page.Receipt
	if receipt.SourceID != binding.SourceID ||
		receipt.ViewID != view.ViewID ||
		receipt.ViewVersion != strconv.FormatInt(view.Version, 10) ||
		receipt.BindingID != binding.BindingID ||
		receipt.BindingVersion != strconv.FormatInt(binding.Version, 10) ||
		receipt.SchemaFingerprint != view.SchemaFingerprint ||
		receipt.Operation != sourceaccess.OperationPage ||
		receipt.ObservedAt.IsZero() {
		return fmt.Errorf("%w: source operation receipt does not match the governed binding revision", ErrAcceptanceInvalid)
	}
	if receipt.Count != int64(len(page.Records)) {
		return fmt.Errorf("%w: source receipt count does not match the bounded page", ErrAcceptanceInvalid)
	}
	return nil
}

func buildPersistedIndicatorAcceptance(binding sourceaccess.BindingRevision, view sourceaccess.ViewRevision, check *monitoring.MonitoringCheck, result *monitoring.MonitoringResult) (PersistedIndicatorAcceptance, error) {
	if check == nil && result == nil {
		return PersistedIndicatorAcceptance{}, nil
	}
	if check == nil || result == nil {
		return PersistedIndicatorAcceptance{}, fmt.Errorf("%w: monitoring check and result must be supplied together", ErrAcceptanceInvalid)
	}
	if check.InputKind != monitoring.InputSource ||
		check.BindingID != binding.BindingID ||
		check.BindingVersion != binding.Version ||
		result.InputKind != monitoring.InputSource ||
		result.MonitoringCheckID != check.ID ||
		result.MonitoringCheckVersion != check.Version ||
		result.Evaluation.Measurement == nil ||
		len(result.SourceReceipt) == 0 {
		return PersistedIndicatorAcceptance{}, fmt.Errorf("%w: persisted Indicator is not bound to the accepted native source revision", ErrAcceptanceInvalid)
	}
	var sourceReceipt sourceaccess.OperationReceipt
	if err := json.Unmarshal(result.SourceReceipt, &sourceReceipt); err != nil {
		return PersistedIndicatorAcceptance{}, errors.Join(ErrAcceptanceInvalid, fmt.Errorf("decode persisted source receipt: %w", err))
	}
	bindingMatched := sourceReceipt.BindingID == binding.BindingID && sourceReceipt.BindingVersion == strconv.FormatInt(binding.Version, 10) && sourceReceipt.SourceID == binding.SourceID
	viewMatched := sourceReceipt.ViewID == view.ViewID && sourceReceipt.ViewVersion == strconv.FormatInt(view.Version, 10)
	schemaMatched := sourceReceipt.SchemaFingerprint == view.SchemaFingerprint
	if !bindingMatched || !viewMatched || !schemaMatched || sourceReceipt.ObservedAt.IsZero() {
		return PersistedIndicatorAcceptance{}, fmt.Errorf("%w: persisted Indicator receipt does not match the accepted source revision", ErrAcceptanceInvalid)
	}
	receiptBytes, err := json.Marshal(sourceReceipt)
	if err != nil {
		return PersistedIndicatorAcceptance{}, errors.Join(ErrAcceptanceInvalid, err)
	}
	return PersistedIndicatorAcceptance{
		Proved:                   true,
		CheckIdentitySHA256:      acceptanceIdentityDigest("monitoring-check", check.ID),
		CheckVersion:             check.Version,
		ResultIdentitySHA256:     acceptanceIdentityDigest("monitoring-result", result.ID),
		ResultEvaluatedAt:        result.EvaluatedAt.UTC(),
		SourceReceiptSHA256:      acceptanceBytesDigest(receiptBytes),
		NativeMeasurement:        true,
		BindingRevisionMatched:   bindingMatched,
		ViewRevisionMatched:      viewMatched,
		SchemaFingerprintMatched: schemaMatched,
	}, nil
}

func acceptanceIdentityDigest(kind, value string) string {
	return acceptanceBytesDigest([]byte(strings.TrimSpace(kind) + "\x1f" + strings.TrimSpace(value)))
}

func acceptanceBytesDigest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}
