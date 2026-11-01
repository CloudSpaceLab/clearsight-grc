// Synthetic browser-review transport. Never import from the customer entry.
(function installAutomationEvidence(global) {
  const scenario = new URLSearchParams(global.location.search).get("automationCase");
  if (!scenario) return;
  const log = global.automationEvidence = {
    kind: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", scenario,
    reads: [], commands: [], pending: 0, contextRead: null, scope: null,
  };
  const definitions = [
    ["evidence", "EVIDENCE-REFRESH", "Evidence refresh", "REVERSIBLE_WRITE", "ACTIVE"],
    ["routing", "ROUTING-REVIEW", "Routing review", "OBSERVE", "DRAFT"],
    ["source", "SOURCE-CHECK", "Source availability check", "OBSERVE", "ACTIVE"],
    ["vendor", "VENDOR-RENEWAL", "Vendor renewal request", "REVERSIBLE_WRITE", "PENDING_APPROVAL"],
    ["control", "CONTROL-REVIEW", "Control review reminder", "NOTIFY", "SUSPENDED"],
    ["policy", "POLICY-REVIEW", "Policy review reminder", "NOTIFY", "APPROVED"],
  ];
  function policies() {
    const values = definitions.map(([key, code, name, action_class, status], index) => ({
      id: "sample-auto-" + key, tenant_id: log.scope.tenant_id, code, name, action_class, status,
      version: index + 1, record_version: index + 2, rollout_mode: "SHADOW",
      eligibility: { required_state: "CURRENT", sensitivity: "INTERNAL", materiality_max: 2 },
      blast_radius_limit: { max_records: index + 10, external_side_effects: false },
      verification_contract: { method: "source_recheck", required: true },
      maker_id: "sample-auto-maker", checker_id: status === "DRAFT" ? undefined : "sample-auto-checker",
      created_at: "2026-10-01T09:00:00Z", updated_at: "2026-10-31T10:00:00Z",
      effective_from: "2026-10-15T00:00:00Z", effective_until: "2027-10-15T00:00:00Z",
    }));
    if (scenario === "long") {
      values[0].name = "Evidence refresh for international settlement operations and independent business continuity review";
      values[0].code = "LONG-UNBROKEN-REFERENCE-" + "A".repeat(120);
      values[0].verification_contract = { method: "source_recheck", reference: "Z".repeat(240), required: true };
      values[0].eligibility = { ["K".repeat(160)]: "Recorded condition" };
    }
    if (scenario === "truncated") {
      for (let index = values.length; index < 200; index++) values.push({ ...values[0], id: "sample-auto-" + index, code: "POLICY-" + String(index).padStart(3, "0"), name: "Additional policy " + index });
    }
    return scenario === "empty" ? [] : values;
  }
  let release;
  const loading = new Promise(resolve => { release = resolve; });
  log.release = () => release();
  const json = (body, status = 200) => new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
  let currentFetch = global.fetch;
  Object.defineProperty(global, "fetch", {
    configurable: true, get: () => currentFetch,
    set: delegate => {
      currentFetch = async (input, init) => {
        const url = new URL(typeof input === "string" || input instanceof URL ? input : input.url, global.location.href);
        const method = String(init?.method ?? input?.method ?? "GET").toUpperCase();
        const entry = { path: url.pathname, method, query: Object.fromEntries(url.searchParams) };
        if (url.pathname.startsWith("/api/") && !["GET", "HEAD", "OPTIONS"].includes(method)) {
          log.commands.push(entry);
          return json({ error: { code: "evidence_command_blocked", message: "This review fixture cannot execute commands." } }, 403);
        }
        if (url.pathname === "/api/v1/context") {
          const response = await delegate(input, init);
          if (!response.ok) return response;
          const context = await response.json();
          if (scenario === "readonly") context.capabilities.config_write = false;
          log.scope = { tenant_id: context.tenant.id, legal_entity_id: context.legal_entity.id, principal_id: context.actor.id };
          log.contextRead = { path: url.pathname, status: response.status, scope: { ...log.scope } };
          return json(context);
        }
        if (url.pathname !== "/api/v1/compliance/automation-policies") return delegate(input, init);
        log.reads.push(entry); log.pending++;
        try {
          if (scenario === "loading") await loading;
          if (scenario === "denied" || scenario === "failure" || scenario === "recovery" && log.reads.length === 1) {
            entry.status = scenario === "denied" ? 403 : 503;
            return json({ error: { code: scenario === "denied" ? "access_denied" : "automation_unavailable", message: "Automation policies could not be loaded." } }, entry.status);
          }
          if (!log.contextRead || !Object.values(log.scope ?? {}).every(value => typeof value === "string" && value)) {
            entry.status = 503;
            return json({ error: { code: "evidence_context_missing", message: "The current identity context is unavailable." } }, 503);
          }
          const items = policies();
          const page = { items, tenant_id: log.scope.tenant_id, scope: "TENANT", revision_basis: "LATEST_PER_CODE", limit: 200, returned_count: items.length, truncated: scenario === "truncated" };
          Object.assign(entry, { status: 200, scope: { ...log.scope }, policyIDs: items.map(item => item.id), count: items.length, truncated: page.truncated, revisionBasis: page.revision_basis });
          return json(page);
        } finally { log.pending--; }
      };
    },
  });
})(globalThis);
