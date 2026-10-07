import { expect, it } from "vitest";
import { formatRCSADate, formatRCSAPeriod, rcsaJourneyStages } from "./rcsaPresentation";
import type { RCSACycleDetail } from "./rcsaTypes";

it("formats RCSA periods deterministically in UTC", () => {
  expect(formatRCSADate("2026-09-30T23:59:59Z")).toBe("30 Sep 2026");
  expect(formatRCSAPeriod("2026-07-01T00:00:00Z", "2026-09-30T23:59:59Z"))
    .toBe("1 Jul 2026 – 30 Sep 2026");
});


function detail(overrides: Partial<RCSACycleDetail> = {}): RCSACycleDetail {
  return {
    cycle: {
      id: "cycle-1",
      tenant_id: "bank",
      legal_entity_id: "entity-a",
      code: "RCSA-Q3",
      name: "Q3 Technology RCSA",
      trigger_kind: "SCHEDULED",
      first_line_owner_principal_id: "owner-1",
      status: "COMPLETED",
      population_checksum: "a".repeat(64),
      first_line_distribution_id: "distribution-1",
      first_line_response_revision_id: "response-1",
      challenge_matter_id: "matter-1",
      version: 5,
      created_at: "2026-07-01T00:00:00Z",
      updated_at: "2026-10-07T00:00:00Z",
    },
    risks: [],
    controls: [],
    challenge_context_complete: true,
    challenge_context: {
      matter_id: "matter-1",
      matter_status: "CLOSED",
      decision_status: "APPROVED",
      decision_option: "ACCEPT_FIRST_LINE",
      open_action_count: 0,
      implemented_action_count: 0,
      blocked_action_count: 0,
      active_verification_count: 0,
      passed_verification_count: 0,
      failed_verification_count: 0,
      inconclusive_verification_count: 0,
    },
    handoff: { stage: "COMPLETE", label: "Completed", target_type: "MATTER", target_id: "matter-1" },
    complete: true,
    ...overrides,
  };
}

it("marks accepted first-line challenge as requiring no remediation", () => {
  const stages = rcsaJourneyStages(detail());
  expect(stages.map((stage) => [stage.id, stage.status])).toEqual([
    ["COLLECTION", "Submitted"],
    ["CHALLENGE", "Decision recorded"],
    ["DECISION", "First line accepted"],
    ["REMEDIATION", "Not required"],
  ]);
});

it("keeps confirmed deficiency separate from open remediation", () => {
  const value = detail({
    challenge_context: {
      ...detail().challenge_context!,
      matter_status: "ACTION_IN_PROGRESS",
      decision_option: "DEFICIENCY_CONFIRMED",
      open_action_count: 2,
      active_verification_count: 1,
    },
  });
  const stages = rcsaJourneyStages(value);
  expect(stages.find((stage) => stage.id === "DECISION")).toMatchObject({ state: "ATTENTION", status: "Deficiency confirmed" });
  expect(stages.find((stage) => stage.id === "REMEDIATION")).toMatchObject({ state: "IN_PROGRESS", status: "Remediation in progress" });
});

it("shows implemented treatment as awaiting verification until current checks pass", () => {
  const value = detail({
    challenge_context: {
      ...detail().challenge_context!,
      decision_option: "REQUIRE_CHANGES",
      implemented_action_count: 2,
      active_verification_count: 1,
      passed_verification_count: 0,
    },
  });
  expect(rcsaJourneyStages(value).find((stage) => stage.id === "REMEDIATION"))
    .toMatchObject({ state: "READY", status: "Verification pending" });

  value.challenge_context = { ...value.challenge_context!, passed_verification_count: 1 };
  expect(rcsaJourneyStages(value).find((stage) => stage.id === "REMEDIATION"))
    .toMatchObject({ state: "COMPLETE", status: "Verified" });
});


it("marks restricted challenge context unavailable instead of inferring outcome state", () => {
  const value = detail({ challenge_context: undefined, challenge_context_complete: false });
  const stages = rcsaJourneyStages(value);
  expect(stages.find((stage) => stage.id === "CHALLENGE")).toMatchObject({ state: "UNAVAILABLE", status: "Unavailable" });
  expect(stages.find((stage) => stage.id === "DECISION")).toMatchObject({ state: "UNAVAILABLE", status: "Unavailable" });
  expect(stages.find((stage) => stage.id === "REMEDIATION")).toMatchObject({ state: "UNAVAILABLE", status: "Unavailable" });
});
