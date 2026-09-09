import { requestJSON } from "./http";
import type { DocumentExtractedElement, DocumentSourceAnchor } from "./documentTypes";
import type { VendorRelationshipPage } from "./vendorTypes";

export type RegisterPerson = { id: string; name: string };
export type RegisterGroup = { id: string; vendor: string; service: string; assessment_date: string };
export type RegisterRow = { id: string; group_id: string; finding: string; recommendation: string; responsibility: string; recorded_status: string; suggested_due_date: string; original: Record<string, string>; inherited: string[]; anchor: DocumentSourceAnchor };
export type RegisterSelection = {
  groups: { group_id: string; relationship_id: string; relationship_version: number }[];
  owners: { name: string; person_id: string; vendor_responsible: boolean }[];
  rows: { row_id: string; include: boolean; due_date: string; owner_name: string }[];
};
export type RegisterDraft = { document_id: string; source_version: number; version: number; status: string; selection: RegisterSelection; receipts: { row_id: string; matter_id: string; action_id: string; title: string }[]; updated_at: string };
export type RegisterView = { draft: RegisterDraft; register: { groups: RegisterGroup[]; rows: RegisterRow[] }; owners: RegisterPerson[]; performers: RegisterPerson[]; can_import: boolean; reason?: string };
const base = import.meta.env.VITE_API_BASE_URL ?? "";
const path = (id: string) => `/api/v1/document-imports/${encodeURIComponent(id)}/risk-register`;
export const loadRegisterMigration = (id: string) => requestJSON<RegisterView>(base, path(id));
export const saveRegisterMigration = (draft: RegisterDraft, selection: RegisterSelection) => requestJSON<RegisterView>(base, `${path(draft.document_id)}/save`, { method: "POST", body: JSON.stringify({ expected_version: draft.version, source_version: draft.source_version, selection }) });
export const commitRegisterMigration = (draft: RegisterDraft) => requestJSON<RegisterDraft>(base, `${path(draft.document_id)}/import`, { method: "POST", body: JSON.stringify({ expected_version: draft.version }) });
const normalized = (value: string) => value.trim().toLocaleLowerCase().replace(/\s+/g, " ");
export function suggestPerson(name: string, people: RegisterPerson[]) {
  const key = normalized(name);
  if (["vendor", "service provider", "unassigned", ""].includes(key)) return undefined;
  const exact = people.filter((p) => normalized(p.name) === key);
  if (exact.length) return exact.length === 1 ? exact[0] : undefined;
  const firstName = people.filter((p) => normalized(p.name).split(" ")[0] === key);
  return firstName.length === 1 ? firstName[0] : undefined;
}
export function suggestRelationship(vendor: string, service: string, page: VendorRelationshipPage) {
  if (page.next_cursor) return undefined;
  const matches = page.items.filter((item) => item.vendor.status === "ACTIVE" && item.relationship.status !== "TERMINATED" && [item.vendor.legal_name, item.vendor.trading_name ?? ""].some((name) => normalized(name) === normalized(vendor)) && normalized(item.relationship.service_name) === normalized(service));
  return matches.length === 1 ? matches[0] : undefined;
}
export function isRiskRegister(elements: DocumentExtractedElement[] = []) {
  return elements.some((e) => e.kind === "TABLE" && e.values?.some((row) => {
    const headers = row.map(normalized);
    return ["service provider", "services offered", "findings", "recommendations"].every((h) => headers.includes(h));
  }));
}
