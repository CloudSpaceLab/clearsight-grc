import { StaticDemoHTTPError, staticDemoEnabled, staticDemoRequest } from "./staticDemo";
import { staticExternalCaptureRequest } from "./staticExternalCapture";

if (staticDemoEnabled) {
  const nativeFetch = globalThis.fetch.bind(globalThis);
  globalThis.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
    const raw = typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;
    const url = new URL(raw, window.location.origin);
    if (!url.pathname.startsWith("/api/")) return nativeFetch(input, init);
    try {
      const external = await staticExternalCaptureRequest(`${url.pathname}${url.search}`, init);
      const body = external === undefined ? await staticDemoRequest<unknown>(`${url.pathname}${url.search}`, init) : external;
      return jsonResponse(body, 200);
    } catch (cause) {
      if (cause instanceof StaticDemoHTTPError) return jsonResponse({ error: { code: cause.code, message: cause.message } }, cause.status);
      if (isStaticExternalError(cause)) return jsonResponse({ error: { code: cause.staticCode, message: cause.message } }, cause.staticStatus);
      return jsonResponse({ error: { code: "static_demo_failed", message: cause instanceof Error ? cause.message : "Static demo request failed" } }, 503);
    }
  };
}

function isStaticExternalError(cause: unknown): cause is Error & { staticStatus: number; staticCode: string } {
  return cause instanceof Error && "staticStatus" in cause && "staticCode" in cause;
}

function jsonResponse(body: unknown, status: number) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}
