import { render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { loadContext, loadDemoAccounts, logoutDemo, type RuntimeContext } from "../api";
import { DemoAuthGate } from "./DemoAuthGate";

vi.mock("../api", () => ({
  loadContext: vi.fn(),
  loadDemoAccounts: vi.fn(),
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
  vi.mocked(logoutDemo).mockResolvedValue(undefined);
});

it("keeps demo authentication while hiding role switching in live preview", async () => {
  render(<DemoAuthGate presentation="live-preview"><div>Workspace</div></DemoAuthGate>);

  expect(await screen.findByText("Workspace")).not.toBeNull();
  expect(screen.queryByRole("button", { name: "Switch demo role" })).toBeNull();
});
