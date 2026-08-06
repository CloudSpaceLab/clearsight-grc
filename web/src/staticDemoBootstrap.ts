import { staticDemoEnabled, staticDemoRequest } from "./staticDemo";

if (staticDemoEnabled) {
  const nativeFetch = globalThis.fetch.bind(globalThis);
  globalThis.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
    const raw = typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;
    const url = new URL(raw, window.location.origin);
    if (!url.pathname.startsWith("/api/")) return nativeFetch(input, init);
    try {
      const body = await staticDemoRequest<unknown>(`${url.pathname}${url.search}`, init);
      return new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } });
    } catch (cause) {
      return new Response(JSON.stringify({ error: { message: cause instanceof Error ? cause.message : "Static demo request failed" } }), { status: 503, headers: { "Content-Type": "application/json" } });
    }
  };
}
