// Isolated synthetic browser evidence. This entry is never loaded by the runtime app.
import { createRoot } from "react-dom/client";
import { installFreshEvidence } from "../src/v210/evidence";
import { Reports } from "../src/v210/Reports";
import "../src/v210/fresh.css";

installFreshEvidence();
const native = globalThis.fetch;
const tenant = "sample-bank", entity = "sample-entity", principal = "sample-person";
const definition = { id: "sample-definition", tenant_id: tenant, legal_entity_id: entity, name: "Sample risk report", description: "Sample data", dataset: "PROGRAMS", status: "ACTIVE", effective: true, current_version: 1, format: "PDF", scope_kind: "LEGAL_ENTITY" };
const reply = (value: unknown) => new Response(JSON.stringify(value), { status: 200, headers: { "Content-Type": "application/json" } });
globalThis.fetch = async (input, init) => {
  const url = new URL(typeof input === "string" ? input : input instanceof URL ? input.href : input.url, window.location.origin);
  if (url.pathname === "/api/v1/authority/resolve") return reply({ principal: { id: principal }, rule_id: "sample-rule", policy_version: "sample-v1" });
  if (url.pathname === "/api/v1/reports/definitions") return reply({ items: [definition] });
  if (url.pathname === "/api/v1/reports/runs" && (init?.method ?? "GET") === "GET") return reply({ items: [] });
  if (url.pathname === "/api/v1/reports/runs" && init?.method === "POST") {
    const body = JSON.parse(String(init.body)) as { idempotency_key: string; definition_id: string; expected_definition_version: number };
    const receiptKey = `sample-report-receipt:${body.idempotency_key}`;
    let run = localStorage.getItem(receiptKey);
    localStorage.setItem("sample-report-calls", String(Number(localStorage.getItem("sample-report-calls") ?? 0) + 1));
    if (!run) {
      localStorage.setItem("sample-report-creates", String(Number(localStorage.getItem("sample-report-creates") ?? 0) + 1));
      run = JSON.stringify({ id: "sample-run", tenant_id: tenant, legal_entity_id: entity, definition_id: body.definition_id, definition_version: body.expected_definition_version, requested_by_ref: principal, status: "QUEUED" });
      localStorage.setItem(receiptKey, run);
    }
    await new Promise((resolve) => setTimeout(resolve, 300));
    if (new URLSearchParams(location.search).get("drop") === "1") throw new Error("sample connection loss");
    return new Response(run, { status: 201, headers: { "Content-Type": "application/json" } });
  }
  return native(input, init);
};
const root = document.getElementById("root");
if (!root) throw new Error("Evidence root unavailable");
createRoot(root).render(<Reports context={{ tenant: { id: tenant, name: "Sample Bank" }, legal_entity: { id: entity, name: "Sample Bank" }, actor: { id: principal, name: "Sample person" }, mode: "sample" }} enableGeneration/>);
