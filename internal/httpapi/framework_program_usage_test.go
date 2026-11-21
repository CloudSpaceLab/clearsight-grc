package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type usageScopeRepository struct {
	*continuity.MemoryRepository
	tenant, entity string
	fail           bool
}

func (r *usageScopeRepository) FrameworkProgramUsage(_ context.Context, tenant, entity string, _ time.Time) ([]continuity.FrameworkProgramUsage, error) {
	r.tenant, r.entity = tenant, entity
	if r.fail {
		return nil, errors.New("read unavailable")
	}
	return []continuity.FrameworkProgramUsage{{TemplateCode: "ISO_27001_2022", ActiveProgramCount: 2}}, nil
}
func TestFrameworkProgramUsageUsesVerifiedScope(t *testing.T) {
	repo := &usageScopeRepository{MemoryRepository: continuity.NewMemoryRepository()}
	repo.RegisterLegalEntity("signed-bank", "entity-a", "EA")
	api := &API{deps: Dependencies{Continuity: continuity.NewService(repo)}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/framework-catalog/program-usage?tenant_id=forged&legal_entity_id=foreign", nil)
	request = request.WithContext(identity.WithActor(context.Background(), identity.Actor{TenantID: "signed-bank", LegalEntityID: "entity-a", PrincipalID: "reader"}))
	result := httptest.NewRecorder()
	api.listFrameworkProgramUsage(result, request)
	if result.Code != 200 || repo.tenant != "signed-bank" || repo.entity != "entity-a" {
		t.Fatalf("status %d scope %s/%s", result.Code, repo.tenant, repo.entity)
	}
	repo.fail = true
	result = httptest.NewRecorder()
	api.listFrameworkProgramUsage(result, request)
	if result.Code != 503 {
		t.Fatal("source failure became a successful count")
	}
	for _, entity := range []string{"", "*"} {
		scoped := request.WithContext(identity.WithActor(context.Background(), identity.Actor{TenantID: "signed-bank", LegalEntityID: entity, PrincipalID: "reader"}))
		result = httptest.NewRecorder()
		api.listFrameworkProgramUsage(result, scoped)
		if result.Code != 403 {
			t.Fatalf("unscoped request status %d", result.Code)
		}
	}
	result = httptest.NewRecorder()
	api.listFrameworkProgramUsage(result, httptest.NewRequest(http.MethodGet, "/", nil))
	if result.Code != 401 {
		t.Fatal("unsigned request accepted")
	}
}
