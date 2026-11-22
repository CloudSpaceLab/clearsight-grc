import { createRoot } from "react-dom/client";
import { FormsRegister } from "../src/v210/FormsRegister";
import "../src/v210/tokens.css";
import "../src/v210/fresh.css";
import "../src/v210/program-register.css";
const items = [
  { template: { id: "f1", name: "Branch control assessment", purpose: "Record control assessment results and supporting documents.", status: "ACTIVE", version: 2, fields: Array.from({ length: 12 }, (_, i) => ({ id: `a${i}` })) } },
  { template: { id: "f2", name: "Technical control verification", purpose: "Review technical control observations.", status: "ACTIVE", version: 1, fields: Array.from({ length: 5 }, (_, i) => ({ id: `b${i}` })) } },
  { template: { id: "f3", name: "Outsourced service assurance", purpose: "Review supplier declarations and service controls.", status: "DRAFT", version: 3, fields: Array.from({ length: 7 }, (_, i) => ({ id: `c${i}` })) } },
];
window.fetch = async (input) => {
  const url = new URL(String(input), location.origin);
  if (url.pathname === "/api/v1/forms/templates") return Response.json({ items: url.searchParams.get("status") ? items.filter((item) => item.template.status === url.searchParams.get("status")) : items, facets: { status: { ACTIVE: 2, DRAFT: 1 } } });
  return new Response("Unavailable", { status: 404 });
};
createRoot(document.getElementById("root")!).render(<><header className="v-topbar"><span className="v-brand"><strong>ClearSight</strong></span><div className="v-entity"><strong>Sample Bank</strong></div></header><aside className="v-sidebar"><p>Sample Bank</p><nav aria-label="Sample navigation"><a href="#">Forms</a></nav></aside><main className="v-main"><p className="v-sample">Sample data</p><FormsRegister/></main></>);
