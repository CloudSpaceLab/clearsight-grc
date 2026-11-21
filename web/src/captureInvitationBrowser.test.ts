import { afterEach, expect, it } from "vitest";
import { consumeCaptureInvitation } from "./captureInvitationBrowser";

afterEach(() => window.history.replaceState({}, "", "/"));

it("removes the invitation fragment while preserving unrelated URL and history state", () => {
  window.history.replaceState({ source: "request" }, "", "/capture?mode=compact#view=response&form_access=secret-token");
  expect(consumeCaptureInvitation(window)).toBe("secret-token");
  expect(window.location.pathname).toBe("/capture");
  expect(window.location.search).toBe("?mode=compact");
  expect(window.location.hash).toBe("#view=response");
  expect(window.history.state).toEqual({ source: "request" });
});

it("does not consume a retired query invitation", () => {
  window.history.replaceState({}, "", "/capture?capture_invite=old-token#response");
  expect(consumeCaptureInvitation(window)).toBeNull();
  expect(window.location.search).toBe("?capture_invite=old-token");
  expect(window.location.hash).toBe("#response");
});

it("removes all duplicate invitation values without choosing one", () => {
  window.history.replaceState({}, "", "/capture#form_access=first&view=response&form_access=second");
  expect(consumeCaptureInvitation(window)).toBe("");
  expect(window.location.hash).toBe("#view=response");
});
