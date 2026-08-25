import { requestJSON } from "./http";
import type { CreateVendorRelationshipInput, UpdateVendorRelationshipInput, VendorRelationshipAggregate, VendorRelationshipPage } from "./vendorTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export function loadVendorRelationships(query: { search?: string; cursor?: string; limit?: number } = {}): Promise<VendorRelationshipPage> {
  const params = new URLSearchParams();
  if (query.search) params.set("search", query.search);
  if (query.cursor) params.set("cursor", query.cursor);
  params.set("limit", String(query.limit ?? 50));
  return requestJSON<VendorRelationshipPage>(apiBase, `/api/v1/vendors?${params.toString()}`);
}

export function loadVendorRelationship(id: string): Promise<VendorRelationshipAggregate> {
  return requestJSON<VendorRelationshipAggregate>(apiBase, `/api/v1/vendors/${encodeURIComponent(id)}`);
}

export function createVendorRelationship(input: CreateVendorRelationshipInput): Promise<VendorRelationshipAggregate> {
  return requestJSON<VendorRelationshipAggregate>(apiBase, "/api/v1/vendors", { method: "POST", body: JSON.stringify(input) });
}

export function updateVendorRelationship(id: string, input: UpdateVendorRelationshipInput): Promise<VendorRelationshipAggregate> {
  return requestJSON<VendorRelationshipAggregate>(apiBase, `/api/v1/vendors/${encodeURIComponent(id)}`, { method: "POST", body: JSON.stringify(input) });
}
