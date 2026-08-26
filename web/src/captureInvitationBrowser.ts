export const EXTERNAL_CAPTURE_LOCATOR_KEY = "clearsight.capture.active-session";

const EXTERNAL_CAPTURE_SESSION_PREFIX = "clearsight.capture.session.";

type CaptureSessionStorage = Pick<Storage, "getItem" | "removeItem" | "setItem">;

export function consumeCaptureInvitation(browser: Pick<Window, "history" | "location">): string | null {
  const url = new URL(browser.location.href);
  const invitationToken = url.searchParams.get("capture_invite");
  if (invitationToken === null) return null;

  url.searchParams.delete("capture_invite");
  browser.history.replaceState(browser.history.state, "", `${url.pathname}${url.search}${url.hash}`);
  return invitationToken;
}

export function saveCaptureSession(
  storage: CaptureSessionStorage,
  sessionToken: string,
  createLocator: () => string = () => crypto.randomUUID(),
): void {
  clearCaptureSession(storage);
  const locator = createLocator();
  storage.setItem(sessionStorageKey(locator), sessionToken);
  storage.setItem(EXTERNAL_CAPTURE_LOCATOR_KEY, locator);
}

export function readCaptureSession(storage: CaptureSessionStorage): string | null {
  const locator = storage.getItem(EXTERNAL_CAPTURE_LOCATOR_KEY);
  return locator ? storage.getItem(sessionStorageKey(locator)) : null;
}

export function hasCaptureSession(storage: CaptureSessionStorage): boolean {
  return readCaptureSession(storage) !== null;
}

export function clearCaptureSession(storage: CaptureSessionStorage): void {
  const locator = storage.getItem(EXTERNAL_CAPTURE_LOCATOR_KEY);
  if (locator) storage.removeItem(sessionStorageKey(locator));
  storage.removeItem(EXTERNAL_CAPTURE_LOCATOR_KEY);
}

function sessionStorageKey(locator: string): string {
  return `${EXTERNAL_CAPTURE_SESSION_PREFIX}${locator}`;
}
