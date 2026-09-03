package aigovernance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

func (s *Service) CreateGatewayTransport(ctx context.Context, input CreateGatewayTransportInput) (GatewayTransportRevision, error) {
	if s == nil || s.repo == nil {
		return GatewayTransportRevision{}, ErrInvalid
	}
	tenantID := strings.TrimSpace(input.TenantID)
	environment := strings.ToUpper(strings.TrimSpace(input.Environment))
	makerID := strings.TrimSpace(input.MakerID)
	changeReason := strings.TrimSpace(input.ChangeReason)
	if tenantID == "" || makerID == "" || changeReason == "" || len(changeReason) > 1000 {
		return GatewayTransportRevision{}, ErrInvalid
	}
	if environment != "DEVELOPMENT" && environment != "TEST" && environment != "PRODUCTION" {
		return GatewayTransportRevision{}, ErrInvalid
	}
	definition, err := aigateway.ValidateTransportDefinition(strings.ToLower(environment), 10*time.Minute, input.Definition)
	if err != nil {
		return GatewayTransportRevision{}, errors.Join(ErrInvalid, err)
	}
	version, err := s.repo.NextGatewayTransportVersion(ctx, tenantID, environment)
	if err != nil {
		return GatewayTransportRevision{}, err
	}
	idValue, err := id.NewUUIDv7()
	if err != nil {
		return GatewayTransportRevision{}, err
	}
	now := s.now().UTC()
	value := GatewayTransportRevision{
		ID: idValue, TenantID: tenantID, Environment: environment, Definition: definition,
		Status: GatewayTransportDraft, MakerID: makerID, ChangeReason: changeReason,
		CreatedAt: now, UpdatedAt: now, Version: version, RecordVersion: 1,
	}
	value.Checksum = checksumGatewayTransport(value)
	return s.repo.CreateGatewayTransport(ctx, value)
}

func (s *Service) GetGatewayTransport(ctx context.Context, tenantID, id string) (GatewayTransportRevision, error) {
	if s == nil || s.repo == nil || strings.TrimSpace(tenantID) == "" || strings.TrimSpace(id) == "" {
		return GatewayTransportRevision{}, ErrInvalid
	}
	return s.repo.GatewayTransport(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(id))
}

func (s *Service) ListGatewayTransports(ctx context.Context, tenantID, environment string, limit int) ([]GatewayTransportRevision, error) {
	if s == nil || s.repo == nil || strings.TrimSpace(tenantID) == "" {
		return nil, ErrInvalid
	}
	environment = strings.ToUpper(strings.TrimSpace(environment))
	if environment != "" && environment != "DEVELOPMENT" && environment != "TEST" && environment != "PRODUCTION" {
		return nil, ErrInvalid
	}
	return s.repo.ListGatewayTransports(ctx, strings.TrimSpace(tenantID), environment, boundedLimit(limit))
}

func (s *Service) ActiveGatewayTransport(ctx context.Context, tenantID, environment string) (GatewayTransportRevision, error) {
	if s == nil || s.repo == nil || strings.TrimSpace(tenantID) == "" {
		return GatewayTransportRevision{}, ErrInvalid
	}
	environment = strings.ToUpper(strings.TrimSpace(environment))
	if environment != "DEVELOPMENT" && environment != "TEST" && environment != "PRODUCTION" {
		return GatewayTransportRevision{}, ErrInvalid
	}
	return s.repo.ActiveGatewayTransport(ctx, strings.TrimSpace(tenantID), environment)
}

func (s *Service) TransitionGatewayTransport(ctx context.Context, action string, input GatewayTransportTransitionInput) (GatewayTransportRevision, error) {
	if s == nil || s.repo == nil {
		return GatewayTransportRevision{}, ErrInvalid
	}
	value, err := s.repo.GatewayTransport(ctx, strings.TrimSpace(input.TenantID), strings.TrimSpace(input.ID))
	if err != nil {
		return GatewayTransportRevision{}, err
	}
	if input.ExpectedVersion > 0 && value.RecordVersion != input.ExpectedVersion {
		return GatewayTransportRevision{}, ErrConflict
	}
	if err := validateGatewayTransportChecksum(value); err != nil {
		return GatewayTransportRevision{}, errors.Join(ErrInvalid, err)
	}
	actor := strings.TrimSpace(input.ActorID)
	if actor == "" {
		return GatewayTransportRevision{}, ErrInvalid
	}
	now := s.now().UTC()
	activate := false
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "submit":
		if value.Status != GatewayTransportDraft {
			return GatewayTransportRevision{}, ErrInvalidTransition
		}
		value.Status = GatewayTransportPendingApproval
		value.SubmittedAt = &now
	case "approve":
		if value.Status != GatewayTransportPendingApproval {
			return GatewayTransportRevision{}, ErrInvalidTransition
		}
		if actor == value.MakerID {
			return GatewayTransportRevision{}, ErrMakerChecker
		}
		value.Status = GatewayTransportApproved
		value.CheckerID = actor
		value.ApprovedAt = &now
	case "activate":
		if value.Status != GatewayTransportApproved && value.Status != GatewayTransportSuspended {
			return GatewayTransportRevision{}, ErrInvalidTransition
		}
		if actor == value.MakerID {
			return GatewayTransportRevision{}, ErrMakerChecker
		}
		if value.CheckerID != "" && value.CheckerID != actor && value.Status == GatewayTransportApproved {
			return GatewayTransportRevision{}, ErrMakerChecker
		}
		value.Status = GatewayTransportActive
		value.CheckerID = actor
		value.ActivatedAt = &now
		value.SuspendedAt = nil
		activate = true
	case "suspend":
		if value.Status != GatewayTransportActive {
			return GatewayTransportRevision{}, ErrInvalidTransition
		}
		if actor == value.MakerID {
			return GatewayTransportRevision{}, ErrMakerChecker
		}
		value.Status = GatewayTransportSuspended
		value.SuspendedAt = &now
	case "retire":
		if value.Status == GatewayTransportRetired {
			return GatewayTransportRevision{}, ErrInvalidTransition
		}
		if actor == value.MakerID && value.Status != GatewayTransportDraft {
			return GatewayTransportRevision{}, ErrMakerChecker
		}
		value.Status = GatewayTransportRetired
		value.RetiredAt = &now
	default:
		return GatewayTransportRevision{}, ErrInvalidTransition
	}
	value.RecordVersion++
	value.UpdatedAt = now
	if activate {
		return s.repo.ActivateGatewayTransport(ctx, value, value.RecordVersion-1)
	}
	return s.repo.UpdateGatewayTransport(ctx, value, value.RecordVersion-1)
}

func checksumGatewayTransport(value GatewayTransportRevision) string {
	payload, err := json.Marshal(struct {
		TenantID    string                        `json:"tenant_id"`
		Environment string                        `json:"environment"`
		Version     int64                         `json:"version"`
		Definition  aigateway.TransportDefinition `json:"definition"`
	}{TenantID: value.TenantID, Environment: value.Environment, Version: value.Version, Definition: value.Definition})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func validateGatewayTransportChecksum(value GatewayTransportRevision) error {
	if value.Checksum == "" || value.Checksum != checksumGatewayTransport(value) {
		return fmt.Errorf("gateway transport checksum mismatch")
	}
	return nil
}
