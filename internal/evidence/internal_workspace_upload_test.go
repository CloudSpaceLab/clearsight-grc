package evidence

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type revokingUploadStore struct {
	*MemoryObjectStore
	revoke func()
	key    string
}

type ambiguousArtifactRepo struct{ *MemoryRepository }

func (r *ambiguousArtifactRepo) CreateArtifact(context.Context, Artifact) (Artifact, error) {
	return Artifact{}, ErrArtifactOutcomeUnknown
}

func (s *revokingUploadStore) Put(ctx context.Context, key string, reader io.Reader, max int64) (ObjectInfo, error) {
	value, err := s.MemoryObjectStore.Put(ctx, key, reader, max)
	if err == nil {
		s.key = key
		s.revoke()
	}
	return value, err
}

func TestInternalWorkspaceUploadRevocationRollsBackObject(t *testing.T) {
	now := time.Now().UTC()
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "person", Kind: "PERSON", IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)}
	request := Request{ID: "request", TenantID: "bank", LegalEntityID: "entity", SubjectType: "PROGRAM", SubjectID: "program", FormTemplateID: "form", FormTemplateVersion: 1, Status: RequestReady, Deadline: now.Add(time.Hour), Recipient: Recipient{Type: RecipientInternalPrincipal, PrincipalID: "person", State: RecipientStateAssigned}, Fields: []Field{{ID: "file", Label: "Document", Type: "file"}}}
	candidate := RecipientCandidate{PrincipalID: "person", TenantID: "bank", LegalEntityIDs: []string{"entity"}, Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"PROGRAM:program": true}}
	repo := NewMemoryRepositoryWithRecipientCandidates(nil, []Request{request}, []RecipientCandidate{candidate})
	distributions := NewMemoryDistributionStore(repo, nil, nil)
	distributions.distributions["distribution"] = FormDistribution{ID: "distribution", TenantID: "bank", LegalEntityID: "entity", SubjectType: "PROGRAM", SubjectID: "program", FormTemplateID: "form", FormTemplateVersion: 1, Status: DistributionOpen, Deadline: now.Add(time.Hour)}
	distributions.workspaces["distribution"] = ResponseWorkspace{ID: "workspace", TenantID: "bank", LegalEntityID: "entity", DistributionID: "distribution", Status: ResponseWorkspaceOpen, Version: 1}
	distributions.recipients["distribution"] = []memoryDistributionRecipient{{safe: DistributionRecipient{ID: "recipient", TenantID: "bank", LegalEntityID: "entity", DistributionID: "distribution", RequestID: "request", Role: RecipientTo, Type: RecipientInternalPrincipal, PrincipalID: "person", State: DistributionRecipientPending}}}
	distributions.requestDistribution["request"] = "distribution"
	objects := &revokingUploadStore{MemoryObjectStore: NewMemoryObjectStore(), revoke: func() {}}
	service := NewService(repo, objects)
	access := (&DistributionAccessService{store: NewMemoryDistributionAccessStore(distributions), now: func() time.Time { return now }}).WithInternalWorkspaceAuthorizer(func(_ context.Context, who identity.Actor, req Request, command string) (string, error) {
		if who.PrincipalID != "person" || req.ID != "request" || command != "forms.response.upload" {
			return "", errors.New("denied")
		}
		return "v1:performer", nil
	})
	ctx := identity.WithActor(context.Background(), actor)
	created, err := service.StoreArtifactForInternalWorkspace(ctx, access, "request", ArtifactInput{FieldID: "file", FileName: "report.txt", MediaType: "text/plain"}, strings.NewReader("sample document"))
	if err != nil || created.CreatedBy != "person" || len(repo.scanJobs) != 1 || len(distributions.events) != 1 || len(distributions.outbox) != 1 || distributions.events[0].Payload["authority_route"] != "v1:performer" {
		t.Fatalf("authorized upload: %+v %v", created, err)
	}
	firstKey := objects.key
	ambiguous := NewService(&ambiguousArtifactRepo{repo}, objects)
	if _, err := ambiguous.StoreArtifactForInternalWorkspace(ctx, access, "request", ArtifactInput{FieldID: "file", FileName: "uncertain.txt", MediaType: "text/plain"}, strings.NewReader("sample document")); !errors.Is(err, ErrArtifactOutcomeUnknown) {
		t.Fatalf("uncertain upload: %v", err)
	}
	if _, err := objects.Open(ctx, objects.key); err != nil {
		t.Fatal("object deleted after uncertain commit")
	}
	if err := objects.Delete(ctx, objects.key); err != nil {
		t.Fatal(err)
	}
	if err := objects.Delete(ctx, firstKey); err != nil {
		t.Fatal(err)
	}
	delete(repo.artifacts, created.ID)
	delete(repo.scanJobs, created.ID)
	distributions.events = nil
	distributions.outbox = nil
	objects.revoke = func() {
		distributions.mu.Lock()
		defer distributions.mu.Unlock()
		distributions.recipients["distribution"][0].safe.State = DistributionRecipientRevoked
	}
	_, err = service.StoreArtifactForInternalWorkspace(ctx, access, "request", ArtifactInput{FieldID: "file", FileName: "report.txt", MediaType: "text/plain"}, strings.NewReader("sample document"))
	if !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("revoked upload: %v", err)
	}
	if objects.key == "" {
		t.Fatal("test did not reach object storage")
	}
	if _, err := objects.Open(ctx, objects.key); err == nil {
		t.Fatal("revoked upload left an object")
	}
	if len(repo.artifacts) != 0 || len(repo.scanJobs) != 0 || len(distributions.events) != 0 || len(distributions.outbox) != 0 {
		t.Fatal("revoked upload left material state")
	}
}
