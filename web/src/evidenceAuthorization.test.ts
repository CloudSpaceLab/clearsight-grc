import { describe, expect, it } from "vitest";
import type { EvidenceRequest } from "./types";
import { canRespondToEvidenceRequest } from "./evidenceAuthorization";

function request(overrides: Partial<EvidenceRequest> = {}): EvidenceRequest {
  return {
    id: "request-1",
    tenant_id: "bank-1",
    subject_type: "PROGRAM",
    subject_id: "program-1",
    title: "Confirm evidence",
    purpose: "Confirm the current evidence.",
    why_you: "The assigned person owns this response.",
    sensitivity: "INTERNAL",
    audience_type: "INTERNAL",
    recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "actor-1", state: "ASSIGNED" },
    created_by: "requester-1",
    estimated_minutes: 3,
    deadline: "2026-08-30T12:00:00Z",
    known_facts: {},
    fields: [],
    status: "READY",
    version: 1,
    created_at: "2026-08-25T12:00:00Z",
    updated_at: "2026-08-25T12:00:00Z",
    ...overrides,
  };
}

describe("canRespondToEvidenceRequest", () => {
  it.each([
    ["READY request", request(), "actor-1", true],
    ["IN_PROGRESS request", request({ status: "IN_PROGRESS" }), "actor-1", true],
    ["missing actor", request(), undefined, false],
    ["empty actor", request(), "", false],
    ["blank actor", request({ recipient: { type: "INTERNAL_PRINCIPAL", principal_id: " ", state: "ASSIGNED" } }), " ", false],
    ["external audience", request({ audience_type: "EXTERNAL" }), "actor-1", false],
    ["missing recipient", request({ recipient: undefined }), "actor-1", false],
    ["external recipient", request({ recipient: { type: "EXTERNAL_AUDIENCE", state: "ASSIGNED" } }), "actor-1", false],
    ["missing assignment state", request({ recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "actor-1" } }), "actor-1", false],
    ["reassignment required", request({ recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "actor-1", state: "REASSIGNMENT_REQUIRED" } }), "actor-1", false],
    ["different principal", request(), "actor-2", false],
    ["submitted request", request({ status: "SUBMITTED" }), "actor-1", false],
  ])("%s", (_label, evidenceRequest, actorPrincipalID, expected) => {
    expect(canRespondToEvidenceRequest(evidenceRequest, actorPrincipalID)).toBe(expected);
  });
});
