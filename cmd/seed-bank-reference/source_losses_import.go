//go:build postgres

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oploss"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Import only canonical historical Loss entries, without re-running unrelated
// Forms, Programs or Matters. All source files remain private operator inputs.
func installSourceLossesOnly(
	ctx context.Context, cfg config.Config, pool *pgxpool.Pool, seed bankverticals.SeedConfig,
) (sourceRecordReceipt, error) {
	var receipt sourceRecordReceipt
	if sourceRecordFiles == nil || pool == nil || strings.TrimSpace(seed.OwnerPrincipalID) == "" {
		return receipt, fmt.Errorf("source loss import requires private manifests, a database and an existing demo owner")
	}
	if cfg.Environment == "production" || !cfg.DemoMode ||
		seed.TenantID != "00000000-0000-4000-8000-000000000001" ||
		seed.LegalEntityID != "00000000-0000-4000-8000-000000000002" {
		return receipt, fmt.Errorf("source losses require the non-production Clear Bank demo scope")
	}
	// Validate the complete source population before the first ledger write.
	// A truncated private manifest must not leave a partial eight-event seed.
	values, err := sourceLossCandidateValues(sourceRecordFiles)
	if err != nil {
		return receipt, err
	}
	lock, err := pool.Acquire(ctx)
	if err != nil {
		return receipt, err
	}
	defer lock.Release()
	var held bool
	if err = lock.QueryRow(ctx, "SELECT pg_try_advisory_lock(842019260910)").Scan(&held); err != nil {
		return receipt, err
	}
	if !held {
		return receipt, fmt.Errorf("source record installation is already running")
	}
	defer func() { _, _ = lock.Exec(context.Background(), "SELECT pg_advisory_unlock(842019260910)") }()

	service := oploss.NewService(oploss.NewPostgresRepository(pool))
	for _, value := range values {
		created, installErr := ensureSourceLoss(ctx, pool, service, seed, value)
		if installErr != nil {
			return receipt, fmt.Errorf("source loss %s: %w", value.Code, installErr)
		}
		receipt.Losses++
		if created {
			receipt.LossesCreated++
		}
	}
	return receipt, nil
}

// Read, reconcile and validate the entire private source package before
// invoking any canonical Loss command. Monthly views are not new events.
func sourceLossCandidateValues(files fs.FS) ([]sourceLossValue, error) {
	seen := make(map[string]sourceLossValue, 8)
	values := make([]sourceLossValue, 0, 8)
	for _, file := range []string{"source_records_ops.json", "source_records_ops_loss.json"} {
		data, readErr := fs.ReadFile(files, file)
		if errors.Is(readErr, fs.ErrNotExist) {
			continue
		}
		if readErr != nil {
			return nil, readErr
		}
		var manifest sourceRecordManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		if manifest.Version != 1 {
			return nil, fmt.Errorf("%s has unsupported manifest version", file)
		}
		for _, group := range manifest.Groups {
			for _, record := range group.Records {
				value, candidate, projectErr := sourceLossProjection(group, record)
				if projectErr != nil {
					return nil, fmt.Errorf("source loss %s: %w", record.Key, projectErr)
				}
				if !candidate {
					continue
				}
				if previous, duplicate := seen[value.Code]; duplicate {
					if previous.Identity != value.Identity || previous.SourceSHA != value.SourceSHA {
						return nil, fmt.Errorf("conflicting monthly source loss %s", value.Code)
					}
					continue
				}
				seen[value.Code] = value
				values = append(values, value)
			}
		}
	}
	if len(values) != 8 {
		return nil, fmt.Errorf("historical OpsRisk loss source has %d distinct events; expected eight", len(values))
	}
	return values, nil
}
