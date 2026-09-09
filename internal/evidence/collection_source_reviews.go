package evidence

import "context"

// CollectionSourceReviewReader exposes only the current review of the exact
// source occurrence already authorized by a bank collection receipt.
type CollectionSourceReviewReader interface {
	ReadCollectionSourceReview(context.Context, string, string, string, string, string) (DocumentReview, string, error)
}

func (s *Service) ConfigureCollectionReviewReader(reader CollectionSourceReviewReader) {
	if configured, ok := s.repo.(interface {
		ConfigureCollectionReviewReader(CollectionSourceReviewReader)
	}); ok {
		configured.ConfigureCollectionReviewReader(reader)
	}
}

func (r *MemoryRepository) ConfigureCollectionReviewReader(reader CollectionSourceReviewReader) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.collectionReviews = reader
}

func (r *MemoryRepository) RefreshCollectionRequestReviews(ctx context.Context, request Request) (Request, error) {
	r.mu.RLock()
	reader := r.collectionReviews
	currency := r.collectionCurrency
	r.mu.RUnlock()
	if currency != nil {
		currency.mu.RLock()
		r.mu.RLock()
		request = refreshCollectionSourceCurrencyMemoryLocked(currency, request)
		r.mu.RUnlock()
		currency.mu.RUnlock()
	} else {
		request.Fields = cloneFields(request.Fields)
		for i := range request.Fields {
			if receipt := request.Fields[i].CollectionResolution; receipt != nil {
				receipt.Source.Current = false
			}
		}
	}
	return refreshCollectionSourceReviews(ctx, request, reader)
}

// The caller holds the distribution and capture repository locks, in that order.
// Refresh only the live view; the original receipt remains reconstructable.
func refreshCollectionSourceCurrencyMemoryLocked(store *MemoryDistributionStore, request Request) Request {
	request.Fields = cloneFields(request.Fields)
	revisions := map[string]ResponseRevision{}
	for _, group := range store.responseRevisions {
		for _, revision := range group {
			revisions[revision.SubmissionID] = revision
		}
	}
	for i := range request.Fields {
		receipt := request.Fields[i].CollectionResolution
		if receipt == nil {
			continue
		}
		source := &receipt.Source
		source.Current = false
		original, ok := store.repo.requests[source.RequestID]
		if !ok || original.TenantID != request.TenantID || original.LegalEntityID != request.LegalEntityID || original.SubjectID != source.RelationshipID {
			continue
		}
		submission, ok := store.repo.submissions[source.SubmissionID]
		if !ok || submission.TenantID != request.TenantID || submission.RequestID != original.ID {
			continue
		}
		present := false
		for _, field := range original.Fields {
			if field.ID == source.FieldID {
				present = true
			}
		}
		if !present {
			continue
		}
		revision := revisions[submission.ID]
		if revision.ID != source.ResponseRevisionID || !collectionMemoryRevisionScope(store, original, revision) {
			continue
		}
		if original.Origin.Type != "THIRD_PARTY_WORK" && original.Origin.Type != "THIRD_PARTY_ASSESSMENT" {
			source.Current = revision.ID != "" && revision.Current
			continue
		}
		prior := documentFieldSource{submission.ID, original.Origin.Version, revision.Revision, submission.SubmittedAt}
		source.Current = true
		for _, candidate := range store.repo.submissions {
			newer, found := store.repo.requests[candidate.RequestID]
			if !found || candidate.TenantID != request.TenantID || newer.TenantID != request.TenantID || newer.LegalEntityID != request.LegalEntityID || newer.Origin.Type != original.Origin.Type || newer.Origin.ID != original.Origin.ID || newer.SubjectType != original.SubjectType || newer.SubjectID != original.SubjectID || newer.FormTemplateID != original.FormTemplateID || newer.FormTemplateVersion != original.FormTemplateVersion {
				continue
			}
			candidateRevision := revisions[candidate.ID]
			if !collectionMemoryRevisionScope(store, newer, candidateRevision) {
				continue
			}
			position := documentFieldSource{candidate.ID, newer.Origin.Version, candidateRevision.Revision, candidate.SubmittedAt}
			if !position.after(prior) {
				continue
			}
			for _, field := range newer.Fields {
				if field.ID == source.FieldID {
					source.Current = false
					break
				}
			}
			if !source.Current {
				break
			}
		}
	}
	return request
}

func collectionMemoryRevisionScope(store *MemoryDistributionStore, request Request, revision ResponseRevision) bool {
	if revision.ID == "" {
		return true
	}
	distribution, found := store.distributions[revision.DistributionID]
	return found && revision.TenantID == request.TenantID && revision.LegalEntityID == request.LegalEntityID && store.requestDistribution[request.ID] == revision.DistributionID && distribution.TenantID == request.TenantID && distribution.LegalEntityID == request.LegalEntityID && distribution.SubjectType == request.SubjectType && distribution.SubjectID == request.SubjectID && distribution.FormTemplateID == request.FormTemplateID && distribution.FormTemplateVersion == request.FormTemplateVersion
}

func refreshCollectionRequestReviews(ctx context.Context, repo Repository, request Request) (Request, error) {
	if reader, ok := repo.(interface {
		RefreshCollectionRequestReviews(context.Context, Request) (Request, error)
	}); ok {
		return reader.RefreshCollectionRequestReviews(ctx, request)
	}
	return refreshCollectionSourceReviews(ctx, request, nil)
}

func refreshCollectionSourceReviews(ctx context.Context, request Request, reader CollectionSourceReviewReader) (Request, error) {
	request.Fields = cloneFields(request.Fields)
	for i := range request.Fields {
		r := request.Fields[i].CollectionResolution
		if r == nil || r.Source.AssessmentID == "" {
			continue
		}
		if reader == nil {
			r.Source.ArtifactStatus = ArtifactQuarantined
			continue
		}
		review, expiry, err := reader.ReadCollectionSourceReview(ctx, request.TenantID, request.LegalEntityID, r.Source.AssessmentID, r.Source.RequestID, r.Source.ArtifactID)
		if err != nil {
			return Request{}, err
		}
		if review.ID != "" {
			r.Source.Review = &review
		}
		if expiry != "" {
			r.Source.ExpiresOn = expiry
		}
	}
	return request, nil
}
