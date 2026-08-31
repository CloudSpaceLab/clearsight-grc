import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { loadContext, loadDemoAccounts, loadSessionStatus, loginDemo, logoutDemo, type RuntimeContext } from "../api";
import { ApiError } from "../http";
import type { RuntimePresentation } from "../runtimePresentation";
import { DemoEnvironmentMenu } from "./DemoEnvironmentMenu";
import { SessionGate } from "./SessionGate";

vi.mock("../api", () => ({
  loadContext: vi.fn(),
  loadDemoAccounts: vi.fn(),
  loadSessionStatus: vi.fn(),
  loginDemo: vi.fn(),
  logoutDemo: vi.fn(),
}));

const demoAccounts = [{
  label: "Chief Risk Officer",
  username: "cro@demo.clearsight.local",
  password: "demo",
  role_codes: ["CHIEF_RISK_OFFICER"],
}, {
  label: "System Administrator",
  username: "system-admin@demo.clearsight.local",
  password: "demo",
  role_codes: ["SYSTEM_ADMIN"],
}];

beforeEach(() => {
  vi.mocked(loadContext).mockResolvedValue({
    tenant: { id: "bank-demo", name: "Clear Bank" },
    legal_entity: { id: "bank-ng", name: "Clear Bank Nigeria" },
    actor: { id: "role-cro", name: "Chief Risk Officer" },
    mode: "postgres",
    demo_mode: true,
  } as RuntimeContext & { demo_mode: boolean });
  vi.mocked(loadDemoAccounts).mockResolvedValue(demoAccounts);
  vi.mocked(loadSessionStatus).mockResolvedValue({ authenticated: true, demo_login_available: true });
  vi.mocked(loginDemo).mockResolvedValue(undefined);
  vi.mocked(logoutDemo).mockResolvedValue(undefined);
});

function renderSession(presentation: RuntimePresentation = "demo") {
  return render(<SessionGate presentation={presentation}>
    <div>Workspace</div>
    <DemoEnvironmentMenu onOpenReferenceJourneys={vi.fn()}/>
  </SessionGate>);
}

async function openDemoEnvironment() {
  fireEvent.click(await screen.findByText("Demo environment", { selector: "summary" }));
}

it("opens demo role login without probing protected context while signed out", async () => {
  vi.mocked(loadSessionStatus).mockResolvedValue({ authenticated: false, demo_login_available: true });
  vi.mocked(loadDemoAccounts).mockResolvedValue([{
    label: "Chief Risk Officer",
    username: "cro@demo.clearsight.local",
    password: "demo",
    role_codes: ["CRO", "EXECUTIVE"],
  }]);

  renderSession();

  expect(await screen.findByRole("heading", { name: "Choose a demo account" })).not.toBeNull();
  expect(screen.getByRole("button", { name: /Chief Risk Officer\s*CRO · Executive/ })).toBeTruthy();
  expect(loadContext).not.toHaveBeenCalled();
});

it("loads protected context after session discovery confirms authentication", async () => {
  renderSession();

  expect(await screen.findByText("Workspace")).not.toBeNull();
  expect(loadSessionStatus).toHaveBeenCalledTimes(1);
  expect(loadContext).toHaveBeenCalledTimes(1);
  await openDemoEnvironment();
  expect(screen.getByText("Chief Risk Officer", { selector: ".demo-environment-account strong" })).toBeTruthy();
});

it("uses the demo catalogue label when runtime context exposes a principal id", async () => {
  vi.mocked(loadContext).mockResolvedValue({
    tenant: { id: "bank-demo", name: "Clear Bank" },
    legal_entity: { id: "bank-ng", name: "Clear Bank Nigeria" },
    actor: { id: "role-admin", name: "00000000-0000-4000-8000-000000000104", role_codes: ["SYSTEM_ADMIN"] },
    mode: "postgres",
    demo_mode: true,
  } as RuntimeContext & { demo_mode: boolean });

  renderSession();
  await openDemoEnvironment();

  expect(screen.getByText("System Administrator", { selector: ".demo-environment-account strong" })).toBeTruthy();
  expect(screen.queryByText("00000000-0000-4000-8000-000000000104")).toBeNull();
});

it("switches directly to another demo account after securely logging out", async () => {
  renderSession();
  await openDemoEnvironment();
  fireEvent.click(screen.getByRole("button", { name: "Switch to System Administrator" }));

  await waitFor(() => expect(logoutDemo).toHaveBeenCalledTimes(1));
  await waitFor(() => expect(loginDemo).toHaveBeenCalledWith("system-admin@demo.clearsight.local", "demo"));
  expect(vi.mocked(logoutDemo).mock.invocationCallOrder[0]).toBeLessThan(vi.mocked(loginDemo).mock.invocationCallOrder[0]!);
});

it("shows the replacement account when demo roles share a broad role", async () => {
  const overlappingAccounts = [{
    label: "Chief Risk Officer", username: "cro@demo.clearsight.local", password: "demo", role_codes: ["CRO", "EXECUTIVE"],
  }, {
    label: "Chief Compliance Officer", username: "cco@demo.clearsight.local", password: "demo", role_codes: ["CCO", "EXECUTIVE", "COMPLIANCE_OFFICER"],
  }];
  vi.mocked(loadDemoAccounts).mockResolvedValue(overlappingAccounts);
  vi.mocked(loadContext)
    .mockResolvedValueOnce({ tenant: { id: "bank-demo", name: "Clear Bank" }, legal_entity: { id: "bank-ng", name: "Clear Bank Nigeria" }, actor: { id: "role-cro", name: "Chief Risk Officer", role_codes: ["CRO", "EXECUTIVE"] }, mode: "memory", demo_mode: true } as RuntimeContext & { demo_mode: boolean })
    .mockResolvedValueOnce({ tenant: { id: "bank-demo", name: "Clear Bank" }, legal_entity: { id: "bank-ng", name: "Clear Bank Nigeria" }, actor: { id: "role-cco", name: "Chief Compliance Officer", role_codes: ["CCO", "EXECUTIVE", "COMPLIANCE_OFFICER"] }, mode: "memory", demo_mode: true } as RuntimeContext & { demo_mode: boolean });

  renderSession();
  await openDemoEnvironment();
  fireEvent.click(screen.getByRole("button", { name: "Switch to Chief Compliance Officer" }));

  expect(await screen.findByText("Workspace")).toBeTruthy();
  await openDemoEnvironment();
  expect(screen.getByText("Chief Compliance Officer", { selector: ".demo-environment-account strong" })).toBeTruthy();
});

it("returns to the account chooser when the replacement account cannot sign in", async () => {
  vi.mocked(loginDemo).mockRejectedValueOnce(new Error("Replacement sign-in failed"));
  renderSession();
  await openDemoEnvironment();
  fireEvent.click(screen.getByRole("button", { name: "Switch to System Administrator" }));

  expect(await screen.findByRole("heading", { name: "Choose a demo account" })).toBeTruthy();
  expect(screen.getByRole("alert").textContent).toContain("Replacement sign-in failed");
});

it("uses the compatibility context flow when session discovery is unavailable", async () => {
  vi.mocked(loadSessionStatus).mockRejectedValue(new Error("not deployed"));

  renderSession();

  expect(await screen.findByText("Workspace")).not.toBeNull();
  expect(loadContext).toHaveBeenCalledTimes(1);
});

it("returns to demo role login when the session expires before context loads", async () => {
  vi.mocked(loadContext).mockRejectedValue(new ApiError(401, "expired"));
  vi.mocked(loadDemoAccounts).mockResolvedValue([{
    label: "Chief Risk Officer",
    username: "cro@demo.clearsight.local",
    password: "demo",
    role_codes: ["CRO"],
  }]);

  renderSession();

  expect(await screen.findByRole("heading", { name: "Choose a demo account" })).not.toBeNull();
  expect(screen.queryByText("Workspace")).toBeNull();
});

it("defaults to enterprise presentation and keeps demo account switching inside explicit demo presentation", async () => {
  render(<SessionGate><div>Workspace</div><DemoEnvironmentMenu onOpenReferenceJourneys={vi.fn()}/></SessionGate>);

  expect(await screen.findByText("Workspace")).not.toBeNull();
  await openDemoEnvironment();
  expect(screen.queryByText("Viewing as")).toBeNull();
  expect(screen.queryByRole("button", { name: /Switch to/ })).toBeNull();
});

it("keeps demo authentication while hiding account switching in live preview", async () => {
  renderSession("live-preview");

  expect(await screen.findByText("Workspace")).not.toBeNull();
  await openDemoEnvironment();
  expect(screen.queryByText("Viewing as")).toBeNull();
  expect(screen.queryByRole("button", { name: /Switch to/ })).toBeNull();
});
