import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { CollectionSummary, FormTemplate, MonitoringCheck } from "../monitoringTypes";
import { CollectionRecord } from "./CollectionRecord";

const form: FormTemplate = {
  id: "form-1", tenant_id: "bank-1", code: "VENDOR", name: "Vendor security and privacy review", purpose: "Confirm vendor safeguards.",
  fields: Array.from({ length: 8 }, (_, index) => ({ id: `q-${index}`, label: `Question ${index}`, type: "text", required: true })),
  status: "ACTIVE", is_current: true, version: 2, created_at: "2026-08-01T00:00:00Z", updated_at: "2026-08-01T00:00:00Z",
};
const check: MonitoringCheck = {
  id: "check-1", tenant_id: "bank-1", program_id: "program-1", code: "VENDOR-CHECK", name: form.name, claim: form.purpose,
  input_kind: "FORM", form_template_id: form.id, form_template_version: form.version,
  collection_policy: { validity_months: 12, renewal_window_days: 30, reminder_count: 3 },
  thresholds: { moderate_from: 25, high_from: 50, critical_from: 75 }, freshness_minutes: 60, minimum_coverage: 1, failure_action: "REVIEW",
  status: "ACTIVE", is_current: true, version: 4, created_at: "2026-08-01T00:00:00Z", updated_at: "2026-08-01T00:00:00Z",
};
const summary: CollectionSummary = {
  monitoring_check_id: check.id, latest_request_id: "request-2", latest_submission_id: "submission-1", latest_submission_at: "2026-08-14T10:32:00Z",
  respondent_label: "Vendor security contact", recipient_hint: "v***@supplier.example", expires_at: "2027-08-14T10:32:00Z", renewal_opens_at: "2027-07-15T10:32:00Z",
  currency_state: "CURRENT", reminders_sent: 0, reminder_count: 3, delivery_state: "DELIVERED", projection_generated_at: "2026-08-14T10:33:00Z", projection_source_version: 4,
};

describe("collection record", () => {
  it("shows one collection with its policy and latest respondent activity", () => {
    render(<CollectionRecord form={form} check={check} summary={summary}/>);
    expect(screen.getByRole("heading", { name: form.name })).toBeTruthy();
    expect(screen.getByText("8 questions · Active · Valid for 12 months")).toBeTruthy();
    expect(screen.getByText("Renewal starts 30 days before expiry · 3 reminders")).toBeTruthy();
    expect(screen.getByText((_, element) => element?.tagName === "SPAN" && element.textContent?.includes("Last submitted") === true && element.textContent.includes("by Vendor security contact"))).toBeTruthy();
    expect(screen.getByText(/Expires/)).toBeTruthy();
  });

  it.each([
    ["NO_RESPONSE_SUBMITTED", "No response submitted"],
    ["RENEWAL_DUE", "Renewal due"],
    ["RESPONSE_POTENTIALLY_EXPIRED", "Response potentially expired"],
    ["AWAITING_RESPONSE", "Awaiting response"],
    ["RENEWAL_BLOCKED", "Renewal blocked"],
  ] as const)("shows %s as %s", (currency_state, label) => {
    render(<CollectionRecord form={form} check={check} summary={{ ...summary, currency_state, last_error_safe: currency_state === "RENEWAL_BLOCKED" ? "The recipient route could not be resolved." : undefined }}/>);
    expect(screen.getByText(label)).toBeTruthy();
    if (currency_state === "RENEWAL_BLOCKED") expect(screen.getByText("The recipient route could not be resolved.")).toBeTruthy();
  });

  it("makes partial summary failure local to the collection records", () => {
    const onRetry = vi.fn();
    render(<CollectionRecord form={form} check={check} summaryUnavailable onRetrySummary={onRetry}/>);
    expect(screen.getByText("Collection dates unavailable")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Retry collection dates" }));
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it("states when a legacy form check has no renewal policy", () => {
    render(<CollectionRecord form={form} check={{ ...check, collection_policy: undefined }}/>);
    expect(screen.getByText("Response expiry is not configured.")).toBeTruthy();
  });
});
