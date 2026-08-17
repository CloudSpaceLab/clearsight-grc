package aigovernance

import (
	"context"
	"time"
)

// RetentionMaintainer runs bounded receipt expiry and execution-grant expiry on
// the existing shared worker runtime; it deliberately does not introduce a
// gateway-specific scheduler.
type RetentionMaintainer struct{ Repo Repository }

func (m *RetentionMaintainer) Maintain(ctx context.Context, now time.Time, batch int) (int, error) {
	if m == nil || m.Repo == nil {
		return 0, nil
	}
	return m.Repo.MaintainRetention(ctx, now.UTC(), batch)
}
