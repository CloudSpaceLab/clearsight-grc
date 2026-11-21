// Return links only accept the Documents route; source links must not become
// arbitrary navigation targets when a URL is copied or modified.
export function documentReturnDestination(): string | undefined {
  const params = new URLSearchParams(window.location.hash.split("?")[1] ?? "");
  const destination = params.get("return_to");
  return destination && destination.length <= 1200 && /^#\/documents(?:\?[^#]*)?$/.test(destination) ? destination : undefined;
}

export function withDocumentReturn(target: string): string {
  const current = window.location.hash;
  if (current.length > 1200 || !/^#\/documents(?:\?[^#]*)?$/.test(current)) return target;
  return `${target}?${new URLSearchParams({ return_to: current })}`;
}

export function preserveDocumentReturn(target: string): string {
  const destination = documentReturnDestination();
  return destination ? `${target}?${new URLSearchParams({ return_to: destination })}` : target;
}

export function scopedDocumentsHref(kind: "vendor" | "response", id: string, assessmentId?: string, distributionId?: string): string | undefined {
  const valid = (value: string) => /^[A-Za-z0-9_-]{1,200}$/.test(value);
  if (!valid(id) || (assessmentId !== undefined && !valid(assessmentId)) || (distributionId !== undefined && !valid(distributionId))) return undefined;
  const params = new URLSearchParams();
  if (kind === "response") { params.set("response_revision_id", id); if (distributionId) params.set("return_to", `#/forms/responses/${encodeURIComponent(id)}?distribution_id=${encodeURIComponent(distributionId)}`); }
  else {
    params.set("relationship_id", id);
    params.set("return_to", `#/vendors/${encodeURIComponent(id)}${assessmentId ? `/assessments/${encodeURIComponent(assessmentId)}` : ""}`);
  }
  return `#/documents?${params}`;
}
