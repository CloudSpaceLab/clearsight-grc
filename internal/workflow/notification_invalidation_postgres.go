//go:build postgres

package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/CloudSpaceLab/clearsight-grc/internal/invalidation"
)

func (r *PostgresRepository) publishNotificationInvalidation(ctx context.Context, record inAppNotificationRecord) error {
	if r == nil || r.pool == nil {
		return ErrNotificationUnavailable
	}
	digest := sha256.Sum256([]byte(record.OutboxEventID + "\x00" + record.Kind))
	payload, err := json.Marshal(invalidation.Event{
		TenantID: record.TenantID, LegalEntityID: record.LegalEntityID,
		PrincipalID: record.PrincipalID, Revision: hex.EncodeToString(digest[:16]),
	})
	if err != nil {
		return fmt.Errorf("encode actor invalidation: %w", err)
	}
	if _, err := r.pool.Exec(ctx, "SELECT pg_notify($1,$2)", invalidation.PostgresChannel, string(payload)); err != nil {
		return fmt.Errorf("publish actor invalidation: %w", err)
	}
	return nil
}
