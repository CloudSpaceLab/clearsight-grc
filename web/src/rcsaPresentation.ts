import type { StatusTone } from "./components/ui";
import type { RCSACycle, RCSACycleDetail, RCSAHandoff, RCSAStatus, RCSATriggerKind } from "./rcsaTypes";

export function rcsaStatusLabel(status: RCSAStatus) {
  switch (status) {
    case "DRAFT": return "Draft";
    case "ASSESSMENT_OPEN": return "First line";
    case "AWAITING_CHALLENGE": return "Challenge";
    case "COMPLETED": return "Completed";
    case "CANCELLED": return "Cancelled";
  }
}

export function rcsaStatusTone(status: RCSAStatus): StatusTone {
  switch (status) {
    case "COMPLETED": return "success";
    case "ASSESSMENT_OPEN": return "info";
    case "AWAITING_CHALLENGE": return "warning";
    case "CANCELLED": return "neutral";
    case "DRAFT": return "unknown";
  }
}

export function rcsaTriggerLabel(kind: RCSATriggerKind) {
  switch (kind) {
    case "SCHEDULED": return "Scheduled";
    case "CHANGE": return "Change";
    case "MANUAL": return "Manual";
  }
}

export function rcsaHandoffTone(handoff: RCSAHandoff): StatusTone {
  switch (handoff.stage) {
    case "COMPLETE": return "success";
    case "CHALLENGE": return "warning";
    case "FIRST_LINE": return "info";
    case "CANCELLED": return "neutral";
    default: return "unknown";
  }
}

export function formatRCSADate(value?: string) {
  if (!value) return "Not recorded";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Not recorded";
  return new Intl.DateTimeFormat("en-GB", { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" }).format(date).replace("Sept", "Sep");
}

export function formatRCSAPeriod(start?: string, end?: string) {
  if (!start && !end) return "Not recorded";
  if (start && end) return `${formatRCSADate(start)} – ${formatRCSADate(end)}`;
  return start ? `From ${formatRCSADate(start)}` : `To ${formatRCSADate(end)}`;
}

export function rcsaOwnerLabel(cycle: RCSACycle, displayName?: string) {
  return displayName?.trim() || cycle.first_line_owner_principal_id || "Not assigned";
}


export type RCSAJourneyStageState = "NOT_STARTED" | "READY" | "IN_PROGRESS" | "COMPLETE" | "ATTENTION" | "NOT_REQUIRED" | "UNAVAILABLE";

export type RCSAJourneyStage = {
  id: "COLLECTION" | "CHALLENGE" | "DECISION" | "REMEDIATION";
  label: string;
  state: RCSAJourneyStageState;
  status: string;
  detail: string;
};

export function rcsaJourneyStages(detail: RCSACycleDetail): RCSAJourneyStage[] {
  const cycle = detail.cycle;
  const challenge = detail.challenge_context;
  const decisionFinal = challenge ? ["APPROVED", "CONDITIONALLY_APPROVED", "REJECTED"].includes(challenge.decision_status ?? "") : false;
  const option = decisionFinal ? challenge?.decision_option : undefined;

  const collection: RCSAJourneyStage = cycle.first_line_response_revision_id
    ? { id: "COLLECTION", label: "First-line collection", state: "COMPLETE", status: "Submitted", detail: "The current first-line response is recorded for this frozen cycle." }
    : cycle.first_line_distribution_id
      ? { id: "COLLECTION", label: "First-line collection", state: "IN_PROGRESS", status: "In progress", detail: "The first-line assessment has been issued and remains open." }
      : { id: "COLLECTION", label: "First-line collection", state: "NOT_STARTED", status: "Not started", detail: "No first-line assessment has been issued." };

  let independentChallenge: RCSAJourneyStage;
  if (!cycle.first_line_response_revision_id) {
    independentChallenge = { id: "CHALLENGE", label: "Independent challenge", state: "NOT_STARTED", status: "Not ready", detail: "First-line collection must finish before independent challenge." };
  } else if (!cycle.challenge_matter_id) {
    independentChallenge = { id: "CHALLENGE", label: "Independent challenge", state: cycle.status === "AWAITING_CHALLENGE" ? "READY" : "NOT_STARTED", status: cycle.status === "AWAITING_CHALLENGE" ? "Ready" : "Not started", detail: cycle.status === "AWAITING_CHALLENGE" ? "Independent challenge is ready to start." : "Independent challenge has not started." };
  } else if (!detail.challenge_context_complete || !challenge) {
    independentChallenge = { id: "CHALLENGE", label: "Independent challenge", state: "UNAVAILABLE", status: "Unavailable", detail: "Challenge work exists but its current details are not available to this view." };
  } else if (decisionFinal && option) {
    independentChallenge = { id: "CHALLENGE", label: "Independent challenge", state: "COMPLETE", status: "Decision recorded", detail: "The independent challenge decision is recorded on the linked issue." };
  } else {
    independentChallenge = { id: "CHALLENGE", label: "Independent challenge", state: "IN_PROGRESS", status: "In progress", detail: "Review the linked challenge issue and record the independent decision." };
  }

  let riskDecision: RCSAJourneyStage;
  if (!cycle.challenge_matter_id) {
    riskDecision = { id: "DECISION", label: "Risk decision", state: "NOT_STARTED", status: "Pending challenge", detail: "No challenge decision has been recorded." };
  } else if (!detail.challenge_context_complete || !challenge) {
    riskDecision = { id: "DECISION", label: "Risk decision", state: "UNAVAILABLE", status: "Unavailable", detail: "The linked challenge decision is not available to this view." };
  } else if (!decisionFinal || !option) {
    riskDecision = { id: "DECISION", label: "Risk decision", state: "IN_PROGRESS", status: "Decision pending", detail: "The independent reviewer has not recorded a final challenge outcome." };
  } else if (option === "ACCEPT_FIRST_LINE") {
    riskDecision = { id: "DECISION", label: "Risk decision", state: "COMPLETE", status: "First line accepted", detail: "Independent challenge accepted the submitted first-line assessment." };
  } else if (option === "REQUIRE_CHANGES") {
    riskDecision = { id: "DECISION", label: "Risk decision", state: "ATTENTION", status: "Changes required", detail: "The challenge requires follow-up changes before the work can be considered resolved." };
  } else if (option === "DEFICIENCY_CONFIRMED") {
    riskDecision = { id: "DECISION", label: "Risk decision", state: "ATTENTION", status: "Deficiency confirmed", detail: "Independent challenge confirmed a deficiency that requires treatment." };
  } else {
    riskDecision = { id: "DECISION", label: "Risk decision", state: "UNAVAILABLE", status: "Outcome unavailable", detail: "The recorded challenge outcome is not recognised by this presentation." };
  }

  let remediation: RCSAJourneyStage;
  if (cycle.challenge_matter_id && (!detail.challenge_context_complete || !challenge)) {
    remediation = { id: "REMEDIATION", label: "Remediation verification", state: "UNAVAILABLE", status: "Unavailable", detail: "Treatment state is not available to this view." };
  } else if (!decisionFinal || !option) {
    remediation = { id: "REMEDIATION", label: "Remediation verification", state: "NOT_STARTED", status: "Not started", detail: "Remediation follows the final challenge decision when treatment is required." };
  } else if (option === "ACCEPT_FIRST_LINE") {
    remediation = { id: "REMEDIATION", label: "Remediation verification", state: "NOT_REQUIRED", status: "Not required", detail: "The independent challenge accepted the first-line assessment without required treatment." };
  } else if (challenge.failed_verification_count > 0) {
    remediation = { id: "REMEDIATION", label: "Remediation verification", state: "ATTENTION", status: "Verification failed", detail: "At least one current outcome check failed; follow-up work remains required." };
  } else if (challenge.inconclusive_verification_count > 0) {
    remediation = { id: "REMEDIATION", label: "Remediation verification", state: "ATTENTION", status: "Verification inconclusive", detail: "At least one current outcome check is inconclusive." };
  } else if (challenge.open_action_count > 0) {
    remediation = { id: "REMEDIATION", label: "Remediation verification", state: "IN_PROGRESS", status: "Remediation in progress", detail: `${challenge.open_action_count} follow-up action${challenge.open_action_count === 1 ? "" : "s"} remain.` };
  } else if (challenge.active_verification_count > 0 && challenge.passed_verification_count === challenge.active_verification_count) {
    remediation = { id: "REMEDIATION", label: "Remediation verification", state: "COMPLETE", status: "Verified", detail: "All current outcome checks passed." };
  } else if (challenge.active_verification_count > 0) {
    remediation = { id: "REMEDIATION", label: "Remediation verification", state: "READY", status: "Verification pending", detail: "Remediation work is recorded; current outcome checks still need a conclusive result." };
  } else if (challenge.implemented_action_count > 0) {
    remediation = { id: "REMEDIATION", label: "Remediation verification", state: "READY", status: "Verification not recorded", detail: "Remediation actions are implemented, but no current outcome check is recorded." };
  } else {
    remediation = { id: "REMEDIATION", label: "Remediation verification", state: "ATTENTION", status: "Treatment not recorded", detail: "The challenge requires treatment, but no remediation action is recorded on the linked issue." };
  }

  return [collection, independentChallenge, riskDecision, remediation];
}

export function rcsaJourneyTone(state: RCSAJourneyStageState): StatusTone {
  switch (state) {
    case "COMPLETE": return "success";
    case "ATTENTION": return "warning";
    case "IN_PROGRESS": return "info";
    case "READY": return "info";
    case "NOT_REQUIRED": return "neutral";
    case "UNAVAILABLE": return "unknown";
    case "NOT_STARTED": return "neutral";
  }
}
