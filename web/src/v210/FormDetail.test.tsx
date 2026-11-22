import { render, screen, within } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { FormDetail } from "./FormDetail";

vi.mock("../formsApi", () => ({ loadFormTemplateItem: vi.fn(async () => ({ template: { id: "form-1", name: "Access review", purpose: "Review privileged accounts", version: 2, status: "DRAFT", fields: [{ id: "q1", label: "Is access reviewed?", type: "yes_no", required: true }] }, authority_available: false })) }));
vi.mock("./FormActions", () => ({ FormActions: () => null }));
vi.mock("./APIQuestionWizard", () => ({ APIQuestionWizard: () => null }));

it("shows the form purpose and question list in separate labelled sections", async () => {
  render(<FormDetail id="form-1"/>);
  const details = await screen.findByRole("region", { name: "Form details" });
  expect(within(details).getByText("Review privileged accounts")).toBeTruthy();
  expect(within(screen.getByRole("region", { name: "Questions" })).getByText("Is access reviewed?")).toBeTruthy();
});
