import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { loadMatterFormLinks, unlinkExistingForm, type FormSubjectLink } from "../../formSubjectLinksApi";
import { MatterFormLinksPanel } from "./MatterFormLinksPanel";

vi.mock("../../formSubjectLinksApi", () => ({
  loadMatterFormLinks: vi.fn(),
  unlinkExistingForm: vi.fn(),
}));
vi.mock("./FormLinkPicker", () => ({ FormLinkPicker: () => <span>Form picker open</span> }));

const link: FormSubjectLink = {
  id: "link-a", subject_type: "MATTER", subject_id: "matter-a", form_template_id: "form-a",
  form_template_version: 3, form_name: "Risk evidence", form_status: "ACTIVE",
  linked_by: "user-a", linked_at: "2026-10-14T12:00:00Z", version: 1,
};

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(loadMatterFormLinks).mockResolvedValue({ items: [link] });
  vi.mocked(unlinkExistingForm).mockResolvedValue({ ...link, ended_at: "2026-10-14T13:00:00Z", version: 2 });
});

it("loads issue-linked forms, opens the exact Forms record and keeps the picker independent", async () => {
  const onLinkedIDsChange = vi.fn();
  const view = render(<MatterFormLinksPanel matterID="matter-a" pickerOpen={false} onClosePicker={vi.fn()} onLinkedIDsChange={onLinkedIDsChange}/>);
  expect(await screen.findByText("Risk evidence")).toBeTruthy();
  expect(screen.getByRole("link", { name: "Open form" }).getAttribute("href")).toBe("#forms/form-a");
  expect(onLinkedIDsChange).toHaveBeenCalledWith(new Set(["form-a"]));
  expect(screen.queryByText("Form picker open")).toBeNull();
  view.rerender(<MatterFormLinksPanel matterID="matter-a" pickerOpen onClosePicker={vi.fn()} onLinkedIDsChange={onLinkedIDsChange}/>);
  expect(screen.getByText("Form picker open")).toBeTruthy();
});

it("opens a request for the exact linked revision without sending on link", async () => {
  const onRequestResponse = vi.fn();
  render(<MatterFormLinksPanel matterID="matter-a" pickerOpen={false} onClosePicker={vi.fn()} onLinkedIDsChange={vi.fn()} onRequestResponse={onRequestResponse}/>);
  expect(await screen.findByText("Risk evidence")).toBeTruthy();
  expect(onRequestResponse).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole("button", { name: "Request response" }));
  expect(onRequestResponse).toHaveBeenCalledWith(link);
});

it("keeps the form intact and confirms unlinking with a reason", async () => {
  render(<MatterFormLinksPanel matterID="matter-a" pickerOpen={false} onClosePicker={vi.fn()} onLinkedIDsChange={vi.fn()}/>);
  expect(await screen.findByText("Risk evidence")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Unlink" }));
  expect(screen.getByRole("button", { name: "Confirm unlink" }).hasAttribute("disabled")).toBe(true);
  fireEvent.change(screen.getByLabelText("Reason for unlinking"), { target: { value: "Not relevant" } });
  fireEvent.click(screen.getByRole("button", { name: "Confirm unlink" }));
  await waitFor(() => expect(unlinkExistingForm).toHaveBeenCalledWith("matter-a", link, "Not relevant"));
  expect(await screen.findByText(/history is unchanged/i)).toBeTruthy();
  expect(screen.queryByRole("link", { name: "Open form" })).toBeNull();
});

it("retains existing links and exposes local retry when a later page fails", async () => {
  vi.mocked(loadMatterFormLinks).mockResolvedValueOnce({ items: [link], next_cursor: "cursor-2" }).mockRejectedValueOnce(new Error("next page failed"));
  render(<MatterFormLinksPanel matterID="matter-a" pickerOpen={false} onClosePicker={vi.fn()} onLinkedIDsChange={vi.fn()}/>);
  expect(await screen.findByText("Risk evidence")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more linked forms" }));
  expect(await screen.findByText(/More linked forms could not be loaded/)).toBeTruthy();
  expect(screen.getByText("Risk evidence")).toBeTruthy();
});
