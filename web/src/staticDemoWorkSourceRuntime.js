// Browser-review transport only. The customer runtime never imports this file.
// Install before the evidence bootstrap; both old and new bundles then receive
// identical source records without changing their compiled application bytes.
(function installWorkSourceEvidence(global) {
  const scenario = new URLSearchParams(global.location.search).get("workSourceCase");
  if (!scenario) return;
  const asOf = "2026-10-31T10:00:00.123456Z";
  const pageAsOf = "2026-10-31T10:00:05.123456Z";
  const nextWeek = "2026-11-07T10:00:00.123456Z";
  const log = global.workSourceEvidence = {
    scenario, reads: [], commands: [], pending: 0,
    scope: { tenant_id: "bank-demo", legal_entity_id: "bank-ng", principal_id: "role-cro" },
    asOf, pageAsOf, totals: {},
  };
  const micros = (value) => {
    if (!value) return null;
    const fraction = /\.(\d+)Z$/.exec(value)?.[1] ?? "";
    return BigInt(Date.parse(value)) * 1000n + BigInt(fraction.padEnd(6, "0").slice(3, 6) || "0");
  };
  const row = (id, title, due_at, index = 0) => ({
    id, title, due_at, state: index % 3 === 0 ? "In progress" : "Ready",
    type: "FINDING_REMEDIATION", intervention_class: "REVIEW",
    owner: "Sample Risk Operations", scope: "Meridian Trust Bank Nigeria",
    evidence: "Review evidence received", why_now: "Review the assigned issue before its deadline.",
    primary_action: "Open issue", action_target_type: "MATTER", action_target_id: "matter-gaid-change",
    organization_scope_id: "sample-work-area",
  });
  const overdue = Array.from({ length: 60 }, (_, i) => row(`sample-overdue-${i + 1}`,
    `Sample overdue assignment ${String(i + 1).padStart(2, "0")}`, `2026-10-30T10:${String(i).padStart(2, "0")}:00Z`, i));
  overdue.push(row("sample-before-start", "Sample just before the deadline window", "2026-10-31T10:00:00.123455Z"));
  const upcoming = [row("sample-at-start", "Sample at the deadline window start", asOf),
    ...Array.from({ length: 60 }, (_, i) => row(`sample-upcoming-${i + 1}`,
      `Sample upcoming assignment ${String(i + 1).padStart(2, "0")}`, `2026-11-03T10:${String(i).padStart(2, "0")}:00Z`, i)),
    row("sample-at-end", "Sample at the deadline window end", nextWeek)];
  const population = scenario === "empty" ? [] : [...overdue, ...upcoming,
    row("sample-after-end", "Sample just after the deadline window", "2026-11-07T10:00:00.123457Z"),
    row("sample-undated", "Sample assignment without a deadline", "")];
  const loaded = scenario === "empty" ? [] : [overdue[0], upcoming[1], {
    ...row("sample-loaded-recovery", "Sample loaded recovery approval", ""), intervention_class: "AUTHORIZATION",
  }, { ...row("sample-loaded-evidence", "Sample loaded evidence request", ""), intervention_class: "EVIDENCE_EXCEPTION" }];
  const start = micros(asOf), end = micros(nextWeek);
  log.totals = {
    total: population.length,
    overdue: population.filter((item) => item.due_at && micros(item.due_at) < start).length,
    due_soon: population.filter((item) => item.due_at && micros(item.due_at) >= start && micros(item.due_at) <= end).length,
    undated: population.filter((item) => !item.due_at).length,
  };
  const json = (body, status = 200) => new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
  let currentFetch = global.fetch;
  Object.defineProperty(global, "fetch", {
    configurable: true,
    get: () => currentFetch,
    set: (delegate) => {
      currentFetch = async (input, init) => {
        const url = new URL(typeof input === "string" || input instanceof URL ? input : input.url, global.location.href);
        const method = String(init?.method ?? input?.method ?? "GET").toUpperCase();
        const source = url.pathname.startsWith("/api/v1/work/");
        const observed = source || url.pathname === "/api/v1/today";
        const entry = { path: url.pathname, query: Object.fromEntries(url.searchParams), method };
        if (observed) { log.reads.push(entry); log.pending++; }
        if (url.pathname.startsWith("/api/") && !["GET", "HEAD", "OPTIONS"].includes(method)) log.commands.push(entry);
        try {
          let body;
          if (method === "GET" && url.pathname === "/api/v1/today") {
            body = { items: loaded, generated_at: asOf, workflow_has_more: population.length > loaded.length, count_basis: "LOADED" };
          } else if (method === "GET" && url.pathname === "/api/v1/work/assignment-counts") {
            if (scenario === "counts-unavailable") { entry.status = 503; return json({ error: { code: "work_counts_unavailable", message: "Assignment totals unavailable." } }, 503); }
            body = { ...log.totals, as_of: asOf, coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "SOURCE_SNAPSHOT",
              scope: { ...log.scope, ...(scenario === "counts-scope" ? { tenant_id: "other-bank" } : {}) } };
          } else if (method === "GET" && url.pathname === "/api/v1/work/assignments") {
            const search = url.searchParams.get("q")?.trim().toLowerCase() ?? "";
            const status = url.searchParams.get("status"), from = micros(url.searchParams.get("due_from")), before = micros(url.searchParams.get("due_before"));
            const selected = population.filter((item) => (!search || item.title.toLowerCase().includes(search))
              && (!status || item.state.toUpperCase().replace(/\s+/g, "_") === status)
              && (from === null || item.due_at && micros(item.due_at) >= from)
              && (before === null || item.due_at && micros(item.due_at) < before));
            const query = JSON.stringify({ search, status, from: String(from), before: String(before), scope: log.scope });
            const rawCursor = url.searchParams.get("cursor");
            const cursor = rawCursor ? JSON.parse(atob(rawCursor)) : { offset: 0, query };
            if (cursor.query !== query) throw new Error("The Work evidence cursor was reused with changed filters or scope.");
            const limit = Math.min(100, Math.max(1, Number(url.searchParams.get("limit") ?? 50)));
            const items = selected.slice(cursor.offset, cursor.offset + limit);
            const hasMore = cursor.offset + limit < selected.length;
            body = { items, has_more: hasMore, next_cursor: hasMore ? btoa(JSON.stringify({ offset: cursor.offset + limit, query })) : undefined,
              as_of: pageAsOf, coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "NOT_AVAILABLE",
              scope: { ...log.scope, ...(scenario === "page-scope" ? { principal_id: "another-person" } : {}) } };
            entry.population = selected.map((item) => item.id);
          } else {
            const response = await delegate(input, init);
            if (url.pathname === "/api/v1/context" && response.ok) {
              const context = await response.clone().json();
              log.scope = { tenant_id: context.tenant.id, legal_entity_id: context.legal_entity.id, principal_id: context.actor.id };
            }
            return response;
          }
          entry.status = 200;
          entry.ids = body.items?.map((item) => item.id);
          entry.scope = body.scope;
          entry.asOf = body.as_of ?? body.generated_at;
          return json(body);
        } finally { if (observed) log.pending--; }
      };
    },
  });
})(globalThis);
