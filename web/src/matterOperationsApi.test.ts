import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadContext } from "./api";
import {
  assignMatter,
  assignMatterAction,
  changeMatterContext,
  defineMatterOutcomeCheck,
  loadMatterOperations,
  updateMatterAction,
  updateMatterDetails,
} from "./matterOperationsApi";
import { requestJSON } from "./http";

vi.mock("./api", () => ({ loadContext: vi.fn() }));
vi.mock("./http", () => ({ requestJSON: vi.fn() }));

describe("Matter operation API", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(loadContext).mockResolvedValue({
      tenant: { id: "tenant-1", name: "Clear Bank" },
      legal_entity: { id: "entity-1", name: "Clear Bank Nigeria" },
      actor: { id: "actor-1", name: "Risk owner" },
      mode: "demo",
    });
    vi.mocked(requestJSON).mockResolvedValue({});
  });

  it("loads server-resolved responsibilities for the exact Matter", async () => {
    await loadMatterOperations("matter / 1");

    expect(requestJSON).toHaveBeenCalledWith(
      "",
      "/api/v1/matters/matter%20%2F%201/operations?tenant_id=tenant-1",
    );
  });

  it.each([
    {
      name: "details",
      run: () => updateMatterDetails("matter-1", 7, {
        title: "Annual return filing",
        summary: "Update the filing process and evidence.",
        priority: 4,
        dueAt: "2026-09-30T16:00:00.000Z",
        scope: { filing_channel: "licensed DPCO" },
        rationale: "Use the confirmed filing instructions.",
      }),
      path: "/api/v1/matters/matter-1/details?tenant_id=tenant-1",
      body: {
        tenant_id: "tenant-1", expected_version: 7, title: "Annual return filing",
        summary: "Update the filing process and evidence.", priority: 4,
        due_at: "2026-09-30T16:00:00.000Z", scope: { filing_channel: "licensed DPCO" },
        rationale: "Use the confirmed filing instructions.",
      },
    },
    {
      name: "context",
      run: () => changeMatterContext("matter-1", 8, {
        kind: "RESOLVE_MISSING", key: "final_checklist", label: "final checklist",
        value: "Checklist v3", evidenceReferences: ["artifact-v3"],
        rationale: "Record the approved checklist.",
      }),
      path: "/api/v1/matters/matter-1/context-changes?tenant_id=tenant-1",
      body: {
        tenant_id: "tenant-1", expected_version: 8, kind: "RESOLVE_MISSING", key: "final_checklist",
        label: "final checklist", value: "Checklist v3", evidence_references: ["artifact-v3"],
        rationale: "Record the approved checklist.",
      },
    },
    {
      name: "Matter assignment",
      run: () => assignMatter("matter-1", 9, "owner-2", "Assign the current compliance owner."),
      path: "/api/v1/matters/matter-1/assignment?tenant_id=tenant-1",
      body: { tenant_id: "tenant-1", expected_version: 9, owner_principal_id: "owner-2", rationale: "Assign the current compliance owner." },
    },
    {
      name: "action update",
      run: () => updateMatterAction("matter-1", "action / 1", 10, {
        title: "Update checklist", description: "Map each section to its current source.",
        dueAt: "2026-09-12T12:00:00.000Z", rationale: "Clarify the work and deadline.",
      }),
      path: "/api/v1/matters/matter-1/actions/action%20%2F%201?tenant_id=tenant-1",
      body: {
        tenant_id: "tenant-1", expected_version: 10, title: "Update checklist",
        description: "Map each section to its current source.", due_at: "2026-09-12T12:00:00.000Z",
        rationale: "Clarify the work and deadline.",
      },
    },
    {
      name: "action assignment",
      run: () => assignMatterAction("matter-1", "action-1", 11, "performer-2", "Assign the evidence owner."),
      path: "/api/v1/matters/matter-1/actions/action-1/assignment?tenant_id=tenant-1",
      body: { tenant_id: "tenant-1", expected_version: 11, owner_principal_id: "performer-2", rationale: "Assign the evidence owner." },
    },
    {
      name: "outcome check definition",
      run: () => defineMatterOutcomeCheck("matter-1", 12, {
        actionID: "action-1", expectedOutcome: "All annual return sections have current evidence.",
        baseline: { description: "Two sections are missing current evidence." },
        scope: { description: "All ten annual return sections." },
        threshold: { success_condition: "Ten of ten sections have approved evidence." },
        measurementSourceID: "source-1", observationPeriodMinutes: 1440,
        reviewerCandidateID: "reviewer-1", failureResponse: "REOPEN",
      }),
      path: "/api/v1/matters/matter-1/verification-contracts?tenant_id=tenant-1",
      body: {
        tenant_id: "tenant-1", expected_version: 12, action_id: "action-1",
        expected_outcome: "All annual return sections have current evidence.",
        baseline: { description: "Two sections are missing current evidence." },
        scope: { description: "All ten annual return sections." },
        threshold: { success_condition: "Ten of ten sections have approved evidence." },
        measurement_source_id: "source-1", observation_period_minutes: 1440,
        reviewer_candidate_id: "reviewer-1", failure_response: "REOPEN",
      },
    },
  ])("submits the $name request without browser-supplied command actors", async ({ run, path, body }) => {
    await run();

    const [, actualPath, init] = vi.mocked(requestJSON).mock.calls[0]!;
    const actualBody = JSON.parse(String(init?.body));
    expect(actualPath).toBe(path);
    expect(init?.method).toBe("POST");
    expect(actualBody).toEqual(body);
    expect(actualBody).not.toHaveProperty("actor_id");
    expect(actualBody).not.toHaveProperty("reviewer_principal_id");
    expect(actualBody).not.toHaveProperty("authority_principal_id");
  });
});
