export function installFreshEvidence() {
  if (import.meta.env.VITE_STATIC_DEMO !== "true" || import.meta.env.VITE_UI_EVIDENCE !== "true") throw new Error("Evidence transport is unavailable");
  const native = globalThis.fetch.bind(globalThis);
  const program = { id: "sample-program", code: "P-01", name: "Sample bank assurance", type: "COMPLIANCE", status: "ACTIVE", owning_function: "Risk", version: 1 };
  const apiQuestionFixture = new URLSearchParams(window.location.search).get("fixture") === "api-question";
  const responseValuesFixture = new URLSearchParams(window.location.search).get("fixture") === "response-values";
  const formApprovalsFixture = new URLSearchParams(window.location.search).get("fixture") === "form-approvals";
  const adminDashboardFixture = new URLSearchParams(window.location.search).get("fixture") === "admin-dashboard";
  const unavailableProgramReviewsFixture = new URLSearchParams(window.location.search).get("fixture") === "program-reviews-unavailable";
  const matterWorkFixture = new URLSearchParams(window.location.search).get("fixture") === "matter-work";
  const form = { id: "sample-form", code: "F-01", name: "Sample vendor review", purpose: "Review service controls.", status: apiQuestionFixture ? "DRAFT" : formApprovalsFixture ? "PENDING_APPROVAL" : "ACTIVE", version: 2, fields: [{ id: "sample-question", label: "Is access reviewed?", type: "yes_no", required: true }] };
  const formItem = apiQuestionFixture ? { template: form, authority_available: true, operations: [{ command: "forms.template.revise", can_act: true }] } : formApprovalsFixture ? { template: form, authority_available: true, operations: [{ command: "forms.template.transition", can_act: true, allowed_targets: ["ACTIVE", "REJECTED"] }] } : { template: form };
  const document = { id: "sample-document", artifact_id: "sample-artifact", submission_id: "sample-submission", field_id: "sample-field", file_name: "Sample access review.png", media_type: "image/png", file_kind: "IMAGE", field_label: "Access review", form_template_id: "sample-form", form_template_version: 2, form_title: "Sample vendor review", artifact_status: "AVAILABLE", uploaded_at: "2026-11-14T09:00:00Z", uploaded_by: "sample-uploader", submitted_at: "2026-11-15T10:00:00Z", submitted_by: "sample-respondent", review: { id: "sample-review", status: "VALIDATED", reviewed_by: "sample-reviewer", reviewed_at: "2026-11-16T11:00:00Z", source: "VENDOR_ASSESSMENT" }, expires_on: "2026-10-31", current: true, response_revision_id: "sample-response", relationship_id: "sample-vendor", assessment_id: "sample-assessment" };
  const vendor = { vendor: { id: "sample-vendor-identity", tenant_id: "sample-bank", legal_name: "Sample Atlas Ltd", version: 1 }, relationship: { id: "sample-vendor", tenant_id: "sample-bank", legal_entity_id: "sample-entity", vendor_id: "sample-vendor-identity", business_owner_principal_id: "sample-person", service_name: "Hosting", criticality: "IMPORTANT", privacy_role: "PROCESSOR", status: "PROPOSED", version: 1 } };
  const response = { id: "sample-response", distribution_id: "sample-distribution", form_template_id: "sample-form", form_template_version: 2, title: "Sample vendor review", subject_type: "VENDOR_RELATIONSHIP", subject_id: "sample-vendor", submitted_by_display_name: "Ada Okafor", revision: 1, current: true, state: "FINAL", completed_at: "2026-11-15T10:00:00Z" };
  const programResponse = { ...response, id: "sample-program-response", title: "Sample access assessment", subject_type: "PROGRAM", subject_id: "sample-program" };
  const assessment = { id: "sample-assessment", relationship_id: "sample-vendor", version: 1, status: "SUBMITTED", review_kind: "ONBOARDING", current_request_id: "sample-request" };
  const risk = { id: "sample-risk", code: "R-01", name: "Payment service disruption", category: "Operational", statement: "Payment services may be unavailable.", impact: "Delayed payments", status: "ACTIVE", version: 1 };
  const activity = { id: "sample-activity", code: "PA-01", name: "Customer onboarding", status: "OPEN", version: 1, purpose: "Open customer accounts", lawful_basis: "Contract", controller: "Sample Bank", processor: "Not recorded", owner_display_name: "Ada Okafor" };
  const cycle = { id: "sample-cycle", code: "RC-01", name: "Sample annual assessment", status: "DRAFT", version: 1 };
  const loss = { id: "sample-loss", code: "LOSS-01", title: "Sample payment outage", event_type: "BUSINESS_DISRUPTION_SYSTEM_FAILURES", currency: "NGN", occurred_at: "2026-11-14T10:00:00Z", status: "ACTIVE", version: 1 };
  const lossTotals = { gross_amount_minor: 12500000, recovered_amount_minor: 2500000, net_loss_minor: 10000000, currency: "NGN", recovery_status: "PARTIAL" };
  const bodies: Record<string, unknown> = {
    "/api/v1/session/status": { authenticated: true, demo_login_available: false },
    "/api/v1/context": { tenant: { id: "sample-bank", name: "Sample Bank" }, legal_entity: { id: "sample-entity", name: "Sample Bank" }, actor: { id: "sample-person", name: "Ada Okafor", ...(adminDashboardFixture ? { role_codes: ["SYSTEM_ADMIN"] } : {}) }, mode: "sample", capabilities: { oversight_read: true } },
    "/api/v1/work/assignment-counts": { scope: { tenant_id: "sample-bank", legal_entity_id: "sample-entity", principal_id: "sample-person" }, coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "SOURCE_SNAPSHOT", as_of: "2026-11-20T10:00:00Z", total: 5, overdue: 1, due_soon: 2, undated: 1 },
    "/api/v1/work/assignments": { scope: { tenant_id: "sample-bank", legal_entity_id: "sample-entity", principal_id: "sample-person" }, coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "NOT_AVAILABLE", as_of: "2026-11-20T10:00:00Z", has_more: false, items: [
      { id: "sample-work-1", title: "Review the sample access assessment", type: "REVIEW", state: "READY", owner: "Ada Okafor", due_at: "2026-11-25T10:00:00Z", ...(matterWorkFixture ? { action_target_type: "MATTER", action_target_id: "sample-issue" } : {}) },
      { id: "sample-work-2", title: "Confirm the sample vendor audit report", type: "REVIEW", state: "IN_PROGRESS", owner: "Ada Okafor", due_at: "2026-11-27T10:00:00Z" },
    ] },
    "/api/v1/metrics/home": { scope_kind: "LEGAL_ENTITY", scope_id: "sample-entity", source_revision: "sample-revision", reporting_period: { start_date: "2026-10-21", end_date: "2026-11-20", mode: "CURRENT_WINDOW", max_days: 365, historical_end_supported: false }, posture_as_of: "2026-11-20T10:00:00Z", items: [
      { id: "critical_high_open", label: "Critical and high", value: 3, condition: "ATTENTION", freshness: "CURRENT", completeness: "COMPLETE" },
      { id: "overdue_open", label: "Overdue", value: 2, condition: "ATTENTION", freshness: "CURRENT", completeness: "COMPLETE" },
      { id: "routing_gaps", label: "Routing gaps", value: 0, condition: "CLEAR", freshness: "CURRENT", completeness: "COMPLETE" },
      { id: "outcome_failures", label: "Outcome failures", value: 1, condition: "ATTENTION", freshness: "CURRENT", completeness: "COMPLETE" },
    ] },
    "/api/v1/oversight": { projection_version: "sample-revision", reporting_period: { start_date: "2026-10-21", end_date: "2026-11-20", mode: "CURRENT_WINDOW", max_days: 365, historical_end_supported: false }, interventions: [{ target_type: "MATTER", target_id: "sample-issue", title: "Review privileged access exceptions", owner_name: "Ada Okafor", due_at: "2026-11-25T10:00:00Z", state: "OPEN", priority: 4 }] },
    "/api/v1/program-summaries": { items: [{ program, state_label: "Review due", projection_stale: false, requirement_count: 8, open_matter_count: 2 }], generated_at: "2026-11-20T10:00:00Z" },
    "/api/v1/programs/sample-program": { program, requirements: [{ id: "sample-requirement", code: "R-01", title: "Review access", statement: "Review privileged access.", status: "APPROVED", source_anchor: "Section 1", effective_from: "2026-01-01T00:00:00Z" }], framework_adoptions: [{ id: "sample-adoption", edition_id: "sample-framework", framework_code: "SECURITY", edition_code: "2026", status: "ACTIVE" }], state_label: "Review due" },
    "/api/v1/programs/sample-program/operations": { program_id: "sample-program", program_version: 1, authority_available: true, operations: [{ command: "program.requirement.supersede", subresource_id: "sample-requirement", can_act: true }, { command: "program.applicability.decide", can_act: true }, { command: "program.framework.mapping.propose", can_act: true }] },
    "/api/v1/programs/sample-program/standards/sample-adoption/candidates": { program_id: "sample-program", program_version: 1, adoption_id: "sample-adoption", edition_id: "sample-framework", next_after_code: "", candidates: [{ clause_id: "sample-clause", clause_code: "A1", source_anchor: "Section 1", status: "REVIEW_CANDIDATE", candidates: [{ requirement_id: "sample-requirement", requirement_code: "R-01", title: "Review access" }] }] },
    "/api/v1/programs/sample-program/standards/sample-adoption/mappings": { adoption_id: "sample-adoption", program_version: 1, items: [], next_after_id: "" },
    "/api/v1/framework-catalog/editions": { items: [{ id: "sample-framework", publisher_code: "Sample publisher", framework_code: "SECURITY", edition_code: "2026", title: "Sample security framework", jurisdiction: "GLOBAL", source_url: "https://example.org/sample-security-framework", source_published_on: "2026-01-01", license_basis: "CITATIONS_ONLY" }] },
    "/api/v1/framework-catalog/editions/sample-framework/clauses": { edition_id: "sample-framework", items: [{ code: "A1", title: "Review privileged access", source_anchor: "Section 1", ordinal: 1 }], next_after_code: "" },
    "/api/v1/risk-indicators": { complete: true, items: [{ indicator: { link: { kind: "KRI" }, check_id: "sample-indicator", check_code: "KRI-01", check_name: "Sample backup failures", program_id: "sample-program", program_name: "Sample bank assurance", state: "UNKNOWN", reason: "Latest monitoring result is stale.", native_measurement: { value: "2", unit: "COUNT", limits: [{ operator: "LESS_OR_EQUAL", expected: "1" }], condition: "BREACHED" }, evaluated_at: "2026-11-15T10:00:00Z", freshness_minutes: 1440 }, risks: [{ id: "sample-risk", code: "R-01", name: "Sample service disruption", status: "ACTIVE" }], risk_count: 1 }] },
    "/api/v1/forms/templates": { items: [formItem] },
    "/api/v1/forms/program-reviews": { scope: { tenant_id: "sample-bank", legal_entity_id: "sample-entity", principal_id: "sample-person" }, coverage: "PROGRAM_RESPONSE_REVIEWS", items: [programResponse] },
    "/api/v1/forms/responses/sample-program-response": { response: programResponse, revision: { id: programResponse.id, revision: 1, current: true, state: "FINAL", created_at: programResponse.completed_at }, answers: [{ field_id: "sample-question", label: "Is access reviewed?", type: "yes_no", value: { text: "Yes" } }], documents: [] },
    "/api/v1/forms/responses/sample-program-response/review": { version: 0, may_review: true, items: [] },
    "/api/v1/forms/templates/sample-form": formItem,
    "/api/v1/forms/documents": { items: [document] },
    "/api/v1/vendors": { items: [vendor] },
    "/api/v1/vendors/sample-vendor": vendor,
    "/api/v1/vendors/sample-vendor/assessments/current": { assessment },
    "/api/v1/vendor-assessments/sample-assessment": { assessment, answers: [{ field_id: "sample-question", label: "Is access reviewed?", visibility: "VISIBLE", value: { text: "Yes" } }], documents: [], matters: [{ matter_id: "sample-finding", type: "VENDOR_DEFICIENCY", status: "INITIAL_REVIEW", title: "Update the sample access review" }], coverage: {} },
    "/api/v1/vendor-assessments/sample-assessment/collection": { assessment_id: "sample-assessment", assessment_version: 1, request_id: "sample-request", request_version: 1, can_reconcile: true, fields: [{ field_id: "sample-document-field", label: "Audit report", type: "vendor_document", collection_state: "MISSING", bank_review_state: "PENDING" }] },
    "/api/v1/forms/responses": { items: [response] },
    "/api/v1/forms/responses/sample-response": { response, revision: { id: "sample-response", revision: 1, current: true, state: "FINAL", created_at: response.completed_at }, answers: responseValuesFixture ? [
      { field_id: "sample-description", label: "Review notes", type: "long_text", value: { text: "Access checked for October.\nTwo exceptions require follow-up." } },
      { field_id: "sample-amount", label: "Sample loss amount", type: "currency", value: { text: "1250000.50" } },
      { field_id: "sample-rate", label: "Accounts reviewed", type: "percentage", value: { text: "98.70" } },
      { field_id: "sample-confirmation", label: "Evidence reviewed", type: "attestation", value: { text: "true" } },
      { field_id: "sample-document", label: "Supporting reports", type: "file", value: { artifact_ids: ["sample-one", "sample-two"] } },
      { field_id: "sample-signature", label: "Signature", type: "signature", value: { artifact_ids: ["sample-signature"] } },
    ] : [{ field_id: "sample-question", label: "Is access reviewed?", type: "yes_no", value: { text: "Yes" } }], documents: responseValuesFixture ? [
      { ...document, id: "sample-document-one", artifact_id: "sample-one", field_id: "sample-document", file_name: "Sample access review.png" },
      { ...document, id: "sample-document-two", artifact_id: "sample-two", field_id: "sample-document", file_name: "Sample exception report.png" },
    ] : [document] },
    "/api/v1/forms/responses/sample-response/review": { version: 0, may_review: false, items: [] },
    "/api/v1/risks": { items: [{ risk }] },
    "/api/v1/matters/sample-issue": { matter: { id: "sample-issue", tenant_id: "sample-bank", legal_entity_id: "sample-entity", reference: "ISS-01", title: "Review privileged access exceptions", summary: "Four privileged accounts need owner approval.", status: "DECISION_REQUIRED", version: 1 }, scope: { tenant_id: "sample-bank", legal_entity_id: "sample-entity", principal_id: "sample-person", record_tenant_id: "sample-bank", record_legal_entity_id: "sample-entity" }, status_label: "Decision needed", owner_display_name: "Ada Okafor", decisions: [] },
    "/api/v1/risks/sample-risk": { risk, owner_display_name: "Ada Okafor", assessments: [], appetite: [] },
    "/api/v1/losses": { items: [{ loss, totals: lossTotals }] },
    "/api/v1/losses/sample-loss": { loss, totals: lossTotals, owner_display_name: "Ada Okafor", recoveries: [{ id: "sample-recovery", kind: "RECOVERY", loss_version: 1, amount_minor: 2500000, currency: "NGN", recovered_at: "2026-11-16T10:00:00Z" }] },
    "/api/v1/rcsa/cycles": { items: [{ cycle, risk_count: 1, control_count: 0, handoff: { label: "Setup" } }], complete: true },
    "/api/v1/rcsa/cycles/sample-cycle": { cycle, risks: [{ risk_id: "sample-risk", code: "R-01", name: "Payment service disruption", risk_version: 1 }], controls: [], first_line_owner_display_name: "Ada Okafor", phase: { label: "Setup" }, handoff: { label: "Setup" }, complete: true },
    "/api/v1/ropa/processing-activities": { rows: [activity], has_more: false },
    "/api/v1/ropa/processing-activities/sample-activity": { activity, state_label: "Open" },
  };
  globalThis.fetch = async (input, init) => {
    const url = new URL(typeof input === "string" ? input : input instanceof URL ? input.href : input.url, window.location.origin);
    if (!url.pathname.startsWith("/api/")) return native(input, init);
    if (url.pathname === "/api/v1/authority/resolve" && init?.method === "POST") return new Response(JSON.stringify({ principal: { id: "sample-person" } }), { status: 200, headers: { "Content-Type": "application/json" } });
    if (url.pathname === "/api/v1/forms/documents/sample-submission/sample-field/sample-artifact/content" && (init?.method ?? "GET") === "GET") {
      const canvas = globalThis.document.createElement("canvas");
      canvas.width = 600; canvas.height = 760;
      const context = canvas.getContext("2d");
      if (!context) return new Response(null, { status: 500 });
      context.fillStyle = "#ffffff"; context.fillRect(0, 0, canvas.width, canvas.height);
      context.fillStyle = "#162d50"; context.font = "bold 28px sans-serif";
      context.fillText("Sample access review", 42, 74);
      context.font = "18px sans-serif";
      context.fillText("Review period: October 2026", 42, 125);
      context.fillText("System: Sample hosting service", 42, 160);
      context.fillText("Owner: Ada Okafor", 42, 195);
      context.fillStyle = "#4c6280"; context.fillText("Sample data for preview only", 42, 250);
      const png = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, "image/png"));
      return png ? new Response(png, { status: 200, headers: { "Content-Type": "image/png" } }) : new Response(null, { status: 500 });
    }
    let value = bodies[url.pathname];
    if (url.pathname === "/api/v1/forms/program-reviews" && unavailableProgramReviewsFixture) value = undefined;
    if (url.pathname === "/api/v1/risk-indicators" && url.searchParams.get("kind") === "KCI") value = { complete: true, items: [] };
    if (url.pathname === "/api/v1/framework-catalog/editions" && url.searchParams.get("jurisdiction") !== "GLOBAL") value = { items: [] };
    const query = url.searchParams.get("search")?.trim().toLowerCase();
    if (url.pathname === "/api/v1/work/assignments") { const work = bodies[url.pathname] as { items: Array<{ title: string }> }; const term = url.searchParams.get("q")?.trim().toLowerCase(); value = { ...work, items: term ? work.items.filter((item) => item.title.toLowerCase().includes(term)) : work.items }; }
    if (url.pathname === "/api/v1/forms/templates" && query && !form.name.toLowerCase().includes(query)) value = { items: [] };
    const programQuery = url.searchParams.get("q")?.trim().toLowerCase();
    if (url.pathname === "/api/v1/program-summaries" && programQuery && !program.name.toLowerCase().includes(programQuery)) value = { items: [] };
    const documentQuery = url.searchParams.get("query")?.trim().toLowerCase();
    if (url.pathname === "/api/v1/forms/documents" && ((documentQuery && !document.file_name.toLowerCase().includes(documentQuery)) || ["AWAITING_REVIEW", "REVIEW_HISTORY"].includes(url.searchParams.get("view") ?? ""))) value = { items: [] };
    if ((init?.method ?? "GET") !== "GET") value = undefined;
    if (url.pathname === "/api/v1/vendors" && query && !`${vendor.vendor.legal_name} ${vendor.relationship.service_name}`.toLowerCase().includes(query)) value = { items: [] };
    return new Response(JSON.stringify(value ?? { error: "not_found", message: "Sample record unavailable." }), { status: value ? 200 : 404, headers: { "Content-Type": "application/json" } });
  };
}
