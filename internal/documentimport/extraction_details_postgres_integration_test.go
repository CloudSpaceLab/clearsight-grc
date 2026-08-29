//go:build postgres && postgresintegration

package documentimport

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresExtractionDetailsRoundTripWithoutLegacyReconstruction(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	const (
		tenantID    = "9d222222-2222-7222-8222-222222222221"
		entityID    = "9d222222-2222-7222-8222-222222222222"
		principalID = "9d222222-2222-7222-8222-222222222223"
		documentID  = "9d222222-2222-7222-8222-222222222224"
	)
	const tenantSlug = "extraction-details-roundtrip-test"
	now := time.Date(2026, 8, 29, 15, 50, 0, 0, time.UTC)

	cleanupExtractionDetailsIntegration(ctx, pool, tenantID)
	defer cleanupExtractionDetailsIntegration(context.Background(), pool, tenantID)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Extraction Details Roundtrip Test')`, tenantID, tenantSlug); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1::uuid,$2::uuid,'EXTRACT-DETAILS','Extraction Details Entity','NG')`, entityID, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($1::uuid,$2::uuid,'PERSON','Document Importer')`, principalID, tenantID); err != nil {
		t.Fatal(err)
	}

	repo := NewPostgresRepository(pool)
	created, err := repo.Create(ctx, Document{
		ID: documentID, TenantID: tenantSlug, LegalEntityID: entityID,
		FileName: "source.docx", MediaType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Purpose: "Preserve structured extraction details", SourceType: "DOCUMENT", SizeBytes: 1,
		SHA256: strings.Repeat("b", 64), StorageKey: "imports/extraction-details-roundtrip-test", ArtifactStatus: "AVAILABLE",
		ExtractionStatus: ExtractionPending, ExtractionMethod: "PENDING", AnalysisStatus: AnalysisPending, AnalysisMethod: "PENDING",
		Limitations: []string{}, Sections: []Section{}, Elements: []ExtractedElement{}, Degradations: []Degradation{}, Proposals: []Proposal{},
		CreatedBy: principalID, CreatedAt: now, UpdatedAt: now, Version: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	processedAt := now.Add(time.Second)
	processed := created
	processed.ExtractionStatus = ExtractionPartial
	processed.ExtractionMethod = "DOCX_XML_STREAM_V3"
	processed.ParserVersion = "DOCX_XML_STREAM_V3"
	processed.AdapterVersion = "DOCX_STRUCTURE_ADAPTER_V1"
	processed.AnalysisStatus = AnalysisUnavailable
	processed.AnalysisMethod = "NONE"
	processed.Elements = []ExtractedElement{{
		Ref: "link-1", Kind: ElementLink, Text: "Policy portal", Target: "https://example.com/policy",
		Anchor: SourceAnchor{Paragraph: "paragraph-4"},
	}}
	processed.Degradations = []Degradation{{
		Code: "DOCX_IMAGES_NOT_EXTRACTED", Message: "An image requires explicit review.", Recoverable: true,
		Anchor: &SourceAnchor{Paragraph: "paragraph-5"},
	}}
	processed.Limitations = []string{"An image requires explicit review."}
	processed.ProcessedAt = &processedAt
	processed.UpdatedAt = processedAt

	saved, err := repo.SaveProcessing(ctx, processed, created.Version)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Version != created.Version+1 {
		t.Fatalf("processing version = %d, want %d", saved.Version, created.Version+1)
	}

	reloaded, err := NewPostgresRepository(pool).Get(ctx, tenantSlug, documentID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Sections) != 0 {
		t.Fatalf("fixture unexpectedly gained legacy sections: %#v", reloaded.Sections)
	}
	if reloaded.ParserVersion != processed.ParserVersion || reloaded.AdapterVersion != processed.AdapterVersion {
		t.Fatalf("parser metadata was not durable: %#v", reloaded)
	}
	if len(reloaded.Elements) != 1 || reloaded.Elements[0].Kind != ElementLink || reloaded.Elements[0].Target != "https://example.com/policy" || reloaded.Elements[0].Anchor.Paragraph != "paragraph-4" {
		t.Fatalf("native element did not round-trip: %#v", reloaded.Elements)
	}
	if len(reloaded.Degradations) != 1 || reloaded.Degradations[0].Code != "DOCX_IMAGES_NOT_EXTRACTED" || reloaded.Degradations[0].Anchor == nil || reloaded.Degradations[0].Anchor.Paragraph != "paragraph-5" {
		t.Fatalf("structured degradation did not round-trip: %#v", reloaded.Degradations)
	}
}

func cleanupExtractionDetailsIntegration(ctx context.Context, pool *pgxpool.Pool, tenantID string) {
	_, _ = pool.Exec(ctx, `DELETE FROM outbox_events WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM document_imports WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
}
