import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { loadFormTemplatePage } from "../../formsApi";
import { linkExistingForm, type FormSubjectLink } from "../../formSubjectLinksApi";
import { FormLinkPicker } from "./FormLinkPicker";

vi.mock("../../formsApi", () => ({ loadFormTemplatePage: vi.fn() }));
vi.mock("../../formSubjectLinksApi", () => ({ linkExistingForm: vi.fn() }));

const linked: FormSubjectLink = {
  id: "link-a", subject_type: "MATTER", subject_id: "matter-a", form_template_id: "form-a",
  form_template_version: 3, form_name: "Risk evidence", form_status: "ACTIVE",
  linked_by: "user-a", linked_at: "2026-10-14T12:00:00Z", version: 1,
};

const approved = {
  template: { id: "form-a", name: "Risk evidence", code: "RISK", version: 4, status: "DRAFT" },
  active_version: 3, active_status: "ACTIVE",
};

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(loadFormTemplatePage).mockResolvedValue({ items: [approved] } as never);
  vi.mocked(linkExistingForm).mockResolvedValue(linked);
});

it("links an approved existing form without dispatching a request or creating a draft", async () => {
  const onLinked = vi.fn();
  render(<FormLinkPicker matterID="matter-a" linkedFormIDs={new Set()} onClose={vi.fn()} onLinked={onLinked}/>);

  expect(await screen.findByText("Risk evidence")).toBeTruthy();
  expect(screen.getByText(/Linking does not send a request/i)).toBeTruthy();
  expect(screen.getByRole("button", { name: "Link form" }).hasAttribute("disabled")).toBe(true);
  fireEvent.click(screen.getByRole("radio", { name: /Risk evidence/ }));
  fireEvent.click(screen.getByRole("button", { name: "Link form" }));
  await waitFor(() => expect(linkExistingForm).toHaveBeenCalledWith("matter-a", "form-a", 3));
  expect(onLinked).toHaveBeenCalledWith(linked);
});

it("prevents linking an already associated form", async () => {
  render(<FormLinkPicker matterID="matter-a" linkedFormIDs={new Set(["form-a"])} onClose={vi.fn()} onLinked={vi.fn()}/>);
  expect(await screen.findByText(/Already linked/)).toBeTruthy();
  expect(screen.getByRole("radio", { name: /Risk evidence/ }).hasAttribute("disabled")).toBe(true);
  expect(linkExistingForm).not.toHaveBeenCalled();
});

it("does not disclose errors or claim success if the association fails", async () => {
  vi.mocked(linkExistingForm).mockRejectedValue(new Error("secret server detail"));
  render(<FormLinkPicker matterID="matter-a" linkedFormIDs={new Set()} onClose={vi.fn()} onLinked={vi.fn()}/>);
  fireEvent.click(await screen.findByRole("radio", { name: /Risk evidence/ }));
  fireEvent.click(screen.getByRole("button", { name: "Link form" }));
  expect(await screen.findByText(/could not be linked/)).toBeTruthy();
  expect(screen.queryByText(/secret server detail/)).toBeNull();
});
