import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { ConfigureWorkspace } from "./ConfigureWorkspace";

const mounts = vi.hoisted(() => ({
  overview: vi.fn(),
  access: vi.fn(),
  authority: vi.fn(),
  data: vi.fn(),
  automation: vi.fn(),
  operations: vi.fn(),
}));

vi.mock("./ConfigureOverview", () => ({
  ConfigureOverview: () => {
    mounts.overview();
    return <div>Configuration overview surface</div>;
  },
}));

vi.mock("./PeopleAccessSection", () => ({
  PeopleAccessSection: () => {
    mounts.access();
    return <div>People and access surface</div>;
  },
}));

vi.mock("./AuthorityRoutingSection", () => ({
  AuthorityRoutingSection: () => {
    mounts.authority();
    return <div>Authority and routing surface</div>;
  },
}));

vi.mock("./DataIntegrationsSection", () => ({
  DataIntegrationsSection: () => {
    mounts.data();
    return <div>Data and integrations surface</div>;
  },
}));

vi.mock("./AutomationAISection", () => ({
  AutomationAISection: () => {
    mounts.automation();
    return <div>Automation and AI surface</div>;
  },
}));

vi.mock("./SystemOperationsSection", () => ({
  SystemOperationsSection: () => {
    mounts.operations();
    return <div>System operations surface</div>;
  },
}));

beforeEach(() => {
  window.history.replaceState(null, "", "#configure");
  for (const mount of Object.values(mounts)) mount.mockClear();
});

it("renders only the selected administrative domain and keeps selection in the route", () => {
  render(<ConfigureWorkspace importsEnabled canReconcileProjection onOpenImports={vi.fn()}/>);

  expect(screen.getByText("Configuration overview surface")).toBeTruthy();
  expect(mounts.overview).toHaveBeenCalled();
  expect(mounts.access).not.toHaveBeenCalled();
  expect(mounts.authority).not.toHaveBeenCalled();
  expect(mounts.data).not.toHaveBeenCalled();
  expect(mounts.automation).not.toHaveBeenCalled();
  expect(mounts.operations).not.toHaveBeenCalled();

  fireEvent.click(screen.getByRole("button", { name: /People & access/ }));
  expect(screen.getByText("People and access surface")).toBeTruthy();
  expect(window.location.hash).toBe("#configure/access");
  expect(mounts.access).toHaveBeenCalled();
  expect(mounts.authority).not.toHaveBeenCalled();
  expect(mounts.data).not.toHaveBeenCalled();
  expect(mounts.automation).not.toHaveBeenCalled();
  expect(mounts.operations).not.toHaveBeenCalled();

  fireEvent.click(screen.getByRole("button", { name: /Automation & AI/ }));
  expect(screen.getByText("Automation and AI surface")).toBeTruthy();
  expect(window.location.hash).toBe("#configure/automation");
  expect(mounts.automation).toHaveBeenCalled();
  expect(mounts.authority).not.toHaveBeenCalled();
  expect(mounts.data).not.toHaveBeenCalled();
  expect(mounts.operations).not.toHaveBeenCalled();
});

it("opens a deep-linked configuration domain without mounting overview first", () => {
  window.history.replaceState(null, "", "#configure/operations");

  render(<ConfigureWorkspace importsEnabled canReconcileProjection onOpenImports={vi.fn()}/>);

  expect(screen.getByText("System operations surface")).toBeTruthy();
  expect(mounts.operations).toHaveBeenCalled();
  expect(mounts.overview).not.toHaveBeenCalled();
  expect(mounts.access).not.toHaveBeenCalled();
  expect(mounts.authority).not.toHaveBeenCalled();
  expect(mounts.data).not.toHaveBeenCalled();
  expect(mounts.automation).not.toHaveBeenCalled();
});
