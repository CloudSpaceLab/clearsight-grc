package activity

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

var (
	ErrExportTooLarge = errors.New("audit export exceeds the synchronous export limit")
	ErrExportInvalid  = errors.New("audit export request is invalid")
)

const (
	ExportFormatCSV    = "CSV"
	ExportFormatNDJSON = "NDJSON"
	ExportStatusReady  = "READY"
	ExportStatusFailed = "FAILED"

	exportPageSize  = 100
	maxExportRows   = 10_000
	maxExportBytes  = int64(32 << 20)
	exportRetention = 7 * 24 * time.Hour
	exportSchema    = "clearsight.audit-export.v1"
)

type ExportFilter struct {
	From          *time.Time `json:"from,omitempty"`
	To            *time.Time `json:"to,omitempty"`
	Category      string     `json:"category,omitempty"`
	EventType     string     `json:"event_type,omitempty"`
	ObjectType    string     `json:"object_type,omitempty"`
	ObjectID      string     `json:"object_id,omitempty"`
	ActorID       string     `json:"actor_id,omitempty"`
	ActorQuery    string     `json:"actor_query,omitempty"`
	ActorKind     string     `json:"actor_kind,omitempty"`
	LegalEntityID string     `json:"legal_entity_id,omitempty"`
}

type ExportReceipt struct {
	ID             string       `json:"id"`
	TenantID       string       `json:"tenant_id"`
	LegalEntityID  string       `json:"legal_entity_id,omitempty"`
	RequestedBy    string       `json:"requested_by"`
	Format         string       `json:"format"`
	Filter         ExportFilter `json:"filter"`
	AsOf           time.Time    `json:"as_of"`
	Status         string       `json:"status"`
	RowCount       int          `json:"row_count"`
	DataSHA256     string       `json:"data_sha256,omitempty"`
	ManifestSHA256 string       `json:"manifest_sha256,omitempty"`
	FailureCode    string       `json:"failure_code,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	CompletedAt    *time.Time   `json:"completed_at,omitempty"`
	ExpiresAt      time.Time    `json:"expires_at"`
	DataObjectKey  string       `json:"-"`
	ManifestKey    string       `json:"-"`
}

type ExportRepository interface {
	CreateExport(context.Context, ExportReceipt) (ExportReceipt, error)
	CompleteExport(context.Context, ExportReceipt) (ExportReceipt, error)
	FailExport(context.Context, string, string, string) error
	GetExport(context.Context, string, string) (ExportReceipt, error)
	RecordDownload(context.Context, string, string, string) error
}

type ExportManifest struct {
	SchemaVersion  string       `json:"schema_version"`
	TenantID       string       `json:"tenant_id"`
	LegalEntityID  string       `json:"legal_entity_id,omitempty"`
	RequestedBy    string       `json:"requested_by"`
	GeneratedAt    time.Time    `json:"generated_at"`
	AsOf           time.Time    `json:"as_of"`
	Filter         ExportFilter `json:"filter"`
	Format         string       `json:"format"`
	RowCount       int          `json:"row_count"`
	DataSHA256     string       `json:"data_sha256"`
	Truncated      bool         `json:"truncated"`
	RetentionUntil time.Time    `json:"retention_until"`
}

type ExportService struct {
	activity *Service
	receipts ExportRepository
	objects  evidence.ObjectStore
	now      func() time.Time
}

func NewExportService(activity *Service, receipts ExportRepository, objects evidence.ObjectStore) *ExportService {
	return &ExportService{activity: activity, receipts: receipts, objects: objects, now: time.Now}
}

func (s *ExportService) Create(ctx context.Context, tenantID, legalEntityID, requestedBy, format string, query Query) (ExportReceipt, error) {
	if s == nil || s.activity == nil || s.receipts == nil || s.objects == nil {
		return ExportReceipt{}, ErrExportInvalid
	}
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	requestedBy = strings.TrimSpace(requestedBy)
	format = strings.ToUpper(strings.TrimSpace(format))
	if tenantID == "" || requestedBy == "" || (format != ExportFormatCSV && format != ExportFormatNDJSON) {
		return ExportReceipt{}, ErrExportInvalid
	}

	asOf := s.now().UTC()
	query.TenantID = tenantID
	query.Cursor = ""
	query.Limit = exportPageSize
	if query.To == nil || query.To.After(asOf) {
		query.To = &asOf
	}
	query = normalizeQuery(query)
	if query.From != nil && query.From.After(*query.To) {
		return ExportReceipt{}, ErrExportInvalid
	}
	filter := exportFilterFromQuery(query)
	receipt, err := s.receipts.CreateExport(ctx, ExportReceipt{
		TenantID: tenantID, LegalEntityID: legalEntityID, RequestedBy: requestedBy,
		Format: format, Filter: filter, AsOf: asOf, ExpiresAt: asOf.Add(exportRetention),
	})
	if err != nil {
		return ExportReceipt{}, err
	}

	data, rowCount, err := s.render(ctx, query, format)
	if err != nil {
		code := "EXPORT_GENERATION_FAILED"
		if errors.Is(err, ErrExportTooLarge) {
			code = "EXPORT_TOO_LARGE"
		}
		_ = s.receipts.FailExport(ctx, tenantID, receipt.ID, code)
		return ExportReceipt{}, err
	}

	extension := "csv"
	if format == ExportFormatNDJSON {
		extension = "ndjson"
	}
	dataKey := fmt.Sprintf("audit-exports/%s/audit.%s", receipt.ID, extension)
	dataInfo, err := s.objects.Put(ctx, dataKey, bytes.NewReader(data), maxExportBytes)
	if err != nil {
		_ = s.receipts.FailExport(ctx, tenantID, receipt.ID, "EXPORT_STORAGE_FAILED")
		return ExportReceipt{}, err
	}

	manifest := ExportManifest{
		SchemaVersion: exportSchema, TenantID: tenantID, LegalEntityID: legalEntityID, RequestedBy: requestedBy,
		GeneratedAt: asOf, AsOf: asOf, Filter: filter, Format: format, RowCount: rowCount,
		DataSHA256: dataInfo.SHA256, Truncated: false, RetentionUntil: receipt.ExpiresAt,
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		_ = s.objects.Delete(ctx, dataKey)
		_ = s.receipts.FailExport(ctx, tenantID, receipt.ID, "EXPORT_MANIFEST_FAILED")
		return ExportReceipt{}, err
	}
	manifestKey := fmt.Sprintf("audit-exports/%s/manifest.json", receipt.ID)
	manifestInfo, err := s.objects.Put(ctx, manifestKey, bytes.NewReader(manifestData), maxExportBytes)
	if err != nil {
		_ = s.objects.Delete(ctx, dataKey)
		_ = s.receipts.FailExport(ctx, tenantID, receipt.ID, "EXPORT_STORAGE_FAILED")
		return ExportReceipt{}, err
	}

	receipt.Status = ExportStatusReady
	receipt.RowCount = rowCount
	receipt.DataObjectKey = dataKey
	receipt.ManifestKey = manifestKey
	receipt.DataSHA256 = dataInfo.SHA256
	receipt.ManifestSHA256 = manifestInfo.SHA256
	completedAt := s.now().UTC()
	receipt.CompletedAt = &completedAt
	completed, err := s.receipts.CompleteExport(ctx, receipt)
	if err != nil {
		_ = s.objects.Delete(ctx, dataKey)
		_ = s.objects.Delete(ctx, manifestKey)
		return ExportReceipt{}, err
	}
	return completed, nil
}

func (s *ExportService) Get(ctx context.Context, tenantID, exportID string) (ExportReceipt, error) {
	if s == nil || s.receipts == nil || strings.TrimSpace(tenantID) == "" || strings.TrimSpace(exportID) == "" {
		return ExportReceipt{}, ErrExportInvalid
	}
	return s.receipts.GetExport(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(exportID))
}

func (s *ExportService) Open(ctx context.Context, tenantID, exportID, downloadedBy string) (ExportReceipt, io.ReadCloser, error) {
	if s == nil || s.objects == nil || strings.TrimSpace(downloadedBy) == "" {
		return ExportReceipt{}, nil, ErrExportInvalid
	}
	receipt, err := s.Get(ctx, tenantID, exportID)
	if err != nil {
		return ExportReceipt{}, nil, err
	}
	if receipt.Status != ExportStatusReady || receipt.DataObjectKey == "" || s.now().UTC().After(receipt.ExpiresAt) {
		return ExportReceipt{}, nil, ErrNotFound
	}
	reader, err := s.objects.Open(ctx, receipt.DataObjectKey)
	if err != nil {
		return ExportReceipt{}, nil, err
	}
	if err := s.receipts.RecordDownload(ctx, strings.TrimSpace(tenantID), receipt.ID, strings.TrimSpace(downloadedBy)); err != nil {
		_ = reader.Close()
		return ExportReceipt{}, nil, err
	}
	return receipt, reader, nil
}

func (s *ExportService) render(ctx context.Context, query Query, format string) ([]byte, int, error) {
	var buffer bytes.Buffer
	var csvWriter *csv.Writer
	var jsonEncoder *json.Encoder
	if format == ExportFormatCSV {
		csvWriter = csv.NewWriter(&buffer)
		if err := csvWriter.Write([]string{"event_id", "occurred_at", "category", "event_type", "action", "outcome", "actor_kind", "actor_id", "actor_display_name", "legal_entity_id", "object_type", "object_id", "request_id", "correlation_id", "source"}); err != nil {
			return nil, 0, err
		}
	} else {
		jsonEncoder = json.NewEncoder(&buffer)
		jsonEncoder.SetEscapeHTML(false)
	}

	rowCount := 0
	cursor := ""
	for {
		query.Cursor = cursor
		page, err := s.activity.List(ctx, query)
		if err != nil {
			return nil, 0, err
		}
		if rowCount+len(page.Items) > maxExportRows {
			return nil, 0, ErrExportTooLarge
		}
		for _, event := range page.Items {
			if csvWriter != nil {
				if err := csvWriter.Write(exportCSVRow(event)); err != nil {
					return nil, 0, err
				}
			} else if err := jsonEncoder.Encode(event); err != nil {
				return nil, 0, err
			}
			rowCount++
		}
		if csvWriter != nil {
			csvWriter.Flush()
			if err := csvWriter.Error(); err != nil {
				return nil, 0, err
			}
		}
		if int64(buffer.Len()) > maxExportBytes {
			return nil, 0, ErrExportTooLarge
		}
		if page.NextCursor == "" {
			break
		}
		if rowCount >= maxExportRows {
			return nil, 0, ErrExportTooLarge
		}
		cursor = page.NextCursor
	}
	return buffer.Bytes(), rowCount, nil
}

func exportFilterFromQuery(query Query) ExportFilter {
	return ExportFilter{
		From: query.From, To: query.To, Category: query.Category, EventType: query.EventType,
		ObjectType: query.ObjectType, ObjectID: query.ObjectID, ActorID: query.ActorID,
		ActorQuery: query.ActorQuery, ActorKind: query.ActorKind, LegalEntityID: query.LegalEntityID,
	}
}

func exportCSVRow(event Event) []string {
	return []string{
		event.ID, event.OccurredAt.UTC().Format(time.RFC3339Nano), event.Category, event.EventType, event.Action,
		event.Outcome, event.ActorKind, event.ActorID, event.ActorDisplayName, event.LegalEntityID,
		event.ObjectType, event.ObjectID, event.RequestID, event.CorrelationID, event.Source,
	}
}
