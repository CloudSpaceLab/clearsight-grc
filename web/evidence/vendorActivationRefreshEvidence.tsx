// Isolated visual evidence; this entry is never loaded by the customer app.
import { createRoot } from "react-dom/client";
import { VendorActivation } from "../src/v210/VendorActivation";
import "../src/v210/fresh.css";

const originalFetch = globalThis.fetch.bind(globalThis);
const relationship = { id: "sample-relationship", tenant_id: "sample-bank", legal_entity_id: "sample-entity", status: "PROPOSED", version: 4 };
const requirements = { eligible: true, relationship, policy: { id: "sample-policy", policy_number: 3, version: 5, status: "ACTIVE", effective_from: "2026-01-01T00:00:00Z" }, gates: [{ code: "REVIEW", satisfied: true, explanation: "The assessment is complete." }] };
globalThis.fetch = async (input, init) => {
  const url = new URL(typeof input === "string" ? input : input instanceof URL ? input.href : input.url, location.href);
  let value: unknown;
  if (url.pathname === "/api/v1/context") value = { tenant: { id: "sample-bank" }, legal_entity: { id: "sample-entity" }, actor: { id: "sample-actor" } };
  else if (url.pathname === "/api/v1/authority/resolve") value = { principal: { id: "sample-actor" } };
  else if (url.pathname === "/api/v1/vendors/sample-relationship/activation") value = requirements;
  else if (url.pathname === "/api/v1/vendors/sample-relationship/activate") value = { ...requirements, relationship: { ...relationship, status: "ACTIVE", version: 5 }, receipt: { id: "sample-receipt", relationship_version: 5, activated_at: "2026-11-21T10:00:00Z" } };
  else return originalFetch(input, init);
  return new Response(JSON.stringify(value), { status: 200, headers: { "content-type": "application/json" } });
};
createRoot(document.getElementById("root")!).render(<main className="v-main"><p className="v-sample">Sample data</p><header className="v-page-heading"><div><p className="v-eyebrow">Vendor relationship</p><h1>Sample hosting service</h1></div></header><VendorActivation relationshipId="sample-relationship" relationshipScope={relationship} onActivated={() => { throw new Error("Simulated refresh failure"); }}/></main>);
