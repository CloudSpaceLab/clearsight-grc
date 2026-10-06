import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { loadCompletedResponses, loadDistributionPage } from "../../formsDistributionApi";
import { SubjectFormActivity } from "./SubjectFormActivity";

vi.mock("../../formsDistributionApi", () => ({
  loadCompletedResponses: vi.fn(),
  loadDistributionPage: vi.fn(),
}));

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(loadDistributionPage).mockResolvedValue({ items: [] });
  vi.mocked(loadCompletedResponses).mockResolvedValue({ items: [] });
});

it("loads bounded Matter requests and responses without treating a page as the total", async () => {
  vi.mocked(loadDistributionPage).mockResolvedValue({
    next_cursor: "more-requests",
    items: [
      {
        id: "distribution-open",
        legal_entity_id: "entity-a",
        form_template_id: "form-a",
        form_template_version: 2,
        subject_type: "MATTER",
        subject_id: "matter-a",
        title: "Control owner confirmation",
        purpose: "Confirm the current owner.",
        access_policy: "DIRECT_MAGIC_LINK",
        status: "OPEN",
        deadline: "2026-10-12T17:00:00Z",
        route_expires_at: "2026-10-12T17:00:00Z",
        version: 1,
        created_at: "2026-10-06T10:00:00Z",
        updated_at: "2026-10-06T10:00:00Z",
      },
      {
        id: "distribution-expired",
        legal_entity_id: "entity-a",
        form_template_id: "form-b",
        form_template_version: 1,
        subject_type: "MATTER",
        subject_id: "matter-a",
        title: "Expired evidence request",
        purpose: "Collect missing evidence.",
        access_policy: "DIRECT_MAGIC_LINK",
        status: "EXPIRED",
        deadline: "2026-10-01T17:00:00Z",
        route_expires_at: "2026-10-01T17:00:00Z",
        version: 2,
        created_at: "2026-09-20T10:00:00Z",
        updated_at: "2026-10-02T10:00:00Z",
      },
    ],
  });
  vi.mocked(loadCompletedResponses).mockResolvedValue({
    next_cursor: "more-responses",
    items: [{
      id: "response-a",
      distribution_id: "distribution-complete",
      form_template_id: "form-c",
      form_template_version: 4,
      title: "Remediation evidence",
      subject_type: "MATTER",
      subject_id: "matter-a",
      revision: 1,
      current: true,
      state: "PROVISIONAL",
      score: { state: "PROVISIONAL" },
      completed_at: "2026-10-06T14:00:00Z",
    }],
  });

  render(<SubjectFormActivity subjectType="MATTER" subjectID="matter-a" subjectLabel="MAT-82BF" limit={6}/>);

  await waitFor(() => expect(loadDistributionPage).toHaveBeenCalledWith({
    subject_type: "MATTER",
    subject_id: "matter-a",
    limit: 6,
  }));
  expect(loadCompletedResponses).toHaveBeenCalledWith({
    subject_type: "MATTER",
    subject_id: "matter-a",
    current_only: true,
    sort: "COMPLETED_DESC",
    limit: 6,
  });

  expect(await screen.findByText("2 shown · More available")).toBeTruthy();
  expect(screen.getByText("1 shown · More available")).toBeTruthy();
  expect(screen.getByText("1 response pending · 1 request expired · 1 submitted response needs review.")).toBeTruthy();
  expect(screen.getByText("Control owner confirmation")).toBeTruthy();
  expect(screen.getByText("Remediation evidence")).toBeTruthy();
  expect(screen.getByRole("link", { name: "Review response" }).getAttribute("href")).toBe("#forms?section=responses&response=response-a");
  expect(screen.queryByText(/3 requests|1 response total|2 requests total/i)).toBeNull();
});

it("keeps available response context visible when request reads fail", async () => {
  vi.mocked(loadDistributionPage).mockRejectedValue(new Error("requests unavailable"));
  vi.mocked(loadCompletedResponses).mockResolvedValue({
    items: [{
      id: "response-a",
      distribution_id: "distribution-a",
      form_template_id: "form-a",
      form_template_version: 1,
      title: "Current issue evidence",
      subject_type: "MATTER",
      subject_id: "matter-a",
      revision: 1,
      current: true,
      state: "FINAL",
      score: { state: "FINAL" },
      completed_at: "2026-10-06T14:00:00Z",
    }],
  });

  render(<SubjectFormActivity subjectType="MATTER" subjectID="matter-a" subjectLabel="MAT-82BF"/>);

  expect(await screen.findByText("Form requests are unavailable. Other issue work remains available.")).toBeTruthy();
  expect(await screen.findByText("Current issue evidence")).toBeTruthy();
  expect(screen.getByRole("link", { name: "Review response" })).toBeTruthy();
});
