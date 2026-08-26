type OperationalCoverageEntry = {
  surface: string;
  states: readonly string[];
  testedBy: readonly string[];
  access?: "USER" | "AUTOMATION_ONLY";
};

export const matterOperationalCoverage = {
  "matter.create": { surface: "MattersWorkspace", states: ["list", "empty", "create"], testedBy: ["MatterSetupWorkspace.test.tsx"] },
  "matter.details.update": { surface: "MatterDetailsPanel", states: ["open", "conflict", "read_only"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.context.change": { surface: "MatterInformationPanel", states: ["facts", "missing", "contradiction", "read_only"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.assign": { surface: "MatterDetailsPanel", states: ["assigned", "candidate_selection"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.transition": { surface: "MatterOutcomePanel", states: ["open", "blocked_from_close", "ready_to_close"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.link": { surface: "MatterDetailsPanel", states: ["unlinked", "linked", "no_visible_programs"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.unlink": { surface: "MatterDetailsPanel", states: ["linked", "confirmation", "preserved_history"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.decision.record": { surface: "MatterDecisionResponsePanel", states: ["proposal", "review", "authorization", "history"], testedBy: ["MatterRecordWorkspace.test.tsx", "OperatingMutations.test.tsx"] },
  "matter.action.add": { surface: "MatterActionsPanel", states: ["empty", "assigned"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.action.update": { surface: "MatterActionsPanel", states: ["planned", "in_progress"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.action.assign": { surface: "MatterActionsPanel", states: ["assigned", "candidate_selection"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.action.transition": { surface: "MatterActionsPanel", states: ["planned", "in_progress", "blocked", "implemented"], testedBy: ["MatterRecordWorkspace.test.tsx", "OperatingMutations.test.tsx"] },
  "matter.response.add": { surface: "MatterDecisionResponsePanel", states: ["empty", "draft"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.response.transition": { surface: "MatterDecisionResponsePanel", states: ["draft", "review", "approval", "transmission", "acknowledgement"], testedBy: ["MatterRecordWorkspace.test.tsx", "OperatingMutations.test.tsx"] },
  "matter.outcome.define": { surface: "MatterOutcomePanel", states: ["empty", "defined"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
  "matter.outcome.supersede": { surface: "MatterOutcomePanel", states: ["active", "replacement", "preserved_history"], testedBy: ["MatterOutcomePanel.test.tsx"] },
  "matter.outcome.retire": { surface: "MatterOutcomePanel", states: ["active", "ended", "preserved_history"], testedBy: ["MatterOutcomePanel.test.tsx"] },
  "matter.outcome.record": { surface: "MatterOutcomePanel", states: ["ready", "pass", "fail", "inconclusive"], testedBy: ["MatterRecordWorkspace.test.tsx"] },
} as const satisfies Record<string, OperationalCoverageEntry>;

export const programOperationalCoverage = {
  "program.create": { surface: "ProgramSetupWorkspace", states: ["list", "empty", "create"], testedBy: ["ProgramsWorkspace.test.tsx"] },
  "program.details.update": { surface: "ProgramDetailsPanel", states: ["current", "edit", "conflict", "read_only"], testedBy: ["ProgramRecordWorkspace.test.tsx"] },
  "program.assign": { surface: "ProgramDetailsPanel", states: ["assigned", "eligible_candidate_selection", "read_only"], testedBy: ["ProgramRecordWorkspace.test.tsx"] },
  "program.approval-authority.assign": { surface: "ProgramDetailsPanel", states: ["assigned", "eligible_candidate_selection", "conflict_separated", "read_only"], testedBy: ["ProgramRecordWorkspace.test.tsx"] },
  "program.transition": { surface: "ProgramStatusPanel", states: ["draft", "active", "paused", "ended", "read_only"], testedBy: ["ProgramRecordWorkspace.test.tsx", "OperatingMutations.test.tsx"] },
  "program.requirement.add": { surface: "ProgramRequirementsPanel", states: ["empty", "source_anchored", "read_only"], testedBy: ["ProgramRecordWorkspace.test.tsx"] },
  "program.requirement.supersede": { surface: "ProgramRequirementsPanel", states: ["current", "replacement", "preserved_history"], testedBy: ["ProgramRecordWorkspace.test.tsx"] },
  "program.applicability.decide": { surface: "ProgramRequirementsPanel", states: ["undecided", "applicable", "partly", "not_applicable"], testedBy: ["ProgramRecordWorkspace.test.tsx"] },
  "program.control-objective.add": { surface: "ProgramSafeguardsPanel", states: ["empty", "defined"], testedBy: ["ProgramRecordWorkspace.test.tsx"] },
  "program.safeguard.add": { surface: "ProgramSafeguardsPanel", states: ["objective_only", "eligible_owner", "planned"], testedBy: ["ProgramSafeguardsPanel.test.tsx"] },
  "program.safeguard.update": { surface: "ProgramSafeguardsPanel", states: ["planned", "implemented_requires_reconfirmation", "read_only"], testedBy: ["ProgramSafeguardsPanel.test.tsx"] },
  "program.safeguard.assign": { surface: "ProgramSafeguardsPanel", states: ["assigned", "eligible_candidate_selection", "read_only"], testedBy: ["ProgramSafeguardsPanel.test.tsx"] },
  "program.safeguard.transition": { surface: "ProgramSafeguardsPanel", states: ["planned", "in_progress", "implemented", "inactive", "retired"], testedBy: ["ProgramSafeguardsPanel.test.tsx"] },
  "program.coverage.link": { surface: "ProgramSafeguardsPanel", states: ["uncovered", "linked", "duplicate_prevented"], testedBy: ["ProgramRecordWorkspace.test.tsx"] },
  "program.safeguard.unlink": { surface: "ProgramSafeguardsPanel", states: ["linked", "confirmation", "preserved_history"], testedBy: ["ProgramSafeguardsPanel.test.tsx"] },
  "program.evidence.define": { surface: "ProgramEvidencePanel", states: ["empty", "source_selected", "thresholds_defined"], testedBy: ["ProgramRecordWorkspace.test.tsx"] },
  "program.evidence.revise": { surface: "ProgramEvidencePanel", states: ["draft", "active_returns_to_draft", "read_only"], testedBy: ["ProgramEvidencePanel.test.tsx"] },
  "program.evidence.transition": { surface: "ProgramEvidencePanel", states: ["draft", "active", "retired", "assigned_reviewer"], testedBy: ["ProgramEvidencePanel.test.tsx"] },
  "program.evidence.assess": { surface: "ProgramEvidencePanel", states: ["not_assessed", "supported", "partly_supported", "expired"], testedBy: ["ProgramRecordWorkspace.test.tsx"] },
  "program.review.accept": { surface: "ProgramReviewDigest", states: ["changed", "acknowledged", "conflict"], testedBy: ["ProgramReviewDigest.test.tsx"] },
  "program.trigger.apply": { surface: "Deterministic Program monitoring", states: ["observed", "deduplicated", "refresh_queued"], testedBy: ["internal/continuity/service_test.go"], access: "AUTOMATION_ONLY" },
  "program.monitoring.define": { surface: "MonitoringSetup", states: ["form_check", "connected_data_check"], testedBy: ["MonitoringSetup.test.tsx", "internal/httpapi/monitoring_handlers_test.go"] },
  "program.monitoring.form.define": { surface: "MonitoringSetup", states: ["choose_input", "draft_form"], testedBy: ["MonitoringSetup.test.tsx", "internal/httpapi/monitoring_handlers_test.go"] },
  "program.monitoring.form.transition": { surface: "MonitoringSetup", states: ["draft", "awaiting_approval", "active", "ended"], testedBy: ["MonitoringSetup.test.tsx", "internal/httpapi/monitoring_handlers_test.go"] },
  "program.monitoring.collect": { surface: "MonitoringSetup", states: ["schedule", "assigned_request"], testedBy: ["MonitoringSetup.test.tsx", "internal/httpapi/monitoring_handlers_test.go"] },
  "program.monitoring.transition": { surface: "MonitoringSetup", states: ["draft", "awaiting_approval", "active", "paused", "ended"], testedBy: ["MonitoringSetup.test.tsx", "internal/httpapi/monitoring_handlers_test.go"] },
  "program.monitoring.evaluate": { surface: "MonitoringSetup", states: ["ready", "evaluated", "not_assessed", "exception"], testedBy: ["MonitoringSetup.test.tsx", "internal/httpapi/monitoring_handlers_test.go"] },
  "program.monitoring.issue.create": { surface: "MonitoringSetup", states: ["eligible", "assigned_elsewhere", "created", "replayed"], testedBy: ["MonitoringSetup.test.tsx", "internal/httpapi/monitoring_linked_issue_test.go"] },
  "monitoring.form.create": { surface: "MonitoringSetup", states: ["choose_input", "draft_form"], testedBy: ["MonitoringSetup.test.tsx"] },
  "monitoring.check.create": { surface: "MonitoringSetup", states: ["form_check", "connected_data_check"], testedBy: ["MonitoringSetup.test.tsx"] },
  "monitoring.check.transition": { surface: "MonitoringSetup", states: ["draft", "awaiting_approval", "active"], testedBy: ["MonitoringSetup.test.tsx"] },
  "monitoring.collection.start": { surface: "MonitoringSetup", states: ["schedule", "assigned_request"], testedBy: ["MonitoringSetup.test.tsx"] },
  "monitoring.source.evaluate": { surface: "MonitoringSetup", states: ["ready", "evaluated", "not_assessed", "exception"], testedBy: ["MonitoringSetup.test.tsx"] },
} as const satisfies Record<string, OperationalCoverageEntry>;
