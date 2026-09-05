export function consumeCaptureInvitation(browser: Pick<Window, "history" | "location">): string | null {
  const url = new URL(browser.location.href);
  const fragment = new URLSearchParams(url.hash.replace(/^#/, ""));
  const fragmentToken = fragment.get("form_access");
  if (fragmentToken !== null) {
    fragment.delete("form_access");
    const remainingFragment = fragment.toString();
    url.hash = remainingFragment ? `#${remainingFragment}` : "";
    browser.history.replaceState(browser.history.state, "", `${url.pathname}${url.search}${url.hash}`);
    return fragmentToken;
  }

  return null;
}
