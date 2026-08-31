import axe from "axe-core";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DistributionComposer } from "./DistributionComposer";
import { ResponsesView } from "./ResponsesView";
import { SentFormsView } from "./SentFormsView";

const formApi = vi.hoisted(() => ({ loadReusableFormTemplateRefs: vi.fn() }));
const distributionApi = vi.hoisted(() => ({
  amendDistribution: vi.fn(), createDistribution: vi.fn(), loadRecipientCandidates: vi.fn(), loadDistributionPage: vi.fn(), loadDistribution: vi.fn(), previewDistributionSupersession: vi.fn(), supersedeDistribution: vi.fn(), transitionDistribution: vi.fn(), loadResponseRevisions: vi.fn(),
}));
vi.mock("../../formsApi", () => formApi);
vi.mock("../../formsDistributionApi", () => distributionApi);

const distribution = {
  id: "dist-a", form_template_id: "form-a", form_template_version: 4, subject_type: "CONTROL", subject_id: "control-a",
  title: "Quarterly control review", purpose: "Collect operating evidence", access_policy: "DIRECT_LINK_EMAIL_OTP", status: "OPEN",
  deadline: "2027-09-01T12:00:00Z", route_expires_at: "2027-09-01T11:00:00Z", version: 2,
  created_at: "2026-08-28T10:00:00Z", updated_at: "2026-08-28T10:00:00Z",
} as const;
const detail = {
  distribution,
  recipients: [
    { id: "r1", role: "TO", type: "INTERNAL_PRINCIPAL", principal_id: "jane", state: "COMPLETED", version: 1 },
    { id: "r2", role: "TO", type: "EXTERNAL_AUDIENCE", audience_hint: "e***@example.com", state: "PENDING", version: 1 },
  ],
  workspace: { id: "workspace-a", status: "OPEN", version: 3, updated_at: "2026-08-28T10:00:00Z" },
} as const;
const responseRevision = {
  id: "revision-2", revision: 2, supersedes_revision_id: "revision-1", achieved_assurance: "EMAIL_VERIFIED",
  signoff_summary: { attested: true }, compliance_score: 92, scored_weight_coverage: 100, state: "FINAL",
  critical_field_results: [], scoring_policy_version: "policy-1", current: true, created_at: "2026-08-28T12:00:00Z",
} as const;

beforeEach(() => {
  window.history.replaceState(null, "", "/?safe=1#forms");
  for (const value of Object.values(formApi)) value.mockReset();
  for (const value of Object.values(distributionApi)) value.mockReset();
  formApi.loadReusableFormTemplateRefs.mockResolvedValue([{ id: "form-a", name: "Control review", code: "CONTROL", version: 4 }]);
  distributionApi.loadRecipientCandidates.mockResolvedValue({ items: [{ principal_id: "jane", display_name: "Jane Reviewer", context_label: "Controls" }], has_more: false });
  distributionApi.createDistribution.mockResolvedValue(detail);
  distributionApi.loadDistributionPage.mockResolvedValue({ items: [distribution] });
  distributionApi.loadDistribution.mockResolvedValue(detail);
  distributionApi.loadResponseRevisions.mockResolvedValue({ items: [] });
  distributionApi.amendDistribution.mockResolvedValue({ detail: { ...detail, distribution: { ...distribution, deadline: "2027-09-02T12:00:00Z", version: 3 } }, impact: { deadline_changed: true, recipients_added: 1 } });
  distributionApi.previewDistributionSupersession.mockResolvedValue({ distribution_id: "dist-a", expected_version: 2, expected_workspace_version: 3, target_form_template_id: "form-a", target_form_version: 5, compatible_fields: [{ field_id: "legal-name" }], excluded_fields: [{ field_id: "certificate", reason: "Field type changed" }] });
  distributionApi.supersedeDistribution.mockResolvedValue({ previous: { ...detail, distribution: { ...distribution, status: "SUPERSEDED", version: 3 } }, replacement: { ...detail, distribution: { ...distribution, id: "dist-b", form_template_version: 5, version: 2 } }, carried_field_ids: ["legal-name"] });
});

describe("Task 11 governed form views", () => {
  it("dispatches the exact active revision with a directory-selected internal recipient", async () => {
    render(<DistributionComposer/>);
    expect(await screen.findByRole("option", { name: /Control review/ })).toBeTruthy();
    fireEvent.change(screen.getByLabelText("Subject identifier"), { target: { value: "control-a" } });
    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Quarterly control review" } });
    fireEvent.change(screen.getByLabelText("Purpose"), { target: { value: "Collect operating evidence" } });
    fireEvent.change(screen.getByLabelText(/Deadline/), { target: { value: "2027-09-01T12:00" } });
    fireEvent.change(screen.getByLabelText(/Access route expiry/), { target: { value: "2027-09-01T11:00" } });
    fireEvent.change(screen.getByLabelText("Find internal recipient"), { target: { value: "Jane" } });
    fireEvent.click(await screen.findByRole("option", { name: /Jane Reviewer/ }));
    fireEvent.click(screen.getByRole("button", { name: "Create and dispatch" }));
    await waitFor(() => expect(distributionApi.createDistribution).toHaveBeenCalledTimes(1));
    expect(distributionApi.createDistribution.mock.calls[0]?.[0]).toMatchObject({
      form_template_id: "form-a", form_template_version: 4, subject_type: "CONTROL", subject_id: "control-a",
      recipients: [{ role: "TO", type: "INTERNAL_PRINCIPAL", principal_id: "jane" }],
    });
  });

  it("renders recipient counts and opens distribution amendment", async () => {
    render(<SentFormsView/>);
    fireEvent.click(await screen.findByRole("button", { name: "Open Quarterly control review" }));
    expect(await screen.findByText("2 To · 0 CC")).toBeTruthy();
    expect(screen.getByText("1/2")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Amend distribution" }));
    fireEvent.change(screen.getByLabelText("Response deadline"), { target: { value: "2027-09-02T12:00" } });
    fireEvent.change(screen.getByLabelText(/Access expiry/), { target: { value: "2027-09-02T11:00" } });
    fireEvent.change(screen.getByLabelText("External email"), { target: { value: "new@example.com" } });
    fireEvent.click(screen.getByRole("button", { name: "Add external To" }));
    fireEvent.click(screen.getByRole("button", { name: "Save amendment" }));
    await waitFor(() => expect(distributionApi.amendDistribution).toHaveBeenCalledWith("dist-a", expect.objectContaining({ expected_version: 2, deadline: expect.any(String), route_expires_at: expect.any(String), add_recipients: [expect.objectContaining({ role: "TO", type: "EXTERNAL_AUDIENCE", address: "new@example.com" })] })));
  });

  it("previews and confirms replacement form versions before superseding", async () => {
    formApi.loadReusableFormTemplateRefs.mockResolvedValue([{ id: "form-a", name: "Control review", code: "CONTROL", version: 5 }]);
    render(<SentFormsView/>);
    fireEvent.click(await screen.findByRole("button", { name: "Open Quarterly control review" }));
    expect(await screen.findByText("2 To · 0 CC")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Replace form version" }));
    fireEvent.click(await screen.findByRole("button", { name: "Preview replacement" }));
    expect(await screen.findByText("1 response can carry forward")).toBeTruthy();
    fireEvent.click(screen.getByLabelText("Carry compatible responses into the replacement"));
    fireEvent.click(screen.getByRole("button", { name: "Confirm replacement" }));
    await waitFor(() => expect(distributionApi.supersedeDistribution).toHaveBeenCalledWith("dist-a", expect.objectContaining({ expected_version: 2, expected_workspace_version: 3, target_form_version: 5, carry_forward: true, confirmed_field_ids: ["legal-name"] })));
  });

  it("shows immutable response revision state without mutation controls", async () => {
    distributionApi.loadResponseRevisions.mockResolvedValue({ items: [responseRevision] });
    render(<ResponsesView/>);
    expect(await screen.findByText("Revision 2 · Current")).toBeTruthy();
    expect(screen.getByText("Email Verified")).toBeTruthy();
    expect(screen.getByText("92%")).toBeTruthy();
    expect((screen.getByRole("button", { name: "Edit response" }) as HTMLButtonElement).disabled).toBe(true);
  });

  it("passes semantic accessibility checks for the new sender and immutable-response surfaces", async () => {
    const sent = render(<SentFormsView/>);
    fireEvent.click(await screen.findByRole("button", { name: "Open Quarterly control review" }));
    expect(await screen.findByText("2 To · 0 CC")).toBeTruthy();
    const sentResults = await axe.run(sent.container, { rules: { "color-contrast": { enabled: false } } });
    expect(sentResults.violations.map((violation) => violation.id)).toEqual([]);
    sent.unmount();

    distributionApi.loadResponseRevisions.mockResolvedValue({ items: [responseRevision] });
    const responses = render(<ResponsesView/>);
    expect(await screen.findByText("Revision 2 · Current")).toBeTruthy();
    const responseResults = await axe.run(responses.container, { rules: { "color-contrast": { enabled: false } } });
    expect(responseResults.violations.map((violation) => violation.id)).toEqual([]);
  });
});
