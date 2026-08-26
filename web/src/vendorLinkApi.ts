import { requestJSON } from "./http";
import type { EndVendorRelationshipLinkInput, LinkVendorRelationshipInput, VendorRelationshipLink, VendorRelationshipLinkPage, VendorRelationshipLinkQuery } from "./vendorLinkTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export function loadVendorRelationshipLinks(query: VendorRelationshipLinkQuery): Promise<VendorRelationshipLinkPage> {
  const params = new URLSearchParams();
  params.set("target_type", query.target_type);
  params.set("target_id", query.target_id);
  if (query.cursor) params.set("cursor", query.cursor);
  params.set("limit", String(query.limit ?? 50));
  return requestJSON<VendorRelationshipLinkPage>(apiBase, `/api/v1/vendor-links?${params.toString()}`);
}

export function linkVendorRelationship(relationshipID: string, input: LinkVendorRelationshipInput): Promise<VendorRelationshipLink> {
  return requestJSON<VendorRelationshipLink>(apiBase, `/api/v1/vendors/${encodeURIComponent(relationshipID)}/links`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function endVendorRelationshipLink(relationshipID: string, linkID: string, input: EndVendorRelationshipLinkInput): Promise<VendorRelationshipLink> {
  return requestJSON<VendorRelationshipLink>(apiBase, `/api/v1/vendors/${encodeURIComponent(relationshipID)}/links/${encodeURIComponent(linkID)}/end`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}
