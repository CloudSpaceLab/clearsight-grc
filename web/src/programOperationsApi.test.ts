import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadContext } from "./api";
import { requestJSON } from "./http";
import {
  addProgramControlImplementation,
  addProgramControlObjective,
  addProgramEvidenceContract,
  addProgramRequirement,
  assignProgram,
  determineProgramApplicability,
  linkProgramRequirementControl,
  loadProgramOperations,
  recordProgramEvidenceAssessment,
  supersedeProgramRequirement,
  transitionProgram,
  updateProgramDetails,
} from "./programOperationsApi";

vi.mock("./api", () => ({ loadContext: vi.fn() }));
vi.mock("./http", () => ({ requestJSON: vi.fn() }));

describe("Program operation API", () => {
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

  it("loads server-resolved responsibilities for the exact Program", async () => {
    await loadProgramOperations("program / 1");
    expect(requestJSON).toHaveBeenCalledWith(
      "",
      "/api/v1/programs/program%20%2F%201/operations?tenant_id=tenant-1",
    );
  });

  it.each([
    {
      name: "details",
      run: () => updateProgramDetails("program-1", 4, {
        name: "Nigeria data protection", owningFunction: "Data Protection Office", jurisdiction: "Nigeria",
        scope: { business_lines: ["Retail", "Corporate"] }, effectiveFrom: "2026-01-01T00:00:00Z",
        rationale: "Confirm the approved operating scope.",
      }),
      path: "/api/v1/programs/program-1/details?tenant_id=tenant-1",
      body: { tenant_id: "tenant-1", expected_version: 4, name: "Nigeria data protection", owning_function: "Data Protection Office", jurisdiction: "Nigeria", scope: { business_lines: ["Retail", "Corporate"] }, effective_from: "2026-01-01T00:00:00Z", rationale: "Confirm the approved operating scope." },
    },
    {
      name: "assignment",
      run: () => assignProgram("program-1", 5, "owner-2", "Assign the current DPO position."),
      path: "/api/v1/programs/program-1/assignment?tenant_id=tenant-1",
      body: { tenant_id: "tenant-1", expected_version: 5, owner_principal_id: "owner-2", rationale: "Assign the current DPO position." },
    },
    {
      name: "requirement supersession",
      run: () => supersedeProgramRequirement("program-1", "requirement / 1", 6, {
        code: "CAR-01", title: "File the annual return", statement: "The bank must file through a licensed DPCO.",
        sourceAnchor: "GAID 2025, section 7.2", modality: "MUST", actor: "The bank", action: "file",
        object: "the annual return", effectiveFrom: "2026-09-01T00:00:00Z", rationale: "The filing channel changed.",
      }),
      path: "/api/v1/programs/program-1/requirements/requirement%20%2F%201/supersede?tenant_id=tenant-1",
      body: { tenant_id: "tenant-1", expected_version: 6, code: "CAR-01", title: "File the annual return", statement: "The bank must file through a licensed DPCO.", source_anchor: "GAID 2025, section 7.2", modality: "MUST", actor: "The bank", action: "file", object: "the annual return", effective_from: "2026-09-01T00:00:00Z", rationale: "The filing channel changed." },
    },
    {
      name: "requirement addition",
      run: () => addProgramRequirement("program-1", 7, {
        code: "CAR-02", title: "Keep filing evidence", statement: "The bank must keep the filing receipt.",
        sourceAnchor: "GAID 2025, section 7.3", modality: "MUST", actor: "The bank", action: "keep",
        object: "the filing receipt", effectiveFrom: "2026-09-01T00:00:00Z",
      }),
      path: "/api/v1/programs/program-1/requirements?tenant_id=tenant-1",
      body: { tenant_id: "tenant-1", expected_version: 7, code: "CAR-02", title: "Keep filing evidence", statement: "The bank must keep the filing receipt.", source_anchor: "GAID 2025, section 7.3", modality: "MUST", actor: "The bank", action: "keep", object: "the filing receipt", status: "APPROVED", effective_from: "2026-09-01T00:00:00Z" },
    },
    {
      name: "applicability decision",
      run: () => determineProgramApplicability("program-1", 8, {
        requirementID: "requirement-1", status: "APPLICABLE", scope: { legal_entities: ["Clear Bank Nigeria"] },
        rationale: "The return applies to the licensed entity.", effectiveFrom: "2026-09-01T00:00:00Z",
      }),
      path: "/api/v1/programs/program-1/applicability?tenant_id=tenant-1",
      body: { tenant_id: "tenant-1", expected_version: 8, requirement_id: "requirement-1", status: "APPLICABLE", scope: { legal_entities: ["Clear Bank Nigeria"] }, rationale: "The return applies to the licensed entity.", effective_from: "2026-09-01T00:00:00Z" },
    },
    {
      name: "control objective",
      run: () => addProgramControlObjective("program-1", 9, { code: "CAR-COMPLETE", name: "Complete return", outcome: "Every required section is filed with current evidence.", status: "ACTIVE" }),
      path: "/api/v1/programs/program-1/control-objectives?tenant_id=tenant-1",
      body: { tenant_id: "tenant-1", expected_version: 9, code: "CAR-COMPLETE", name: "Complete return", outcome: "Every required section is filed with current evidence.", status: "ACTIVE" },
    },
    {
      name: "control implementation",
      run: () => addProgramControlImplementation("program-1", 10, {
        objectiveID: "objective-1", name: "Annual return checklist", description: "Owners confirm each section and attach current evidence.",
        implementationType: "CHECKLIST", ownerPrincipalID: "owner-2", scope: { legal_entity: "Clear Bank Nigeria" },
        status: "IMPLEMENTED", effectiveFrom: "2026-09-01T00:00:00Z",
      }),
      path: "/api/v1/programs/program-1/control-implementations?tenant_id=tenant-1",
      body: { tenant_id: "tenant-1", expected_version: 10, objective_id: "objective-1", name: "Annual return checklist", description: "Owners confirm each section and attach current evidence.", implementation_type: "CHECKLIST", owner_principal_id: "owner-2", scope: { legal_entity: "Clear Bank Nigeria" }, status: "IMPLEMENTED", effective_from: "2026-09-01T00:00:00Z" },
    },
    {
      name: "control link",
      run: () => linkProgramRequirementControl("program-1", 11, "requirement-1", "implementation-1"),
      path: "/api/v1/programs/program-1/control-links?tenant_id=tenant-1",
      body: { tenant_id: "tenant-1", expected_version: 11, requirement_id: "requirement-1", implementation_id: "implementation-1" },
    },
    {
      name: "evidence check",
      run: () => addProgramEvidenceContract("program-1", 12, {
        requirementID: "requirement-1", controlImplementationID: "implementation-1", code: "CAR-EVIDENCE",
        name: "Annual return evidence", claim: "Every required section has current evidence.", acceptableSourceIDs: ["source-1"],
        populationScope: { period: "2026" }, freshnessMinutes: 1440, minimumCoverage: 1, independenceRequired: true,
        contradictionPolicy: "FAIL", failureAction: "CREATE_MATTER", status: "ACTIVE",
      }),
      path: "/api/v1/programs/program-1/evidence-contracts?tenant_id=tenant-1",
      body: { tenant_id: "tenant-1", expected_version: 12, requirement_id: "requirement-1", control_implementation_id: "implementation-1", code: "CAR-EVIDENCE", name: "Annual return evidence", claim: "Every required section has current evidence.", acceptable_source_ids: ["source-1"], population_scope: { period: "2026" }, freshness_minutes: 1440, minimum_coverage: 1, independence_required: true, contradiction_policy: "FAIL", failure_action: "CREATE_MATTER", status: "ACTIVE" },
    },
    {
      name: "evidence assessment",
      run: () => recordProgramEvidenceAssessment("program-1", 13, {
        contractID: "contract-1", conclusion: "SUPPORTED", coverage: 1, basis: { receipt: "artifact-1" },
        assessedAt: "2026-09-02T10:00:00Z",
      }),
      path: "/api/v1/programs/program-1/evidence-assessments?tenant_id=tenant-1",
      body: { tenant_id: "tenant-1", expected_version: 13, contract_id: "contract-1", conclusion: "SUPPORTED", coverage: 1, basis: { receipt: "artifact-1" }, assessed_at: "2026-09-02T10:00:00Z" },
    },
    {
      name: "lifecycle transition",
      run: () => transitionProgram("program-1", 14, "ACTIVE", "Initial requirements and safeguards are approved."),
      path: "/api/v1/programs/program-1/transition?tenant_id=tenant-1",
      body: { tenant_id: "tenant-1", expected_version: 14, to: "ACTIVE", rationale: "Initial requirements and safeguards are approved." },
    },
  ])("submits the $name request without browser-supplied command actors", async ({ run, path, body }) => {
    await run();
    const [, actualPath, init] = vi.mocked(requestJSON).mock.calls[0]!;
    const actualBody = JSON.parse(String(init?.body));
    expect(actualPath).toBe(path);
    expect(init?.method).toBe("POST");
    expect(actualBody).toEqual(body);
    expect(actualBody).not.toHaveProperty("actor_id");
    expect(actualBody).not.toHaveProperty("approved_by");
    expect(actualBody).not.toHaveProperty("assessed_by");
    expect(actualBody).not.toHaveProperty("authority_principal_id");
  });
});
