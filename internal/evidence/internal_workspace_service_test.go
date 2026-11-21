package evidence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestInternalWorkspaceServiceRejectsUnverifiedIdentity(t *testing.T) {
	service := &DistributionAccessService{store: NewMemoryDistributionAccessStore(nil), now: time.Now}
	if _, _, err := service.GetInternalWorkspace(context.Background(), "request"); !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("read: %v", err)
	}
	if _, err := service.SaveInternalWorkspace(context.Background(), "request", SaveWorkspaceInput{ExpectedVersion: 1}); !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("save: %v", err)
	}
	if _, err := service.SubmitInternalWorkspace(context.Background(), "request", SubmitWorkspaceInput{ExpectedVersion: 1}); !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("submit: %v", err)
	}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "person", Kind: "SERVICE", ExpiresAt: time.Now().Add(time.Hour)}
	if _, _, err := service.GetInternalWorkspace(identity.WithActor(context.Background(), actor), "request"); !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("service identity: %v", err)
	}
}
