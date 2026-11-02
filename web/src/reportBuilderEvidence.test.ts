import { afterEach, describe, expect, it, vi } from "vitest";
import { installReportingEvidence } from "./reportingEvidence";
import type { ReportDefinition, ReportDefinitionAction, ReportDefinitionInput, ReportRun } from "./reportingTypes";

const base = "/api/v1/reports";
const input: ReportDefinitionInput = {
  code: "SAMPLE_PROGRAM_EXCEPTIONS", name: "Sample Program exceptions", description: "Sample Program attention report.",
  dataset: "PROGRAMS", scope_kind: "LEGAL_ENTITY", format: "XLSX",
  filter: { kind: "group", operator: "or", children: [
    { kind: "condition", field: "overall_state", operator: "is", value: "AT_RISK" },
    { kind: "condition", field: "overall_state", operator: "is", value: "UNKNOWN" },
  ] },
};

afterEach(() => {
  vi.unstubAllGlobals();
  window.history.replaceState(null, "", "/");
});

function install(fixture = "report-builder-flow") {
  const previous = vi.fn(async () => new Response(JSON.stringify({ delegated: true }), { headers: { "Content-Type": "application/json" } }));
  vi.stubGlobal("fetch", previous);
  window.history.replaceState(null, "", `/?fixture=${fixture}`);
  installReportingEvidence();
  return previous;
}

async function post<T>(path: string, body: unknown, status = 200): Promise<T> {
  const response = await fetch(path, { method: "POST", body: JSON.stringify(body) });
  expect(response.status).toBe(status);
  return response.json() as Promise<T>;
}

async function read<T>(path: string): Promise<T> {
  const response = await fetch(path);
  expect(response.status).toBe(200);
  return response.json() as Promise<T>;
}

async function advance(definition: ReportDefinition, action: ReportDefinitionAction) {
  return post<ReportDefinition>(`${base}/definitions/${definition.id}/${action}`, {
    expected_version: definition.version, checksum_seen: definition.checksum,
  });
}

async function createActive() {
  let definition = await post<ReportDefinition>(`${base}/definitions`, input, 201);
  for (const action of ["submit", "review", "activate"] as const) definition = await advance(definition, action);
  return definition;
}

describe("isolated report builder command evidence", () => {
  it("retains the create, review and activation chain and queues the returned definition revision", async () => {
    const previous = install();
    const request = new Request(new URL(`${base}/definitions`, window.location.origin), { method: "POST", body: JSON.stringify(input) });
    const response = await fetch(request);
    expect(response.status).toBe(201);
    const created = await response.json() as ReportDefinition;
    expect(created).toMatchObject({ ...input, id: "sample-report-builder-1", status: "DRAFT", version: 1, current_version: 1, effective: false, maker_id: "Sample proposer" });
    expect(created.checksum).toMatch(/^[0-9a-f]{64}$/);
    const chain = [created];
    for (const [action, status] of [["submit", "PENDING_REVIEW"], ["review", "REVIEWED"], ["activate", "ACTIVE"]] as const) {
      const before = chain.at(-1)!;
      const after = await advance(before, action);
      expect(after).toMatchObject({ id: created.id, name: input.name, filter: input.filter, status,
        version: before.version + 1, current_version: created.current_version, checksum: created.checksum,
        effective: status === "ACTIVE" });
      const persisted = await read<{ items: ReportDefinition[] }>(`${base}/definitions?include_retired=true`);
      expect(persisted.items.find((item) => item.id === created.id)).toEqual(after);
      chain.push(after);
    }
    const active = chain.at(-1)!;
    expect(active).toMatchObject({ reviewer_id: "Sample independent reviewer", checker_id: "Sample authorizer" });
    const parameters = { start_date: "2026-09-01", end_date: "2026-09-30", owner_principal_id: "owner-program" };
    const run = await post<ReportRun>(`${base}/runs`, { definition_id: active.id, expected_definition_version: active.current_version, parameters }, 201);
    expect(run).toMatchObject({ id: "sample-report-builder-run-1", definition_id: active.id, definition_version: 1,
      definition_checksum: active.checksum, dataset: "PROGRAMS", format: "XLSX", status: "QUEUED", attempt_count: 0, row_count: 0, parameters });
    expect(run.completed_at).toBeUndefined();
    expect(run.data_sha256).toBeUndefined();
    expect(run.manifest_sha256).toBeUndefined();
    expect(run.data_object_key).toBeUndefined();
    expect(run.source_boundary).toEqual({ captured_at: "", projection_version: "", source_high_water: {}, population: 0, population_complete: false });
    expect((await read<{ items: ReportRun[] }>(`${base}/runs?definition_id=${active.id}&limit=50`)).items).toEqual([run]);
    expect((await fetch(`${base}/runs/${run.id}/download`)).status).toBe(409);

    expect(window.reportingEvidenceCommands).toEqual([
      { method: "POST", path: `${base}/definitions`, body: input, status: 201, result: created },
      ...["submit", "review", "activate"].map((action, index) => ({
        method: "POST", path: `${base}/definitions/${created.id}/${action}`,
        body: { expected_version: chain[index]!.version, checksum_seen: chain[index]!.checksum }, status: 200, result: chain[index + 1],
      })),
      { method: "POST", path: `${base}/runs`, body: { definition_id: active.id, expected_definition_version: active.current_version, parameters }, status: 201, result: run },
    ]);
    expect(window.reportingEvidenceReads).toHaveLength(5);
    expect(window.reportingEvidenceReads).not.toContain(`${base}/definitions/${created.id}/submit`);
    expect(previous).not.toHaveBeenCalled();
  });

  it("rejects stale transition versions and checksums without changing the stored definition", async () => {
    install();
    const created = await post<ReportDefinition>(`${base}/definitions`, input, 201);
    const submitted = await advance(created, "submit");
    for (const command of [
      { expected_version: created.version, checksum_seen: submitted.checksum },
      { expected_version: submitted.version, checksum_seen: "0".repeat(64) },
    ]) {
      expect(await post(`${base}/definitions/${created.id}/review`, command, 409)).toMatchObject({ error: { code: "report_version_conflict" } });
      expect(await read(`${base}/definitions/${created.id}`)).toEqual(submitted);
    }
    expect(window.reportingEvidenceCommands?.slice(-2).map((item) => item.status)).toEqual([409, 409]);
  });

  it("rejects client-supplied actor and identity fields on every command boundary", async () => {
    install();
    for (const field of ["actor_id", "maker_id", "reviewer_id", "checker_id", "authorizer_id", "tenant_id", "legal_entity_id"]) {
      expect(await post(`${base}/definitions`, { ...input, [field]: "forged-identity" }, 400)).toMatchObject({ error: { code: "report_fixture_identity_payload" } });
    }
    const created = await post<ReportDefinition>(`${base}/definitions`, input, 201);
    expect(created.id).toBe("sample-report-builder-1");
    expect(await post(`${base}/definitions/${created.id}/submit`, { expected_version: created.version, checksum_seen: created.checksum, actor_id: "forged-proposer" }, 400)).toMatchObject({ error: { code: "report_fixture_identity_payload" } });
    expect(await read(`${base}/definitions/${created.id}`)).toEqual(created);
    const submitted = await advance(created, "submit");
    const reviewed = await advance(submitted, "review");
    const active = await advance(reviewed, "activate");
    expect(await post(`${base}/runs`, { definition_id: active.id, expected_definition_version: active.current_version, requested_by_ref: "forged-performer" }, 400)).toMatchObject({ error: { code: "report_fixture_identity_payload" } });
    expect((await read<{ items: ReportRun[] }>(`${base}/runs?definition_id=${created.id}`)).items).toEqual([]);
  });

  it("distinguishes the definition revision used to generate from its command version", async () => {
    install();
    const active = await createActive();
    expect(active.version).toBe(4);
    expect(active.current_version).toBe(1);
    expect(await post(`${base}/runs`, { definition_id: active.id, expected_definition_version: active.version, parameters: {} }, 409)).toMatchObject({ error: { code: "report_version_conflict" } });
    expect((await read<{ items: ReportRun[] }>(`${base}/runs?definition_id=${active.id}`)).items).toEqual([]);
    expect(await post(`${base}/runs`, { definition_id: active.id, expected_definition_version: active.current_version, parameters: {} }, 201)).toMatchObject({ status: "QUEUED", definition_version: 1 });
  });

  it("keeps a denied review pending with its name, source filter and version intact", async () => {
    install("report-builder-blocked");
    const created = await post<ReportDefinition>(`${base}/definitions`, input, 201);
    const submitted = await advance(created, "submit");
    const request = { expected_version: submitted.version, checksum_seen: submitted.checksum };
    for (let attempt = 0; attempt < 2; attempt++) {
      expect(await post(`${base}/definitions/${submitted.id}/review`, request, 403)).toMatchObject({ error: { code: "report_governance_blocked" } });
      expect(await read(`${base}/definitions/${submitted.id}`)).toEqual(submitted);
    }
    expect((await read<{ items: ReportDefinition[] }>(`${base}/definitions`)).items[0]).toMatchObject({ name: input.name, filter: input.filter, status: "PENDING_REVIEW", version: 2 });
    expect(await post(`${base}/runs`, { definition_id: submitted.id, expected_definition_version: submitted.current_version, parameters: {} }, 422)).toMatchObject({ error: { code: "report_governance_blocked" } });
    expect((await read<{ items: ReportRun[] }>(`${base}/runs?definition_id=${submitted.id}`)).items).toEqual([]);
  });

  it("seeds a Sample board brief and rejects date or owner overrides before queuing it", async () => {
    install();
    const list = await read<{ items: ReportDefinition[] }>(`${base}/definitions`);
    expect(list.items.every((item) => item.name.startsWith("Sample "))).toBe(true);
    const brief = list.items.find((item) => item.dataset === "MATTER_BOARD_BRIEF")!;
    const run = { definition_id: brief.id, expected_definition_version: brief.current_version };
    expect(await post(`${base}/runs`, { ...run, parameters: { start_date: "2026-09-01", owner_principal_id: "owner-work" } }, 400)).toMatchObject({ error: { code: "report_request_invalid" } });
    expect(await post(`${base}/runs`, { ...run, parameters: {} }, 201)).toMatchObject({ status: "QUEUED", dataset: "MATTER_BOARD_BRIEF", format: "PDF", parameters: {}, scope_ref: brief.scope_ref });
  });

  it.each(["report-definitions", "report-builder-flow-extra", "other"])("leaves commands delegated outside the exact fixture name: %s", async (fixture) => {
    const previous = install(fixture);
    const body = JSON.stringify(input);
    const response = await fetch(`${base}/definitions`, { method: "POST", body });
    expect(await response.json()).toEqual({ delegated: true });
    expect(previous).toHaveBeenCalledExactlyOnceWith(`${base}/definitions`, { method: "POST", body });
    expect(window.reportingEvidenceCommands).toEqual([]);
    expect(window.reportingEvidenceReads).toEqual([]);
  });

  it("delegates unrelated mutations and resets fixture state and logs on a new install", async () => {
    const previous = install();
    await fetch("/api/v1/context", { method: "POST", body: "{}" });
    await fetch(`${base}/definitions/report-active`, { method: "PUT", body: "{}" });
    expect(previous).toHaveBeenCalledTimes(2);
    expect(window.reportingEvidenceCommands).toEqual([]);
    const first = await post<ReportDefinition>(`${base}/definitions`, input, 201);
    await advance(first, "submit");
    install();
    expect(window.reportingEvidenceCommands).toEqual([]);
    expect((await read<{ items: ReportDefinition[] }>(`${base}/definitions`)).items.some((item) => item.id === first.id)).toBe(false);
    expect(await post(`${base}/definitions`, input, 201)).toMatchObject({ id: first.id, status: "DRAFT", version: 1 });
  });
});
