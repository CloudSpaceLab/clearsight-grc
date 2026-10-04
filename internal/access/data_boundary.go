package access

import (
	"context"
	"sort"
	"strings"
	"time"
)

type DetailTransferMode string

const (
	DetailTransferAggregateOnly DetailTransferMode = "AGGREGATE_ONLY"
	DetailTransferAllowlist     DetailTransferMode = "ALLOWLIST"
)

type LegalEntityDataBoundary struct {
	LegalEntityID             string             `json:"legal_entity_id"`
	LegalEntityCode           string             `json:"legal_entity_code"`
	LegalEntityName           string             `json:"legal_entity_name"`
	Jurisdiction              string             `json:"jurisdiction,omitempty"`
	ResidencyRegion           string             `json:"residency_region,omitempty"`
	DetailTransferMode        DetailTransferMode `json:"detail_transfer_mode"`
	AllowedDestinationRegions []string           `json:"allowed_destination_regions"`
	Version                   int64              `json:"version"`
	Configured                bool               `json:"configured"`
	UpdatedAt                 *time.Time         `json:"updated_at,omitempty"`
}

type LegalEntityDataBoundaryRevision struct {
	ID                         string             `json:"id"`
	LegalEntityID              string             `json:"legal_entity_id"`
	BaseVersion                int64              `json:"base_version"`
	ProposedResidencyRegion    string             `json:"proposed_residency_region"`
	ProposedDetailTransferMode DetailTransferMode `json:"proposed_detail_transfer_mode"`
	ProposedDestinationRegions []string           `json:"proposed_destination_regions"`
	MakerID                    string             `json:"maker_id"`
	CheckerID                  string             `json:"checker_id,omitempty"`
	Status                     string             `json:"status"`
	Rationale                  string             `json:"rationale,omitempty"`
	CreatedAt                  time.Time          `json:"created_at"`
	DecidedAt                  *time.Time         `json:"decided_at,omitempty"`
	AppliedAt                  *time.Time         `json:"applied_at,omitempty"`
}

type DataBoundaryReader interface {
	GetLegalEntityDataBoundary(context.Context, string, string) (LegalEntityDataBoundary, error)
}

type DataBoundaryAdministrator interface {
	ProposeLegalEntityDataBoundary(context.Context, ProposeLegalEntityDataBoundaryInput) (LegalEntityDataBoundaryRevision, error)
	ApproveLegalEntityDataBoundary(context.Context, DecideLegalEntityDataBoundaryInput) error
	RejectLegalEntityDataBoundary(context.Context, DecideLegalEntityDataBoundaryInput) error
}

type ProposeLegalEntityDataBoundaryInput struct {
	TenantID                  string             `json:"tenant_id"`
	LegalEntityID             string             `json:"legal_entity_id"`
	ResidencyRegion           string             `json:"residency_region"`
	DetailTransferMode        DetailTransferMode `json:"detail_transfer_mode"`
	AllowedDestinationRegions []string           `json:"allowed_destination_regions,omitempty"`
	ExpectedVersion           int64              `json:"expected_version"`
	ActorID                   string             `json:"-"`
}

type DecideLegalEntityDataBoundaryInput struct {
	TenantID      string `json:"tenant_id"`
	LegalEntityID string `json:"legal_entity_id"`
	RevisionID    string `json:"revision_id"`
	ActorID       string `json:"-"`
	Rationale     string `json:"rationale"`
}

func NormalizeLegalEntityDataBoundaryInput(input ProposeLegalEntityDataBoundaryInput) (ProposeLegalEntityDataBoundaryInput, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.LegalEntityID = strings.TrimSpace(input.LegalEntityID)
	input.ResidencyRegion = normalizeDataRegion(input.ResidencyRegion)
	input.ActorID = strings.TrimSpace(input.ActorID)
	if input.TenantID == "" || input.LegalEntityID == "" || input.ActorID == "" ||
		input.ExpectedVersion < 0 || !validDataRegion(input.ResidencyRegion) {
		return ProposeLegalEntityDataBoundaryInput{}, ErrAdminInvalid
	}
	switch input.DetailTransferMode {
	case DetailTransferAggregateOnly:
		input.AllowedDestinationRegions = []string{}
	case DetailTransferAllowlist:
		values := make([]string, 0, len(input.AllowedDestinationRegions))
		seen := make(map[string]struct{}, len(input.AllowedDestinationRegions))
		for _, raw := range input.AllowedDestinationRegions {
			value := normalizeDataRegion(raw)
			if !validDataRegion(value) {
				return ProposeLegalEntityDataBoundaryInput{}, ErrAdminInvalid
			}
			if _, exists := seen[value]; exists {
				continue
			}
			seen[value] = struct{}{}
			values = append(values, value)
		}
		sort.Strings(values)
		if len(values) == 0 || len(values) > 64 {
			return ProposeLegalEntityDataBoundaryInput{}, ErrAdminInvalid
		}
		input.AllowedDestinationRegions = values
	default:
		return ProposeLegalEntityDataBoundaryInput{}, ErrAdminInvalid
	}
	return input, nil
}

func LegalEntityDetailTransferAllowed(boundary LegalEntityDataBoundary, destinationRegion string) bool {
	if !boundary.Configured || boundary.DetailTransferMode != DetailTransferAllowlist {
		return false
	}
	destinationRegion = normalizeDataRegion(destinationRegion)
	if !validDataRegion(destinationRegion) {
		return false
	}
	for _, allowed := range boundary.AllowedDestinationRegions {
		if normalizeDataRegion(allowed) == destinationRegion {
			return true
		}
	}
	return false
}

func normalizeDataRegion(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func validDataRegion(value string) bool {
	if len(value) < 2 || len(value) > 32 {
		return false
	}
	for _, r := range value {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}
