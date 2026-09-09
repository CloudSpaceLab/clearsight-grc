package registermigration

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

type sourceStub struct{ document documentimport.Document }

func (s *sourceStub) GetVisible(_ context.Context, tenant, entity, id string) (documentimport.Document, error) {
	if tenant != s.document.TenantID || entity != s.document.LegalEntityID || id != s.document.ID {
		return documentimport.Document{}, documentimport.ErrNotFound
	}
	return s.document, nil
}

type vendorReader struct {
	repo *thirdparty.MemoryAssessmentRepository
}

func (v vendorReader) GetRelationship(ctx context.Context, a thirdparty.Actor, id string) (thirdparty.Aggregate, error) {
	return v.repo.GetRelationship(ctx, thirdparty.Scope{TenantID: a.TenantID, LegalEntityID: a.LegalEntityID}, id)
}

type routeStub struct {
	authority.Service
	denied string
}

func (r *routeStub) Resolve(_ context.Context, in authority.ResolveInput) (authority.Resolution, error) {
	if in.DecisionType == r.denied {
		return authority.Resolution{}, authority.ErrNoRoute
	}
	return authority.Resolution{Principal: authority.Principal{ID: "actor", Kind: "PERSON", DisplayName: "Importer"}, CandidatePrincipals: []authority.Principal{{ID: "owner", Kind: "PERSON", DisplayName: "Alex Ade"}}}, nil
}

type fixture struct {
	ctx     context.Context
	service *Service
	source  *sourceStub
	routes  *routeStub
	repo    *MemoryRepository
	matters *continuity.MemoryRepository
	vendors *thirdparty.MemoryAssessmentRepository
}

func setup(t *testing.T) fixture {
	t.Helper()
	actor := identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "actor", ExpiresAt: time.Now().Add(time.Hour)}
	ctx := identity.WithActor(context.Background(), actor)
	d := documentimport.Document{ID: "source", TenantID: "tenant", LegalEntityID: "entity", SHA256: strings.Repeat("a", 64), Version: 2, ExtractionStatus: documentimport.ExtractionExtracted}
	rows := [][]string{{"S/N", "SERVICE PROVIDER", "SERVICES OFFERED", "FINDINGS", "RECOMMENDATIONS", "RESPONSIBILITY", "TIMELINE", "STATUS"}, {"1", "Example Ltd", "Payments", "Missing certificate", "Obtain certificate", "Alex ", "March 31st 2026", "Open"}, {"", "", "", "Missing report", "Review report", "", "2026-03-31", "Closed"}, {"2", "Example Ltd", "Hosting", "Missing plan", "Obtain plan", "Vendor", "", "Open"}}
	for i, row := range rows {
		d.Sections = append(d.Sections, documentimport.Section{ID: stableID(strings.Join(row, "|")), Sheet: "Register", RowStart: i + 1, RowEnd: i + 1})
		d.Elements = append(d.Elements, documentimport.ExtractedElement{Kind: documentimport.ElementTable, Values: [][]string{row}, Anchor: documentimport.SourceAnchor{Sheet: "Register", RowStart: i + 1, RowEnd: i + 1}})
	}
	d.SectionsTotal = len(d.Sections)
	source := &sourceStub{d}
	routes := &routeStub{}
	vendors := thirdparty.NewMemoryAssessmentRepository()
	for _, id := range []string{"payments", "hosting"} {
		_, err := vendors.CreateRelationship(ctx, thirdparty.CreateRecord{Vendor: thirdparty.Vendor{ID: "vendor", TenantID: "tenant", Status: thirdparty.VendorActive}, Relationship: thirdparty.Relationship{ID: id, VendorID: "vendor", TenantID: "tenant", LegalEntityID: "entity", ServiceName: id, Status: thirdparty.RelationshipActive, Version: 1}})
		if err != nil {
			t.Fatal(err)
		}
	}
	matters := continuity.NewMemoryRepository()
	repo := NewMemoryRepository(matters, vendors)
	return fixture{ctx, New(repo, source, vendorReader{vendors}, routes), source, routes, repo, matters, vendors}
}
func (f fixture) save(t *testing.T) View {
	t.Helper()
	v, err := f.service.View(f.ctx, "source")
	if err != nil {
		t.Fatal(err)
	}
	for i := range v.Draft.Selection.Groups {
		v.Draft.Selection.Groups[i].RelationshipID = []string{"payments", "hosting"}[i]
		v.Draft.Selection.Groups[i].RelationshipVersion = 1
	}
	for i := range v.Draft.Selection.Owners {
		v.Draft.Selection.Owners[i].PersonID = "owner"
	}
	saved, err := f.service.Save(f.ctx, "source", v.Draft.Version, 2, v.Draft.Selection)
	if err != nil {
		t.Fatal(err)
	}
	return saved
}
func TestImportCreatesDistinctSourceBackedFindingsAndReplaysReceipt(t *testing.T) {
	f := setup(t)
	v := f.save(t)
	if len(v.Draft.Selection.Owners) != 2 {
		t.Fatal("repeated owner not collapsed")
	}
	d, err := f.service.Import(f.ctx, "source", v.Draft.Version)
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != "IMPORTED" || len(d.Receipts) != 3 {
		t.Fatalf("bad receipt %#v", d)
	}
	for _, receipt := range d.Receipts {
		aggregate, err := f.matters.GetMatter(f.ctx, "tenant", receipt.MatterID)
		if err != nil {
			t.Fatal(err)
		}
		if aggregate.Matter.Type != continuity.MatterVendorDeficiency || aggregate.Matter.Status != continuity.MatterInitialReview || len(aggregate.Actions) != 1 || aggregate.Actions[0].Status != continuity.ActionPlanned || aggregate.Actions[0].OwnerPrincipalID != "owner" {
			t.Fatalf("incorrect material state %#v", aggregate)
		}
		if !strings.Contains(string(aggregate.Matter.KnownFacts), `"source_sha256"`) {
			t.Fatal("source facts missing")
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			replay, e := f.service.Import(f.ctx, "source", v.Draft.Version)
			if e != nil || replay.Receipts[0].MatterID != d.Receipts[0].MatterID {
				t.Errorf("replay: %v", e)
			}
		}()
	}
	wg.Wait()
	f.source.document.ID = "same-bytes"
	f.source.document.Version = 7
	replay, err := f.service.View(f.ctx, "same-bytes")
	if err != nil || replay.Draft.Status != "IMPORTED" || len(replay.Draft.Receipts) != 3 {
		t.Fatal("duplicate upload lost receipt", err)
	}
}
func TestImportRejectsStaleSourceAndForeignAssignmentsWithoutFindings(t *testing.T) {
	for _, kind := range []string{"source", "relationship", "owner", "authority", "scope", "no-identity"} {
		t.Run(kind, func(t *testing.T) {
			f := setup(t)
			v := f.save(t)
			switch kind {
			case "source":
				f.source.document.Version++
			case "relationship":
				f.repo.drafts[draftKey("tenant", "entity", f.source.document.SHA256)].Selection.Groups[0].RelationshipVersion++
			case "owner":
				f.repo.drafts[draftKey("tenant", "entity", f.source.document.SHA256)].Selection.Owners[0].PersonID = "foreign"
			case "authority":
				f.routes.denied = CommitCommand
			case "scope":
				f.ctx = identity.WithActor(f.ctx, identity.Actor{TenantID: "tenant", LegalEntityID: "other", PrincipalID: "actor", ExpiresAt: time.Now().Add(time.Hour)})
			case "no-identity":
				f.ctx = context.Background()
			}
			if _, err := f.service.Import(f.ctx, "source", v.Draft.Version); err == nil {
				t.Fatal("unsafe import accepted")
			}
			stored, _ := f.repo.Get(f.ctx, "tenant", "entity", f.source.document.SHA256)
			if stored.Status != "DRAFT" || len(stored.Receipts) != 0 {
				t.Fatal("failed import committed")
			}
		})
	}
}

type revokeRepository struct {
	Repository
	routes *routeStub
}

func (r revokeRepository) Commit(ctx context.Context, c Commit) (Draft, error) {
	r.routes.denied = "thirdparty.relationship.link"
	return r.Repository.Commit(ctx, c)
}
func TestImportRechecksAuthorityAtCommit(t *testing.T) {
	f := setup(t)
	v := f.save(t)
	f.service.repo = revokeRepository{f.repo, f.routes}
	if _, err := f.service.Import(f.ctx, "source", v.Draft.Version); !errors.Is(err, ErrAuthority) {
		t.Fatalf("revoked route: %v", err)
	}
	d, _ := f.repo.Get(f.ctx, "tenant", "entity", f.source.document.SHA256)
	if d.Status != "DRAFT" {
		t.Fatal("revoked import committed")
	}
}
func TestSaveRejectsUnknownSourceRows(t *testing.T) {
	f := setup(t)
	v := f.save(t)
	v.Draft.Selection.Rows[0].RowID = "foreign"
	if _, err := f.service.Save(f.ctx, "source", v.Draft.Version, 2, v.Draft.Selection); !errors.Is(err, ErrInvalid) {
		t.Fatalf("foreign source row accepted: %v", err)
	}
}

func TestImportRequiresActionCreationAuthority(t *testing.T) {
	f := setup(t)
	v := f.save(t)
	f.routes.denied = "matter.action.add"
	if _, err := f.service.Import(f.ctx, "source", v.Draft.Version); err == nil {
		t.Fatal("action creation route bypassed")
	}
}
func TestImportIgnoresUnmatchedExcludedAssessments(t *testing.T) {
	f := setup(t)
	v := f.save(t)
	v.Draft.Selection.Groups[1].RelationshipID = ""
	v.Draft.Selection.Rows[2].Include = false
	v, err := f.service.Save(f.ctx, "source", v.Draft.Version, 2, v.Draft.Selection)
	if err != nil {
		t.Fatal(err)
	}
	d, err := f.service.Import(f.ctx, "source", v.Draft.Version)
	if err != nil || len(d.Receipts) != 2 {
		t.Fatal("excluded assessment blocked valid import", err)
	}
}

func TestImportIgnoresExcludedVersionOfTheSameRelationship(t *testing.T) {
	f := setup(t)
	v := f.save(t)
	v.Draft.Selection.Groups[1].RelationshipID = "payments"
	v.Draft.Selection.Groups[1].RelationshipVersion = 999
	v.Draft.Selection.Rows[2].Include = false
	v, err := f.service.Save(f.ctx, "source", v.Draft.Version, 2, v.Draft.Selection)
	if err != nil {
		t.Fatal(err)
	}
	d, err := f.service.Import(f.ctx, "source", v.Draft.Version)
	if err != nil || len(d.Receipts) != 2 {
		t.Fatal("excluded version blocked import", err)
	}
}

type unavailableAfterSave struct {
	Repository
	source *sourceStub
}

func (r unavailableAfterSave) Save(ctx context.Context, d Draft, version int64) (Draft, error) {
	saved, err := r.Repository.Save(ctx, d, version)
	r.source.document.ID = "unavailable"
	return saved, err
}
func TestSaveReturnsCommittedReceiptWithoutASecondSourceRead(t *testing.T) {
	f := setup(t)
	v := f.save(t)
	f.service.repo = unavailableAfterSave{f.repo, f.source}
	saved, err := f.service.Save(f.ctx, "source", v.Draft.Version, 2, v.Draft.Selection)
	if err != nil || saved.Draft.Version != 2 {
		t.Fatal("committed save reported failure", err)
	}
}

func TestEquivalentResponsibilityNamesShareTheSameChoice(t *testing.T) {
	f := setup(t)
	f.source.document.Elements[2].Values[0][5] = "  aLeX  "
	v, err := f.service.View(f.ctx, "source")
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Draft.Selection.Owners) != 2 || v.Draft.Selection.Rows[0].OwnerName != v.Draft.Selection.Rows[1].OwnerName {
		t.Fatal("equivalent names produced different row choices")
	}
}
