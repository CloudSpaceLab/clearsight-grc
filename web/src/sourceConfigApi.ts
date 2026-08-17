import { loadContext } from "./api";
import { requestJSON } from "./http";
import type { EvidenceSource } from "./types";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type NativeField = { name: string; native_type: string; nullable: boolean };
export type SourceConnection = { connection_id: string; source_id: string; version: number; code: string; name: string; status: string };
export type SourceView = { view_id: string; connection_id: string; connection_version: number; source_id: string; version: number; code: string; name: string; native_schema: NativeField[]; schema_fingerprint?: string };
export type SourceBinding = { binding_id: string; view_id: string; view_version: number; source_id: string; version: number; code: string; name: string; status: string; selected_fields: string[] };
export type PreparedRESTSource = { source: EvidenceSource; connection: SourceConnection; view: SourceView };

async function scoped<T>(path: string, init?: RequestInit): Promise<T> {
  const context = await loadContext();
  const separator = path.includes("?") ? "&" : "?";
  return requestJSON<T>(apiBase, `${path}${separator}tenant_id=${encodeURIComponent(context.tenant.id)}`, init);
}

export async function prepareRESTSource(input: { code: string; name: string; endpoint: string; freshnessMinutes: number }): Promise<PreparedRESTSource> {
  const context = await loadContext();
  const endpoint = new URL(input.endpoint);
  if (endpoint.protocol !== "https:") throw new Error("Use an HTTPS endpoint.");
  const source = await scoped<EvidenceSource>("/api/v1/evidence/sources", {
    method: "POST", body: JSON.stringify({
      legal_entity_id: context.legal_entity.id, code: input.code, name: input.name, type: "SYSTEM",
      authority_class: "INTERNAL_CONTROL", owner_principal_id: context.actor.id,
      endpoint: endpoint.toString(), expected_freshness_minutes: input.freshnessMinutes,
    }),
  });
  const connection = await scoped<SourceConnection>(`/api/v1/config/sources/${encodeURIComponent(source.id)}/connections`, {
    method: "POST", body: JSON.stringify({
      code: `${input.code}-REST`, name: `${input.name} endpoint`, adapter_kind: "REST_JSON", adapter_version: "rest-json-v1",
      definition: { base_url: endpoint.origin, authentication: { kind: "NONE" } }, declared_capabilities: ["INSPECT", "PAGE"],
    }),
  });
  const fixedQuery = Object.fromEntries(endpoint.searchParams.entries());
  const view = await scoped<SourceView>(`/api/v1/config/source-connections/${encodeURIComponent(connection.connection_id)}/views`, {
    method: "POST", body: JSON.stringify({
      connection_version: connection.version, code: `${input.code}-STATUS`, name: `${input.name} status`,
      definition: { path: endpoint.pathname || "/", fixed_query: fixedQuery, pagination: { mode: "NONE" } }, output_kind: "RECORDS",
    }),
  });
  const inspected = await scoped<{ view: SourceView }>(`/api/v1/config/source-views/${encodeURIComponent(view.view_id)}/inspect?version=${view.version}`, {
    method: "POST", body: JSON.stringify({ stable_keys: [] }),
  });
  return { source, connection, view: inspected.view };
}

export async function createRESTBinding(prepared: PreparedRESTSource, selectedField: string): Promise<SourceBinding> {
  const inspected = await scoped<{ view: SourceView }>(`/api/v1/config/source-views/${encodeURIComponent(prepared.view.view_id)}/inspect?version=${prepared.view.version}`, {
    method: "POST", body: JSON.stringify({ stable_keys: [selectedField] }),
  });
  return scoped<SourceBinding>(`/api/v1/config/source-views/${encodeURIComponent(inspected.view.view_id)}/bindings`, {
    method: "POST", body: JSON.stringify({
      view_version: inspected.view.version, code: `${prepared.source.code}-MONITOR`, name: `${prepared.source.name} monitoring`, purpose: "monitoring",
      operations: ["PAGE"], selected_fields: [selectedField], key_fields: [selectedField],
      limits: { page_rows: 2, response_bytes: 65536, lookup_values: 1, timeout: 5000000000 },
      required_freshness_minutes: prepared.source.expected_freshness_minutes, completeness: "REQUIRE_COMPLETE",
    }),
  });
}
