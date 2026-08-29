import { describe, expect, it } from "vitest";
import type { FormResponseWorkspacePayload } from "./captureApi";
import { buildCaptureRecoveryContext } from "./captureRecoveryContext";

function payload(overrides: Record<string, unknown> = {}): FormResponseWorkspacePayload {
  return {
    session: {
      id: "session-1",
      distribution_id: "distribution-1",
      request_id: "request-1",
      audience_hint: "o***@example.test",
      assurance: "LINK_POSSESSION",
      expires_at: "2026-09-02T12:00:00.000Z",
      created_at: "2026-09-01T12:00:00.000Z",
    },
    request: {
      id: "request-1",
      title: "Evidence request",
      purpose: "Confirm evidence.",
      why_you: "You were assigned this request.",
      audience_type: "EXTERNAL",
      status: "READY",
      sensitivity: "INTERNAL",
      estimated_minutes: 2,
      deadline: "2026-09-03T12:00:00.000Z",
      known_facts: {},
      fields: [],
      version: 99,
    },
    workspace: {
      workspace: {
        id: "workspace-1",
        distribution_id: "distribution-1",
        status: "OPEN",
        version: 4,
        created_at: "2026-09-01T12:00:00.000Z",
        updated_at: "2026-09-01T12:00:00.000Z",
      },
      answers: {},
      presentation_mode: "AUTOMATIC",
      field_sequences: {},
    },
    recovery_context: {
      legal_entity_id: "entity-1",
      distribution_id: "distribution-1",
      schema_version: 7,
      route_expires_at: "2026-09-02T12:00:00.000Z",
    },
    ...overrides,
  } as FormResponseWorkspacePayload;
}

describe("capture recovery authority", () => {
  it("enables encrypted recovery only from the server-authoritative binding", () => {
    expect(buildCaptureRecoveryContext(payload(), "https://capture.example.test")).toEqual({
      origin: "https://capture.example.test",
      legalEntityID: "entity-1",
      distributionID: "distribution-1",
      schemaVersion: 7,
      workspaceID: "workspace-1",
      serverVersion: 4,
      authorized: true,
      deadline: "2026-09-03T12:00:00.000Z",
      routeExpiresAt: "2026-09-02T12:00:00.000Z",
      cachePolicy: "ENCRYPTED_BROWSER_CACHE",
    });
  });

  it("does not fall back to compatibility request metadata when recovery_context is absent", () => {
    const malformed = payload({ recovery_context: undefined }) as FormResponseWorkspacePayload & {
      request: FormResponseWorkspacePayload["request"] & { legal_entity_id: string; form_template_version: number };
    };
    malformed.request.legal_entity_id = "legacy-entity";
    malformed.request.form_template_version = 42;

    expect(buildCaptureRecoveryContext(malformed, "https://capture.example.test")).toMatchObject({
      legalEntityID: "",
      distributionID: "",
      schemaVersion: 0,
      authorized: false,
      cachePolicy: "NO_BROWSER_CACHE",
    });
  });

  it("fails closed when the authoritative distribution disagrees with the session or workspace", () => {
    const mismatched = payload({
      recovery_context: {
        legal_entity_id: "entity-1",
        distribution_id: "distribution-other",
        schema_version: 7,
        route_expires_at: "2026-09-02T12:00:00.000Z",
      },
    });

    expect(buildCaptureRecoveryContext(mismatched, "https://capture.example.test")).toMatchObject({
      distributionID: "distribution-other",
      authorized: false,
      cachePolicy: "NO_BROWSER_CACHE",
    });
  });

  it("rejects malformed schema and expiry authority instead of manufacturing defaults", () => {
    const malformed = payload({
      recovery_context: {
        legal_entity_id: "entity-1",
        distribution_id: "distribution-1",
        schema_version: 0,
        route_expires_at: "not-a-time",
      },
    });

    expect(buildCaptureRecoveryContext(malformed, "https://capture.example.test")).toMatchObject({
      schemaVersion: 0,
      routeExpiresAt: "not-a-time",
      authorized: false,
      cachePolicy: "NO_BROWSER_CACHE",
    });
  });
});
