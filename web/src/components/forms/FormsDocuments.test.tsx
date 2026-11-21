import { fireEvent, render, screen, within } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { FreshApp } from "../../v210/FreshApp";

const api = vi.hoisted(() => ({ loadContext: vi.fn(), loadSessionStatus: vi.fn() }));
const forms = vi.hoisted(() => ({ loadFormTemplatePage: vi.fn() }));
const documents = vi.hoisted(() => ({ loadDocuments: vi.fn(), documentContentURL: vi.fn(), documentEligibility: vi.fn() }));
vi.mock("../../api", () => api);
vi.mock("../../formsApi", () => forms);
vi.mock("../../submittedDocumentApi", () => documents);

it("opens the shared Documents workspace from Forms and displays submitted documents", async () => {
  window.location.hash = "#/forms";
  api.loadSessionStatus.mockResolvedValue({ authenticated: true, demo_login_available: false });
  api.loadContext.mockResolvedValue({ tenant: { id: "bank", name: "Bank" }, legal_entity: { id: "entity", name: "Bank" }, actor: { id: "person", name: "Ada" }, capabilities: {} });
  forms.loadFormTemplatePage.mockResolvedValue({ items: [] });
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "doc", file_name: "Access review.pdf", form_title: "Access assessment", artifact_status: "AVAILABLE", submitted_at: "2026-11-20T10:00:00Z", current: true }] });
  render(<FreshApp/>);
  const views = await screen.findByRole("navigation", { name: "Forms views" });
  const link = within(views).getByRole("link", { name: "Documents" });
  expect(link.getAttribute("href")).toBe("#/documents");
  fireEvent.click(link);
  expect(await screen.findByRole("heading", { name: "Documents", level: 1 })).toBeTruthy();
  expect(await screen.findByRole("button", { name: "Open Access review.pdf" })).toBeTruthy();
  expect(documents.loadDocuments).toHaveBeenCalledWith(expect.objectContaining({ limit: 25 }), expect.any(AbortSignal));
});
