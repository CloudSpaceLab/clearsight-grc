// Synthetic browser evidence only; imported through reportingEvidence/evidenceMain.
// These closures exercise UI commands, not backend authority or independent actors.
import type { ReportDefinition, ReportDefinitionAction, ReportDefinitionInput, ReportDefinitionStatus, ReportRun, ReportRunParameters } from "./reportingTypes";

export type ReportingEvidenceCommand = {
  method: string;
  path: string;
  body: unknown;
  status: number;
  result: unknown;
};

declare global {
  interface Window {
    reportingEvidenceCommands?: ReportingEvidenceCommand[];
  }
}

type Seed = { definitions: readonly ReportDefinition[]; runs: readonly ReportRun[] };
const base = "/api/v1/reports";
const fixtureTime = Date.parse("2026-10-21T12:00:00Z");
const identityFields = new Set(["actor", "actor_id", "principal_id", "maker_id", "reviewer_id", "authorizer_id", "checker_id", "requested_by_ref", "tenant_id", "legal_entity_id"]);
const definitionKeys = new Set(["code", "name", "description", "dataset", "scope_kind", "scope_ref", "format", "filter", "effective_from", "effective_until"]);
const transitionKeys = new Set(["expected_version", "checksum_seen", "note", "effective_from"]);
const runKeys = new Set(["definition_id", "expected_definition_version", "parameters"]);
const parameterKeys = new Set(["start_date", "end_date", "owner_principal_id"]);

export function createReportBuilderEvidence(fixture: string | null, seed: Seed, commands: ReportingEvidenceCommand[]) {
  if (fixture !== "report-builder-flow" && fixture !== "report-builder-blocked") return undefined;
  const definitions = clone(seed.definitions).map((item) => ({ ...item, name: sampleLabel(item.name), description: sampleLabel(item.description) }));
  const runs = clone([...seed.runs]);
  let definitionSequence = 0;
  let runSequence = 0;
  let mutationSequence = 0;
  const now = () => new Date(fixtureTime + mutationSequence++ * 1000).toISOString();

  return async (input: RequestInfo | URL, init?: RequestInit): Promise<Response | undefined> => {
    const raw = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    const url = new URL(raw, window.location.origin);
    const path = url.pathname;
    const method = (init?.method ?? (input instanceof Request ? input.method : "GET")).toUpperCase();
    if (method === "GET") {
      if (path === `${base}/definitions`) return json({ items: definitions.filter((item) => url.searchParams.get("include_retired") === "true" || item.status !== "RETIRED") });
      const definitionRead = /^\/api\/v1\/reports\/definitions\/([^/]+)$/.exec(path);
      if (definitionRead) return found(definitions.find((item) => item.id === decodeURIComponent(definitionRead[1]!)));
      if (path === `${base}/runs`) {
        const definitionID = url.searchParams.get("definition_id");
        const limit = Math.min(100, Math.max(1, Number(url.searchParams.get("limit")) || 50));
        return json({ items: runs.filter((item) => !definitionID || item.definition_id === definitionID).slice(0, limit) });
      }
      const runRead = /^\/api\/v1\/reports\/runs\/([^/]+)$/.exec(path);
      if (runRead) return found(runs.find((item) => item.id === decodeURIComponent(runRead[1]!)));
      if (/^\/api\/v1\/reports\/runs\/sample-report-builder-run-\d+\/download$/.test(path)) {
        return json(error("report_run_not_ready", "Sample report remains queued. No file has been generated."), 409);
      }
      return undefined;
    }
    const transition = /^\/api\/v1\/reports\/definitions\/([^/]+)\/(submit|review|activate|reject|retire)$/.exec(path);
    if (method !== "POST" || (path !== `${base}/definitions` && path !== `${base}/runs` && !transition)) return undefined;

    let body: unknown = null;
    const reply = (result: unknown, status = 200) => {
      commands.push({ method, path, body: clone(body), status, result: clone(result) });
      return json(result, status);
    };
    const invalid = () => reply(error("report_request_invalid", "Sample request has unsupported or missing report fields."), 400);
    const conflict = () => reply(error("report_version_conflict", "Sample report changed. Refresh setups before retrying."), 409);
    const blocked = () => reply(error("report_governance_blocked", "Sample setup has not completed the required review and activation."), 422);
    try {
      const text = init?.body !== undefined && init.body !== null
        ? await new Response(init.body).text()
        : input instanceof Request ? await input.clone().text() : "";
      body = JSON.parse(text);
    } catch {
      return invalid();
    }
    if (!record(body)) return invalid();
    if (Object.keys(body).some((key) => identityFields.has(key))) {
      return reply(error("report_fixture_identity_payload", "Sample commands must not supply actor, tenant or legal-entity identity."), 400);
    }

    if (path === `${base}/definitions`) {
      if (!keysAllowed(body, definitionKeys) || !validDefinition(body)) return invalid();
      const value = body as unknown as ReportDefinitionInput;
      if (definitions.some((item) => item.code === value.code && item.status !== "RETIRED")) return conflict();
      const sequence = ++definitionSequence;
      const createdAt = now();
      const created: ReportDefinition = {
        id: `sample-report-builder-${sequence}`, tenant_id: "bank-demo", legal_entity_id: "bank-ng",
        code: value.code, name: sampleLabel(value.name.trim()), description: sampleLabel(value.description || "Report builder command fixture."),
        dataset: value.dataset, scope_kind: value.scope_kind, ...(value.scope_ref ? { scope_ref: value.scope_ref } : {}),
        format: value.format, filter: clone(value.filter ?? { kind: "group", operator: "and", children: [] }),
        status: "DRAFT", current_version: 1, effective: false, version: 1,
        // Deliberately deterministic sample token, not a backend digest proof.
        checksum: `${"b".repeat(56)}${sequence.toString(16).padStart(8, "0")}`,
        maker_id: "Sample proposer", created_at: createdAt, updated_at: createdAt,
      };
      definitions.unshift(created);
      return reply(created, 201);
    }

    if (transition) {
      if (!keysAllowed(body, transitionKeys)) return invalid();
      const definition = definitions.find((item) => item.id === decodeURIComponent(transition[1]!));
      if (!definition) return reply(error("report_not_found", "Sample setup is unavailable."), 404);
      if (body.expected_version !== definition.version || body.checksum_seen !== definition.checksum) return conflict();
      const action = transition[2] as ReportDefinitionAction;
      const next = nextStatus(definition.status, action);
      if (!next) return blocked();
      if (fixture === "report-builder-blocked" && action === "review") {
        return reply(error("report_governance_blocked", "Sample review permission denied. No changes made."), 403);
      }
      const at = now();
      const updated: ReportDefinition = { ...definition, status: next, version: definition.version + 1,
        effective: next === "ACTIVE", updated_at: at,
        ...(action === "submit" ? { submitted_at: at } : {}),
        ...(action === "review" ? { reviewer_id: "Sample independent reviewer", reviewer_note: "Sample review recorded for UI evidence only." } : {}),
        ...(action === "activate" ? { checker_id: "Sample authorizer", approved_at: at, effective_from: at } : {}),
        ...(next === "RETIRED" ? { retired_at: at } : {}),
      };
      definitions.splice(definitions.indexOf(definition), 1, updated);
      return reply(updated);
    }

    if (!keysAllowed(body, runKeys)) return invalid();
    const definition = definitions.find((item) => item.id === body.definition_id);
    if (!definition) return reply(error("report_not_found", "Sample setup is unavailable."), 404);
    if (body.expected_definition_version !== definition.current_version) return conflict();
    if (definition.status !== "ACTIVE" || !definition.effective) return blocked();
    const parameters = body.parameters ?? {};
    if (!record(parameters) || !keysAllowed(parameters, parameterKeys)
      || Object.values(parameters).some((value) => typeof value !== "string")
      || (definition.dataset === "MATTER_BOARD_BRIEF" && Object.keys(parameters).length > 0)) return invalid();
    const createdAt = now();
    const run: ReportRun = {
      id: `sample-report-builder-run-${++runSequence}`, tenant_id: "bank-demo", legal_entity_id: "bank-ng",
      definition_id: definition.id, definition_version: definition.current_version, definition_code: definition.code,
      definition_checksum: definition.checksum, scope_kind: definition.scope_kind, scope_ref: definition.scope_ref,
      requested_by_ref: "Sample report performer", as_of: createdAt,
      filter: clone(definition.filter ?? { kind: "group", operator: "and", children: [] }),
      parameters: clone(parameters) as ReportRunParameters, dataset: definition.dataset, format: definition.format,
      status: "QUEUED", attempt_count: 0, row_count: 0, created_at: createdAt,
      expires_at: "2026-10-28T12:00:00Z",
      source_boundary: { captured_at: "", projection_version: "", source_high_water: {}, population: 0, population_complete: false },
    };
    runs.unshift(run);
    return reply(run, 201);
  };
}

function nextStatus(status: ReportDefinitionStatus, action: ReportDefinitionAction): ReportDefinitionStatus | undefined {
  if (action === "submit" && status === "DRAFT") return "PENDING_REVIEW";
  if (action === "review" && status === "PENDING_REVIEW") return "REVIEWED";
  if (action === "activate" && status === "REVIEWED") return "ACTIVE";
  if (action === "reject" && (status === "PENDING_REVIEW" || status === "REVIEWED")) return "RETIRED";
  if (action === "retire" && status === "ACTIVE") return "RETIRED";
  return undefined;
}

function validDefinition(value: Record<string, unknown>) {
  return typeof value.code === "string" && value.code.length > 0
    && typeof value.name === "string" && value.name.trim().length >= 3
    && ["VENDORS", "PROGRAMS", "MATTERS", "MATTER_EXCEPTIONS", "MATTER_BOARD_BRIEF", "PROCESSING_ACTIVITIES", "PROCESSING_ACTIVITY_EXCEPTIONS"].includes(String(value.dataset))
    && ["LEGAL_ENTITY", "PROGRAM", "MATTER"].includes(String(value.scope_kind))
    && ["XLSX", "PDF", "CSV", "NDJSON"].includes(String(value.format))
    && (value.scope_kind === "LEGAL_ENTITY" || (typeof value.scope_ref === "string" && value.scope_ref.length > 0))
    && (value.dataset !== "MATTER_BOARD_BRIEF" || (value.scope_kind === "MATTER" && value.format === "PDF"))
    && (value.filter === undefined || record(value.filter));
}
function record(value: unknown): value is Record<string, unknown> { return typeof value === "object" && value !== null && !Array.isArray(value); }
function keysAllowed(value: Record<string, unknown>, allowed: Set<string>) { return Object.keys(value).every((key) => allowed.has(key)); }
function sampleLabel(value: string) { return /^Sample\b/i.test(value) ? value : `Sample ${value}`; }
function clone<T>(value: T): T { return JSON.parse(JSON.stringify(value)) as T; }
function error(code: string, message: string) { return { error: { code, message } }; }
function json(value: unknown, status = 200) { return new Response(JSON.stringify(value), { status, headers: { "Content-Type": "application/json" } }); }
function found(value: unknown) { return value ? json(value) : json(error("report_not_found", "Sample report is unavailable."), 404); }
