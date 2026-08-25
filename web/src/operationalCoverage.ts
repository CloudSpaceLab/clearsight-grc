type OperationalCoverageEntry = {
  surface: string;
  states: readonly string[];
  testedBy: readonly string[];
};

export const matterOperationalCoverage = {
  "matter.create": { surface: "MattersWorkspace", states: ["list", "empty", "create"], testedBy: ["MatterSetupWorkspace.test.tsx"] },
  "matter.details.update": { surface: "MatterDetailsPanel", states: ["open", "conflict", "read_only"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.context.change": { surface: "MatterInformationPanel", states: ["facts", "missing", "contradiction", "read_only"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.assign": { surface: "MatterDetailsPanel", states: ["assigned", "candidate_selection"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.transition": { surface: "MatterOutcomePanel", states: ["open", "blocked_from_close", "ready_to_close"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.link": { surface: "MatterDetailsPanel", states: ["unlinked", "linked", "no_visible_programs"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.decision.record": { surface: "MatterDecisionResponsePanel", states: ["proposal", "review", "authorization", "history"], testedBy: ["MatterRecordWorkspace.test.tsx", "OperatingMutations.test.tsx"] },
  "matter.action.add": { surface: "MatterActionsPanel", states: ["empty", "assigned"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.action.update": { surface: "MatterActionsPanel", states: ["planned", "in_progress"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.action.assign": { surface: "MatterActionsPanel", states: ["assigned", "candidate_selection"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.action.transition": { surface: "MatterActionsPanel", states: ["planned", "in_progress", "blocked", "implemented"], testedBy: ["MatterRecordWorkspace.test.tsx", "OperatingMutations.test.tsx"] },
  "matter.response.add": { surface: "MatterDecisionResponsePanel", states: ["empty", "draft"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.response.transition": { surface: "MatterDecisionResponsePanel", states: ["draft", "review", "approval", "transmission", "acknowledgement"], testedBy: ["MatterRecordWorkspace.test.tsx", "OperatingMutations.test.tsx"] },
  "matter.outcome.define": { surface: "MatterOutcomePanel", states: ["empty", "defined"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.outcome.record": { surface: "MatterOutcomePanel", states: ["ready", "pass", "fail", "inconclusive"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
} as const satisfies Record<string, OperationalCoverageEntry>;
