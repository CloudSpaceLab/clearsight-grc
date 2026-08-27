import { describe, expect, it } from "vitest";
import type { FormTemplate } from "../../formsTypes";
import type { CreateFormTemplateInput } from "../../monitoringTypes";
import { preserveLibraryRevisionMetadata } from "./formRevisionInput";

const contract: CreateFormTemplateInput = {
  code: "VENDOR",
  name: "Vendor review",
  purpose: "Collect current vendor evidence.",
  scoring_mode: "NONE",
  presentation: { default_mode: "CLASSIC", allow_mode_switch: false },
  sections: [{ id: "identity", title: "Identity" }],
  fields: [{ id: "name", section_id: "identity", label: "Registered name", type: "short_text", required: true }],
};

const template: FormTemplate = {
  id: "form-1",
  tenant_id: "bank-1",
  legal_entity_id: "entity-1",
  program_id: "program-1",
  code: "VENDOR",
  name: "Vendor review",
  purpose: "Collect current vendor evidence.",
  owner_principal_id: "owner-1",
  responsible_team: "Third-party risk",
  approved_uses: ["VENDOR_DUE_DILIGENCE"],
  tags: ["critical-vendor"],
  jurisdiction: "NG",
  industry: "BANKING",
  sensitivity: "CONFIDENTIAL",
  scoring_mode: "NONE",
  next_review_at: "2027-08-27T00:00:00Z",
  presentation: { default_mode: "CLASSIC", allow_mode_switch: false },
  sections: [{ id: "identity", title: "Identity" }],
  fields: contract.fields,
  status: "DRAFT",
  is_current: false,
  version: 4,
  created_at: "2026-08-27T00:00:00Z",
  updated_at: "2026-08-27T00:00:00Z",
};

describe("preserveLibraryRevisionMetadata", () => {
  it("keeps non-builder library metadata on immutable revisions", () => {
    const next = preserveLibraryRevisionMetadata(template, contract);
    expect(next).toMatchObject({
      program_id: "program-1",
      owner_principal_id: "owner-1",
      responsible_team: "Third-party risk",
      approved_uses: ["VENDOR_DUE_DILIGENCE"],
      tags: ["critical-vendor"],
      jurisdiction: "NG",
      industry: "BANKING",
      sensitivity: "CONFIDENTIAL",
      next_review_at: "2027-08-27T00:00:00Z",
    });
    expect(next.fields).toEqual(contract.fields);
    expect(next.approved_uses).not.toBe(template.approved_uses);
    expect(next.tags).not.toBe(template.tags);
  });
});
