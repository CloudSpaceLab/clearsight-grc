package httpapi

import (
	"context"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func TestAssessmentDocumentAvailableRequiresExactCurrentRequestAndAvailableArtifact(t *testing.T) {
	view := thirdparty.AssessmentReviewView{
		Assessment: thirdparty.Assessment{CurrentRequestID: "request-1"},
		Requests:   []thirdparty.AssessmentReviewRequest{{RequestID: "request-1"}},
		Response:   &thirdparty.AssessmentReviewResponse{RequestID: "request-1"},
		Documents:  []thirdparty.AssessmentReviewDocument{{ArtifactID: "artifact-1", ArtifactStatus: evidence.ArtifactAvailable}},
	}
	if !assessmentDocumentAvailable(view, "request-1", "artifact-1") {
		t.Fatal("expected exact artifact to be available")
	}
	if assessmentDocumentAvailable(view, "request-other", "artifact-1") {
		t.Fatal("wrong request must not be available")
	}
	view.Documents[0].ArtifactStatus = evidence.ArtifactQuarantined
	if assessmentDocumentAvailable(view, "request-1", "artifact-1") {
		t.Fatal("quarantined artifact must not be available")
	}
}

func TestAssessmentCollectionDocumentRequiresExactReceipt(t *testing.T) {
	view := thirdparty.AssessmentReviewView{Answers: []thirdparty.AssessmentReviewAnswer{{CollectionResolution: &evidence.CollectionResolution{SourceArtifactRequestID: "original", Source: evidence.DocumentOccurrence{ArtifactID: "report", ArtifactStatus: evidence.ArtifactAvailable}}}}}
	if assessmentCollectionDocument(view, "original", "report") == nil {
		t.Fatal("receipt source unavailable")
	}
	if assessmentCollectionDocument(view, "other", "report") != nil || assessmentCollectionDocument(view, "original", "other") != nil {
		t.Fatal("receipt widened document access")
	}
	view.Answers[0].CollectionResolution.Source.ArtifactStatus = evidence.ArtifactQuarantined
	if assessmentCollectionDocument(view, "original", "report") != nil {
		t.Fatal("quarantined source available")
	}
}

func TestAssessmentDocumentDemoUnscannedGates(t *testing.T) {
	for _, test := range []struct {
		name             string
		status           evidence.ArtifactStatus
		enabled, allowed bool
	}{
		{"default off", evidence.ArtifactStoredUnscanned, false, false}, {"demo enabled", evidence.ArtifactStoredUnscanned, true, true}, {"quarantined", evidence.ArtifactQuarantined, true, false}, {"deleted", evidence.ArtifactDeleted, true, false}, {"scanned", evidence.ArtifactAvailable, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			view := thirdparty.AssessmentReviewView{Assessment: thirdparty.Assessment{CurrentRequestID: "request"}, Response: &thirdparty.AssessmentReviewResponse{RequestID: "request"}, Requests: []thirdparty.AssessmentReviewRequest{{RequestID: "request"}}, Documents: []thirdparty.AssessmentReviewDocument{{ArtifactID: "artifact", ArtifactStatus: test.status, DemoUnscannedAllowed: test.enabled}}, Answers: []thirdparty.AssessmentReviewAnswer{{CollectionResolution: &evidence.CollectionResolution{SourceArtifactRequestID: "source", Source: evidence.DocumentOccurrence{ArtifactID: "artifact", ArtifactStatus: test.status, DemoUnscannedAllowed: test.enabled}}}}}
			if got := assessmentDocumentAvailable(view, "request", "artifact"); got != test.allowed {
				t.Fatalf("direct document allowed=%v", got)
			}
			if got := assessmentCollectionDocument(view, "source", "artifact") != nil; got != test.allowed {
				t.Fatalf("reused document allowed=%v", got)
			}
			if assessmentDocumentAvailable(view, "other", "artifact") || assessmentDocumentAvailable(view, "request", "other") || assessmentCollectionDocument(view, "other", "artifact") != nil || assessmentCollectionDocument(view, "source", "other") != nil {
				t.Fatal("scope widened")
			}
		})
	}
}

type assessmentOpenEvidenceReader struct {
	thirdparty.AssessmentReviewEvidenceReader
	receipt *evidence.CollectionResolution
	status  evidence.ArtifactStatus
}

func (s assessmentOpenEvidenceReader) GetRequest(ctx context.Context, tenant, id string) (evidence.Request, error) {
	request, err := s.AssessmentReviewEvidenceReader.GetRequest(ctx, tenant, id)
	if err == nil && s.receipt != nil {
		request.Fields = append(append([]evidence.Field(nil), request.Fields...), evidence.Field{ID: "held_report", SectionID: request.Fields[0].SectionID, Label: "Held report", Type: "vendor_document", CollectionResolution: s.receipt})
	}
	return request, err
}
func (s assessmentOpenEvidenceReader) GetArtifact(ctx context.Context, tenant, request, id string) (evidence.Artifact, error) {
	artifact, err := s.AssessmentReviewEvidenceReader.GetArtifact(ctx, tenant, request, id)
	if s.status != "" {
		artifact.Status = s.status
	}
	return artifact, err
}

func TestOpenAssessmentDocumentUsesActivePolicyAndFreshCollectionSource(t *testing.T) {
	for _, reused := range []bool{false, true} {
		for _, test := range []struct {
			name                          string
			review, capture, distribution bool
			change                        string
			want                          int
		}{
			{"allowed", true, true, true, "", 200}, {"default off", false, false, false, "", 404}, {"artifact opener revoked", true, false, true, "", 404}, {"quarantined", true, true, true, "QUARANTINED", 404}, {"deleted", true, true, true, "DELETED", 404}, {"wrong request", true, true, true, "request", 404},
			{"fresh access revoked", true, true, true, "denied", 404}, {"distribution policy revoked", true, true, false, "", 404},
		} {
			if !reused && (test.change == "denied" || test.name == "distribution policy revoked") {
				continue
			}
			name := test.name
			if reused {
				name = "reused/" + name
			} else {
				name = "direct/" + name
			}
			t.Run(name, func(t *testing.T) {
				fixture := newReviewHTTPFixture(t, true)
				artifactID := reviewArtifactID(t, fixture)
				requestID := fixture.assessment.CurrentRequestID
				artifact, err := fixture.base.evidence.GetArtifact(context.Background(), "bank", requestID, artifactID)
				if err != nil {
					t.Fatal(err)
				}
				source := evidence.DocumentOccurrence{ID: "source", SubmissionChannel: "MAGIC_LINK", RequestID: requestID, ArtifactRequestID: requestID, ArtifactID: artifactID, SubmissionID: fixture.assessment.SubmissionID, FieldID: "assurance_report", RelationshipID: fixture.assessment.RelationshipID, ArtifactStatus: artifact.Status, SHA256: artifact.SHA256, SizeBytes: artifact.SizeBytes, FileName: artifact.FileName, MediaType: artifact.MediaType, Current: true}
				reader := assessmentOpenEvidenceReader{AssessmentReviewEvidenceReader: fixture.base.evidence}
				if test.change == "QUARANTINED" || test.change == "DELETED" {
					reader.status = evidence.ArtifactStatus(test.change)
					source.ArtifactStatus = reader.status
				}
				if reused {
					reader.receipt = &evidence.CollectionResolution{ID: "receipt", Version: 1, Source: source, SourceArtifactRequestID: requestID, BankReviewState: "PENDING"}
				}
				reviews := thirdparty.NewAssessmentReviewService(fixture.base.service, fixture.base.repository, reader, nil)
				reviews.ConfigureDemoUnscannedArtifacts(test.review)
				fixture.base.evidence.ConfigureDemoUnscannedArtifacts(test.capture)
				documents := &documentContentStore{MemoryDistributionStore: evidence.NewMemoryDistributionStore(evidence.NewMemoryRepository(nil, nil), nil, nil), value: source, denied: test.change == "denied"}
				distributions := evidence.NewDistributionService(documents)
				distributions.ConfigureDemoUnscannedArtifacts(test.distribution)
				request := httptest.NewRequest(http.MethodGet, "/document?demo_unscanned_allowed=true", nil)
				request = request.WithContext(identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "verified-owner", Kind: "PERSON", AuthenticationMethod: "TEST", AssuranceLevel: "HIGH", SessionID: "document-open-test", IssuedAt: time.Now().Add(-time.Minute), ExpiresAt: time.Now().Add(time.Hour)}))
				request.SetPathValue("id", fixture.assessment.ID)
				request.SetPathValue("request_id", requestID)
				request.SetPathValue("artifact_id", artifactID)
				if test.change == "request" {
					request.SetPathValue("request_id", "other")
				}
				response := httptest.NewRecorder()
				(&API{deps: Dependencies{ThirdPartyAssessmentReviews: reviews, Evidence: fixture.base.evidence, FormDistributions: distributions}}).openVendorAssessmentDocument(response, request)
				if response.Code != test.want {
					t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
				}
				if test.want == 200 && (response.Body.Len() != int(artifact.SizeBytes) || response.Header().Get("Cache-Control") != "private, no-store") {
					t.Fatal("document bytes or protection changed")
				}
				if reused && (test.want == 200 || test.change == "denied" || test.name == "distribution policy revoked") {
					if documents.query.PrincipalID != "verified-owner" || documents.query.RelationshipID != source.RelationshipID || documents.query.SubmissionID != source.SubmissionID || documents.query.ArtifactID != artifactID || documents.query.FieldID != source.FieldID {
						t.Fatalf("missing exact fresh authorization query: %+v", documents.query)
					}
				}
			})
		}
	}
}
