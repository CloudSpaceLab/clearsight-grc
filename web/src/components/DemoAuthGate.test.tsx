import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { loadContext, loadDemoAccounts, loadSessionStatus, loginDemo, logoutDemo, type RuntimeContext } from "../api";
import { ApiError } from "../http";
import { DemoAuthGate } from "./DemoAuthGate";

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
    tenant: { id: "bank-demo", name: "Demo Bank" },
    legal_entity: { id: "bank-ng", name: "Demo Bank Nigeria" },
    actor: { id: "role-cro", name: "Chief Risk Officer" },
    mode: "postgres",
    demo_mode: true,
  } as RuntimeContext & { demo_mode: boolean });
  vi.mocked(loadDemoAccounts).mockResolvedValue(demoAccounts);
  vi.mocked(loadSessionStatus).mockResolvedValue({ authenticated: true, demo_login_available: true });
  vi.mocked(loginDemo).mockResolvedValue(undefined);
  vi.mocked(logoutDemo).mockResolvedValue(undefined);
});

it("opens demo role login without probing protected context while signed out", async () => {
  vi.mocked(loadSessionStatus).mockResolvedValue({ authenticated: false, demo_login_available: true });
  vi.mocked(loadDemoAccounts).mockResolvedValue([{
    label: "Chief Risk Officer",
    username: "cro@demo.clearsight.local",
    password: "demo",
    role_codes: ["CHIEF_RISK_OFFICER"],
  }]);

  render(<DemoAuthGate><div>Workspace</div></DemoAuthGate>);

  expect(await screen.findByRole("heading", { name: "See the bank from a real role" })).not.toBeNull();
  expect(loadContext).not.toHaveBeenCalled();
});

it("loads protected context after session discovery confirms authentication", async () => {
  render(<DemoAuthGate><div>Workspace</div></DemoAuthGate>);

  expect(await screen.findByText("Workspace")).not.toBeNull();
  expect(loadSessionStatus).toHaveBeenCalledTimes(1);
  expect(loadContext).toHaveBeenCalledTimes(1);
  await waitFor(() => expect(screen.getByRole("button", { name: "Viewing as Chief Risk Officer" })).not.toBeNull());
});

it("switches directly to another demo account after securely logging out", async () => {
  render(<DemoAuthGate><div>Workspace</div></DemoAuthGate>);

  fireEvent.click(await screen.findByRole("button", { name: "Viewing as Chief Risk Officer" }));
  fireEvent.click(screen.getByRole("button", { name: "Switch to System Administrator" }));

  await waitFor(() => expect(logoutDemo).toHaveBeenCalledTimes(1));
  await waitFor(() => expect(loginDemo).toHaveBeenCalledWith("system-admin@demo.clearsight.local", "demo"));
  expect(vi.mocked(logoutDemo).mock.invocationCallOrder[0]).toBeLessThan(vi.mocked(loginDemo).mock.invocationCallOrder[0]!);
});

it("returns to the account chooser when the replacement account cannot sign in", async () => {
  vi.mocked(loginDemo).mockRejectedValueOnce(new Error("Replacement sign-in failed"));
  render(<DemoAuthGate><div>Workspace</div></DemoAuthGate>);

  fireEvent.click(await screen.findByRole("button", { name: "Viewing as Chief Risk Officer" }));
  fireEvent.click(screen.getByRole("button", { name: "Switch to System Administrator" }));

  expect(await screen.findByRole("heading", { name: "See the bank from a real role" })).toBeTruthy();
  expect(screen.getByRole("alert").textContent).toContain("Replacement sign-in failed");
});

it("uses the compatibility context flow when session discovery is unavailable", async () => {
  vi.mocked(loadSessionStatus).mockRejectedValue(new Error("not deployed"));

  render(<DemoAuthGate><div>Workspace</div></DemoAuthGate>);

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

  render(<DemoAuthGate><div>Workspace</div></DemoAuthGate>);

  expect(await screen.findByRole("heading", { name: "See the bank from a real role" })).not.toBeNull();
  expect(screen.queryByText("Workspace")).toBeNull();
});

it("keeps demo authentication while hiding role switching in live preview", async () => {
  render(<DemoAuthGate presentation="live-preview"><div>Workspace</div></DemoAuthGate>);

  expect(await screen.findByText("Workspace")).not.toBeNull();
  expect(screen.queryByRole("button", { name: /Viewing as/ })).toBeNull();
});
