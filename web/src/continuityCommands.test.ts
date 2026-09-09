import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadContext } from "./api";
import { addProgramRequirement, createMatter, createProgram, loadProgramSetupCandidates } from "./continuityCommands";
import { requestJSON } from "./http";

vi.mock("./api", () => ({ loadContext: vi.fn(), resolveAuthority: vi.fn() }));
vi.mock("./http", () => ({ requestJSON: vi.fn() }));

describe("continuity commands", () => {
  it("preserves the obligated party, action and object without making that party the command actor", async () => {
    await addProgramRequirement("program-1", 3, { code: "RETENTION", title: "Retain delivery records", statement: "The carrier should retain delivery records.", actor: "The carrier", action: "retain", object: "delivery records", modality: "SHOULD" });
    const body = JSON.parse(String(vi.mocked(requestJSON).mock.calls[0]![2]?.body));
    expect(body).toMatchObject({ actor: "The carrier", action: "retain", object: "delivery records", expected_version: 3, modality: "SHOULD" });
    expect(body).not.toHaveProperty("actor_id");
    expect(body).not.toHaveProperty("actor_principal_id");
  });

  it.each(["actor", "action", "object"])("rejects absent requirement meaning: %s", async (field) => {
    const input = { code: "RETENTION", title: "Retain delivery records", statement: "The carrier must retain delivery records.", actor: "The carrier", action: "retain", object: "delivery records", modality: "MUST", [field]: "  " };
    await expect(addProgramRequirement("program-1", 3, input)).rejects.toThrow("Enter who must act, what they must do and what the requirement applies to.");
    expect(requestJSON).not.toHaveBeenCalled();
  });
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(loadContext).mockResolvedValue({
      tenant: { id: "tenant-1", name: "Clear Bank" }, legal_entity: { id: "entity-1", name: "Clear Bank Nigeria" },
      actor: { id: "actor-1", name: "Risk owner" }, mode: "demo",
    });
    vi.mocked(requestJSON).mockResolvedValue({});
  });

  it("maps work creation to the actor-bound Matter command", async () => {
    await createMatter({
      type: "CONTROL_GAP", priority: 4, title: "Face verification is unavailable",
      summary: "The mobile channel did not return a successful result.", affectedArea: "Mobile banking",
      knownInformation: "The status check failed.", missingInformation: ["Confirm SDK version"],
      dueAt: "2026-09-30T22:59:59.999Z", programID: "program-mobile",
    });

    expect(requestJSON).toHaveBeenCalledTimes(1);
    const [base, path, init] = vi.mocked(requestJSON).mock.calls[0]!;
    expect(base).toBe("");
    expect(path).toBe("/api/v1/matters?tenant_id=tenant-1");
    expect(init?.method).toBe("POST");
    expect(JSON.parse(String(init?.body))).toEqual({
      tenant_id: "tenant-1", type: "CONTROL_GAP", priority: 4, title: "Face verification is unavailable",
      summary: "The mobile channel did not return a successful result.", scope: { access: "INTERNAL", area: "Mobile banking" },
      known_facts: { notes: "The status check failed." }, missing_facts: ["Confirm SDK version"], contradictions: [],
      owner_principal_id: "actor-1", due_at: "2026-09-30T22:59:59.999Z", program_id: "program-mobile",
    });
  });

  it("submits separate server-issued Program responsibility selections without actor or approval fields", async () => {
	await createProgram({
	  code: "NDPA", name: "Data protection", type: "PRIVACY", owningFunction: "Privacy",
	  scopeDescription: "Nigeria privacy obligations", ownerCandidateID: "owner-1", approvalAuthorityCandidateID: "cro-1",
	});

	const [, path, init] = vi.mocked(requestJSON).mock.calls[0]!;
	expect(path).toBe("/api/v1/programs?tenant_id=tenant-1");
	const body = JSON.parse(String(init?.body));
	expect(body).toMatchObject({ owner_candidate_id: "owner-1", approval_authority_candidate_id: "cro-1" });
	expect(body).not.toHaveProperty("owner_principal_id");
	expect(body).not.toHaveProperty("authority_principal_id");
	expect(body).not.toHaveProperty("actor_id");
  });

  it("loads Program responsibility candidates for the exact legal entity", async () => {
	await loadProgramSetupCandidates();
	expect(vi.mocked(requestJSON).mock.calls[0]?.[1]).toBe("/api/v1/programs/setup-candidates?tenant_id=tenant-1&scope_legal_entity_id=entity-1");
  });
});
