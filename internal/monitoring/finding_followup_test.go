package monitoring

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
)

func TestFindingFollowUpReceiptIdentity(t *testing.T) {
	doc := proposalSourceDocument()
	first := FormTemplateProposal{ID: "first", TenantID: doc.TenantID, LegalEntityID: doc.LegalEntityID, SourceKind: FormProposalSourceDocument, SourceDocumentID: doc.ID, SourceDocumentVersion: doc.Version, SourceSHA256: doc.SHA256, FindingAssessmentID: "assessment-1", Status: FormProposalGenerating, CreatedBy: "maker-a", CreatedAt: testProposalTime(), UpdatedAt: testProposalTime(), Version: 1}
	second := first
	second.ID = "second"
	second.FindingAssessmentID = "assessment-2"
	store := NewMemoryFormProposalStore()
	if _, err := store.Create(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	got, err := store.Create(context.Background(), second)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID == first.ID || sameProposalSource(first, second) {
		t.Fatal("different assessments share a proposal receipt")
	}
	retry, err := store.Create(context.Background(), second)
	if err != nil || retry.ID != got.ID {
		t.Fatalf("retry = %#v, %v", retry, err)
	}
	general := first
	general.ID = "general"
	general.FindingAssessmentID = ""
	got, err = store.Create(context.Background(), general)
	if err != nil || got.ID != general.ID {
		t.Fatalf("general proposal collided: %#v %v", got, err)
	}
}

func findingProposalDocument() documentimport.Document {
	doc := proposalSourceDocument()
	doc.FileName = "sample-findings.xlsx"
	headers := []string{"S/N", "SERVICE PROVIDER", "SERVICES OFFERED", "DATE OF ASSESSMENT", "FINDINGS", "RECOMMENDATIONS", "STATUS", "RESPONSIBILITY"}
	rows := [][]string{headers, {"1", "Sample A", "Payments", "2025-03-01", "No recovery test", "Run a test", "Open", "Service owner"}, {"2", "Sample B", "Hosting", "2025-04-01", "Old certificate", "Supply certificate", "Open", "Security owner"}}
	doc.Elements = nil
	fields := make([]documentimport.TabularField, len(headers))
	for i, h := range headers {
		fields[i] = documentimport.TabularField{Name: h}
	}
	doc.Tabular = &documentimport.TabularMetadata{Format: documentimport.TabularXLSX, Resources: []documentimport.TabularResource{{Name: "Findings", Fields: fields, RowsTotal: len(rows) - 1}}}
	for i, row := range rows {
		text := strings.Join(row, " | ")
		doc.Elements = append(doc.Elements, documentimport.ExtractedElement{Kind: documentimport.ElementTable, Text: text, Values: [][]string{row}, Anchor: documentimport.SourceAnchor{Sheet: "Findings", RowStart: i + 1, RowEnd: i + 1}})
		doc.Sections = append(doc.Sections, documentimport.Section{Sheet: "Findings", RowStart: i + 1, RowEnd: i + 1, Text: text})
	}
	return doc
}

func TestFindingFollowUpRequestSelectsOneAssessmentAndKeepsDefault(t *testing.T) {
	docs := &proposalDocumentStub{document: findingProposalDocument()}
	service := NewFormProposalService(NewMemoryFormProposalStore(), docs, libraryService(t, NewMemoryRepository(), "maker-a"))
	ctx := formActorContext("bank-a", "entity-a", "maker-a")
	general, err := service.RequestFromDocument(ctx, docs.document.ID, RequestDocumentFormProposalInput{ExpectedDocumentVersion: docs.document.Version})
	if err != nil {
		t.Fatal(err)
	}
	if general.Provenance.ProposalVersion != "FORM_TEMPLATE_PROPOSAL_V2" || len(general.FieldChanges) != 2 {
		t.Fatalf("default changed: %#v", general)
	}
	options, err := documentimport.FindingFollowUpAssessments(docs.document)
	if err != nil || len(options) != 2 {
		t.Fatalf("assessment options: %#v %v", options, err)
	}
	var prior string
	for _, option := range options {
		input := RequestDocumentFormProposalInput{ExpectedDocumentVersion: docs.document.Version, FindingAssessmentID: option.ID}
		got, err := service.RequestFromDocument(ctx, docs.document.ID, input)
		if err != nil {
			t.Fatal(err)
		}
		if got.ID == general.ID || got.ID == prior || got.FindingAssessmentID != option.ID || len(got.FieldChanges) != 5 || got.Provenance.ProposalVersion != "FINDING_FOLLOW_UP_V2" {
			t.Fatalf("wrong follow-up receipt: %#v", got)
		}
		retry, err := service.RequestFromDocument(ctx, docs.document.ID, input)
		if err != nil || retry.ID != got.ID {
			t.Fatalf("retry differs: %#v %v", retry, err)
		}
		ids := make([]string, len(got.FieldChanges))
		for i, c := range got.FieldChanges {
			ids[i] = c.ID
		}
		accepted, err := service.Accept(ctx, got.ID, AcceptFormProposalInput{ExpectedVersion: got.Version, ChangeIDs: ids, AssessmentConfirmed: true})
		if err != nil || accepted.Status != FormProposalAccepted {
			t.Fatalf("accept = %#v %v", accepted, err)
		}
		again, err := service.Accept(ctx, got.ID, AcceptFormProposalInput{ExpectedVersion: got.Version, ChangeIDs: ids, AssessmentConfirmed: true})
		if err != nil || again.ResultTemplateID != accepted.ResultTemplateID {
			t.Fatalf("accept retry = %#v %v", again, err)
		}
		prior = got.ID
	}
	_, err = service.RequestFromDocument(ctx, docs.document.ID, RequestDocumentFormProposalInput{ExpectedDocumentVersion: docs.document.Version, FindingAssessmentID: "not-an-assessment"})
	if !errors.Is(err, ErrFormProposalSelection) {
		t.Fatalf("unknown assessment allowed: %v", err)
	}
}

func TestFindingFollowUpDraftCodesDoNotShareUUIDTimestampPrefix(t *testing.T) {
	doc := proposalSourceDocument()
	first := FormTemplateProposal{ID: "01990000-0000-7000-8000-000000000001", FindingAssessmentID: "assessment-1"}
	second := first
	second.ID = "01990000-0000-7000-8000-000000000002"
	if proposalFormInput(FormTemplate{}, &doc, first, first.ProposedContract).Code == proposalFormInput(FormTemplate{}, &doc, second, second.ProposedContract).Code {
		t.Fatal("distinct assessments have the same draft code")
	}
}

func TestFindingFollowUpAcceptanceRechecksSourceScopeAuthorityAndRejection(t *testing.T) {
	for _, kind := range []string{"version", "digest", "scope", "authority", "rejected"} {
		t.Run(kind, func(t *testing.T) {
			docs := &proposalDocumentStub{document: findingProposalDocument()}
			store := NewMemoryFormProposalStore()
			repo := NewMemoryRepository()
			service := NewFormProposalService(store, docs, libraryService(t, repo, "maker-a"))
			ctx := formActorContext("bank-a", "entity-a", "maker-a")
			choices, err := documentimport.FindingFollowUpAssessments(docs.document)
			if err != nil {
				t.Fatal(err)
			}
			proposal, err := service.RequestFromDocument(ctx, docs.document.ID, RequestDocumentFormProposalInput{ExpectedDocumentVersion: docs.document.Version, FindingAssessmentID: choices[0].ID})
			if err != nil {
				t.Fatal(err)
			}
			ids := make([]string, len(proposal.FieldChanges))
			for i, change := range proposal.FieldChanges {
				ids[i] = change.ID
			}
			expected := ErrFormProposalSourceChanged
			switch kind {
			case "version":
				docs.document.Version++
			case "digest":
				docs.document.SHA256 = strings.Repeat("b", 64)
			case "scope":
				ctx = formActorContext("bank-a", "other-entity", "maker-a")
				expected = ErrNotFound
			case "authority":
				service.forms = libraryService(t, repo, "replacement-owner")
				expected = commandauth.ErrNotAuthorized
			case "rejected":
				proposal, err = service.Reject(ctx, proposal.ID, RejectFormProposalInput{ExpectedVersion: proposal.Version})
				if err != nil {
					t.Fatal(err)
				}
				expected = ErrFormProposalState
			}
			_, err = service.Accept(ctx, proposal.ID, AcceptFormProposalInput{ExpectedVersion: proposal.Version, ChangeIDs: ids, AssessmentConfirmed: true})
			if !errors.Is(err, expected) {
				t.Fatalf("expected %v got %v", expected, err)
			}
			stored, err := store.Get(context.Background(), "bank-a", "entity-a", proposal.ID)
			if err != nil || stored.ResultTemplateID != "" {
				t.Fatalf("denied acceptance recorded draft: %#v %v", stored, err)
			}
		})
	}
}

func TestFindingFollowUpAcceptanceRequiresWholeAssessmentAndConfirmation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		confirmed bool
		selection string
	}{{"not confirmed", false, "all"}, {"partial", true, "partial"}, {"unknown", true, "unknown"}} {
		t.Run(tc.name, func(t *testing.T) {
			docs := &proposalDocumentStub{document: proposalSourceDocument()}
			store := NewMemoryFormProposalStore()
			service := NewFormProposalService(store, docs, libraryService(t, NewMemoryRepository(), "maker-a"))
			ctx := formActorContext("bank-a", "entity-a", "maker-a")
			proposal, err := service.RequestFromDocument(ctx, docs.document.ID, RequestDocumentFormProposalInput{ExpectedDocumentVersion: docs.document.Version})
			if err != nil {
				t.Fatal(err)
			}
			// A persisted specialized receipt must not permit the generic partial-selection path.
			key := formProposalKey(proposal.TenantID, proposal.LegalEntityID, proposal.ID)
			stored := store.values[key]
			stored.FindingAssessmentID = "assessment-1"
			store.values[key] = stored
			ids := []string{proposal.FieldChanges[0].ID, proposal.FieldChanges[1].ID}
			if tc.selection == "partial" {
				ids = ids[:1]
			}
			if tc.selection == "unknown" {
				ids = append(ids, "not-a-field")
			}
			_, err = service.Accept(ctx, proposal.ID, AcceptFormProposalInput{ExpectedVersion: proposal.Version, ChangeIDs: ids, AssessmentConfirmed: tc.confirmed})
			if !errors.Is(err, ErrFormProposalSelection) {
				t.Fatalf("selection should be rejected, got %v", err)
			}
		})
	}
}
