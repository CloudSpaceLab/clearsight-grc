//go:build postgres && postgresintegration

package thirdparty

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestPostgresDeficiencyLinkRejectsDifferentEntityWithoutWrites(t *testing.T) {
	pool := assessmentPostgresPool(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 26, 13, 0, 0, 0, time.UTC)
	relationship := seedAssessmentRelationship(t, pool, "Finding scope")
	repo := NewPostgresRepository(pool)
	record := postgresAssessmentRecord(assessmentOneID, relationship, now)
	assessment, err := repo.CreateAssessment(ctx, record)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `UPDATE third_party_assessments SET status='UNDER_REVIEW' WHERE id=$1::uuid`, assessment.ID)
	if err != nil {
		t.Fatal(err)
	}
	const matterID = "33333333-3333-7333-8333-333333333349"
	const trigger = "vendor-deficiency:scope-test"
	_, err = pool.Exec(ctx, `INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,trigger_type,trigger_id,trigger_key) VALUES($1::uuid,$2::uuid,$3::uuid,'TEST-FINDING','VENDOR_DEFICIENCY','TRIAGE',3,'Finding','Details','VENDOR_ASSESSMENT_DEFICIENCY',$4,$5)`, matterID, thirdPartyTenantID, thirdPartyEntityB, assessment.ID, trigger)
	if err != nil {
		t.Fatal(err)
	}
	link := LinkAssessmentDeficiencyRecord{Scope: record.Scope, AssessmentID: assessment.ID, ExpectedVersion: assessment.Version, ActorPrincipalID: thirdPartyPrincipal, MatterID: matterID, MatterTriggerKey: trigger, LinkedAt: now.Add(time.Minute)}
	_, _, err = repo.LinkAssessmentDeficiency(ctx, link)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity link error = %v", err)
	}
	var links, relationships, events, outbox int
	var version int64
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM third_party_assessment_matter_links WHERE assessment_id=$1::uuid),(SELECT count(*) FROM third_party_relationship_matter_links WHERE matter_id=$2::uuid),(SELECT version FROM third_party_assessments WHERE id=$1::uuid),(SELECT count(*) FROM third_party_events WHERE aggregate_id=$1::uuid AND event_type='AssessmentDeficiencyLinked'),(SELECT count(*) FROM outbox_events WHERE aggregate_id=$1::uuid AND event_type='AssessmentDeficiencyLinked')`, assessment.ID, matterID).Scan(&links, &relationships, &version, &events, &outbox); err != nil {
		t.Fatal(err)
	}
	if links != 0 || relationships != 0 || version != assessment.Version || events != 0 || outbox != 0 {
		t.Fatalf("cross-entity link wrote state: links=%d relationships=%d version=%d events=%d outbox=%d", links, relationships, version, events, outbox)
	}
	// A corrupted existing link must not bypass the same-entity proof on replay.
	const relationshipLinkID = "33333333-3333-7333-8333-333333333351"
	_, err = pool.Exec(ctx, `
INSERT INTO third_party_relationship_matter_links(id,tenant_id,legal_entity_id,relationship_id,matter_id,purpose_code,purpose_label,state,created_by_principal_id,version,created_at,updated_at)
VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,'VENDOR_DEFICIENCY','Vendor finding','ACTIVE',$6::uuid,1,$7,$7);
INSERT INTO third_party_assessment_matter_links(tenant_id,legal_entity_id,assessment_id,matter_id,relationship_link_id,link_kind,created_at)
VALUES($2::uuid,$3::uuid,$8::uuid,$5::uuid,$1::uuid,'DEFICIENCY',$7);
UPDATE third_party_assessments SET version=version+1 WHERE id=$8::uuid;
INSERT INTO third_party_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,actor_principal_id,event_type,payload,occurred_at)
VALUES($2::uuid,'THIRD_PARTY_ASSESSMENT',$8::uuid,$9,$6::uuid,'AssessmentDeficiencyLinked',jsonb_build_object('deficiency_matter_id',$5::text),$7)`, pgx.QueryExecModeSimpleProtocol, relationshipLinkID, thirdPartyTenantID, thirdPartyEntityA, relationship.Relationship.ID, matterID, thirdPartyPrincipal, now.Add(time.Minute), assessment.ID, assessment.Version+1)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = repo.LinkAssessmentDeficiency(ctx, link)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity replay error = %v", err)
	}
	_, err = pool.Exec(ctx, `DELETE FROM third_party_assessment_matter_links WHERE assessment_id=$1::uuid AND matter_id=$2::uuid; DELETE FROM third_party_relationship_matter_links WHERE id=$3::uuid; DELETE FROM third_party_events WHERE aggregate_id=$1::uuid AND event_type='AssessmentDeficiencyLinked'; UPDATE third_party_assessments SET version=$4 WHERE id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, assessment.ID, matterID, relationshipLinkID, assessment.Version)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `UPDATE matters SET status='CANCELLED' WHERE id=$1::uuid`, matterID)
	if err != nil {
		t.Fatal(err)
	}
	const allowedMatterID = "33333333-3333-7333-8333-333333333350"
	_, err = pool.Exec(ctx, `INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,trigger_type,trigger_id,trigger_key) VALUES($1::uuid,$2::uuid,$3::uuid,'TEST-FINDING-ALLOWED','VENDOR_DEFICIENCY','TRIAGE',3,'Finding','Details','VENDOR_ASSESSMENT_DEFICIENCY',$4,$5)`, allowedMatterID, thirdPartyTenantID, thirdPartyEntityA, assessment.ID, trigger)
	if err != nil {
		t.Fatal(err)
	}
	link.MatterID = allowedMatterID
	result, current, err := repo.LinkAssessmentDeficiency(ctx, link)
	if err != nil || result.MatterID != allowedMatterID || current.Version != assessment.Version+1 {
		t.Fatalf("same-entity link failed: %+v %+v %v", result, current, err)
	}
	replayed, replayAssessment, err := repo.LinkAssessmentDeficiency(ctx, link)
	if err != nil || replayed.RelationshipLinkID != result.RelationshipLinkID || replayAssessment.Version != current.Version {
		t.Fatalf("same-entity replay failed: %+v %+v %v", replayed, replayAssessment, err)
	}
}
