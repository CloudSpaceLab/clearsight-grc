package people

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type repositoryStub struct {
	profile Profile
	err     error
	got     Scope
}

func (r *repositoryStub) Profile(_ context.Context, scope Scope) (Profile, error) {
	r.got = scope
	if r.err != nil {
		return Profile{}, r.err
	}
	return r.profile, nil
}

func (r *repositoryStub) Work(context.Context, PageQuery) (WorkPage, error) { return WorkPage{}, nil }
func (r *repositoryStub) Assignments(context.Context, PageQuery) (AssignmentPage, error) {
	return AssignmentPage{}, nil
}
func (r *repositoryStub) Activity(context.Context, PageQuery) (ActivityPage, error) {
	return ActivityPage{}, nil
}

func TestProfileAllowsSelfAndPreservesVerifiedViewer(t *testing.T) {
	repo := &repositoryStub{profile: Profile{Person: Person{ID: "hakeem", DisplayName: "Hakeem", Status: "ACTIVE"}}}
	service := NewService(repo)
	viewer := actor("hakeem")

	profile, err := service.Profile(context.Background(), Scope{Viewer: viewer, PersonID: "hakeem"})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Person.ID != "hakeem" || repo.got.Viewer.PrincipalID != "hakeem" || repo.got.PersonID != "hakeem" {
		t.Fatalf("profile request lost identity context: profile=%#v scope=%#v", profile, repo.got)
	}
}

func TestProfileAllowsOversightReader(t *testing.T) {
	repo := &repositoryStub{profile: Profile{Person: Person{ID: "hakeem", DisplayName: "Hakeem", Status: "ACTIVE"}}}
	service := NewService(repo)
	viewer := actor("cro")
	viewer.PermissionCodes = []string{identity.PermissionOversightRead}

	if _, err := service.Profile(context.Background(), Scope{Viewer: viewer, PersonID: "hakeem"}); err != nil {
		t.Fatalf("oversight reader should access profile: %v", err)
	}
}

func TestProfileHidesUnauthorizedAndUnknownPeopleTheSameWay(t *testing.T) {
	viewer := actor("peer")
	unauthorized := NewService(&repositoryStub{profile: Profile{Person: Person{ID: "hakeem"}}})
	unknown := NewService(&repositoryStub{err: ErrNotFound})

	if _, err := unauthorized.Profile(context.Background(), Scope{Viewer: viewer, PersonID: "hakeem"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unauthorized profile error=%v, want not found", err)
	}
	if _, err := unknown.Profile(context.Background(), Scope{Viewer: viewer, PersonID: "unknown"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown profile error=%v, want not found", err)
	}
}

func actor(principalID string) identity.Actor {
	return identity.Actor{
		TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: principalID, Kind: "PERSON",
		IssuedAt: time.Now().Add(-time.Minute), ExpiresAt: time.Now().Add(time.Hour),
	}
}
