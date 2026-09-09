package evidence

import (
	"context"
	"fmt"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"time"
)

func ValidateCollectionSourceForField(field Field, source DocumentOccurrence) error {
	if field.Type != "vendor_document" {
		return fmt.Errorf("Select a document requirement")
	}
	if err := validateArtifactForField(field, Artifact{Status: source.ArtifactStatus, MediaType: source.MediaType, SizeBytes: source.SizeBytes}); err != nil {
		return err
	}
	if maximum := field.Constraints.MaxFileBytes; maximum != nil && source.SizeBytes > *maximum {
		return fmt.Errorf("%s file exceeds the size limit", field.Label)
	}
	if maximum := field.Constraints.MaxTotalFileBytes; maximum != nil && source.SizeBytes > *maximum {
		return fmt.Errorf("%s file exceeds the total size limit", field.Label)
	}
	if minimum := field.Constraints.MinFiles; minimum != nil && *minimum > 1 {
		return fmt.Errorf("%s requires more than one file", field.Label)
	}
	return nil
}

// CollectionResolution records bank use of an immutable, previously submitted
// occurrence. It is never a respondent answer or a document acceptance.
type CollectionResolution struct {
	ID                      string                      `json:"id"`
	Version                 int64                       `json:"version"`
	SupersedesID            string                      `json:"supersedes_id,omitempty"`
	Source                  DocumentOccurrence          `json:"source"`
	SourceArtifactRequestID string                      `json:"source_artifact_request_id"`
	Document                formcontract.DocumentAnswer `json:"document"`
	ReconciledBy            string                      `json:"reconciled_by"`
	ReconciledAt            time.Time                   `json:"reconciled_at"`
	Rationale               string                      `json:"rationale"`
	BankReviewState         string                      `json:"bank_review_state"`
	ReviewedBy              string                      `json:"reviewed_by,omitempty"`
	ReviewedAt              *time.Time                  `json:"reviewed_at,omitempty"`
}

func CollectionFieldFulfilled(field Field, now time.Time) bool {
	r := field.CollectionResolution
	if field.Type != "vendor_document" || r == nil || r.ID == "" || r.Version < 1 || r.Source.ArtifactStatus != ArtifactAvailable || !r.Source.Current || (r.BankReviewState != "PENDING" && r.BankReviewState != "VALIDATED") {
		return false
	}
	if r.Source.Review != nil && (r.Source.Review.Status == "REJECTED" || r.Source.Review.Status == "EXPIRED") {
		return false
	}
	for _, value := range []string{r.Source.ExpiresOn, r.Document.ExpiresOn} {
		if value != "" {
			expires, err := time.Parse("2006-01-02", value)
			if err != nil || now.UTC().Format("2006-01-02") > expires.Format("2006-01-02") {
				return false
			}
		}
	}
	return true
}

// RefreshCollectionResolutions checks the immutable original artifact under
// the caller's transaction/lock. A receipt never grants inventory access.
func RefreshCollectionResolutions(ctx context.Context, request Request, loader func(context.Context, string, string, string) (Artifact, error), now time.Time) Request {
	request.Fields = cloneFields(request.Fields)
	for i := range request.Fields {
		r := request.Fields[i].CollectionResolution
		if r == nil {
			continue
		}
		if loader == nil {
			r.Source.ArtifactStatus = ArtifactQuarantined
			continue
		}
		artifact, err := loader(ctx, request.TenantID, r.SourceArtifactRequestID, r.Source.ArtifactID)
		if err != nil || artifact.ID != r.Source.ArtifactID || artifact.RequestID != r.SourceArtifactRequestID || artifact.SHA256 != r.Source.SHA256 || artifact.SizeBytes != r.Source.SizeBytes {
			r.Source.ArtifactStatus = ArtifactQuarantined
			continue
		}
		r.Source.ArtifactStatus = artifact.Status
	}
	return request
}

// MutateCollectionRequest is the in-memory transaction seam for the owning
// bank workflow. The callback must finish every fallible check before mutation.
func (s *Service) MutateCollectionRequest(ctx context.Context, tenant, requestID string, expected int64, candidate *CollectionResolution, apply func(*Request) error) (Request, error) {
	repo, ok := s.repo.(interface {
		MutateCollectionRequest(context.Context, string, string, int64, *CollectionResolution, func(*Request) error) (Request, error)
	})
	if !ok {
		return Request{}, ErrNotFound
	}
	return repo.MutateCollectionRequest(ctx, tenant, requestID, expected, candidate, apply)
}

func (r *MemoryRepository) MutateCollectionRequest(ctx context.Context, tenant, requestID string, expected int64, candidate *CollectionResolution, apply func(*Request) error) (Request, error) {
	r.mu.RLock()
	currency := r.collectionCurrency
	r.mu.RUnlock()
	if currency != nil {
		currency.mu.RLock()
		defer currency.mu.RUnlock()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.requests[requestID]
	if !ok || current.TenantID != tenant {
		return Request{}, ErrNotFound
	}
	if current.Version != expected {
		return Request{}, ErrVersionConflict
	}
	if candidate == nil {
		return Request{}, ErrNotFound
	}
	probe := current
	probe.Fields = []Field{{Type: "vendor_document", CollectionResolution: candidate}}
	if currency != nil {
		probe = refreshCollectionSourceCurrencyMemoryLocked(currency, probe)
	} else {
		candidate.Source.Current = false
	}
	probe, err := refreshCollectionSourceReviews(ctx, probe, r.collectionReviews)
	if err != nil {
		return Request{}, err
	}
	probe = RefreshCollectionResolutions(ctx, probe, func(_ context.Context, tenant, requestID, artifactID string) (Artifact, error) {
		artifact, ok := r.artifacts[artifactID]
		if !ok || artifact.TenantID != tenant || artifact.RequestID != requestID {
			return Artifact{}, ErrNotFound
		}
		return artifact, nil
	}, time.Now().UTC())
	*candidate = *probe.Fields[0].CollectionResolution
	current.Fields = cloneFields(current.Fields)
	if err := apply(&current); err != nil {
		return Request{}, err
	}
	r.requests[requestID] = current
	current.Fields = cloneFields(current.Fields)
	return current, nil
}
