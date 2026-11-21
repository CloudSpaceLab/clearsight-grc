import { loadContext } from "./api";
import { requestJSON } from "./http";
import type { EvidenceSource } from "./types";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type NativeField = { name: string; native_type: string; nullable: boolean };
export type SourceConnection = { connection_id: string; source_id: string; version: number; code: string; name: string; status: string };
export type SourceView = { view_id: string; connection_id: string; connection_version: number; source_id: string; version: number; code: string; name: string; native_schema: NativeField[]; schema_fingerprint?: string };
export type SourceBinding = { binding_id: string; view_id: string; view_version: number; source_id: string; version: number; code: string; name: string; status: string; selected_fields: string[] };
export type RESTQuestionResultType = "BOOLEAN" | "NUMBER";
export type PreparedRESTSource = { source: EvidenceSource; connection: SourceConnection; view: SourceView; resultType?: RESTQuestionResultType };
export type RESTPreparationPhase = "source" | "connection" | "view" | "inspection";
export type CommittedRESTDrafts = { source?: Pick<EvidenceSource, "id">; connection?: Pick<SourceConnection, "connection_id" | "version">; view?: Pick<SourceView, "view_id" | "version"> };
export class RESTSourcePreparationError extends Error {
  constructor(readonly phase: RESTPreparationPhase, readonly committed: CommittedRESTDrafts) {
    super("Source setup could not be confirmed. Check existing drafts before trying again.");
    this.name = "RESTSourcePreparationError";
  }
}

async function scoped<T>(path: string, init?: RequestInit): Promise<T> {
  const context = await loadContext();
  const separator = path.includes("?") ? "&" : "?";
  return requestJSON<T>(apiBase, `${path}${separator}tenant_id=${encodeURIComponent(context.tenant.id)}`, init);
}

export async function prepareRESTSource(input: { code: string; name: string; endpoint: string; freshnessMinutes: number; resultType?: RESTQuestionResultType }): Promise<PreparedRESTSource> {
  let endpoint: URL;
  try { endpoint = new URL(input.endpoint); } catch { throw new Error("Enter a valid HTTPS endpoint."); }
  if (endpoint.protocol !== "https:") throw new Error("Use an HTTPS endpoint.");
  if (endpoint.username || endpoint.password) throw new Error("Remove credentials from the endpoint URL.");
  if (input.endpoint.includes("#") || (input.resultType && input.endpoint.includes("?"))) throw new Error("Remove query parameters and fragments from the endpoint URL.");
  const context = await loadContext();
  const committed: CommittedRESTDrafts = {};
  let phase: RESTPreparationPhase = "source";
  try {
    const source = await scoped<EvidenceSource>("/api/v1/evidence/sources", {
      method: "POST", body: JSON.stringify({
        legal_entity_id: context.legal_entity.id, code: input.code, name: input.name, type: "SYSTEM",
        authority_class: "INTERNAL_CONTROL", owner_principal_id: context.actor.id,
        expected_freshness_minutes: input.freshnessMinutes,
      }),
    });
    committed.source = { id: source.id }; phase = "connection";
    const connection = await scoped<SourceConnection>(`/api/v1/config/sources/${encodeURIComponent(source.id)}/connections`, {
      method: "POST", body: JSON.stringify({
        code: `${input.code}-REST`, name: `${input.name} endpoint`, adapter_kind: "REST_JSON", adapter_version: "rest-json-v1",
        definition: { base_url: endpoint.origin, authentication: { kind: "NONE" } }, declared_capabilities: ["INSPECT", "PAGE"],
      }),
    });
    committed.connection = { connection_id: connection.connection_id, version: connection.version }; phase = "view";
    const fixedQuery = Object.fromEntries(endpoint.searchParams.entries());
    const view = await scoped<SourceView>(`/api/v1/config/source-connections/${encodeURIComponent(connection.connection_id)}/views`, {
      method: "POST", body: JSON.stringify({
        connection_version: connection.version, code: `${input.code}-STATUS`, name: `${input.name} status`,
        definition: { path: endpoint.pathname || "/", fixed_query: fixedQuery, pagination: { mode: "NONE" }, ...(input.resultType ? { singleton: true, result_type: input.resultType } : {}) }, output_kind: "RECORDS",
      }),
    });
    committed.view = { view_id: view.view_id, version: view.version }; phase = "inspection";
    const inspected = await scoped<{ view: SourceView }>(`/api/v1/config/source-views/${encodeURIComponent(view.view_id)}/inspect?version=${view.version}`, {
      method: "POST", body: JSON.stringify({ stable_keys: [] }),
    });
    return { source, connection, view: inspected.view, resultType: input.resultType };
  } catch { throw new RESTSourcePreparationError(phase, committed); }
}

export async function createRESTBinding(prepared: PreparedRESTSource, selectedField: string): Promise<SourceBinding> {
  if (prepared.resultType && selectedField !== "result") throw new Error("Select the result field.");
  const inspected = prepared.resultType ? { view: prepared.view } : await scoped<{ view: SourceView }>(`/api/v1/config/source-views/${encodeURIComponent(prepared.view.view_id)}/inspect?version=${prepared.view.version}`, {
    method: "POST", body: JSON.stringify({ stable_keys: [selectedField] }),
  });
  return scoped<SourceBinding>(`/api/v1/config/source-views/${encodeURIComponent(inspected.view.view_id)}/bindings`, {
    method: "POST", body: JSON.stringify({
      view_version: inspected.view.version, code: `${prepared.source.code}-MONITOR`, name: `${prepared.source.name} monitoring`, purpose: "monitoring",
      operations: ["PAGE"], selected_fields: [selectedField], key_fields: prepared.resultType ? [] : [selectedField],
      limits: { page_rows: 2, response_bytes: 65536, lookup_values: 1, timeout: 5000000000 },
      required_freshness_minutes: prepared.source.expected_freshness_minutes, completeness: "REQUIRE_COMPLETE",
    }),
  });
}
