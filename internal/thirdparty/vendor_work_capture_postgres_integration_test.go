//go:build postgres && postgresintegration

package thirdparty

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/jackc/pgx/v5"
)

func TestPostgresAttachVendorWorkCapturePersistsIntegerSequenceAndBigintOriginVersion(t *testing.T) {
	pool := assessmentPostgresPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	const (
		programID = "33333333-3333-7333-8333-333333333371"
		linkID    = "33333333-3333-7333-8333-333333333372"
		workID    = "33333333-3333-7333-8333-333333333373"
		captureID = "33333333-3333-7333-8333-333333333374"
	)
	relationship := seedAssessmentRelationship(t, pool, "Address verification attachment")
	if _, err := pool.Exec(ctx, `
		INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,jurisdiction,effective_from)
		VALUES($1::uuid,$2::uuid,$3::uuid,'VENDOR-CAPTURE','Vendor capture','COMPLIANCE','ACTIVE','Third-party risk','NG',$4);
		INSERT INTO third_party_relationship_program_links(id,tenant_id,legal_entity_id,relationship_id,program_id,purpose_code,purpose_label,state,created_by_principal_id,version,created_at,updated_at)
		VALUES($5::uuid,$2::uuid,$3::uuid,$6::uuid,$1::uuid,'ADDRESS_VERIFICATION','Address verification','ACTIVE',$7::uuid,1,$4,$4)`,
		pgx.QueryExecModeSimpleProtocol, programID, thirdPartyTenantID, thirdPartyEntityA, now, linkID, relationship.Relationship.ID, thirdPartyPrincipal); err != nil {
		t.Fatal(err)
	}

	repository := NewPostgresRepository(pool)
	created, err := repository.CreateVendorWork(ctx, VendorWorkRequest{
		ID: workID, TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA,
		RelationshipID: relationship.Relationship.ID, RelationshipLinkID: linkID,
		TargetType: LinkTargetProgram, TargetID: programID, RequestKind: VendorWorkAddressVerification,
		Purpose: "Verify the vendor address.", Instructions: "Provide confirmation and supporting evidence.",
		OwnerPrincipalID: thirdPartyPrincipal, FormTemplateID: assessmentTemplateID, FormTemplateVersion: 3,
		Presentation: formcontract.PresentationWizard, State: VendorWorkPreparing, DeliveryState: VendorWorkDeliveryNotSent,
		DueAt: now.Add(24 * time.Hour), Version: 1, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	origin := evidence.RequestOrigin{Type: VendorWorkOrigin, ID: created.ID, Version: 1}
	evidenceContext := evidence.WithRequestOriginAuthority(ctx, origin.Type)
	evidenceService := evidence.NewService(evidence.NewPostgresRepository(pool), evidence.NewMemoryObjectStore())
	request, err := evidenceService.CreateRequest(evidenceContext, evidence.CreateRequestInput{
		TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA,
		SubjectType: "VENDOR_RELATIONSHIP", SubjectID: relationship.Relationship.ID,
		Title: "Confirm the vendor address", Purpose: "Record the address verification result.",
		WhyYou:      "You are responsible for confirming the address and providing evidence.",
		Sensitivity: "CONFIDENTIAL", AudienceType: "VENDOR",
		Recipient:        evidence.RecipientInput{Type: evidence.RecipientExternalAudience, Audience: "review@vendor.example"},
		EstimatedMinutes: 10, Deadline: now.Add(24 * time.Hour), Origin: origin,
		Presentation:   formcontract.Presentation{DefaultMode: formcontract.PresentationWizard},
		Sections:       []formcontract.Section{{ID: "address", Title: "Address verification"}},
		Fields:         []evidence.Field{{ID: "confirmed", SectionID: "address", Label: "Was the address verified?", Type: string(formcontract.TypeYesNo), Required: true}},
		FormTemplateID: assessmentTemplateID, FormTemplateVersion: 3, CreatedBy: thirdPartyPrincipal,
	})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := repository.AttachVendorWorkCapture(ctx, Scope{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA}, created.ID, created.Version, VendorWorkCaptureLink{
		ID: captureID, TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA,
		WorkRequestID: created.ID, RequestID: request.ID, Sequence: 1, Purpose: "INITIAL", OriginVersion: 1, CreatedAt: now,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if updated.CurrentRequestID != request.ID || updated.CurrentCaptureSequence != 1 || updated.Version != 2 {
		t.Fatalf("updated work = %#v", updated)
	}
	captures, err := repository.ListVendorWorkCaptures(ctx, Scope{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA}, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(captures) != 1 || captures[0].Sequence != 1 || captures[0].OriginVersion != 1 || captures[0].RequestID != request.ID {
		t.Fatalf("captures = %#v", captures)
	}
}
