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

type paginatedFrameworkReader struct{ testFrameworkEditionReader }

func (reader paginatedFrameworkReader) ListClauses(_ context.Context, editionID, after string, limit int) ([]frameworkcatalog.Clause, error) {
	if editionID != "edition-1" || after != "" || limit != 2 {
		return nil, frameworkcatalog.ErrInvalidPublication
	}
	return []frameworkcatalog.Clause{{ID: "private-a", Code: "A", Title: "Access", SourceAnchor: "Section 1"}, {ID: "private-b", Code: "B", Title: "Review", SourceAnchor: "Section 2"}}, nil
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

func TestFrameworkClauseCatalogueRequiresScopedIdentityAndValidFilters(t *testing.T) {
	items := []frameworkcatalog.Edition{{ID: "edition-1", PublisherCode: "EXAMPLE", FrameworkCode: "SECURITY", EditionCode: "2026"}}
	for _, tt := range []struct {
		name, entity, path string
		status             int
	}{
		{"scoped", "bank-ng", "/api/v1/framework-catalog/editions/edition-1/clauses?limit=100", http.StatusOK},
		{"unscoped", "*", "/api/v1/framework-catalog/editions/edition-1/clauses?limit=100", http.StatusForbidden},
		{"invalid limit", "bank-ng", "/api/v1/framework-catalog/editions/edition-1/clauses?limit=101", http.StatusBadRequest},
		{"invalid cursor", "bank-ng", "/api/v1/framework-catalog/editions/edition-1/clauses?after_code=%20A", http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "owner", tt.entity), FrameworkCatalog: testFrameworkEditionReader{items: items}})
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if rec.Code != tt.status {
				t.Fatalf("status=%d want %d body=%s", rec.Code, tt.status, rec.Body.String())
			}
			if tt.status == http.StatusOK {
				var got struct {
					EditionID string                    `json:"edition_id"`
					Items     []frameworkcatalog.Clause `json:"items"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				if got.EditionID != "edition-1" || len(got.Items) != 1 || got.Items[0].Code != "A" {
					t.Fatalf("unexpected clauses: %+v", got)
				}
			}
		})
	}
}

func TestFrameworkClauseCatalogueUsesBoundedCursorAndOnlyPublishedMetadata(t *testing.T) {
	reader := paginatedFrameworkReader{testFrameworkEditionReader{items: []frameworkcatalog.Edition{{ID: "edition-1"}}}}
	h := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "owner", "bank-ng"), FrameworkCatalog: reader})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/framework-catalog/editions/edition-1/clauses?limit=2", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		Items []map[string]any `json:"items"`
		Next  string           `json:"next_after_code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 2 || got.Next != "B" {
		t.Fatalf("unexpected page: %+v", got)
	}
	if _, ok := got.Items[0]["id"]; ok {
		t.Fatal("database identity exposed")
	}
}
