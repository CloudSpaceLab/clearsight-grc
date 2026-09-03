import { loadContext } from "./api";
import { requestBlob, requestJSON } from "./http";
import type { AuditExportFormat, AuditExportReceipt, SystemActivityPage, SystemActivityQuery } from "./operationsTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export async function loadSystemActivity(query: SystemActivityQuery = {}): Promise<SystemActivityPage> {
  await loadContext();
  const params = new URLSearchParams({ limit: String(query.limit ?? 50) });
  if (query.cursor) params.set("cursor", query.cursor);
  if (query.from) params.set("from", query.from);
  if (query.to) params.set("to", query.to);
  if (query.category) params.set("category", query.category);
  if (query.eventType) params.set("event_type", query.eventType);
  if (query.objectType) params.set("object_type", query.objectType);
  if (query.objectID) params.set("object_id", query.objectID);
  if (query.actorID) params.set("actor_id", query.actorID);
  if (query.actor) params.set("actor", query.actor);
  if (query.actorKind) params.set("actor_kind", query.actorKind);
  if (query.legalEntityID) params.set("legal_entity_id", query.legalEntityID);
  return requestJSON<SystemActivityPage>(apiBase, `/api/v1/system-activity?${params.toString()}`);
}

export async function createAuditExport(format: AuditExportFormat, query: SystemActivityQuery): Promise<AuditExportReceipt> {
  await loadContext();
  return requestJSON<AuditExportReceipt>(apiBase, "/api/v1/audit-exports", {
    method: "POST",
    body: JSON.stringify({
      format,
      from: query.from,
      to: query.to,
      category: query.category || undefined,
      event_type: query.eventType,
      object_type: query.objectType,
      object_id: query.objectID,
      actor_id: query.actorID,
      actor: query.actor,
      actor_kind: query.actorKind || undefined,
      legal_entity_id: query.legalEntityID,
    }),
  });
}

export async function getAuditExport(id: string): Promise<AuditExportReceipt> {
  await loadContext();
  return requestJSON<AuditExportReceipt>(apiBase, `/api/v1/audit-exports/${encodeURIComponent(id)}`);
}

export async function downloadAuditExport(id: string): Promise<{ blob: Blob; filename?: string }> {
  await loadContext();
  return requestBlob(apiBase, `/api/v1/audit-exports/${encodeURIComponent(id)}/download`);
}
