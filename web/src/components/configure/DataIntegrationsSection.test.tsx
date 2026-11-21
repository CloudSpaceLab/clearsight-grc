import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { DataIntegrationsSection } from "./DataIntegrationsSection";

it("shows source availability and routes each source view without inventing records", () => {
  const navigate = vi.fn();
  const openImports = vi.fn();
  const { rerender } = render(<DataIntegrationsSection hash="#configure/data" onNavigate={navigate} importsEnabled onOpenImports={openImports}/>);
  expect(screen.getByRole("heading", { name: "Integrations & data sources" })).toBeTruthy();
  expect(screen.getByRole("tab", { name: "Sources" }).getAttribute("aria-selected")).toBe("true");
  expect(screen.getByText("Connected sources are unavailable.")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Open document imports" }));
  expect(openImports).toHaveBeenCalledTimes(1);
  fireEvent.click(screen.getByRole("tab", { name: "Field mapping" }));
  expect(navigate).toHaveBeenCalledWith("#configure/data?tab=mapping");
  rerender(<DataIntegrationsSection hash="#configure/data?tab=mapping" onNavigate={navigate} importsEnabled onOpenImports={openImports}/>);
  expect(screen.getByText("Field mappings are unavailable.")).toBeTruthy();
  fireEvent.click(screen.getByRole("tab", { name: "Sync history" }));
  expect(navigate).toHaveBeenCalledWith("#configure/data?tab=history");
  rerender(<DataIntegrationsSection hash="#configure/data?tab=history" onNavigate={navigate} importsEnabled onOpenImports={openImports}/>);
  expect(screen.getByText("Sync history is unavailable.")).toBeTruthy();
});

it("explains unavailable document imports and does not activate them", () => {
  const openImports = vi.fn();
  render(<DataIntegrationsSection hash="#configure/data?tab=unknown" onNavigate={vi.fn()} importsEnabled={false} onOpenImports={openImports}/>);
  expect(screen.getByRole("tab", { name: "Sources" }).getAttribute("aria-selected")).toBe("true");
  expect(screen.getByText("Document imports are unavailable in this deployment.")).toBeTruthy();
  expect(screen.getByRole("button", { name: "Open document imports" }).hasAttribute("disabled")).toBe(true);
  expect(openImports).not.toHaveBeenCalled();
});
