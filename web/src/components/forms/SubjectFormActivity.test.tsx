import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { loadCompletedResponses, loadDistributionPage, type CompletedResponseSummary, type Distribution } from "../../formsDistributionApi";
import { SubjectFormActivity } from "./SubjectFormActivity";

vi.mock("../../formsDistributionApi", () => ({
  loadCompletedResponses: vi.fn(),
  loadDistributionPage: vi.fn(),
}));

const openRequest: Distribution = {
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
};

const secondRequest: Distribution = {
  ...openRequest,
  id: "distribution-second",
  title: "Evidence completeness check",
  status: "READY",
};

const firstResponse: CompletedResponseSummary = {
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
};

const secondResponse: CompletedResponseSummary = {
  ...firstResponse,
  id: "response-b",
  distribution_id: "distribution-second-complete",
  title: "Owner evidence response",
  state: "FINAL",
  score: { state: "FINAL" },
};

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(loadDistributionPage).mockResolvedValue({ items: [] });
  vi.mocked(loadCompletedResponses).mockResolvedValue({ items: [] });
});

it("pages Matter requests and responses independently and opens their exact Forms records", async () => {
  vi.mocked(loadDistributionPage)
    .mockResolvedValueOnce({ items: [openRequest], next_cursor: "request-cursor-2" })
    .mockResolvedValueOnce({ items: [secondRequest] });
  vi.mocked(loadCompletedResponses)
    .mockResolvedValueOnce({ items: [firstResponse], next_cursor: "response-cursor-2" })
    .mockResolvedValueOnce({ items: [secondResponse] });

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

  expect(await screen.findByText("Control owner confirmation")).toBeTruthy();
  expect(screen.getByText("Remediation evidence")).toBeTruthy();
  expect(screen.getAllByText("1 shown")).toHaveLength(2);
  expect(screen.queryByText(/More available/i)).toBeNull();

  const sentFormLink = screen.getByRole("link", { name: "Open sent form" });
  expect(sentFormLink.getAttribute("href")).toBe("#forms?section=sent-forms&distribution=distribution-open");
  expect(screen.getByRole("link", { name: "Review response" }).getAttribute("href"))
    .toBe("#forms?section=responses&response=response-a");

  fireEvent.click(screen.getByRole("button", { name: "Load more form requests" }));
  await waitFor(() => expect(loadDistributionPage).toHaveBeenLastCalledWith({
    subject_type: "MATTER",
    subject_id: "matter-a",
    limit: 6,
    cursor: "request-cursor-2",
  }));
  expect(await screen.findByText("Evidence completeness check")).toBeTruthy();
  expect(screen.getByText("Control owner confirmation")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Load more form requests" })).toBeNull();

  fireEvent.click(screen.getByRole("button", { name: "Load more submitted responses" }));
  await waitFor(() => expect(loadCompletedResponses).toHaveBeenLastCalledWith({
    subject_type: "MATTER",
    subject_id: "matter-a",
    current_only: true,
    sort: "COMPLETED_DESC",
    limit: 6,
    cursor: "response-cursor-2",
  }));
  expect(await screen.findByText("Owner evidence response")).toBeTruthy();
  expect(screen.getByText("Remediation evidence")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Load more submitted responses" })).toBeNull();
  expect(screen.getAllByRole("link", { name: "Review response" })).toHaveLength(2);
  expect(screen.queryByText(/2 requests total|2 responses total/i)).toBeNull();
});

it("retains request rows and offers local retry when an additional page fails", async () => {
  vi.mocked(loadDistributionPage)
    .mockResolvedValueOnce({ items: [openRequest], next_cursor: "request-cursor-2" })
    .mockRejectedValueOnce(new Error("page unavailable"))
    .mockResolvedValueOnce({ items: [secondRequest] });

  render(<SubjectFormActivity subjectType="MATTER" subjectID="matter-a" subjectLabel="MAT-82BF"/>);

  expect(await screen.findByText("Control owner confirmation")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more form requests" }));

  expect(await screen.findByText("More form requests could not be loaded.")).toBeTruthy();
  expect(screen.getByText("Control owner confirmation")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Retry form requests" }));

  expect(await screen.findByText("Evidence completeness check")).toBeTruthy();
  expect(screen.getByText("Control owner confirmation")).toBeTruthy();
  await waitFor(() => expect(screen.queryByText("More form requests could not be loaded.")).toBeNull());
});

it("keeps available response context visible and retries an initial request read locally", async () => {
  vi.mocked(loadDistributionPage)
    .mockRejectedValueOnce(new Error("requests unavailable"))
    .mockResolvedValueOnce({ items: [openRequest] });
  vi.mocked(loadCompletedResponses).mockResolvedValue({ items: [secondResponse] });

  render(<SubjectFormActivity subjectType="MATTER" subjectID="matter-a" subjectLabel="MAT-82BF"/>);

  expect(await screen.findByText(/Form requests are unavailable\. Other issue work remains available\./)).toBeTruthy();
  expect(await screen.findByText("Owner evidence response")).toBeTruthy();
  expect(screen.getByRole("link", { name: "Review response" })).toBeTruthy();

  fireEvent.click(screen.getByRole("button", { name: "Retry form requests" }));
  expect(await screen.findByText("Control owner confirmation")).toBeTruthy();
  await waitFor(() => expect(loadDistributionPage).toHaveBeenCalledTimes(2));
});

it("keeps available request context visible and retries an initial response read locally", async () => {
  vi.mocked(loadDistributionPage).mockResolvedValue({ items: [openRequest] });
  vi.mocked(loadCompletedResponses)
    .mockRejectedValueOnce(new Error("responses unavailable"))
    .mockResolvedValueOnce({ items: [firstResponse] });

  render(<SubjectFormActivity subjectType="MATTER" subjectID="matter-a" subjectLabel="MAT-82BF"/>);

  expect(await screen.findByText(/Submitted responses are unavailable\. Other issue work remains available\./)).toBeTruthy();
  expect(screen.getByText("Control owner confirmation")).toBeTruthy();
  expect(screen.getByRole("link", { name: "Open sent form" })).toBeTruthy();

  fireEvent.click(screen.getByRole("button", { name: "Retry submitted responses" }));
  expect(await screen.findByText("Remediation evidence")).toBeTruthy();
  await waitFor(() => expect(loadCompletedResponses).toHaveBeenCalledTimes(2));
});
