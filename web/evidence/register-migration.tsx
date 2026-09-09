import { createRoot } from "react-dom/client";
import { RiskRegisterMigration } from "../src/components/imports/RiskRegisterMigration";
import { registerFixture, registerVendorFixture } from "./register-migration-fixture";
import { DisplayPreferencesRoot } from "../src/components/DisplayPreferences";
import { DocumentImportWorkspace } from "../src/components/DocumentImportWorkspace";
import type { DocumentImport } from "../src/documentTypes";
import "../src/styles.css";
import "../src/design-system/index.css";
import "../src/ui-preferences.css";
import "../src/enterprise-shell.css";
import "../src/document-import.css";
const state = new URLSearchParams(location.search).get("state");
const view = structuredClone(registerFixture);
const document: DocumentImport = { id: "source", tenant_id: "sample", legal_entity_id: "entity", file_name: "Sample third-party risk register.xlsx", media_type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", purpose: "Review vendor findings and remediation ownership", source_type: "VENDOR", size_bytes: 1000, sha256: "a".repeat(64), storage_key: "sample-register", artifact_status: "AVAILABLE", extraction_status: "EXTRACTED", extraction_method: "XLSX", analysis_status: "NO_PROPOSALS", analysis_method: "DETERMINISTIC", limitations: [], sections: [], proposals: [], elements: [{ kind: "TABLE", anchor: { sheet: "Sample register", row_start: 1, row_end: 1 }, values: [["SERVICE PROVIDER", "SERVICES OFFERED", "FINDINGS", "RECOMMENDATIONS"]] }], created_by: "sample-owner", created_at: "2026-09-09T10:00:00Z", updated_at: "2026-09-09T10:00:00Z", version: 2 };
let attempts = 0;
if (state === "authority") { view.can_import = false; view.reason = "No eligible import route is available. Ask a GRC administrator to check responsibility routing."; }
if (state === "ambiguous") view.owners.push({ id: "namesake", name: "Hakeem Musa" }), view.performers.push({ id: "namesake", name: "Hakeem Musa" });
const receipt = () => ({ ...view.draft, version: view.draft.version + 1, status: "IMPORTED", receipts: view.register.rows.map((row) => ({ row_id: row.id, matter_id: `issue-${row.id}`, action_id: `action-${row.id}`, title: row.finding })) });
if (state === "receipt") view.draft = receipt();
window.fetch = async (input, init) => {
  const url = String(input);
  let data: unknown = view;
  if (url.includes("risk-register/save")) {
    const body = JSON.parse(String(init?.body)); view.draft.selection = body.selection; view.draft.version++; data = view;
  } else if (url.includes("risk-register/import")) {
    if (state === "retry" && attempts++ === 0) return new Response(JSON.stringify({ error: { code: "connection_lost", message: "The connection was interrupted. Retry the import to check its result." } }), { status: 503, headers: { "content-type": "application/json" } });
    data = receipt();
  } else if (url.includes("/vendors?")) data = { items: state === "missing" ? [] : [registerVendorFixture] };
  else if (url.includes("/vendors/")) data = registerVendorFixture;
  else if (url.includes("/document-imports?")) data = { items: [document] };
  else if (url.includes("/coverage")) data = { tenant_id: "sample", document_id: "source", document_sha256: document.sha256, status: "ASSESSED", version: 1, candidates: [], suggestions: [], matters: [], limitations: [], metrics: Object.fromEntries(["estimated_verified", "verified", "requirement_mapped", "control_implemented", "evidence_supported"].map((key) => [key, { numerator: 0, denominator: 0 }])) };
  else if (url.endsWith("/document-imports/source")) data = document;
  else if (!url.includes("risk-register")) throw new Error("Unexpected evidence request");
  return new Response(JSON.stringify(data), { headers: { "content-type": "application/json" } });
};
createRoot(window.document.getElementById("root")!).render(<DisplayPreferencesRoot><main style={{ maxWidth: 1280, margin: "0 auto", padding: 16 }}><h1>Sample vendor risk migration</h1>{state === "workspace" ? <DocumentImportWorkspace /> : <><h2>Third-party risk register.xlsx</h2><RiskRegisterMigration documentID="source" /></>}</main></DisplayPreferencesRoot>);
