import { createRoot } from "react-dom/client";
import { FreshApp } from "../src/v210/FreshApp";
import "../src/v210/tokens.css";
import "../src/v210/fresh.css";
const samples = [
  ["Core banking service interruption", "Service continuity", "BREACHED"],
  ["Unauthorized privileged access", "Information security", "APPROACHING"],
  ["Third-party data exposure", "Third party", "WITHIN"],
  ["Delayed payment recovery", "Service continuity", "UNKNOWN"],
].map(([name, category, position], index) => ({ risk: { id: `r${index}`, code: `RSK-0${index + 1}`, name, category, status: "ACTIVE", version: 2 }, ...(position !== "UNKNOWN" ? { latest_assessment: { risk_version: 2, appetite_position: position, appetite_statement_id: "appetite" }, active_appetite: { id: "appetite" } } : {}) }));
window.fetch = async (input) => {
  const url = new URL(String(input), location.origin);
  if (url.pathname === "/api/v1/session/status") return Response.json({ authenticated: true, demo_login_available: false });
  if (url.pathname === "/api/v1/context") return Response.json({ tenant: { id: "bank", name: "Sample Bank" }, legal_entity: { id: "entity", name: "Sample Bank" }, actor: { id: "actor", name: "Ada Bello", kind: "PERSON" }, capabilities: { oversight_read: true }, mode: "sample" });
  if (url.pathname === "/api/v1/risks") return Response.json({ items: url.searchParams.get("search") ? samples.filter((item) => item.risk.name.toLowerCase().includes(url.searchParams.get("search")!.toLowerCase())) : samples, next_cursor: "next" });
  return new Response("Unavailable", { status: 404 });
};
location.hash = "#/risks";
createRoot(document.getElementById("root")!).render(<FreshApp/>);
