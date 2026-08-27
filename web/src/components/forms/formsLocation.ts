import "../../forms.css";
import type { FormTemplateQuery } from "../../formsTypes";
import type { LifecycleStatus } from "../../monitoringTypes";

const DEFAULT_LIMIT = 25;

export function readFormsQuery(hash: string, fallbackSearch?: string): FormTemplateQuery {
  const raw = hash.includes("?") ? hash.slice(hash.indexOf("?") + 1) : "";
  const params = new URLSearchParams(raw);
  const limitValue = Number(params.get("limit") || String(DEFAULT_LIMIT));
  return {
    search: params.get("search") || fallbackSearch || undefined,
    status: (params.get("status") || undefined) as LifecycleStatus | undefined,
    owner: params.get("owner") || undefined,
    program: params.get("program") || undefined,
    use: params.get("use") || undefined,
    tag: params.get("tag") || undefined,
    limit: Number.isFinite(limitValue) && limitValue >= 1 && limitValue <= 100 ? limitValue : DEFAULT_LIMIT,
  };
}

export function clearedFormsQuery(query: FormTemplateQuery): FormTemplateQuery {
  return { limit: query.limit ?? DEFAULT_LIMIT };
}

export function writeFormsLocation(query: FormTemplateQuery, targetID?: string, replace = true) {
  const params = new URLSearchParams();
  if (query.search?.trim()) params.set("search", query.search.trim());
  if (query.status) params.set("status", query.status);
  if (query.owner?.trim()) params.set("owner", query.owner.trim());
  if (query.program?.trim()) params.set("program", query.program.trim());
  if (query.use?.trim()) params.set("use", query.use.trim());
  if (query.tag?.trim()) params.set("tag", query.tag.trim());
  if (query.limit && query.limit !== DEFAULT_LIMIT) params.set("limit", String(query.limit));
  const encoded = params.toString();
  const hash = `#forms${targetID ? `/${encodeURIComponent(targetID)}` : ""}${encoded ? `?${encoded}` : ""}`;
  if (replace) window.history.replaceState(null, "", hash); else window.history.pushState(null, "", hash);
}
