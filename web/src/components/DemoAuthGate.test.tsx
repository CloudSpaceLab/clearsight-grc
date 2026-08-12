import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { loadContext, loadDemoAccounts, loadSessionStatus, logoutDemo, type RuntimeContext } from "../api";
import { ApiError } from "../http";
import { DemoAuthGate } from "./DemoAuthGate";

vi.mock("../api", () => ({
  loadContext: vi.fn(),
  loadDemoAccounts: vi.fn(),
  loadSessionStatus: vi.fn(),
  logoutDemo: vi.fn(),
}));

beforeEach(() => {
  vi.mocked(loadContext).mockResolvedValue({
    tenant: { id: "bank-demo", name: "Demo Bank" },
    legal_entity: { id: "bank-ng", name: "Demo Bank Nigeria" },
    actor: { id: "role-cro", name: "Chief Risk Officer" },
    mode: "postgres",
    demo_mode: true,
  } as RuntimeContext & { demo_mode: boolean });
  vi.mocked(loadDemoAccounts).mockResolvedValue([]);
  vi.mocked(loadSessionStatus).mockResolvedValue({ authenticated: true, demo_login_available: true });
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
  await waitFor(() => expect(screen.getByRole("button", { name: "Switch demo role" })).not.toBeNull());
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
  expect(screen.queryByRole("button", { name: "Switch demo role" })).toBeNull();
});
