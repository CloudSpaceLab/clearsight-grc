import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { EvidenceRecipientCandidateSelect } from "./EvidenceRecipientCandidateSelect";

const { listEvidenceRecipientCandidates } = vi.hoisted(() => ({ listEvidenceRecipientCandidates: vi.fn() }));
vi.mock("../evidenceRequestAdminApi", () => ({ listEvidenceRecipientCandidates }));

beforeEach(() => vi.clearAllMocks());

it("loads labelled candidates without rendering principal identifiers", async () => {
  listEvidenceRecipientCandidates.mockResolvedValue({ items: [{ principal_id: "person-internal-1", display_name: "Ada Okafor", context_label: "Privacy Operations Lead" }], has_more: false });
  render(<EvidenceRecipientCandidateSelect requestID="request-1" value="" onChange={vi.fn()}/>);

  expect(await screen.findByRole("option", { name: "Ada Okafor · Privacy Operations Lead" })).toBeTruthy();
  expect(screen.queryByText("person-internal-1")).toBeNull();
});

it("searches the bounded server population and explains when more matches exist", async () => {
  listEvidenceRecipientCandidates
    .mockResolvedValueOnce({ items: [{ principal_id: "person-1", display_name: "Ada Okafor", context_label: "Privacy Operations Lead" }], has_more: true })
    .mockResolvedValueOnce({ items: [{ principal_id: "person-2", display_name: "Ada Okafor", context_label: "Risk Assurance Manager" }], has_more: false });
  render(<EvidenceRecipientCandidateSelect requestID="request-1" value="" onChange={vi.fn()}/>);

  expect(await screen.findByText(/More eligible people match this request/)).toBeTruthy();
  fireEvent.change(screen.getByRole("textbox", { name: "Find eligible person" }), { target: { value: "risk assurance" } });
  fireEvent.click(screen.getByRole("button", { name: "Search people" }));

  await waitFor(() => expect(listEvidenceRecipientCandidates).toHaveBeenLastCalledWith("request-1", "risk assurance"));
  expect(await screen.findByRole("option", { name: "Ada Okafor · Risk Assurance Manager" })).toBeTruthy();
});

it("fails closed with a recovery message when candidates cannot be loaded", async () => {
  listEvidenceRecipientCandidates.mockRejectedValue(new Error("unavailable"));
  render(<EvidenceRecipientCandidateSelect requestID="request-1" value="" onChange={vi.fn()}/>);

  expect((await screen.findByRole("combobox", { name: "New assigned person" }) as HTMLSelectElement).disabled).toBe(true);
  expect(screen.getByRole("status").textContent).toMatch(/eligible people could not be loaded.*reload the evidence request/i);
});
