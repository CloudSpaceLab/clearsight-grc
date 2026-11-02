import { useState } from "react";
import type { GatewayProxyInfo, GatewayRuntimeStatus } from "../../aiGatewayTransportApi";
import "./AIGatewayProxyCard.css";

export function AIGatewayProxyCard({
  proxy,
  runtimeStatus,
  loading,
}: {
  proxy: GatewayProxyInfo;
  runtimeStatus: GatewayRuntimeStatus | null;
  loading: boolean;
}) {
  const [copyState, setCopyState] = useState<"idle" | "copied" | "failed">("idle");

  async function copyBaseURL() {
    if (!proxy.base_url) return;
    try {
      await navigator.clipboard.writeText(proxy.base_url);
      setCopyState("copied");
    } catch {
      setCopyState("failed");
    }
  }

  return <section className="ai-gateway-proxy" aria-labelledby="ai-gateway-proxy-heading">
    <div className="ai-gateway-proxy__heading">
      <div>
        <span className="eyebrow">Application endpoint</span>
        <h4 id="ai-gateway-proxy-heading">Governed proxy</h4>
        <p>Gateway URL and model aliases for approved applications.</p>
      </div>
      <span className="ai-gateway-proxy__state">{loading ? "Checking…" : proxy.configured ? "Published" : "Not published"}</span>
    </div>

    {proxy.configured && proxy.base_url ? <div className="ai-gateway-proxy__endpoint">
      <code>{proxy.base_url}</code>
      <button className="secondary-button" type="button" onClick={() => void copyBaseURL()}>Copy base URL</button>
    </div> : <div className="ai-gateway-proxy__empty">
      <strong>No public gateway URL configured.</strong>
      <span>Set a public gateway URL before connecting applications.</span>
    </div>}

    <div className="ai-gateway-proxy__capabilities">
      <div><strong>Supported ingress</strong><span>Available application routes.</span></div>
      <div className="ai-gateway-proxy__routes" aria-label="Supported gateway ingress routes">
        {proxy.ingress.map((route) => <code key={`${route.method}-${route.path}`}><b>{route.method}</b> {route.path}</code>)}
      </div>
    </div>

    <p className="ai-gateway-proxy__note">
      {runtimeStatus?.available
        ? "Runtime connected. Check provider health separately."
        : "Runtime status unavailable. Check gateway and provider health."}
    </p>
    {copyState !== "idle" && <span className="ai-gateway-proxy__copy-result" aria-live="polite">{copyState === "copied" ? "Base URL copied." : "Copy failed. Select the URL and copy it manually."}</span>}
  </section>;
}
