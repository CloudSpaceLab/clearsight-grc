import { parseJSON, requestJSON } from "./http";
import type { CommittedCommandReceipt, CreateVendorRelationshipInput, UpdateVendorIdentityInput, UpdateVendorRelationshipInput, VendorIdentityMutationOutcome, VendorIdentityPresentation, VendorRelationshipAggregate, VendorRelationshipPage } from "./vendorTypes";

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

export function loadVendorIdentity(vendorID: string): Promise<VendorIdentityPresentation> {
  return requestJSON<VendorIdentityPresentation>(apiBase, `/api/v1/vendor-identities/${encodeURIComponent(vendorID)}`);
}

export function updateVendorIdentity(vendorID: string, input: UpdateVendorIdentityInput): Promise<VendorIdentityMutationOutcome> {
  return requestJSON<VendorIdentityMutationOutcome>(apiBase, `/api/v1/vendor-identities/${encodeURIComponent(vendorID)}`, { method: "PUT", body: JSON.stringify(input) });
}

export async function uploadApprovedVendorLogo(vendorID: string, file: File, expectedVersion: number, idempotencyKey = newVendorBrandIdempotencyKey()): Promise<VendorIdentityMutationOutcome> {
  return parseJSON<VendorIdentityMutationOutcome>(await fetch(`${apiBase}/api/v1/vendor-identities/${encodeURIComponent(vendorID)}/brand`, {
    method: "PUT",
    body: file,
    credentials: "include",
    headers: { "Content-Type": file.type || logoMediaTypeFromName(file.name), "If-Match": `"${expectedVersion}"`, "Idempotency-Key": idempotencyKey },
  }));
}

export async function removeApprovedVendorLogo(vendorID: string, expectedVersion: number, idempotencyKey = newVendorBrandIdempotencyKey()): Promise<VendorIdentityMutationOutcome> {
  return parseJSON<VendorIdentityMutationOutcome>(await fetch(`${apiBase}/api/v1/vendor-identities/${encodeURIComponent(vendorID)}/brand`, {
    method: "DELETE",
    credentials: "include",
    headers: { "If-Match": `"${expectedVersion}"`, "Idempotency-Key": idempotencyKey },
  }));
}

export function isCommittedCommandReceipt(value: unknown): value is CommittedCommandReceipt {
  if (!value || typeof value !== "object") return false;
  const receipt = value as Partial<CommittedCommandReceipt>;
  return receipt.status === "COMMITTED" && receipt.response_degraded === true && typeof receipt.aggregate_type === "string" && typeof receipt.aggregate_id === "string" && typeof receipt.version === "number";
}

export function newVendorBrandIdempotencyKey() {
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  return `vendor-brand-${[...bytes].map((value) => value.toString(16).padStart(2, "0")).join("")}`;
}

function logoMediaTypeFromName(name: string) {
  if (/\.png$/i.test(name)) return "image/png";
  if (/\.jpe?g$/i.test(name)) return "image/jpeg";
  if (/\.webp$/i.test(name)) return "image/webp";
  if (/\.ico$/i.test(name)) return "image/x-icon";
  return "application/octet-stream";
}

export function vendorBrandURL(vendorID: string, assetToken?: string): string | undefined {
  const token = assetToken?.trim();
  if (!token || /^(?:https?:)?\/\//i.test(token) || /[\u0000-\u0020]/.test(token)) return undefined;
  return `/api/v1/vendor-identities/${encodeURIComponent(vendorID)}/brand?version=${encodeURIComponent(token)}`;
}
