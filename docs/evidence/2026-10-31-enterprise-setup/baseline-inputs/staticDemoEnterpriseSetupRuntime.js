// Browser-review transport only. Inject before the isolated evidence bootstrap.
// Customer applications must never import this file or these synthetic records.
(function installEnterpriseSetupEvidence(global) {
  const scenario = new URLSearchParams(global.location.search).get("enterpriseSetupCase");
  if (!scenario) return;
  const now = "2026-10-31T10:00:00Z";
  const log = global.enterpriseSetupEvidence = {
    kind: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", scenario,
    reads: [], commands: [], pending: 0, overviewReads: 0,
    scope: { tenant_id: "bank-demo", legal_entity_id: "bank-ng", principal_id: "role-cro" },
  };
  const definitions = [
    ["head-office", "Head Office", "BUSINESS_UNIT", null, "HO"],
    ["technology", "Technology", "DEPARTMENT", "head-office", "TECH"],
    ["security", "Information Security", "FUNCTION", "technology", "SEC"],
    ["infrastructure", "Infrastructure", "FUNCTION", "technology", "INFRA"],
    ["risk", "Risk & Compliance", "DEPARTMENT", "head-office", "RISK"],
    ["finance", "Finance", "DEPARTMENT", "head-office", "FIN"],
    ["people", "People & Culture", "DEPARTMENT", "head-office", "PEOPLE"],
    ["payments", "Payments processing", "CRITICAL_SERVICE", "head-office", "PAY"],
    ["victoria-island", "Victoria Island", "BRANCH", null, "VI"],
    ["lekki", "Lekki", "BRANCH", null, "LEK"],
    ["abuja", "Abuja Central", "BRANCH", null, "ABJ"],
    ["imported", "Imported operations", "ORGANIZATION_UNIT", null, "IMPORTED"],
  ];
  const id = name => "sample-org-" + name;
  function records() {
    const byName = new Map(definitions.map(row => [row[0], row]));
    const path = name => { const row = byName.get(name); return [...(row[3] ? path(row[3]) : ["BANK"]), row[4]]; };
    const scopes = definitions.map(([name, label, kind, parent, code]) => ({
      id: id(name), legal_entity_id: log.scope.legal_entity_id, code, name: label, kind,
      ...(parent ? { parent_scope_id: id(parent) } : {}), department_path: path(name),
      origin: name === "imported" ? "LEGACY_DEPARTMENT_PATH" : "MANAGED",
      status: "ACTIVE", valid_from: "2026-10-01T00:00:00Z", version: 3,
    }));
    if (scenario === "hierarchy-gaps") scopes.push(
      { ...scopes[0], id: id("outside-parent"), name: "Area with parent outside this list", code: "OUTSIDE", parent_scope_id: id("not-loaded"), department_path: ["BANK", "OUTSIDE"] },
      { ...scopes[0], id: id("cycle-a"), name: "Review area A", code: "CYCLE-A", parent_scope_id: id("cycle-b"), department_path: ["BANK", "CYCLE-A"] },
      { ...scopes[0], id: id("cycle-b"), name: "Review area B", code: "CYCLE-B", parent_scope_id: id("cycle-a"), department_path: ["BANK", "CYCLE-B"] },
    );
    const positions = [
      { id: "sample-org-position-cro", code: "CRO", title: "Chief Risk Officer", organization_scope_id: id("risk"), department_path: path("risk"), occupant_principal_id: "role-cro", occupant_name: "Amina Yusuf", role_codes: ["RISK_REVIEWER"] },
      { id: "sample-org-position-security", code: "SEC-LEAD", title: "Information Security Lead", organization_scope_id: id("security"), department_path: path("security"), occupant_principal_id: "sample-person-chidi", occupant_name: "Chidi Okonkwo", role_codes: ["RISK_REVIEWER"] },
      { id: "sample-org-position-branch", code: "VI-RISK", title: "Branch Risk Coordinator", organization_scope_id: id("victoria-island"), department_path: path("victoria-island"), role_codes: [] },
    ].map(row => ({ ...row, valid_from: "2026-10-01T00:00:00Z", version: 2 }));
    const revision = (name, maker) => ({ id: "sample-org-revision-" + name, scope_id: id(name), operation: "UPDATE", base_version: 3,
      proposed_name: name === "technology" ? "Technology services" : "Finance operations", maker_id: maker, status: "PENDING",
      impact: { child_scopes: name === "technology" ? 2 : 0, positions: 0, access_mappings: 0, open_matters: 0 }, created_at: now });
    return { scopes, positions, revisions: [revision("technology", "sample-other-maker"), revision("finance", log.scope.principal_id)] };
  }
  const json = (body, status = 200) => new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
  let currentFetch = global.fetch;
  Object.defineProperty(global, "fetch", {
    configurable: true,
    get: () => currentFetch,
    set: delegate => {
      currentFetch = async (input, init) => {
        const url = new URL(typeof input === "string" || input instanceof URL ? input : input.url, global.location.href);
        const method = String(init?.method ?? input?.method ?? "GET").toUpperCase();
        const observed = url.pathname === "/api/v1/access/overview";
        const entry = { path: url.pathname, query: Object.fromEntries(url.searchParams), method };
        if (url.pathname.startsWith("/api/") && !["GET", "HEAD", "OPTIONS"].includes(method)) {
          log.commands.push(entry);
          return json({ error: { code: "evidence_command_blocked", message: "This review fixture does not execute commands." } }, 403);
        }
        if (observed) { log.reads.push(entry); log.pending++; log.overviewReads++; }
        try {
          if (observed && (scenario === "denied" || scenario === "failure" || scenario === "recovery" && log.overviewReads === 1)) {
            entry.status = scenario === "denied" ? 403 : 503;
            return json({ error: { code: scenario === "denied" ? "access_denied" : "access_unavailable", message: "Organization settings could not be loaded." } }, entry.status);
          }
          const response = await delegate(input, init);
          if (url.pathname === "/api/v1/context" && response.ok) {
            const context = await response.clone().json();
            log.scope = { tenant_id: context.tenant.id, legal_entity_id: context.legal_entity.id, principal_id: context.actor.id };
          }
          if (!observed || !response.ok) { if(observed) entry.status=response.status; return response; }
          const base = await response.json(), data = records(), empty = scenario === "empty";
          const result = { ...base, actor_principal_id: log.scope.principal_id, can_configure_organization: scenario !== "readonly",
            organization_scopes: empty ? [] : data.scopes, organization_scopes_truncated: scenario === "truncated",
            positions: empty ? [] : data.positions, organization_scope_revisions: empty ? [] : data.revisions,
            organization_position_revisions: [], organization_position_history: [], organization_position_role_revisions: [],
            data_boundary_revisions: [],
          };
          entry.status = 200; entry.scope = { ...log.scope }; entry.scopeIDs = result.organization_scopes.map(row => row.id);
          entry.positionIDs = result.positions.map(row => row.id); entry.revisionIDs = result.organization_scope_revisions.map(row => row.id);
          entry.truncated = result.organization_scopes_truncated; entry.canConfigure = result.can_configure_organization;
          return json(result);
        } finally { if (observed) log.pending--; }
      };
    },
  });
})(globalThis);
