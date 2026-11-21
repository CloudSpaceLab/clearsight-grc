import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { Frameworks } from "./Frameworks";

const catalog = vi.hoisted(() => ({ loadFrameworkEditionPage: vi.fn(), loadFrameworkClauses: vi.fn() }));
vi.mock("../programOperationsApi", () => catalog);
const edition = { id: "e1", publisher_code: "CBN", framework_code: "SEC", edition_code: "2026", title: "Security standard", jurisdiction: "GLOBAL", source_published_on: "2026-01-01", source_url: "https://example.org/standard", license_basis: "CITATIONS_ONLY" };
beforeEach(() => { vi.resetAllMocks(); catalog.loadFrameworkClauses.mockResolvedValue({ edition_id: "e1", items: [{ code: "A", title: "Review access", source_anchor: "Section 1" }], next_after_code: "" }); });

it("loads published editions and displays the selected source and rights", async () => {
  catalog.loadFrameworkEditionPage.mockResolvedValue({ items: [edition], next_after_id: "" });
  render(<Frameworks/>);
  expect(await screen.findByRole("heading", { name: "Framework library" })).toBeTruthy();
  fireEvent.click(await screen.findByRole("button", { name: /Security standard/ }));
  expect(screen.getByText("Citations only")).toBeTruthy();
  expect(screen.getByRole("link", { name: "View published source" }).getAttribute("href")).toBe("https://example.org/standard");
  expect(await screen.findByText("Review access")).toBeTruthy();
  expect(catalog.loadFrameworkClauses).toHaveBeenCalledWith("e1", "");
  expect(catalog.loadFrameworkEditionPage).toHaveBeenCalledWith("GLOBAL", "");
});

it("keeps source metadata visible when requirements cannot be loaded", async () => {
  catalog.loadFrameworkEditionPage.mockResolvedValue({ items: [edition], next_after_id: "" });
  catalog.loadFrameworkClauses.mockRejectedValue(new Error("unavailable"));
  render(<Frameworks/>);
  fireEvent.click(await screen.findByRole("button", { name: /Security standard/ }));
  expect(await screen.findByText("Requirements unavailable.")).toBeTruthy();
  expect(screen.getByRole("link", { name: "View published source" })).toBeTruthy();
});

it("searches only loaded editions and loads the next catalog page", async () => {
  catalog.loadFrameworkEditionPage.mockResolvedValueOnce({ items: [edition], next_after_id: "e1" }).mockResolvedValueOnce({ items: [{ ...edition, id: "e2", title: "Privacy standard" }], next_after_id: "" });
  render(<Frameworks/>);
  await screen.findByRole("button", { name: /Security standard/ });
  fireEvent.change(screen.getByLabelText("Search loaded editions"), { target: { value: "Privacy" } });
  expect(screen.getByText("No loaded editions match this search.")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more editions" }));
  expect(await screen.findByRole("button", { name: /Privacy standard/ })).toBeTruthy();
  expect(catalog.loadFrameworkEditionPage).toHaveBeenLastCalledWith("GLOBAL", "e1");
});

it("clears the selected edition when the jurisdiction changes", async () => {
  catalog.loadFrameworkEditionPage.mockResolvedValueOnce({ items: [edition], next_after_id: "" }).mockResolvedValueOnce({ items: [], next_after_id: "" });
  render(<Frameworks/>);
  fireEvent.click(await screen.findByRole("button", { name: /Security standard/ }));
  fireEvent.change(screen.getByLabelText("Jurisdiction"), { target: { value: "ng" } });
  fireEvent.click(screen.getByRole("button", { name: "Apply jurisdiction" }));
  await waitFor(() => expect(catalog.loadFrameworkEditionPage).toHaveBeenLastCalledWith("NG", ""));
  expect(await screen.findByText("No published editions found for NG.")).toBeTruthy();
  expect(screen.queryByRole("link", { name: "View published source" })).toBeNull();
});

it("ignores a late catalog response from the previous jurisdiction", async () => {
  let resolveOld!: (value: { items: typeof edition[]; next_after_id: string }) => void;
  catalog.loadFrameworkEditionPage.mockImplementationOnce(() => new Promise((resolve) => { resolveOld = resolve; })).mockResolvedValueOnce({ items: [{ ...edition, id: "ng", title: "Nigerian standard", jurisdiction: "NG" }], next_after_id: "" });
  render(<Frameworks/>);
  fireEvent.change(screen.getByLabelText("Jurisdiction"), { target: { value: "NG" } });
  fireEvent.click(screen.getByRole("button", { name: "Apply jurisdiction" }));
  expect(await screen.findByRole("button", { name: /Nigerian standard/ })).toBeTruthy();
  resolveOld({ items: [edition], next_after_id: "" });
  await waitFor(() => expect(screen.queryByRole("button", { name: /Security standard/ })).toBeNull());
});

it("does not expose unsafe source links or show an unavailable catalog as empty", async () => {
  catalog.loadFrameworkEditionPage.mockResolvedValueOnce({ items: [{ ...edition, source_url: "javascript:alert(1)" }], next_after_id: "" });
  const view = render(<Frameworks/>);
  fireEvent.click(await screen.findByRole("button", { name: /Security standard/ }));
  expect(screen.queryByRole("link", { name: "View published source" })).toBeNull();
  expect(screen.getByText("Source unavailable.")).toBeTruthy();
  view.unmount();
  catalog.loadFrameworkEditionPage.mockRejectedValueOnce(new Error("denied"));
  render(<Frameworks/>);
  expect((await screen.findByRole("alert")).textContent).toContain("Framework editions unavailable.");
  expect(screen.queryByText(/No published editions found/)).toBeNull();
});
