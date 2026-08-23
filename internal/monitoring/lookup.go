package monitoring

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// CheckRevisionForScope returns one exact monitoring-check revision for
// server-side scope/authority resolution. It deliberately does not weaken the
// normal actor-facing service methods or create a second monitoring read model.
func (s *Service) CheckRevisionForScope(ctx context.Context, tenantID, checkID string, version int64) (MonitoringCheck, error) {
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(checkID) == "" || version < 1 {
		return MonitoringCheck{}, errors.Join(ErrInvalid, fmt.Errorf("tenant, monitoring check and positive version are required"))
	}
	return s.repo.CheckRevision(ctx, tenantID, strings.TrimSpace(checkID), version)
}
