package evidence

import (
	"context"
	"sort"
	"strings"
	"time"
)

func (s *MemoryDistributionStore) ListDocuments(ctx context.Context, q DocumentQuery) (DocumentPage, error) {
	cursor, err := normalizeDocumentQuery(&q)
	if err != nil {
		return DocumentPage{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.repo == nil {
		return DocumentPage{}, ErrNotFound
	}
	s.repo.mu.RLock()
	submissions := make([]Submission, 0, len(s.repo.submissions))
	for _, submission := range s.repo.submissions {
		if submission.TenantID == q.TenantID {
			submissions = append(submissions, submission)
		}
	}
	s.repo.mu.RUnlock()
	revisions := map[string]ResponseRevision{}
	for _, items := range s.responseRevisions {
		for _, revision := range items {
			if revision.TenantID == q.TenantID && revision.LegalEntityID == q.LegalEntityID {
				revisions[revision.SubmissionID] = revision
			}
		}
	}
	values := []DocumentOccurrence{}
	latestFields := map[string]documentFieldSource{}
	for _, submission := range submissions {
		request, err := s.repo.GetRequest(ctx, q.TenantID, submission.RequestID)
		if err != nil || request.LegalEntityID != q.LegalEntityID {
			continue
		}
		revision := revisions[submission.ID]
		d := s.distributions[revision.DistributionID]
		dc := DocumentContext{Current: revision.Current}
		legacy := request.Origin.Type == "THIRD_PARTY_ASSESSMENT" || request.Origin.Type == "THIRD_PARTY_WORK"
		if legacy {
			if s.documentContexts == nil {
				continue
			}
			dc, err = s.documentContexts.ResolveDocumentContext(ctx, q, request, submission)
			if err != nil {
				continue
			}
		} else {
			if revision.ID == "" || d.TenantID != q.TenantID || d.LegalEntityID != q.LegalEntityID {
				continue
			}
			if request.SubjectType != d.SubjectType || request.SubjectID != d.SubjectID || request.FormTemplateID != d.FormTemplateID || request.FormTemplateVersion != d.FormTemplateVersion {
				continue
			}
			switch strings.ToUpper(d.SubjectType) {
			case "PROGRAM", "MATTER", "VENDOR_RELATIONSHIP":
			default:
				continue
			}
			allowed, accessErr := s.completedResponseSubjectVisible(ctx, q.TenantID, q.PrincipalID, d.SubjectType, d.SubjectID)
			if accessErr != nil {
				return DocumentPage{}, accessErr
			}
			if !allowed && !(q.assessmentRead && q.ResponseRevisionID == revision.ID && d.SubjectType == "VENDOR_RELATIONSHIP") {
				continue
			}
			if d.SubjectType == "VENDOR_RELATIONSHIP" {
				dc.RelationshipID = d.SubjectID
			}
		}
		prefix := revision.ID
		if prefix == "" {
			prefix = "legacy"
		}
		for _, field := range request.Fields {
			if legacy {
				key := documentFieldKey(dc.WorkRequestID, dc.AssessmentID, field.ID)
				source := documentFieldSource{submission.ID, request.Origin.Version, revision.Revision, submission.SubmittedAt}
				if current, ok := latestFields[key]; !ok || source.after(current) {
					latestFields[key] = source
				}
			}
			if field.Type != "file" && field.Type != "photo" && field.Type != "vendor_document" {
				continue
			}
			answer := submission.Answers[field.ID]
			ids := answer.ArtifactIDs
			expiry := ""
			if field.Type == "vendor_document" {
				ids = nil
				if answer.Document != nil {
					ids = []string{answer.Document.ArtifactID}
					expiry = answer.Document.ExpiresOn
				}
			}
			seen := map[string]bool{}
			for _, artifactID := range ids {
				if seen[artifactID] {
					continue
				}
				seen[artifactID] = true
				s.repo.mu.RLock()
				artifact, ok := s.repo.artifacts[artifactID]
				s.repo.mu.RUnlock()
				if !ok || artifact.TenantID != q.TenantID {
					continue
				}
				if artifact.RequestID != request.ID && (d.ID == "" || s.requestDistribution[artifact.RequestID] != d.ID) {
					continue
				}
				v := DocumentOccurrence{SubmissionChannel: submission.Channel, ID: prefix + ":" + submission.ID + ":" + field.ID + ":" + artifact.ID, ArtifactID: artifact.ID, RequestID: request.ID, SubmissionID: submission.ID, FieldID: field.ID, ResponseRevisionID: revision.ID, DistributionID: d.ID, RelationshipID: dc.RelationshipID, AssessmentID: dc.AssessmentID, WorkRequestID: dc.WorkRequestID, FormTemplateID: request.FormTemplateID, FormTemplateVersion: request.FormTemplateVersion, FormTitle: request.Title, FieldLabel: field.Label, FileName: artifact.FileName, MediaType: artifact.MediaType, FileKind: DocumentKindForMediaType(artifact.MediaType), SizeBytes: artifact.SizeBytes, SHA256: artifact.SHA256, ArtifactStatus: artifact.Status, UploadedAt: artifact.CreatedAt, UploadedBy: artifact.CreatedBy, SubmittedAt: submission.SubmittedAt, SubmittedBy: submission.SubmittedBy, ExpiresOn: expiry, Current: dc.Current, ArtifactRequestID: artifact.RequestID}
				if review, ok := dc.Reviews[artifact.ID]; ok && artifact.SubmissionID == submission.ID {
					v.Review = &review
					if expiry, ok := dc.Expiries[artifact.ID]; ok {
						v.ExpiresOn = expiry
					}
				}
				values = append(values, v)
			}
		}
	}
	filtered := values[:0]
	for _, value := range values {
		if value.WorkRequestID != "" || value.AssessmentID != "" {
			value.Current = latestFields[documentFieldKey(value.WorkRequestID, value.AssessmentID, value.FieldID)].SubmissionID == value.SubmissionID
		}
		if documentMatches(value, q, cursor) {
			filtered = append(filtered, value)
		}
	}
	values = filtered
	sort.Slice(values, func(i, j int) bool {
		if !values[i].SubmittedAt.Equal(values[j].SubmittedAt) {
			return values[i].SubmittedAt.After(values[j].SubmittedAt)
		}
		return values[i].ID > values[j].ID
	})
	return documentPage(values, q.Limit), nil
}

type documentFieldSource struct {
	SubmissionID       string
	Sequence, Revision int64
	SubmittedAt        time.Time
}

func (s documentFieldSource) after(other documentFieldSource) bool {
	if s.Sequence != other.Sequence {
		return s.Sequence > other.Sequence
	}
	if s.Revision != other.Revision {
		return s.Revision > other.Revision
	}
	if !s.SubmittedAt.Equal(other.SubmittedAt) {
		return s.SubmittedAt.After(other.SubmittedAt)
	}
	return s.SubmissionID > other.SubmissionID
}
func documentFieldKey(work, assessment, field string) string {
	return work + "\x00" + assessment + "\x00" + field
}
func documentMatches(v DocumentOccurrence, q DocumentQuery, c documentCursor) bool {
	return (!q.CurrentOnly || v.Current) && (q.FileKind == "" || v.FileKind == q.FileKind) && (q.Query == "" || strings.Contains(strings.ToLower(v.FileName), strings.ToLower(q.Query))) && (q.FormTemplateID == "" || v.FormTemplateID == q.FormTemplateID) && (q.RelationshipID == "" || v.RelationshipID == q.RelationshipID) && (q.ResponseRevisionID == "" || v.ResponseRevisionID == q.ResponseRevisionID) && (q.SubmissionID == "" || v.SubmissionID == q.SubmissionID) && (q.FieldID == "" || v.FieldID == q.FieldID) && (q.ArtifactID == "" || v.ArtifactID == q.ArtifactID) && (c.ID == "" || v.SubmittedAt.Before(c.SubmittedAt) || v.SubmittedAt.Equal(c.SubmittedAt) && v.ID < c.ID)
}
