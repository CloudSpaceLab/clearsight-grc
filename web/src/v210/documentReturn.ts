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

export function scopedDocumentsHref(kind: "vendor" | "response", id: string, assessmentId?: string, distributionId?: string, filters = ""): string | undefined {
  const valid = (value: string) => /^[A-Za-z0-9_-]{1,200}$/.test(value);
  if (!valid(id) || (assessmentId !== undefined && !valid(assessmentId)) || (distributionId !== undefined && !valid(distributionId))) return undefined;
  const params = new URLSearchParams();
  if (kind === "response") { params.set("response_revision_id", id); const query = new URLSearchParams(filters); if (distributionId) query.set("distribution_id", distributionId); if (query.size) params.set("return_to", `#/forms/responses/${encodeURIComponent(id)}?${query}`); }
  else {
    params.set("relationship_id", id);
    params.set("return_to", `#/vendors/${encodeURIComponent(id)}${assessmentId ? `/assessments/${encodeURIComponent(assessmentId)}` : ""}`);
  }
  return `#/documents?${params}`;
}

export function scopedResponseReturn(target: string, id: string): string | undefined {
  const base = `#/forms/responses/${encodeURIComponent(id)}`;
  if (target.length > 1200 || !target.startsWith(`${base}?`)) return undefined;
  const query = new URLSearchParams(target.slice(base.length + 1));
  const allowed = new Set(["distribution_id", "response_search", "response_concern", "response_sort"]);
  for (const key of query.keys()) if (!allowed.has(key) || query.getAll(key).length !== 1) return undefined;
  const distribution = query.get("distribution_id"), search = query.get("response_search"), concern = query.get("response_concern"), sort = query.get("response_sort");
  if ((distribution !== null && !/^[A-Za-z0-9_-]{1,200}$/.test(distribution)) || (search !== null && new TextEncoder().encode(search.trim()).length > 200) || (concern !== null && !["LOW", "MODERATE", "HIGH", "CRITICAL"].includes(concern)) || (sort !== null && !["COMPLETED_DESC", "CONCERN_DESC"].includes(sort))) return undefined;
  return `${base}?${query}`;
}
