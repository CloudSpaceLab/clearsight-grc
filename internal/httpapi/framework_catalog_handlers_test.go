package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/frameworkcatalog"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type testFrameworkEditionReader struct {
	items []frameworkcatalog.Edition
}

func (reader testFrameworkEditionReader) ListEditions(_ context.Context, jurisdiction, cursor string, limit int) ([]frameworkcatalog.Edition, error) {
	if jurisdiction != "GLOBAL" || cursor != "" || limit != 100 {
		return nil, frameworkcatalog.ErrInvalidPublication
	}
	return reader.items, nil
}

func (reader testFrameworkEditionReader) GetFrameworkEdition(_ context.Context, id string) (continuity.FrameworkEditionReference, error) {
	for _, entry := range reader.items {
		if entry.ID == id {
			return continuity.FrameworkEditionReference{
				ID: entry.ID, PublisherCode: entry.PublisherCode,
				FrameworkCode: entry.FrameworkCode, EditionCode: entry.EditionCode,
				ReleaseSHA256: entry.ReleaseSHA256,
			}, nil
		}
	}
	return continuity.FrameworkEditionReference{}, continuity.ErrNotFound
}

func (reader testFrameworkEditionReader) ListClauses(_ context.Context, editionID, after string, limit int) ([]frameworkcatalog.Clause, error) {
	if editionID != "edition-1" || after != "" || limit != 100 {
		return nil, frameworkcatalog.ErrInvalidPublication
	}
	return []frameworkcatalog.Clause{{Code: "A", SourceAnchor: "Section 1", Title: "Access"}}, nil
}

func TestFrameworkEditionCatalogueReadRequiresExactLegalEntity(t *testing.T) {
	items := []frameworkcatalog.Edition{{
		ID: "edition-1", PublisherCode: "EXAMPLE", FrameworkCode: "SECURITY",
		EditionCode: "2026", Title: "Reference security standard",
		SourceURL: "https://example.org/standard", Jurisdiction: "GLOBAL",
	}}
	for _, test := range []struct {
		name   string
		entity string
		status int
	}{
		{"scoped", "bank-ng", http.StatusOK},
		{"unscoped", "*", http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := New(Dependencies{
				Logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
				Identity:         identity.NewDevelopmentAuthenticator("bank", "owner", test.entity),
				FrameworkCatalog: testFrameworkEditionReader{items: items},
			})
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
				"/api/v1/framework-catalog/editions?jurisdiction=GLOBAL&limit=100", nil))
			if rec.Code != test.status {
				t.Fatalf("read status=%d want %d body=%s", rec.Code, test.status, rec.Body.String())
			}
			if test.status == http.StatusOK {
				var payload struct {
					Items []frameworkcatalog.Edition `json:"items"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
					t.Fatal(err)
				}
				if len(payload.Items) != 1 || payload.Items[0].ID != "edition-1" {
					t.Fatalf("unexpected catalog metadata: %+v", payload.Items)
				}
			}
		})
	}
}
