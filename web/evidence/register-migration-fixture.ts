import type { RegisterView } from "../src/registerMigrationApi";
import type { VendorRelationshipAggregate } from "../src/vendorTypes";
export const registerFixture: RegisterView = {
  draft: { document_id: "source", source_version: 2, version: 0, status: "DRAFT", updated_at: "2026-09-09T10:00:00Z", receipts: [], selection: {
    groups: [{ group_id: "g1", relationship_id: "", relationship_version: 0 }],
    owners: [{ name: "Hakeem", person_id: "", vendor_responsible: false }, { name: "Vendor", person_id: "", vendor_responsible: true }],
    rows: [{ row_id: "r1", include: true, due_date: "2026-03-31", owner_name: "Hakeem" }, { row_id: "r2", include: true, due_date: "2026-03-31", owner_name: "Vendor" }],
  } },
  register: { groups: [{ id: "g1", vendor: "Example Ltd", service: "Payments", assessment_date: "13 February 2026" }], rows: [
    { id: "r1", group_id: "g1", finding: "Annual assurance certificate was not provided.", recommendation: "Obtain the current certificate and verify its scope.", responsibility: "Hakeem", recorded_status: "Open", suggested_due_date: "2026-03-31", original: { RESPONSIBILITY: "Hakeem ", STATUS: "Open", TIMELINE: "March 31st 2026" }, inherited: [], anchor: { sheet: "Sample register", row_start: 2, row_end: 2 } },
    { id: "r2", group_id: "g1", finding: "The latest penetration test remains outstanding.", recommendation: "Obtain the report and review unresolved findings.", responsibility: "Vendor", recorded_status: "Closed", suggested_due_date: "2026-03-31", original: { RESPONSIBILITY: "Vendor", STATUS: "Closed", TIMELINE: "March 31st 2026" }, inherited: ["Vendor and service"], anchor: { sheet: "Sample register", row_start: 3, row_end: 3 } },
  ] },
  owners: [{ id: "owner", name: "Hakeem Ade" }, { id: "other", name: "Joel Obi" }], performers: [{ id: "owner", name: "Hakeem Ade" }, { id: "other", name: "Joel Obi" }], can_import: true,
};
export const registerVendorFixture: VendorRelationshipAggregate = {
  vendor: { id: "vendor", tenant_id: "sample", legal_name: "Example Ltd", status: "ACTIVE", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z", version: 1 },
  relationship: { id: "relationship", tenant_id: "sample", legal_entity_id: "entity", vendor_id: "vendor", service_name: "Payments", business_owner_principal_id: "other", status: "ACTIVE", criticality: "IMPORTANT", privacy_role: "PROCESSOR", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z", version: 4 },
};
