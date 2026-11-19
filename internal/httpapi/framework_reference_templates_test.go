package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/frameworkcatalog"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type testReferenceReader struct{ testFrameworkEditionReader }

func (testReferenceReader) ListReferenceTemplates(_ context.Context, j string) ([]frameworkcatalog.ReferenceTemplate, error) {
	if j == "INVALID VALUE" {
		return nil, frameworkcatalog.ErrInvalidPublication
	}
	return []frameworkcatalog.ReferenceTemplate{{Code: "TEST", Title: "Test", ReviewStatus: "REVIEW_REQUIRED", Areas: []frameworkcatalog.ReferenceArea{{Code: "A", Title: "Review"}}}}, nil
}
func TestReferenceTemplatesRequireScopeAndReturnGuidance(t *testing.T) {
	for _, tc := range []struct {
		name, entity, path string
		status             int
	}{
		{"scope", "entity", "/api/v1/framework-catalog/reference-templates", 200},
		{"unscoped", "*", "/api/v1/framework-catalog/reference-templates", 403},
		{"invalid", "entity", "/api/v1/framework-catalog/reference-templates?jurisdiction=INVALID%20VALUE", 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "owner", tc.entity), FrameworkCatalog: testReferenceReader{}})
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rec.Code != tc.status {
				t.Fatalf("status=%d want %d: %s", rec.Code, tc.status, rec.Body.String())
			}
			if tc.status == 200 && !strings.Contains(rec.Body.String(), "REVIEW_REQUIRED") {
				t.Fatalf("missing review status: %s", rec.Body.String())
			}
		})
	}
}
