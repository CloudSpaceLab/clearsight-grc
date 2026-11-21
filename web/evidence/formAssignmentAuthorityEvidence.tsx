// Isolated synthetic evidence; this entry is never loaded by the customer app.
import { createRoot } from "react-dom/client";
import { FormAssignment } from "../src/v210/FormAssignment";
import "../src/v210/fresh.css";

const originalFetch = globalThis.fetch.bind(globalThis);
globalThis.fetch = async (input, init) => {
  const url = new URL(typeof input === "string" ? input : input instanceof URL ? input.href : input.url, location.href);
  let value: unknown;
  if (url.pathname === "/api/v1/context") value = { tenant: { id: "sample-bank" }, legal_entity: { id: "sample-entity" }, actor: { id: "sample-owner", kind: "PERSON" } };
  else if (url.pathname === "/api/v1/forms/templates/sample-form") value = { template: { id: "sample-form", name: "Sample access review" }, active_version: 2, active_status: "ACTIVE" };
  else if (url.pathname === "/api/v1/program-summaries") value = { items: [{ program: { id: "sample-program", name: "Sample bank assurance" } }] };
  else if (url.pathname === "/api/v1/authority/resolve") value = { principal: { id: new URLSearchParams(location.search).has("denied") ? "other" : "sample-owner" } };
  else if (url.pathname === "/api/v1/forms/recipient-candidates") value = { items: [{ principal_id: "sample-person", display_name: "Ada Okafor", context_label: "Risk" }] };
  else if (url.pathname === "/api/v1/forms/distributions") return new Promise<Response>(() => {});
  else return originalFetch(input, init);
  return new Response(JSON.stringify(value), { status: 200, headers: { "content-type": "application/json" } });
};
createRoot(document.getElementById("root")!).render(<main className="v-main"><p className="v-sample">Sample data</p><FormAssignment formId="sample-form"/></main>);
