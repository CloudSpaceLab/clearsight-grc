const LEGACY_CAPTURE_LOCATOR_KEY = "clearsight.capture.active-session";
const LEGACY_CAPTURE_SESSION_PREFIX = "clearsight.capture.session.";

type CaptureSessionStorage = Pick<Storage, "getItem" | "removeItem" | "key" | "length">;

export function consumeCaptureInvitation(browser: Pick<Window, "history" | "location">): string | null {
  const url = new URL(browser.location.href);
  const invitationToken = url.searchParams.get("capture_invite");
  if (invitationToken === null) return null;

  url.searchParams.delete("capture_invite");
  browser.history.replaceState(browser.history.state, "", `${url.pathname}${url.search}${url.hash}`);
  return invitationToken;
}

/**
 * Removes bearer material written by pre-distribution external-capture builds.
 * New policy-driven form sessions are intentionally memory-only.
 */
export function purgeLegacyCaptureSession(storage: CaptureSessionStorage): void {
  const keys: string[] = [];
  for (let index = 0; index < storage.length; index += 1) {
    const key = storage.key(index);
    if (key?.startsWith(LEGACY_CAPTURE_SESSION_PREFIX)) keys.push(key);
  }
  const locator = storage.getItem(LEGACY_CAPTURE_LOCATOR_KEY);
  if (locator) keys.push(`${LEGACY_CAPTURE_SESSION_PREFIX}${locator}`);
  for (const key of new Set(keys)) storage.removeItem(key);
  storage.removeItem(LEGACY_CAPTURE_LOCATOR_KEY);
}
