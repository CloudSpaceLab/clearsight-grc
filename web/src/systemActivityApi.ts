import { loadContext } from "./api";
import { requestJSON } from "./http";
import type { SystemActivityPage, SystemActivityQuery } from "./operationsTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export async function loadSystemActivity(query: SystemActivityQuery = {}): Promise<SystemActivityPage> {
  const context = await loadContext();
  const params = new URLSearchParams({ limit: String(query.limit ?? 50) });
  if (query.cursor) params.set("cursor", query.cursor);
  if (query.from) params.set("from", query.from);
  if (query.to) params.set("to", query.to);
  if (query.category) params.set("category", query.category);
  if (query.eventType) params.set("event_type", query.eventType);
  if (query.objectType) params.set("object_type", query.objectType);
  if (query.objectID) params.set("object_id", query.objectID);
  if (query.actorID) params.set("actor_id", query.actorID);
  if (query.legalEntityID) params.set("legal_entity_id", query.legalEntityID);

  // Tenant scope is deliberately not sent as authority. The server binds the
  // query to the verified actor; this value is used only to invalidate stale
  // cached context when a caller switches organizations in demo/development.
  void context.tenant.id;
  return requestJSON<SystemActivityPage>(apiBase, `/api/v1/system-activity?${params.toString()}`);
}
