package evidence

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"mime"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/demodocuments"
)

type DocumentFileKind string

const (
	DocumentPDF         DocumentFileKind = "PDF"
	DocumentImage       DocumentFileKind = "IMAGE"
	DocumentWord        DocumentFileKind = "WORD"
	DocumentSpreadsheet DocumentFileKind = "SPREADSHEET"
	DocumentOther       DocumentFileKind = "OTHER"
)

// DocumentKindForMediaType classifies validated MIME types, never file extensions.
func DocumentKindForMediaType(value string) DocumentFileKind {
	media, _, err := mime.ParseMediaType(strings.ToLower(strings.TrimSpace(value)))
	if err != nil {
		return DocumentOther
	}
	switch media {
	case "application/pdf":
		return DocumentPDF
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/tiff", "image/bmp":
		return DocumentImage
	case "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return DocumentWord
	case "application/vnd.ms-excel", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "text/csv":
		return DocumentSpreadsheet
	default:
		return DocumentOther
	}
}

type DocumentReview struct {
	ID         string     `json:"id"`
	Status     string     `json:"status"`
	ReviewedBy string     `json:"reviewed_by"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
	Source     string     `json:"source"`
}
type DocumentOccurrence struct {
	SubmissionChannel    string           `json:"submission_channel"`
	ID                   string           `json:"id"`
	ArtifactID           string           `json:"artifact_id"`
	RequestID            string           `json:"request_id"`
	SubmissionID         string           `json:"submission_id"`
	FieldID              string           `json:"field_id"`
	ResponseRevisionID   string           `json:"response_revision_id,omitempty"`
	DistributionID       string           `json:"distribution_id,omitempty"`
	RelationshipID       string           `json:"relationship_id,omitempty"`
	AssessmentID         string           `json:"assessment_id,omitempty"`
	WorkRequestID        string           `json:"work_request_id,omitempty"`
	FormTemplateID       string           `json:"form_template_id,omitempty"`
	FormTemplateVersion  int64            `json:"form_template_version"`
	FormTitle            string           `json:"form_title"`
	FieldLabel           string           `json:"field_label"`
	FileName             string           `json:"file_name"`
	MediaType            string           `json:"media_type"`
	FileKind             DocumentFileKind `json:"file_kind"`
	SizeBytes            int64            `json:"size_bytes"`
	SHA256               string           `json:"sha256"`
	ArtifactStatus       ArtifactStatus   `json:"artifact_status"`
	DemoPreviewAvailable bool             `json:"demo_preview_available,omitempty"`
	UploadedAt           time.Time        `json:"uploaded_at"`
	UploadedBy           string           `json:"uploaded_by,omitempty"`
	SubmittedAt          time.Time        `json:"submitted_at"`
	SubmittedBy          string           `json:"submitted_by,omitempty"`
	ExpiresOn            string           `json:"expires_on,omitempty"`
	Current              bool             `json:"current"`
	Review               *DocumentReview  `json:"review,omitempty"`
	ArtifactRequestID    string           `json:"-"`
}
type DocumentQuery struct {
	// Set only after the service verifies the current review route for the exact
	// generic vendor response. This never expands a relationship-wide inventory.
	assessmentRead                                            bool
	TenantID, LegalEntityID, PrincipalID                      string
	FileKind                                                  DocumentFileKind
	Query, FormTemplateID, RelationshipID, ResponseRevisionID string
	CurrentOnly                                               bool
	Cursor                                                    string
	Limit                                                     int
	// Exact selectors are set only by the protected content handler.
	SubmissionID, FieldID, ArtifactID string
}
type DocumentPage struct {
	Items      []DocumentOccurrence `json:"items"`
	NextCursor string               `json:"next_cursor,omitempty"`
}
type documentCursor struct {
	SubmittedAt time.Time `json:"at"`
	ID          string    `json:"id"`
}
type documentStore interface {
	ListDocuments(context.Context, DocumentQuery) (DocumentPage, error)
}

// DocumentContextReader delegates memory-mode legacy reads to their owning
// workflow's exact link checks and read authority. PostgreSQL does this in SQL.
type DocumentContextReader interface {
	ResolveDocumentContext(context.Context, DocumentQuery, Request, Submission) (DocumentContext, error)
}
type DocumentContext struct {
	RelationshipID, AssessmentID, WorkRequestID string
	Current                                     bool
	Reviews                                     map[string]DocumentReview
	Expiries                                    map[string]string
}

func (s *DistributionService) ConfigureDocumentContexts(reader DocumentContextReader) {
	if s != nil {
		if store, ok := s.store.(*MemoryDistributionStore); ok {
			store.documentContexts = reader
		}
	}
}

func normalizeDocumentQuery(q *DocumentQuery) (documentCursor, error) {
	if q == nil || strings.TrimSpace(q.TenantID) == "" || strings.TrimSpace(q.LegalEntityID) == "" || q.LegalEntityID == "*" || strings.TrimSpace(q.PrincipalID) == "" || q.Limit < 1 || q.Limit > 100 || len(q.Query) > 200 || len(q.Cursor) > 2048 {
		return documentCursor{}, ErrDistributionInvalid
	}
	switch q.FileKind {
	case "", DocumentPDF, DocumentImage, DocumentWord, DocumentSpreadsheet, DocumentOther:
	default:
		return documentCursor{}, ErrDistributionInvalid
	}
	q.Query = strings.TrimSpace(q.Query)
	if q.Cursor == "" {
		return documentCursor{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(q.Cursor)
	if err != nil {
		return documentCursor{}, ErrDistributionInvalid
	}
	var c documentCursor
	if json.Unmarshal(raw, &c) != nil || c.ID == "" || c.SubmittedAt.IsZero() {
		return documentCursor{}, ErrDistributionInvalid
	}
	return c, nil
}
func documentPage(values []DocumentOccurrence, limit int) DocumentPage {
	page := DocumentPage{Items: values}
	if page.Items == nil {
		page.Items = []DocumentOccurrence{}
	}
	if len(values) > limit {
		page.Items = values[:limit]
		last := page.Items[limit-1]
		raw, _ := json.Marshal(documentCursor{last.SubmittedAt, last.ID})
		page.NextCursor = base64.RawURLEncoding.EncodeToString(raw)
	}
	return page
}
func (s *DistributionService) ListDocuments(ctx context.Context, q DocumentQuery) (DocumentPage, error) {
	if _, err := normalizeDocumentQuery(&q); err != nil {
		return DocumentPage{}, err
	}
	if s == nil || s.store == nil {
		return DocumentPage{}, ErrDistributionInvalid
	}
	reader, ok := s.store.(documentStore)
	if !ok {
		return DocumentPage{}, ErrDistributionInvalid
	}
	if q.ResponseRevisionID != "" {
		if assessments, ok := s.store.(responseAssessmentStore); ok {
			authorize := s.assessmentReadAuthority(ctx, q.TenantID, q.LegalEntityID, q.PrincipalID)
			m, err := assessments.ReadResponseAssessment(ctx, q.TenantID, q.LegalEntityID, q.PrincipalID, q.ResponseRevisionID, authorize)
			if err == nil && m.Summary.SubjectType == "VENDOR_RELATIONSHIP" && m.Request.Origin.Type != "THIRD_PARTY_WORK" && m.Request.Origin.Type != "THIRD_PARTY_ASSESSMENT" {
				q.assessmentRead = authorize(ctx, m)
			}
		}
	}
	page, err := reader.ListDocuments(ctx, q)
	if err != nil {
		return DocumentPage{}, err
	}
	// Decorate only the repository-authorized bounded page. Recompute even false
	// capabilities, and copy the slice so shared repository results stay unchanged.
	items := make([]DocumentOccurrence, len(page.Items))
	copy(items, page.Items)
	for i := range items {
		v := &items[i]
		v.DemoPreviewAvailable = s.demoSamplePreview && v.ArtifactStatus == ArtifactStoredUnscanned && demodocuments.Matches(v.FileName, v.MediaType, v.SHA256, v.SizeBytes)
	}
	page.Items = items
	return page, nil
}
