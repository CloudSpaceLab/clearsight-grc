import { useMemo, useState } from "react";
import type { FormEvent } from "react";
import type {
  GatewayEnvironment,
  GatewayModelConfig,
  GatewayProviderConfig,
  GatewayProviderKind,
  GatewayTransportDefinition,
} from "../../aiGatewayTransportApi";

const newProvider = (): GatewayProviderConfig => ({
  id: "openai-primary",
  name: "Primary OpenAI",
  kind: "OPENAI",
  base_url: "https://api.openai.com",
  secret_ref: "env:OPENAI_API_KEY",
  timeout_ms: 90000,
  require_usage: true,
  regions: [],
  state: "ENABLED",
});

const newModel = (providerId: string): GatewayModelConfig => ({
  alias: "safe-chat",
  routes: [{
    id: "safe-chat-primary",
    provider_id: providerId,
    model: "gpt-5",
    weight: 100,
    input_microusd_per_million_tokens: 0,
    output_microusd_per_million_tokens: 0,
  }],
});

export function AIGatewayTransportDraftForm({
  environment,
  busy,
  disabled,
  onSubmit,
}: {
  environment: GatewayEnvironment;
  busy: boolean;
  disabled: boolean;
  onSubmit: (definition: GatewayTransportDefinition, changeReason: string) => Promise<void>;
}) {
  const initialProvider = useMemo(() => newProvider(), []);
  const [providers, setProviders] = useState<GatewayProviderConfig[]>([initialProvider]);
  const [models, setModels] = useState<GatewayModelConfig[]>([newModel(initialProvider.id)]);
  const [failureThreshold, setFailureThreshold] = useState(3);
  const [openDurationMs, setOpenDurationMs] = useState(30000);
  const [changeReason, setChangeReason] = useState("Establish governed organization AI provider routing");

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (busy || disabled) return;
    await onSubmit({
      circuit_breaker: { failure_threshold: failureThreshold, open_duration_ms: openDurationMs },
      providers: providers.map((provider) => ({ ...provider, regions: provider.regions?.filter(Boolean) ?? [] })),
      models,
    }, changeReason);
  }

  function updateProvider(index: number, patch: Partial<GatewayProviderConfig>) {
    setProviders((current) => current.map((provider, position) => position === index ? { ...provider, ...patch } : provider));
  }

  function removeProvider(index: number) {
    const removed = providers[index];
    if (!removed || providers.length === 1) return;
    setProviders((current) => current.filter((_, position) => position !== index));
    setModels((current) => current.map((model) => ({
      ...model,
      routes: model.routes.filter((route) => route.provider_id !== removed.id),
    })).filter((model) => model.routes.length > 0));
  }

  function addProvider() {
    const suffix = providers.length + 1;
    setProviders((current) => [...current, {
      ...newProvider(),
      id: `provider-${suffix}`,
      name: `Provider ${suffix}`,
      secret_ref: `env:AI_PROVIDER_${suffix}_KEY`,
    }]);
  }

  function updateModel(index: number, patch: Partial<GatewayModelConfig>) {
    setModels((current) => current.map((model, position) => position === index ? { ...model, ...patch } : model));
  }

  function updateRoute(modelIndex: number, routeIndex: number, patch: Partial<GatewayModelConfig["routes"][number]>) {
    setModels((current) => current.map((model, position) => position !== modelIndex ? model : {
      ...model,
      routes: model.routes.map((route, routePosition) => routePosition === routeIndex ? { ...route, ...patch } : route),
    }));
  }

  function addRoute(modelIndex: number) {
    const providerId = providers[0]?.id ?? "";
    setModels((current) => current.map((model, position) => position !== modelIndex ? model : {
      ...model,
      routes: [...model.routes, {
        id: `${model.alias || "model"}-route-${model.routes.length + 1}`,
        provider_id: providerId,
        model: "upstream-model",
        weight: 1,
        input_microusd_per_million_tokens: 0,
        output_microusd_per_million_tokens: 0,
      }],
    }));
  }

  function addModel() {
    const providerId = providers[0]?.id ?? "";
    const suffix = models.length + 1;
    setModels((current) => [...current, {
      alias: `model-${suffix}`,
      routes: [{
        id: `model-${suffix}-primary`,
        provider_id: providerId,
        model: "upstream-model",
        weight: 100,
        input_microusd_per_million_tokens: 0,
        output_microusd_per_million_tokens: 0,
      }],
    }]);
  }

  return <form className="ai-gateway-transport__form" onSubmit={submit}>
    <section className="ai-gateway-transport__section" aria-labelledby="gateway-provider-heading">
      <div className="ai-gateway-transport__section-header">
        <div><span className="eyebrow">Upstream boundary</span><h4 id="gateway-provider-heading">Providers</h4><p>Register fixed provider origins and secret-manager references. Credentials never enter ClearSight.</p></div>
        <button className="secondary-button" type="button" onClick={addProvider} disabled={busy}>Add provider</button>
      </div>
      <div className="ai-gateway-transport__cards">
        {providers.map((provider, index) => <div className="ai-gateway-transport__card" key={`${index}-${provider.id}`}>
          <div className="ai-gateway-transport__card-title"><strong>{provider.name || `Provider ${index + 1}`}</strong>{providers.length > 1 && <button type="button" className="text-button" onClick={() => removeProvider(index)}>Remove</button>}</div>
          <div className="ai-gateway-transport__grid">
            <label><span>Display name</span><input value={provider.name} maxLength={160} onChange={(event) => updateProvider(index, { name: event.target.value })} required/></label>
            <label><span>Provider ID</span><input value={provider.id} maxLength={128} onChange={(event) => updateProvider(index, { id: normalizeIdentifier(event.target.value) })} required/></label>
            <label><span>Adapter</span><select value={provider.kind} onChange={(event) => updateProvider(index, { kind: event.target.value as GatewayProviderKind })}><option value="OPENAI">OpenAI-compatible</option><option value="ANTHROPIC">Anthropic</option></select></label>
            <label><span>State</span><select value={provider.state} onChange={(event) => updateProvider(index, { state: event.target.value as "ENABLED" | "SUSPENDED" })}><option value="ENABLED">Enabled</option><option value="SUSPENDED">Suspended</option></select></label>
          </div>
          <label><span>Fixed provider origin</span><input type="url" value={provider.base_url} placeholder="https://api.example.com" onChange={(event) => updateProvider(index, { base_url: event.target.value })} required/><small>{environment === "PRODUCTION" ? "Production accepts HTTPS origins only." : "HTTPS is preferred; loopback HTTP is allowed only outside production."}</small></label>
          <div className="ai-gateway-transport__grid">
            <label><span>Secret reference</span><input value={provider.secret_ref} placeholder="env:OPENAI_API_KEY" onChange={(event) => updateProvider(index, { secret_ref: event.target.value })} required/><small>Reference only. Never paste a key or token.</small></label>
            <label><span>Regions</span><input value={(provider.regions ?? []).join(", ")} placeholder="US, EU" onChange={(event) => updateProvider(index, { regions: event.target.value.split(",").map((value) => normalizeIdentifier(value.trim().toUpperCase())).filter(Boolean) })}/></label>
          </div>
          <details className="ai-gateway-transport__advanced"><summary>Transport details</summary><div className="ai-gateway-transport__grid">
            <label><span>API version</span><input value={provider.api_version ?? ""} maxLength={128} onChange={(event) => updateProvider(index, { api_version: event.target.value })}/></label>
            <label><span>Timeout (ms)</span><input type="number" min={1000} max={600000} value={provider.timeout_ms ?? 90000} onChange={(event) => updateProvider(index, { timeout_ms: Number(event.target.value) })}/></label>
          </div></details>
        </div>)}
      </div>
    </section>

    <section className="ai-gateway-transport__section" aria-labelledby="gateway-model-heading">
      <div className="ai-gateway-transport__section-header"><div><span className="eyebrow">Application contract</span><h4 id="gateway-model-heading">Logical models</h4><p>Applications request stable aliases. Only approved routes decide the actual provider/model.</p></div><button className="secondary-button" type="button" onClick={addModel} disabled={busy}>Add alias</button></div>
      <div className="ai-gateway-transport__cards">
        {models.map((model, modelIndex) => <div className="ai-gateway-transport__card" key={`${modelIndex}-${model.alias}`}>
          <label><span>Alias exposed to applications</span><input value={model.alias} onChange={(event) => updateModel(modelIndex, { alias: normalizeIdentifier(event.target.value) })} required/></label>
          <div className="ai-gateway-transport__routes">
            {model.routes.map((route, routeIndex) => <div className="ai-gateway-transport__route" key={`${routeIndex}-${route.id}`}>
              <div className="ai-gateway-transport__grid">
                <label><span>Route ID</span><input value={route.id} onChange={(event) => updateRoute(modelIndex, routeIndex, { id: normalizeIdentifier(event.target.value) })} required/></label>
                <label><span>Provider</span><select value={route.provider_id} onChange={(event) => updateRoute(modelIndex, routeIndex, { provider_id: event.target.value })}>{providers.map((provider) => <option value={provider.id} key={provider.id}>{provider.name || provider.id}</option>)}</select></label>
                <label><span>Upstream model</span><input value={route.model} onChange={(event) => updateRoute(modelIndex, routeIndex, { model: normalizeIdentifier(event.target.value) })} required/></label>
                <label><span>Weight</span><input type="number" min={1} max={100000} value={route.weight} onChange={(event) => updateRoute(modelIndex, routeIndex, { weight: Number(event.target.value) })}/></label>
              </div>
              <details className="ai-gateway-transport__advanced"><summary>Cost metadata</summary><div className="ai-gateway-transport__grid">
                <label><span>Input µUSD / 1M tokens</span><input type="number" min={0} value={route.input_microusd_per_million_tokens} onChange={(event) => updateRoute(modelIndex, routeIndex, { input_microusd_per_million_tokens: Number(event.target.value) })}/></label>
                <label><span>Output µUSD / 1M tokens</span><input type="number" min={0} value={route.output_microusd_per_million_tokens} onChange={(event) => updateRoute(modelIndex, routeIndex, { output_microusd_per_million_tokens: Number(event.target.value) })}/></label>
              </div></details>
            </div>)}
          </div>
          <button className="text-button" type="button" onClick={() => addRoute(modelIndex)} disabled={busy}>Add fallback route</button>
        </div>)}
      </div>
    </section>

    <details className="ai-gateway-transport__advanced ai-gateway-transport__section"><summary>Circuit breaker</summary><div className="ai-gateway-transport__grid">
      <label><span>Failure threshold</span><input type="number" min={1} max={100} value={failureThreshold} onChange={(event) => setFailureThreshold(Number(event.target.value))}/></label>
      <label><span>Open duration (ms)</span><input type="number" min={100} max={600000} value={openDurationMs} onChange={(event) => setOpenDurationMs(Number(event.target.value))}/></label>
    </div></details>

    <label><span>Change reason</span><textarea rows={3} maxLength={1000} value={changeReason} onChange={(event) => setChangeReason(event.target.value)} required/><small>Stored with the governed revision for reviewer context and audit reconstruction.</small></label>
    <div className="ai-gateway-transport__actions"><button className="primary-button" type="submit" disabled={busy || disabled || !changeReason.trim()}>{busy ? "Creating…" : `Create ${environment.toLowerCase()} revision`}</button><span>Draft only · independent approval required</span></div>
  </form>;
}

function normalizeIdentifier(value: string) {
  return value.trim().replace(/[^A-Za-z0-9._:/-]/g, "-").replace(/-+/g, "-");
}
