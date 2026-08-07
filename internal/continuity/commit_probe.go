package continuity

import (
	"context"
	"fmt"
	"strings"
)

// CurrentVersionRepository exposes only the normalized authoritative version
// required to distinguish an uncommitted command failure from a post-commit
// response/read failure. It intentionally does not reconstruct event history.
type CurrentVersionRepository interface {
	CurrentProgramVersion(context.Context, string, string) (int64, error)
	CurrentMatterVersion(context.Context, string, string) (int64, error)
}

func (s *Service) CurrentVersion(ctx context.Context, tenant, aggregateType, aggregateID string) (int64, error) {
	if s == nil || s.repo == nil {
		return 0, fmt.Errorf("continuity repository is unavailable")
	}
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(aggregateID) == "" {
		return 0, fmt.Errorf("tenant and aggregate id are required")
	}
	repo, ok := s.repo.(CurrentVersionRepository)
	if !ok {
		return 0, fmt.Errorf("authoritative version probe is not supported")
	}
	switch strings.ToUpper(strings.TrimSpace(aggregateType)) {
	case "PROGRAM":
		return repo.CurrentProgramVersion(ctx, tenant, aggregateID)
	case "MATTER":
		return repo.CurrentMatterVersion(ctx, tenant, aggregateID)
	default:
		return 0, fmt.Errorf("unsupported aggregate type %q", aggregateType)
	}
}
