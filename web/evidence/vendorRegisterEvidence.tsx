import { createRoot } from "react-dom/client";
import { Vendors } from "../src/v210/Vendors";
import "../src/v210/tokens.css";
import "../src/v210/fresh.css";

const context = { tenant: { id: "bank", name: "Sample Bank" }, legal_entity: { id: "entity", name: "Sample Bank" }, actor: { id: "actor", name: "Ada" }, mode: "sample" };
const samples = [
  ["Atlas Cloud Services", "Core banking hosting", "PROPOSED", "CRITICAL", "2027-03-31T00:00:00Z"],
  ["Delta Payments", "Payment switching", "ACTIVE", "IMPORTANT", "2027-06-30T00:00:00Z"],
  ["Cedar Data Services", "Document processing", "UNDER_REVIEW", "STANDARD", ""],
  ["Northstar Facilities", "Branch facilities", "SUSPENDED", "STANDARD", ""],
].map(([name, service, status, criticality, renewal], index) => ({ vendor: { id: `v${index}`, tenant_id: "bank", legal_name: name }, relationship: { id: `r${index}`, vendor_id: `v${index}`, tenant_id: "bank", legal_entity_id: "entity", service_name: service, status, criticality, renewal_at: renewal } }));
window.fetch = async (input) => {
  const url = String(input);
  if (url.includes("/authority/resolve")) return Response.json({ principal: { id: "actor" } });
  if (url.includes("/api/v1/vendors")) return Response.json({ items: samples, next_cursor: "next" });
  return new Response("Unavailable", { status: 404 });
};
createRoot(document.getElementById("root")!).render(<><header className="v-topbar"><span className="v-brand"><strong>ClearSight</strong></span><div className="v-entity"><strong>Sample Bank</strong></div></header><aside className="v-sidebar"><p>Sample Bank</p><nav aria-label="Sample navigation"><a href="#">Vendors</a></nav></aside><main className="v-main"><p className="v-sample">Sample data</p><Vendors context={context}/></main></>);
