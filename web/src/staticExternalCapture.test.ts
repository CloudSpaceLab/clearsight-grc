import { beforeEach, describe, expect, it } from "vitest";
import { resetStaticExternalCaptureDraft, staticExternalCaptureRequest } from "./staticExternalCapture";

describe("static external capture draft", () => {
  beforeEach(() => resetStaticExternalCaptureDraft());

  it("keeps an optimistic request-scoped draft behind the session bearer", async () => {
    const headers = { Authorization: "Bearer static-field-agent-session" };
    expect(await staticExternalCaptureRequest("/api/v1/evidence/session/draft", { headers })).toEqual({ answers: {}, presentation_mode: "AUTOMATIC", version: 0 });
    expect(await staticExternalCaptureRequest("/api/v1/evidence/session/draft", {
      method: "PUT", headers, body: JSON.stringify({ answers: { present: { text: "Yes" } }, presentation_mode: "CLASSIC", expected_version: 0 }),
    })).toEqual({ answers: { present: { text: "Yes" } }, presentation_mode: "CLASSIC", version: 1 });
    await expect(staticExternalCaptureRequest("/api/v1/evidence/session/draft", {
      method: "PUT", headers, body: JSON.stringify({ answers: {}, presentation_mode: "CLASSIC", expected_version: 0 }),
    })).rejects.toMatchObject({ staticStatus: 409, staticCode: "draft_conflict" });
    expect(await staticExternalCaptureRequest("/api/v1/evidence/session/draft", { headers: { Authorization: "Bearer another-session" } })).toBeUndefined();
  });
});
