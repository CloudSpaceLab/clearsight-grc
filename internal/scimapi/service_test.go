package scimapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeRepository struct {
	source        Source
	tokenHash     [32]byte
	user          User
	group         Group
	replacedUser  User
	replacedGroup Group
}

func (f *fakeRepository) AuthenticateSource(_ context.Context, digest []byte) (Source, error) {
	if !bytes.Equal(digest, f.tokenHash[:]) {
		return Source{}, ErrNotFound
	}
	return f.source, nil
}
func (f *fakeRepository) CreateUser(_ context.Context, _ Source, user User) (User, error) {
	user.ID, user.PrincipalID = "11111111-1111-7111-8111-111111111111", "22222222-2222-7222-8222-222222222222"
	user.CreatedAt, user.UpdatedAt = time.Unix(1, 0).UTC(), time.Unix(1, 0).UTC()
	f.user = user
	return user, nil
}
func (f *fakeRepository) GetUser(context.Context, Source, string) (User, error) { return f.user, nil }
func (f *fakeRepository) ListUsers(context.Context, Source, UserFilter, int, int) ([]User, int, error) {
	return []User{f.user}, 1, nil
}
func (f *fakeRepository) ReplaceUser(_ context.Context, _ Source, _ string, user User) (User, error) {
	user.ID, user.PrincipalID, user.CreatedAt, user.UpdatedAt = f.user.ID, f.user.PrincipalID, f.user.CreatedAt, time.Unix(2, 0).UTC()
	f.replacedUser, f.user = user, user
	return user, nil
}
func (f *fakeRepository) DeleteUser(context.Context, Source, string) error { return nil }
func (f *fakeRepository) CreateGroup(_ context.Context, _ Source, group Group) (Group, error) {
	group.ID, group.CreatedAt, group.UpdatedAt = "33333333-3333-7333-8333-333333333333", time.Unix(1, 0).UTC(), time.Unix(1, 0).UTC()
	f.group = group
	return group, nil
}
func (f *fakeRepository) GetGroup(context.Context, Source, string) (Group, error) { return f.group, nil }
func (f *fakeRepository) ListGroups(context.Context, Source, GroupFilter, int, int) ([]Group, int, error) {
	return []Group{f.group}, 1, nil
}
func (f *fakeRepository) ReplaceGroup(_ context.Context, _ Source, _ string, group Group) (Group, error) {
	group.ID, group.CreatedAt, group.UpdatedAt = f.group.ID, f.group.CreatedAt, time.Unix(2, 0).UTC()
	f.replacedGroup, f.group = group, group
	return group, nil
}
func (f *fakeRepository) DeleteGroup(context.Context, Source, string) error { return nil }

func newTestService(t *testing.T) (*Service, *fakeRepository, string) {
	t.Helper()
	token := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	digest := HashToken(token)
	repository := &fakeRepository{
		source:    Source{ID: "aaaaaaaa-aaaa-7aaa-8aaa-aaaaaaaaaaaa", TenantID: "bbbbbbbb-bbbb-7bbb-8bbb-bbbbbbbbbbbb"},
		tokenHash: digest,
	}
	service, err := New(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return service, repository, token
}

func TestSCIMRequiresTenantScopedBearerToken(t *testing.T) {
	service, _, _ := newTestService(t)
	recorder := httptest.NewRecorder()
	service.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/scim/v2/ServiceProviderConfig", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["status"] != "401" {
		t.Fatalf("expected SCIM error payload, got %#v", payload)
	}
}

func TestSCIMDiscoveryAndUserCreate(t *testing.T) {
	service, repository, token := newTestService(t)

	discovery := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/scim/v2/ServiceProviderConfig", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	service.ServeHTTP(discovery, req)
	if discovery.Code != http.StatusOK {
		t.Fatalf("discovery failed: %d %s", discovery.Code, discovery.Body.String())
	}

	body := `{"schemas":["urn:ietf:params:scim:schemas:core:2.0:User"],"externalId":"oidc-sub-1","userName":"alice@example.com","displayName":"Alice","active":true}`
	recorder := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/scim/v2/Users", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/scim+json")
	service.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create failed: %d %s", recorder.Code, recorder.Body.String())
	}
	if repository.user.UserName != "alice@example.com" || repository.user.ExternalID != "oidc-sub-1" || !repository.user.Active {
		t.Fatalf("unexpected user projection: %#v", repository.user)
	}
}

func TestSCIMRejectsNestedGroupMembership(t *testing.T) {
	service, _, token := newTestService(t)
	body := `{"schemas":["urn:ietf:params:scim:schemas:core:2.0:Group"],"displayName":"Risk","members":[{"value":"group-2","type":"Group"}]}`
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/scim+json")
	service.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("nested group must be rejected, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestSCIMGroupPatchRemovesExactMember(t *testing.T) {
	service, repository, token := newTestService(t)
	repository.group = Group{
		ID: "33333333-3333-7333-8333-333333333333", DisplayName: "Risk",
		Members:   []GroupMember{{UserID: "u1"}, {UserID: "u2"}},
		CreatedAt: time.Unix(1, 0).UTC(), UpdatedAt: time.Unix(1, 0).UTC(),
	}
	body := `{"schemas":["urn:ietf:params:scim:api:messages:2.0:PatchOp"],"Operations":[{"op":"remove","path":"members[value eq \"u1\"]"}]}`
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Groups/33333333-3333-7333-8333-333333333333", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/scim+json")
	service.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("patch failed: %d %s", recorder.Code, recorder.Body.String())
	}
	if len(repository.replacedGroup.Members) != 1 || repository.replacedGroup.Members[0].UserID != "u2" {
		t.Fatalf("unexpected patched membership: %#v", repository.replacedGroup.Members)
	}
}

func TestNormalizeUserRequiresConfiguredOIDCSubject(t *testing.T) {
	_, err := normalizeUser(Source{IdentityIssuer: "https://issuer.example", SubjectAttribute: "externalId"}, User{UserName: "alice@example.com"})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected missing externalId to fail, got %v", err)
	}
}
