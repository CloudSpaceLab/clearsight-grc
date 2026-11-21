import { homeEvidenceMetrics, homeEvidenceSnapshot } from "./components/oversight/OversightEvidencePage";

// Installed only by the evidence entrypoint for named non-production fixtures.
export function installHomeShellEvidence(fixture: string | null) {
  if (fixture !== "home-shell" && fixture !== "home-shell-unavailable" && fixture !== "home-shell-insights") return;
  const previous = globalThis.fetch.bind(globalThis);
  globalThis.fetch = async (input, init) => {
    const raw = typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;
    const url = new URL(raw, window.location.origin);
    if (url.pathname === "/api/v1/context") {
      const response = await previous(input, init);
      if (!response.ok) return response;
      const context = await response.json();
      return Response.json({ ...context, tenant: { ...context.tenant, name: "Sample · Meridian Trust Bank" },
        capabilities: { ...context.capabilities, oversight_read: true } });
    }
    if (fixture === "home-shell-insights" && url.pathname === "/api/v1/metrics/domain") {
      if (url.searchParams.has("organization_scope_id")) return Response.json({ error: { message: "Sample data unavailable for this scope." } }, { status: 403 });
      return Response.json(sampleDomainMetrics());
    }
    if (url.pathname === "/api/v1/oversight" || url.pathname === "/api/v1/metrics/home") {
      if (fixture === "home-shell-unavailable") return Response.json({ error: { message: "Unavailable" } }, { status: 503 });
      const period = homeEvidenceSnapshot.reporting_period;
      if (url.searchParams.has("organization_scope_id") ||
          (url.searchParams.has("start_date") && url.searchParams.get("start_date") !== period.start_date) ||
          (url.searchParams.has("end_date") && url.searchParams.get("end_date") !== period.end_date)) {
        return Response.json({ error: { message: "Sample data unavailable for this scope or period." } }, { status: 403 });
      }
      return Response.json(url.pathname.endsWith("/oversight") ? homeEvidenceSnapshot : homeEvidenceMetrics);
    }
    return previous(input, init);
  };
}

function sampleDomainMetrics() {
  const at = "2026-11-20T09:00:00Z";
  const metric = (id: string, label: string, value: number, population: number) => ({ id, label, value, unit: "COUNT", condition: value ? "ATTENTION" : "CLEAR",
    freshness: "CURRENT", completeness: "COMPLETE", population, excluded: 0, unknown: 0, generated_at: at,
    source_revision: "sample-domain-v1", definition_revision: "enterprise-domain-v1", basis: "CURRENT_POSTURE",
    drill: { workspace: "risks", filter: id, consistency: "SOURCE_SNAPSHOT" } });
  return { generated_at: at, posture_as_of: at, scope_id: "bank-ng", scope_kind: "LEGAL_ENTITY", source_id: "sample-source",
    source_revision: "sample-domain-v1", definition_revision: "enterprise-domain-v1",
    items: [metric("risks_outside_appetite", "Outside appetite", 4, 8), metric("indicator_breaches", "Indicator breaches", 3, 6), metric("assurance_failures", "Assurance failures", 1, 8)] };
}
