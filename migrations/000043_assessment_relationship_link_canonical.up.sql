BEGIN;

ALTER TABLE third_party_assessment_matter_links
    ADD COLUMN relationship_link_id uuid;

INSERT INTO third_party_relationship_matter_links(
    id,tenant_id,legal_entity_id,relationship_id,matter_id,purpose_code,purpose_label,state,
    created_by_principal_id,version,created_at,updated_at
)
SELECT uuidv7(),legacy.tenant_id,legacy.legal_entity_id,legacy.relationship_id,legacy.matter_id,
       CASE legacy.link_kind WHEN 'REVIEW' THEN 'ASSESSMENT_REVIEW' ELSE 'ASSESSMENT_DEFICIENCY' END,
       CASE legacy.link_kind WHEN 'REVIEW' THEN 'Due diligence review' ELSE 'Due diligence finding' END,
       'ACTIVE',legacy.actor_principal_id,1,legacy.created_at,legacy.created_at
FROM (
    SELECT DISTINCT ON (l.tenant_id,l.legal_entity_id,a.relationship_id,l.matter_id)
           l.tenant_id,l.legal_entity_id,a.relationship_id,l.matter_id,l.link_kind,l.created_at,
           COALESCE((
               SELECT e.actor_principal_id
               FROM third_party_events e
               WHERE e.tenant_id=l.tenant_id
                 AND e.aggregate_type='THIRD_PARTY_ASSESSMENT'
                 AND e.aggregate_id=l.assessment_id
                 AND e.actor_principal_id IS NOT NULL
               ORDER BY e.aggregate_version DESC
               LIMIT 1
           ),a.started_by_principal_id) AS actor_principal_id
    FROM third_party_assessment_matter_links l
    JOIN third_party_assessments a
      ON a.id=l.assessment_id AND a.tenant_id=l.tenant_id AND a.legal_entity_id=l.legal_entity_id
    WHERE NOT EXISTS (
        SELECT 1
        FROM third_party_relationship_matter_links existing
        WHERE existing.tenant_id=l.tenant_id
          AND existing.legal_entity_id=l.legal_entity_id
          AND existing.relationship_id=a.relationship_id
          AND existing.matter_id=l.matter_id
          AND existing.state='ACTIVE'
    )
    ORDER BY l.tenant_id,l.legal_entity_id,a.relationship_id,l.matter_id,l.created_at,l.assessment_id
) legacy;

INSERT INTO third_party_relationship_link_events(
    tenant_id,legal_entity_id,link_id,relationship_id,target_type,target_id,link_version,
    actor_principal_id,event_type,payload,occurred_at
)
SELECT link.tenant_id,link.legal_entity_id,link.id,link.relationship_id,'MATTER',link.matter_id,1,
       link.created_by_principal_id,'VendorRelationshipLinked',
       jsonb_build_object('purpose_code',link.purpose_code,'state',link.state,'reason',''),link.created_at
FROM third_party_relationship_matter_links link
WHERE link.state='ACTIVE'
  AND link.purpose_code IN ('ASSESSMENT_REVIEW','ASSESSMENT_DEFICIENCY')
  AND NOT EXISTS (
      SELECT 1 FROM third_party_relationship_link_events event
      WHERE event.tenant_id=link.tenant_id AND event.link_id=link.id AND event.link_version=1
  );

INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
SELECT link.tenant_id,'VENDOR_RELATIONSHIP_LINK',link.id,'VendorRelationshipLinked',
       jsonb_build_object('version',1,'relationship_id',link.relationship_id::text,'target_type','MATTER',
                          'target_id',link.matter_id::text,'state',link.state),
       link.created_at,link.created_at
FROM third_party_relationship_matter_links link
WHERE link.state='ACTIVE'
  AND link.purpose_code IN ('ASSESSMENT_REVIEW','ASSESSMENT_DEFICIENCY')
  AND NOT EXISTS (
      SELECT 1 FROM outbox_events event
      WHERE event.tenant_id=link.tenant_id
        AND event.aggregate_type='VENDOR_RELATIONSHIP_LINK'
        AND event.aggregate_id=link.id
        AND event.event_type='VendorRelationshipLinked'
  );

UPDATE third_party_assessment_matter_links assessment_link
SET relationship_link_id=relationship_link.id
FROM third_party_assessments assessment,
     third_party_relationship_matter_links relationship_link
WHERE assessment.id=assessment_link.assessment_id
  AND assessment.tenant_id=assessment_link.tenant_id
  AND assessment.legal_entity_id=assessment_link.legal_entity_id
  AND relationship_link.tenant_id=assessment_link.tenant_id
  AND relationship_link.legal_entity_id=assessment_link.legal_entity_id
  AND relationship_link.relationship_id=assessment.relationship_id
  AND relationship_link.matter_id=assessment_link.matter_id
  AND relationship_link.state='ACTIVE';

ALTER TABLE third_party_assessment_matter_links
    ALTER COLUMN relationship_link_id SET NOT NULL,
    ADD CONSTRAINT third_party_assessment_matter_relationship_link_fk
        FOREIGN KEY (relationship_link_id,tenant_id,legal_entity_id)
        REFERENCES third_party_relationship_matter_links(id,tenant_id,legal_entity_id);

CREATE INDEX third_party_assessment_matter_relationship_link_idx
    ON third_party_assessment_matter_links(tenant_id,legal_entity_id,relationship_link_id);

COMMIT;
