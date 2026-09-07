import { render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { FormsTabContent } from "./FormsTabContent";
import { formsTabs } from "./FormsNavigation";
vi.mock("../documents/DocumentBrowser", () => ({ DocumentBrowser: ({ scopeLabel }: { scopeLabel: string }) => <section aria-label={scopeLabel}/> }));
it("offers the shared document browser as a Forms section", () => {
  expect(formsTabs).toContain("Documents");
  render(<FormsTabContent tab="Documents"/>);
  expect(screen.getByRole("region", { name: "Documents submitted through forms" })).toBeTruthy();
});
