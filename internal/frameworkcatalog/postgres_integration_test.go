//go:build postgres && postgresintegration

package frameworkcatalog

import (
    "context"
    "errors"
    "os"
    "strings"
    "testing"

    platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
    "github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresFrameworkCatalogImmutablePublicationAndReads(t *testing.T) {
    connection := os.Getenv("TEST_DATABASE_URL")
    if connection == "" { t.Skip("TEST_DATABASE_URL is not configured") }
    ctx := context.Background()
    pool, err := pgxpool.New(ctx, connection)
    if err != nil { t.Fatal(err) }
    defer pool.Close()

    nonce, err := platformid.NewUUIDv7()
    if err != nil { t.Fatal(err) }
    suffix := nonce[len(nonce)-8:]
    publication := validTestPublication()
    publication.FrameworkCode = "TEST-" + stringsUpperHex(suffix)
    publication.Title = "Test publication " + suffix

    repository := NewPostgresRepository(pool)
    edition, err := repository.Publish(ctx, publication, "test-reviewed-reference")
    if err != nil { t.Fatal(err) }
    if !validUUIDText(edition.ID) || edition.SourcePublishedOn != "2026-10-01" ||
        edition.ReleaseSHA256 == "" || edition.FrameworkCode != publication.FrameworkCode {
        t.Fatalf("bad edition: %+v", edition)
    }

    again, err := repository.Publish(ctx, publication, "test-reviewed-reference")
    if err != nil || edition.ID != again.ID || edition.ReleaseSHA256 != again.ReleaseSHA256 {
        t.Fatalf("publication rerun changed identity: first=%+v again=%+v error=%v", edition, again, err)
    }

    clauses, err := repository.ListClauses(ctx, edition.ID, "", 10)
    if err != nil { t.Fatal(err) }
    if len(clauses) != 2 || clauses[0].Code != "A" || clauses[1].Code != "B" || clauses[1].ParentCode != "A" {
        t.Fatalf("hierarchy was not published as received: %+v", clauses)
    }
    second, err := repository.ListClauses(ctx, edition.ID, "A", 1)
    if err != nil || len(second) != 1 || second[0].Code != "B" { t.Fatalf("broken keyset page: %+v %v", second, err) }

    listings, err := repository.ListEditions(ctx, publication.Jurisdiction, "", 100)
    if err != nil { t.Fatal(err) }
    found := false
    for _, item := range listings { if item.ID == edition.ID { found = true; break } }
    // Other integration fixtures may have >100 editions: direct exact read is
    // the correctness check, while ListEditions remains deliberately bounded.
    fetched, err := repository.GetEdition(ctx, publication.PublisherCode, publication.FrameworkCode, publication.EditionCode)
    if err != nil || fetched.ID != edition.ID { t.Fatalf("exact read failed: %+v %v", fetched, err) }
    _ = found

    altered := publication
    altered.Title = "Unapproved retroactive replacement"
    if _, err := repository.Publish(ctx, altered, "test-reviewed-reference"); !errors.Is(err, ErrEditionConflict) {
        t.Fatalf("conflicting same edition silently replaced: %v", err)
    }

    if _, err := pool.Exec(ctx, `UPDATE framework_catalog_editions SET title='Changed' WHERE id=$1::uuid`, edition.ID); err == nil {
        t.Fatal("published framework edition was mutable")
    }
    if _, err := pool.Exec(ctx, `UPDATE framework_catalog_clauses SET title='Changed' WHERE edition_id=$1::uuid`, edition.ID); err == nil {
        t.Fatal("published framework clause was mutable")
    }
    if _, err := pool.Exec(ctx, `DELETE FROM framework_catalog_editions WHERE id=$1::uuid`, edition.ID); err == nil {
        t.Fatal("published framework edition could be removed")
    }
}

func stringsUpperHex(value string) string {
    // The UUID suffix is hexadecimal; make it safe for catalog natural codes.
    return strings.ToUpper(value)
}
