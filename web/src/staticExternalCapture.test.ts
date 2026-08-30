import { beforeEach, describe, expect, it } from "vitest";
import { resetStaticExternalCaptureDraft, staticExternalCaptureRequest } from "./staticExternalCapture";

describe("static external capture", () => {
  beforeEach(() => resetStaticExternalCaptureDraft());

  it("supports the protected direct-link ceremony and shared workspace without audience input", async () => {
    expect(await staticExternalCaptureRequest("/api/v1/evidence/access/start", {
      method: "POST",
      body: JSON.stringify({ route_selector: "field-agent-demo" }),
    })).toMatchObject({ policy: "DIRECT_MAGIC_LINK" });

    const redeemed = await staticExternalCaptureRequest("/api/v1/evidence/access/redeem", {
      method: "POST",
      body: JSON.stringify({ route_selector: "field-agent-demo" }),
    }) as { session_token: string };
    const headers = { Authorization: `Bearer ${redeemed.session_token}` };

    const initial = await staticExternalCaptureRequest("/api/v1/evidence/session/workspace", { headers }) as {
      workspace: { workspace: { version: number } };
    };
    expect(initial.workspace.workspace.version).toBe(0);

    const saved = await staticExternalCaptureRequest("/api/v1/evidence/session/workspace", {
      method: "PATCH",
      headers,
      body: JSON.stringify({
        expected_version: 0,
        presentation_mode: "AUTOMATIC",
        edits: [{ field_id: "address_matches", value: { text: "Yes" }, base_sequence: 0 }],
      }),
    }) as {
      answers: Record<string, unknown>;
      field_sequences: Record<string, number>;
      workspace: { version: number };
    };
    expect(saved).toMatchObject({
      answers: { address_matches: { text: "Yes" } },
      field_sequences: { address_matches: 1 },
      workspace: { version: 1 },
    });

    const submitted = await staticExternalCaptureRequest("/api/v1/evidence/session/workspace/submissions", {
      method: "POST",
      headers,
      body: JSON.stringify({ expected_version: 1 }),
    }) as { revision: { current: boolean }; submission: { status: string } };
    expect(submitted).toMatchObject({ revision: { current: true }, submission: { status: "SUBMITTED" } });
  });

  it("keeps the legacy optimistic draft behind the bearer for compatibility fixtures", async () => {
    const headers = { Authorization: "Bearer static-field-agent-session" };
    expect(await staticExternalCaptureRequest("/api/v1/evidence/session/draft", { headers }))
      .toEqual({ answers: {}, presentation_mode: "AUTOMATIC", version: 0 });
    expect(await staticExternalCaptureRequest("/api/v1/evidence/session/draft", {
      method: "PUT",
      headers,
      body: JSON.stringify({ answers: { present: { text: "Yes" } }, presentation_mode: "CLASSIC", expected_version: 0 }),
    })).toEqual({ answers: { present: { text: "Yes" } }, presentation_mode: "CLASSIC", version: 1 });
    await expect(staticExternalCaptureRequest("/api/v1/evidence/session/draft", {
      method: "PUT",
      headers,
      body: JSON.stringify({ answers: {}, presentation_mode: "CLASSIC", expected_version: 0 }),
    })).rejects.toMatchObject({ staticStatus: 409, staticCode: "draft_conflict" });
    expect(await staticExternalCaptureRequest("/api/v1/evidence/session/draft", {
      headers: { Authorization: "Bearer another-session" },
    })).toBeUndefined();
  });

  it.each([
    ["forms-otp-expired", 410, "otp_expired"],
    ["forms-otp-exhausted", 429, "otp_attempts_exhausted"],
  ])("provides the terminal %s OTP fixture through the production access ceremony", async (fixture, status, code) => {
    window.history.replaceState(null, "", `/?fixture=${fixture}`);
    const route = `task22-${fixture}`;
    expect(await staticExternalCaptureRequest("/api/v1/evidence/access/start", { method: "POST", body: JSON.stringify({ route_selector: route }) }))
      .toMatchObject({ policy: "DIRECT_LINK_EMAIL_OTP" });
    expect(await staticExternalCaptureRequest("/api/v1/evidence/access/otp/send", { method: "POST", body: JSON.stringify({ route_selector: route, recipient_selector: "task22-recipient" }) }))
      .toMatchObject({ challenge_id: "task22-challenge" });
    await expect(staticExternalCaptureRequest("/api/v1/evidence/access/otp/verify", { method: "POST", body: JSON.stringify({ route_selector: route, challenge_id: "task22-challenge", code: "123456" }) }))
      .rejects.toMatchObject({ staticStatus: status, staticCode: code });
  });

  it("authorizes Task 22 recovery fixtures and returns an exact conflict body", async () => {
    window.history.replaceState(null, "", "/?fixture=forms-recovery-conflict");
    const route = "task22-forms-recovery-conflict";
    const redeemed = await staticExternalCaptureRequest("/api/v1/evidence/access/redeem", { method: "POST", body: JSON.stringify({ route_selector: route }) }) as { session_token: string };
    const headers = { Authorization: `Bearer ${redeemed.session_token}` };
    const initial = await staticExternalCaptureRequest("/api/v1/evidence/session/workspace", { headers }) as { recovery_context: { distribution_id: string; schema_version: number } };
    expect(initial.recovery_context).toMatchObject({ distribution_id: "field-distribution-1", schema_version: 1 });
    await expect(staticExternalCaptureRequest("/api/v1/evidence/session/workspace", { method: "PATCH", headers, body: JSON.stringify({ expected_version: 0, presentation_mode: "AUTOMATIC", edits: [{ field_id: "address_matches", value: { text: "Yes" }, base_sequence: 0 }] }) }))
      .rejects.toMatchObject({ staticStatus: 409, staticBody: { current_version: 1, changed_fields: [{ field_id: "address_matches", server_value: { text: "No" }, sequence: 1 }] } });
  });
});
