//go:build postgres && postgresintegration

package thirdparty

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestPostgresVendorWorkListAppliesCurrentVisibilityBeforePagination(t *testing.T) {
	pool := assessmentPostgresPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	const (
		actorID         = "33333333-3333-7333-8333-333333333351"
		routeOwnerID    = "33333333-3333-7333-8333-333333333352"
		otherOwnerID    = "33333333-3333-7333-8333-333333333353"
		reviewerID      = "33333333-3333-7333-8333-333333333354"
		programID       = "33333333-3333-7333-8333-333333333355"
		visibleLinkID   = "33333333-3333-7333-8333-333333333356"
		hiddenLinkID    = "33333333-3333-7333-8333-333333333357"
		visibleWorkID   = "33333333-3333-7333-8333-333333333358"
		hiddenWorkID    = "33333333-3333-7333-8333-333333333359"
		conflictOwnerID = "33333333-3333-7333-8333-333333333360"
		roleID          = "33333333-3333-7333-8333-333333333361"
		positionID      = "33333333-3333-7333-8333-333333333362"
		bindingID       = "33333333-3333-7333-8333-333333333363"
	)
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
		($1::uuid,$5::uuid,'PERSON','Delegated reviewer','ACTIVE',$6),
		($2::uuid,$5::uuid,'PERSON','Review route owner','ACTIVE',$6),
		($3::uuid,$5::uuid,'PERSON','Work owner','ACTIVE',$6),
		($4::uuid,$5::uuid,'PERSON','Recorded reviewer','ACTIVE',$6),
		($7::uuid,$5::uuid,'PERSON','Conflicting route owner','ACTIVE',$6)`, actorID, routeOwnerID, otherOwnerID, reviewerID, thirdPartyTenantID, now.Add(-time.Hour), conflictOwnerID); err != nil {
		t.Fatal(err)
	}
	visibleRelationship := seedAssessmentRelationship(t, pool, "Visible delegated vendor work")
	hiddenRelationship := seedAssessmentRelationship(t, pool, "Hidden newer vendor work")
	if _, err := pool.Exec(ctx, `
		INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,jurisdiction,effective_from)
		VALUES($1::uuid,$2::uuid,$3::uuid,'VENDOR-WORK-LIST','Vendor work list','COMPLIANCE','ACTIVE','Third-party risk','NG',$4);
		INSERT INTO third_party_relationship_program_links(id,tenant_id,legal_entity_id,relationship_id,program_id,purpose_code,purpose_label,state,created_by_principal_id,version,created_at,updated_at) VALUES
		($5::uuid,$2::uuid,$3::uuid,$6::uuid,$1::uuid,'VENDOR_ACTION','Vendor action','ACTIVE',$7::uuid,1,$4,$4),
		($8::uuid,$2::uuid,$3::uuid,$9::uuid,$1::uuid,'VENDOR_ACTION','Vendor action','ACTIVE',$7::uuid,1,$4,$4)`,
		programID, thirdPartyTenantID, thirdPartyEntityA, now.Add(-30*time.Minute), visibleLinkID, visibleRelationship.Relationship.ID, thirdPartyPrincipal, hiddenLinkID, hiddenRelationship.Relationship.ID); err != nil {
		t.Fatal(err)
	}
	repository := NewPostgresRepository(pool)
	base := VendorWorkRequest{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA, TargetType: LinkTargetProgram, TargetID: programID,
		Purpose: "Complete the required vendor action.", Instructions: "Provide the requested record.", OwnerPrincipalID: otherOwnerID,
		FormTemplateID: assessmentTemplateID, FormTemplateVersion: 3, Presentation: formcontract.PresentationWizard,
		State: VendorWorkPreparing, DeliveryState: VendorWorkDeliveryNotSent, DueAt: now.Add(24 * time.Hour), Version: 1}
	visible := base
	visible.ID, visible.RelationshipID, visible.RelationshipLinkID = visibleWorkID, visibleRelationship.Relationship.ID, visibleLinkID
	visible.CreatedAt, visible.UpdatedAt = now.Add(-2*time.Minute), now.Add(-2*time.Minute)
	if _, err := repository.CreateVendorWork(ctx, visible); err != nil {
		t.Fatal(err)
	}
	hidden := base
	hidden.ID, hidden.RelationshipID, hidden.RelationshipLinkID = hiddenWorkID, hiddenRelationship.Relationship.ID, hiddenLinkID
	hidden.CreatedAt, hidden.UpdatedAt = now.Add(-time.Minute), now.Add(-time.Minute)
	if _, err := repository.CreateVendorWork(ctx, hidden); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,priority,valid_from,policy_version,decision_type)
		VALUES($1::uuid,$2::uuid,$3::uuid,'REVIEWER','VENDOR_RELATIONSHIP',$4::uuid,100,$5,'vendor-work-list:v1','thirdparty.work.review');
		INSERT INTO delegations(tenant_id,from_principal_id,to_principal_id,responsibility,scope,starts_at,ends_at,status,created_by,approved_by,approved_at,version)
		VALUES($1::uuid,$3::uuid,$6::uuid,'REVIEWER',jsonb_build_object('legal_entity_id',$2::text,'object_type','VENDOR_RELATIONSHIP','object_id',$4::text,'decision_type','thirdparty.work.review'),$5,$7,'ACTIVE',$3::uuid,$8::uuid,$5,1);
		INSERT INTO authority_grants(tenant_id,legal_entity_id,principal_id,decision_type,limits,valid_from,policy_version)
		VALUES($1::uuid,$2::uuid,$3::uuid,'thirdparty.work.review','{"max_materiality":3}'::jsonb,$5,'vendor-work-list:grant')`,
		thirdPartyTenantID, thirdPartyEntityA, routeOwnerID, visibleRelationship.Relationship.ID, now.Add(-time.Minute), actorID, now.Add(time.Hour), thirdPartyPrincipal); err != nil {
		t.Fatal(err)
	}

	input := VendorWorkListInput{TargetType: LinkTargetProgram, TargetID: programID, Limit: 1}
	page, err := repository.ListVendorWork(vendorWorkListActorContext(now, actorID), Scope{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA}, input)
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != visibleWorkID || page.NextCursor != "" {
		t.Fatalf("delegated page = %#v err=%v", page, err)
	}

	if _, err := pool.Exec(ctx, `UPDATE delegations SET status='EXPIRED' WHERE tenant_id=$1::uuid AND from_principal_id=$2::uuid`, thirdPartyTenantID, routeOwnerID); err != nil {
		t.Fatal(err)
	}
	page, err = repository.ListVendorWork(vendorWorkListActorContext(now, actorID), Scope{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA}, input)
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("expired delegation page = %#v err=%v", page, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE delegations SET status='ACTIVE' WHERE tenant_id=$1::uuid AND from_principal_id=$2::uuid;
		INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,priority,valid_from,policy_version,decision_type)
		VALUES($1::uuid,$3::uuid,$4::uuid,'REVIEWER','VENDOR_RELATIONSHIP',$5::uuid,100,$6,'vendor-work-list:v2','thirdparty.work.review')`, thirdPartyTenantID, routeOwnerID, thirdPartyEntityA, conflictOwnerID, visibleRelationship.Relationship.ID, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	page, err = repository.ListVendorWork(vendorWorkListActorContext(now, actorID), Scope{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA}, input)
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("ambiguous route page = %#v err=%v", page, err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM responsibility_assignments WHERE tenant_id=$1::uuid AND policy_version='vendor-work-list:v2';
		INSERT INTO role_templates(id,tenant_id,code,name,description,responsibilities,valid_from) VALUES($2::uuid,$1::uuid,'VENDOR_WORK_CONFLICT','Vendor work conflict','',ARRAY['REVIEWER'],$3);
		INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id,valid_from) VALUES($4::uuid,$1::uuid,$5::uuid,'VENDOR-WORK-CONFLICT','Vendor work conflict',$6::uuid,$3);
		INSERT INTO position_role_bindings(id,tenant_id,position_id,role_template_id,priority,valid_from) VALUES($7::uuid,$1::uuid,$4::uuid,$2::uuid,100,$3);
		INSERT INTO segregation_rules(tenant_id,code,responsibility,prohibited_role_code,status,valid_from) VALUES($1::uuid,'NO-VENDOR-WORK-CONFLICT','REVIEWER','VENDOR_WORK_CONFLICT','ACTIVE',$3)`,
		thirdPartyTenantID, roleID, now.Add(-time.Minute), positionID, thirdPartyEntityA, actorID, bindingID); err != nil {
		t.Fatal(err)
	}
	page, err = repository.ListVendorWork(vendorWorkListActorContext(now, actorID), Scope{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA}, input)
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("segregated reviewer page = %#v err=%v", page, err)
	}

	if _, err := pool.Exec(ctx, `UPDATE third_party_work_requests SET reviewer_principal_id=$1::uuid WHERE id=$2::uuid`, reviewerID, hiddenWorkID); err != nil {
		t.Fatal(err)
	}
	for _, direct := range []string{otherOwnerID, reviewerID, thirdPartyPrincipal} {
		page, err = repository.ListVendorWork(vendorWorkListActorContext(now, direct), Scope{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA}, input)
		if err != nil || len(page.Items) != 1 || page.Items[0].ID != hiddenWorkID {
			t.Fatalf("direct principal %s page = %#v err=%v", direct, page, err)
		}
	}

	if _, err := repository.ListVendorWork(context.Background(), Scope{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA}, input); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing verified identity returned %v", err)
	}
}

func vendorWorkListActorContext(now time.Time, principalID string) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA, PrincipalID: principalID, Kind: "PERSON", IssuedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour)})
}
