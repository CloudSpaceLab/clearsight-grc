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
        <p>Approved applications use this stable base URL and logical model aliases. Provider routes can change behind it without changing the application contract.</p>
      </div>
      <span className="ai-gateway-proxy__state">{loading ? "Checking…" : proxy.configured ? "Published" : "Not published"}</span>
    </div>

    {proxy.configured && proxy.base_url ? <div className="ai-gateway-proxy__endpoint">
      <code>{proxy.base_url}</code>
      <button className="secondary-button" type="button" onClick={() => void copyBaseURL()}>Copy base URL</button>
    </div> : <div className="ai-gateway-proxy__empty">
      <strong>No public proxy URL is configured for this deployment.</strong>
      <span>Set the deployment public gateway base URL before asking applications to route AI traffic through ClearSight.</span>
    </div>}

    <div className="ai-gateway-proxy__capabilities">
      <div><strong>Supported ingress</strong><span>Derived from the gateway's executable workload route registry.</span></div>
      <div className="ai-gateway-proxy__routes" aria-label="Supported gateway ingress routes">
        {proxy.ingress.map((route) => <code key={`${route.method}-${route.path}`}><b>{route.method}</b> {route.path}</code>)}
      </div>
    </div>

    <p className="ai-gateway-proxy__note">
      {runtimeStatus?.available
        ? "Runtime status is connected. Published endpoint metadata is separate from provider health and does not by itself prove a provider request will succeed."
        : "Runtime apply status is not currently available. Publishing a base URL does not imply the gateway or an upstream provider is healthy."}
    </p>
    {copyState !== "idle" && <span className="ai-gateway-proxy__copy-result" aria-live="polite">{copyState === "copied" ? "Base URL copied." : "Copy failed. Select the URL and copy it manually."}</span>}
  </section>;
}
