//go:build postgres

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oploss"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const opsLossSourceFile = "LOSS DATA BASE.xlsx"

var opsLossAmountPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,2})?$`)

type sourceLossValue struct {
	Code         string
	Title        string
	EventType    oploss.EventType
	Cause        string
	Description  string
	AmountMinor  int64
	Currency     string
	OccurredAt   time.Time
	DiscoveredAt time.Time
	Identity     string
	SourceSHA    string
}

// The historical OpsRisk workbook repeats the same events in monthly display
// sections. Month, quarter and reporting-year columns are not event identity.
func sourceLossProjection(group sourceRecordGroup, record sourceRecord) (sourceLossValue, bool, error) {
	if !strings.EqualFold(path.Base(strings.ReplaceAll(group.SourceFile, "\\", "/")), opsLossSourceFile) {
		return sourceLossValue{}, false, nil
	}
	sourceSHA := strings.ToLower(strings.TrimSpace(group.SourceSHA256))
	if len(sourceSHA) != 64 {
		return sourceLossValue{}, true, fmt.Errorf("OpsRisk loss source digest is missing")
	}
	if _, err := hex.DecodeString(sourceSHA); err != nil {
		return sourceLossValue{}, true, fmt.Errorf("OpsRisk loss source digest is invalid")
	}
	get := func(labels ...string) string { return sourceFieldValue(record, labels...) }
	account := get("ACCT_NAME")
	groupName := get("LOSS GROUP")
	particular := get("TRAN_PARTICULAR")
	branch := get("Branch")
	if account == "" || groupName == "" || particular == "" || branch == "" {
		return sourceLossValue{}, true, fmt.Errorf("source loss %s has incomplete event identity", record.Key)
	}
	currencyName := strings.ToUpper(get("CURRENCY OF LOSS"))
	var currency string
	switch currencyName {
	case "NAIRA", "NGN":
		currency = "NGN"
	default:
		return sourceLossValue{}, true, fmt.Errorf("source loss %s has unrecognized currency", record.Key)
	}
	amount, err := opsLossMinor(get("Amount", "AMOUNT"))
	if err != nil {
		return sourceLossValue{}, true, fmt.Errorf("source loss %s amount: %w", record.Key, err)
	}
	occurred, err := opsLossDate(get("DATE OF OCCURRENCE"))
	if err != nil {
		return sourceLossValue{}, true, fmt.Errorf("source loss %s occurrence date: %w", record.Key, err)
	}
	recognized, err := opsLossDate(get("DATE OF RECOGNITION"))
	if err != nil || recognized.Before(occurred) {
		return sourceLossValue{}, true, fmt.Errorf("source loss %s has invalid recognition date", record.Key)
	}
	eventParts := []string{
		strings.ToUpper(strings.TrimSpace(account)), strings.ToUpper(strings.TrimSpace(groupName)),
		strings.ToUpper(strings.TrimSpace(particular)), strings.ToUpper(strings.TrimSpace(branch)),
		occurred.Format("2006-01-02"),
	}
	eventDigest := sha256.Sum256([]byte(strings.Join(eventParts, "\x1f")))
	identityParts := append(append([]string{}, eventParts...), strconv.FormatInt(amount, 10), currency, recognized.Format("2006-01-02"))
	identityDigest := sha256.Sum256([]byte(strings.Join(identityParts, "\x1f")))
	identity := hex.EncodeToString(identityDigest[:])
	kind := oploss.EventOther
	if strings.EqualFold(strings.TrimSpace(groupName), "CASH SHORTAGES") {
		kind = oploss.EventExecutionDeliveryProcess
	}
	cause := get("ROOT CAUSE ANALYSIS")
	if cause == "" {
		cause = "Not recorded in source"
	}
	// A textual recovery statement is not a dated, evidenced recovery entry.
	// Keep the note visible but never turn it into a ledger transaction.
	notes := []string{
		"Imported historical OpsRisk source; monthly views deduplicated by event identity.",
		"Source SHA-256: " + sourceSHA,
		"Source row: " + strings.TrimSpace(record.SourceRange),
		"Source identity: " + identity,
		"Loss group: " + groupName,
		"Account: " + account,
		"Branch: " + branch,
		"Regional bank: " + get("Regional Bank"),
		"Directorate: " + get("Directorate"),
	}
	if recovered := get("RECOVERY DATA"); recovered != "" {
		notes = append(notes, "Unverified recovery note (not posted): "+recovered)
	}
	if recoveryDate := get("DATE OF RECOVERY"); recoveryDate != "" {
		notes = append(notes, "Source recovery date (no amount posted): "+recoveryDate)
	}
	return sourceLossValue{
		Code: "OPSL-" + strings.ToUpper(hex.EncodeToString(eventDigest[:10])),
		Title: sourceShort(particular, 240),
		EventType: kind, Cause: cause, Description: strings.Join(notes, "\n"),
		AmountMinor: amount, Currency: currency, OccurredAt: occurred, DiscoveredAt: recognized,
		Identity: identity, SourceSHA: sourceSHA,
	}, true, nil
}

func opsLossMinor(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if !opsLossAmountPattern.MatchString(value) {
		return 0, fmt.Errorf("requires a positive amount with at most two decimal places")
	}
	number := new(big.Rat)
	if _, ok := number.SetString(value); !ok {
		return 0, fmt.Errorf("invalid decimal")
	}
	minor := new(big.Rat).Mul(number, big.NewRat(100, 1))
	if !minor.IsInt() || !minor.Num().IsInt64() || minor.Num().Sign() <= 0 {
		return 0, fmt.Errorf("amount is outside exact NGN minor-unit range")
	}
	return minor.Num().Int64(), nil
}

func opsLossDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("source date is missing")
	}
	for _, layout := range []string{"2006-01-02", "02/01/2006", "2/1/2006", "02-01-2006", "2-1-2006"} {
		if parsed, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return parsed, nil
		}
	}
	if days, err := strconv.Atoi(value); err == nil && days >= 1 && days <= 100000 {
		return time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC).AddDate(0, 0, days), nil
	}
	return time.Time{}, fmt.Errorf("unsupported source date")
}

// Existing demo records are verified and left unchanged. A code collision,
// changed workbook version or manually altered financial entry fails closed.
func ensureSourceLoss(
	ctx context.Context,
	pool *pgxpool.Pool,
	service *oploss.Service,
	seed bankverticals.SeedConfig,
	value sourceLossValue,
) (bool, error) {
	var id string
	err := pool.QueryRow(ctx, `
		SELECT loss.id::text
		FROM operational_losses loss
		JOIN tenants tenant ON tenant.id=loss.tenant_id
		JOIN legal_entities entity ON entity.id=loss.legal_entity_id AND entity.tenant_id=loss.tenant_id
		WHERE (tenant.id::text=$1 OR tenant.slug=$1)
		  AND (entity.id::text=$2 OR entity.code=$2)
		  AND loss.code=$3
	`, seed.TenantID, seed.LegalEntityID, value.Code).Scan(&id)
	if err == nil {
		existing, getErr := service.Get(ctx, oploss.Scope{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID}, id)
		if getErr != nil {
			return false, getErr
		}
		row := existing.Loss
		if row.Version != 1 || row.Status != oploss.StatusActive ||
			row.Code != value.Code || row.Title != value.Title ||
			row.EventType != value.EventType || row.Cause != value.Cause ||
			row.GrossAmountMinor != value.AmountMinor || row.Currency != value.Currency ||
			!row.OccurredAt.Equal(value.OccurredAt) || !row.DiscoveredAt.Equal(value.DiscoveredAt) ||
			!strings.Contains(row.Description, "Source SHA-256: "+value.SourceSHA+"\n") ||
			!strings.Contains(row.Description, "Source identity: "+value.Identity+"\n") {
			return false, fmt.Errorf("source loss %s already exists with different provenance or an edited state", value.Code)
		}
		return false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	// No synthetic Risk, Matter, recovered amount or bank branch UUID is assigned.
	_, err = service.Create(ctx, oploss.CreateInput{
		TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID,
		Code: value.Code, Title: value.Title, EventType: value.EventType,
		Cause: value.Cause, Description: value.Description,
		GrossAmountMinor: value.AmountMinor, Currency: value.Currency,
		OccurredAt: value.OccurredAt, DiscoveredAt: value.DiscoveredAt,
		OwnerPrincipalID: seed.OwnerPrincipalID, ActorID: seed.ActorID,
	})
	return err == nil, err
}
