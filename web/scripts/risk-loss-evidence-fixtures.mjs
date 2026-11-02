// Dedicated test-only Risk records. Other families reuse their existing fixture
// payloads unchanged; this observer records the real App's reads and commands.
export const riskLossFixtureRisks = [
  ["network", "Sample network resilience", "NET-RES-01", "BREACHED", "High"],
  ["settlement", "Sample settlement processing", "OPS-SET-02", "APPROACHING", "Moderate"],
  ["access", "Sample privileged access review", "CYB-ACC-03", "WITHIN", "Low"],
  ["supplier", "Sample supplier continuity", "TPR-CON-04", "UNKNOWN", "Unknown"],
].map(([key, name, code, position, rating], index) => {
  const risk = { id: `sample-risk-${key}`, tenant_id: "bank-demo", legal_entity_id: "bank-ng", organization_scope_id: "sample-scope-risk-ops",
    code, name, category: index === 2 ? "Cyber" : "Operational resilience", statement: "Sample record: interruption may exceed the approved service tolerance.",
    cause: "Service dependency failure", event: "Service interruption", impact: "Customer service interruption and recovery cost.", scope: { service: name },
    owner_principal_id: "role-cro", status: "ACTIVE", version: 3, created_at: "2026-10-01T08:00:00Z", updated_at: "2026-10-05T09:00:00Z" };
  const active_appetite = { id: `sample-appetite-${key}`, risk_id: risk.id, risk_version: 3, version: 2, statement: "Service recovery within the approved tolerance.",
    rule: { max_minutes: 30 }, rationale: "Protect customer services.", owner_principal_id: "role-cro", authority_principal_id: "role-cro", status: "ACTIVE",
    effective_from: "2026-10-01T08:00:00Z", created_at: "2026-10-01T08:00:00Z" };
  const latest_assessment = position === "UNKNOWN" ? undefined : { id: `sample-assessment-${key}`, risk_id: risk.id, risk_version: 3, kind: "RESIDUAL", method_code: "QUAL-5X5", method_version: "v1",
    dimensions: { likelihood: 4 - index, impact: 5 - index, risk_level: rating }, assumptions: {}, evidence_references: [], assessed_by: "role-cro",
    appetite_statement_id: active_appetite.id, appetite_position: position, appetite_rationale: "Sample assessment against the approved tolerance.", assessed_at: "2026-10-05T09:00:00Z", created_at: "2026-10-05T09:00:00Z" };
  return { risk, active_appetite, latest_assessment };
});

export async function installRiskLossObservation(page, scenario) {
  await page.evaluate(({ view, check, risks }) => {
    const original = window.fetch;
    const log = { view, check, reads: [], commands: [], pending: 0, documentToken: `risk-loss-${Math.random()}`, responsePopulation: [], responses: [] };
    window.riskLossEvidence = log;
    const endpoint = { risks: "/api/v1/risks", rcsa: "/api/v1/rcsa/cycles", insights: "/api/v1/risk-indicators", losses: "/api/v1/losses" }[view];
    const json = (body, status = 200) => new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
    window.fetch = async (input, init) => {
      const url = new URL(typeof input === "string" || input instanceof URL ? input : input.url, location.href);
      const method = String(init?.method ?? (typeof input === "object" ? input.method : null) ?? "GET").toUpperCase();
      const relevant = /^\/api\/v1\/(?:risks|rcsa\/cycles|risk-indicators|losses|monitoring-checks)/.test(url.pathname);
      const entry = { path: url.pathname, query: Object.fromEntries(url.searchParams), method };
      if (relevant && method === "GET") log.reads.push(entry);
      else if (!["GET", "HEAD", "OPTIONS"].includes(method) && url.pathname.startsWith("/api/v1/") && url.pathname !== "/api/v1/authority/resolve") log.commands.push(entry);
      if (relevant) log.pending += 1;
      try {
        const list = url.pathname === endpoint && method === "GET" && !url.searchParams.has("check_id");
        if (list && check === "denied") { entry.status = 403; return json({ error: { code: "permission_denied", message: "Sample records are unavailable." } }, 403); }
        let response;
        if (method === "GET" && url.pathname === "/api/v1/risks") {
          let items = risks.filter((item) => (!url.searchParams.has("search") || `${item.risk.name} ${item.risk.code} ${item.risk.category}`.toLowerCase().includes(url.searchParams.get("search").toLowerCase()))
            && (!url.searchParams.has("status") || item.risk.status === url.searchParams.get("status"))
            && (!url.searchParams.has("appetite_position") || (item.latest_assessment?.appetite_position ?? "UNKNOWN") === url.searchParams.get("appetite_position")));
          response = json({ items });
        } else if (method === "GET" && risks.some((item) => url.pathname === `/api/v1/risks/${item.risk.id}`)) {
          const item = risks.find((item) => url.pathname === `/api/v1/risks/${item.risk.id}`);
          response = json({ risk: item.risk, owner_display_name: "Sample Risk Operations", assessments: item.latest_assessment ? [item.latest_assessment] : [], appetite: [item.active_appetite], active_appetite: item.active_appetite,
            controls: [], control_details: [], control_details_complete: true, indicators: [], indicator_details: [], indicator_details_complete: true, notification_history: [], notification_history_complete: true });
        } else response = await original(input, init);
        entry.status = response.status;
        if (!list || !response.ok) return response;
        const body = await response.clone().json();
        if (check === "empty") body.items = [];
        if (check === "partial") {
          body.next_cursor = "sample-risk-loss-next-page";
          if (view === "rcsa" || view === "insights") body.complete = false;
          if (view === "risks") { body.items[0].latest_assessment.risk_version = 2; }
          if (view === "insights") { body.items[0].indicator.native_measurement = undefined; body.items[0].indicator.score = undefined; body.items[0].indicator.state = "UNKNOWN"; }
          if (view === "losses") {
            const foreign = structuredClone(body.items[0]); foreign.loss.id = "sample-loss-usd"; foreign.loss.code = "LOSS-USD-02"; foreign.loss.title = "Sample foreign-currency settlement loss";
            foreign.loss.currency = "USD"; foreign.loss.gross_amount_minor = 120000; foreign.totals = { currency: "USD", gross_amount_minor: 120000, recovered_amount_minor: 20000, net_loss_minor: 100000, recovery_status: "PARTIAL" };
            body.items.push(foreign);
          }
        }
        if (check === "filters") {
          if (view === "losses" && body.items.length === 1) {
            const second = structuredClone(body.items[0]); second.loss.id = "sample-loss-second"; second.loss.code = "LOSS-SAMPLE-02"; second.loss.title = "Sample duplicate payment correction"; body.items.push(second);
          }
          if (url.searchParams.has("cursor")) { body.items = body.items.slice(1); delete body.next_cursor; }
          else { if (body.items.length > 1) body.next_cursor = "sample-risk-loss-next-page"; body.items = body.items.slice(0, 1); }
        }
        log.responsePopulation = body.items.map((item) => (item.risk ?? item.cycle ?? item.indicator ?? item.loss).id ?? item.indicator?.check_id);
        log.responses.push({ path: url.pathname, query: entry.query, ids: log.responsePopulation,
          titles: body.items.map((item) => item.risk?.name ?? item.cycle?.name ?? item.indicator?.check_name ?? item.loss?.title),
          complete: body.complete ?? null, nextCursor: body.next_cursor ?? null });
        return json(body);
      } finally { if (relevant) log.pending -= 1; }
    };
  }, { view: scenario.riskLossView, check: scenario.riskLossCheck, risks: riskLossFixtureRisks });
}
