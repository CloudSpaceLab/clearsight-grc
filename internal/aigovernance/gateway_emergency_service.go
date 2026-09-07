package aigovernance

import (
	"context"
	"errors"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

func (s *Service) GatewayEmergencyControl(ctx context.Context, tenantID, environment string) (GatewayEmergencyControl, error) {
	if s == nil || s.repo == nil {
		return GatewayEmergencyControl{}, ErrInvalid
	}
	repo, ok := s.repo.(gatewayEmergencyRepository)
	if !ok {
		return GatewayEmergencyControl{}, ErrInvalid
	}
	tenantID = strings.TrimSpace(tenantID)
	environment = normalizeGatewayEnvironment(environment)
	if tenantID == "" || environment == "" {
		return GatewayEmergencyControl{}, ErrInvalid
	}
	value, err := repo.GatewayEmergencyControl(ctx, tenantID, environment)
	if errors.Is(err, ErrNotFound) {
		return GatewayEmergencyControl{TenantID: tenantID, Environment: environment}, nil
	}
	return value, err
}

func (s *Service) SetGatewayEmergencyControl(ctx context.Context, input SetGatewayEmergencyControlInput) (GatewayEmergencyControl, error) {
	if s == nil || s.repo == nil {
		return GatewayEmergencyControl{}, ErrInvalid
	}
	repo, ok := s.repo.(gatewayEmergencyRepository)
	if !ok {
		return GatewayEmergencyControl{}, ErrInvalid
	}
	tenantID := strings.TrimSpace(input.TenantID)
	environment := normalizeGatewayEnvironment(input.Environment)
	actorID := strings.TrimSpace(input.ActorID)
	reason := strings.TrimSpace(input.Reason)
	if tenantID == "" || environment == "" || actorID == "" || reason == "" || len(reason) > 1000 || input.ExpectedVersion < 0 {
		return GatewayEmergencyControl{}, ErrInvalid
	}

	current, err := repo.GatewayEmergencyControl(ctx, tenantID, environment)
	exists := err == nil
	if err != nil && !errors.Is(err, ErrNotFound) {
		return GatewayEmergencyControl{}, err
	}
	if !exists {
		if !input.Frozen || input.ExpectedVersion != 0 {
			return GatewayEmergencyControl{}, ErrInvalidTransition
		}
		controlID, err := id.NewUUIDv7()
		if err != nil {
			return GatewayEmergencyControl{}, err
		}
		current = GatewayEmergencyControl{ID: controlID, TenantID: tenantID, Environment: environment}
	} else {
		if current.RecordVersion != input.ExpectedVersion {
			return GatewayEmergencyControl{}, ErrConflict
		}
		if current.Frozen == input.Frozen {
			return GatewayEmergencyControl{}, ErrInvalidTransition
		}
	}

	now := s.now().UTC()
	current.Frozen = input.Frozen
	current.Reason = reason
	current.ActorID = actorID
	current.UpdatedAt = now
	current.RecordVersion++
	return repo.SetGatewayEmergencyControl(ctx, current, input.ExpectedVersion)
}

func normalizeGatewayEnvironment(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	switch value {
	case "DEVELOPMENT", "TEST", "PRODUCTION":
		return value
	default:
		return ""
	}
}
