export const EXTERNAL_CAPTURE_SESSION_KEY = "clearsight.capture.external-session";

export function consumeCaptureInvitation(browser: Pick<Window, "history" | "location">): string | null {
  const url = new URL(browser.location.href);
  const invitationToken = url.searchParams.get("capture_invite");
  if (invitationToken === null) return null;

  url.searchParams.delete("capture_invite");
  browser.history.replaceState(browser.history.state, "", `${url.pathname}${url.search}${url.hash}`);
  return invitationToken;
}
