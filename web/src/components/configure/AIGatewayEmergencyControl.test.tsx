import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { AIGatewayEmergencyControl } from "./AIGatewayEmergencyControl";

const gatewayApi = vi.hoisted(() => ({ setGatewayEmergencyControl: vi.fn() }));
vi.mock("../../aiGatewayTransportApi", async () => {
  const actual = await vi.importActual<typeof import("../../aiGatewayTransportApi")>("../../aiGatewayTransportApi");
  return { ...actual, setGatewayEmergencyControl: gatewayApi.setGatewayEmergencyControl };
});

it("requires an explicit reason and acknowledgement before emergency freeze", async () => {
  gatewayApi.setGatewayEmergencyControl.mockResolvedValue({ tenant_id: "bank", environment: "PRODUCTION", frozen: true, record_version: 1 });
  const onChanged = vi.fn().mockResolvedValue(undefined);
  render(<AIGatewayEmergencyControl
    environment="PRODUCTION"
    control={{ tenant_id: "bank", environment: "PRODUCTION", frozen: false, record_version: 0 }}
    runtimeStatus={{ configured: true, available: true, tenant_id: "bank", environment: "PRODUCTION", desired_revision: 3, applied_revision: 3, emergency_supported: true, emergency_revision: 0, outbound_frozen: false, degraded: false }}
    canConfigure
    loading={false}
    onChanged={onChanged}
  />);

  expect(screen.getByText("Enabled · applied")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Emergency freeze" }));
  const submit = screen.getByRole("button", { name: "Freeze outbound AI" });
  expect((submit as HTMLButtonElement).disabled).toBe(true);

  fireEvent.change(screen.getByRole("textbox", { name: "Reason" }), { target: { value: "Potential provider credential compromise" } });
  expect((submit as HTMLButtonElement).disabled).toBe(true);
  fireEvent.click(screen.getByRole("checkbox"));
  expect((submit as HTMLButtonElement).disabled).toBe(false);
  fireEvent.click(submit);

  await waitFor(() => expect(gatewayApi.setGatewayEmergencyControl).toHaveBeenCalledWith({
    environment: "PRODUCTION",
    frozen: true,
    reason: "Potential provider credential compromise",
    expectedVersion: 0,
  }));
  expect(onChanged).toHaveBeenCalledOnce();
});

it("does not claim propagation when ClearSight is newer than the gateway", () => {
  render(<AIGatewayEmergencyControl
    environment="PRODUCTION"
    control={{ tenant_id: "bank", environment: "PRODUCTION", frozen: true, reason: "Incident containment", record_version: 4 }}
    runtimeStatus={{ configured: true, available: true, tenant_id: "bank", environment: "PRODUCTION", desired_revision: 3, applied_revision: 3, emergency_supported: true, emergency_revision: 3, outbound_frozen: false, degraded: false }}
    canConfigure={false}
    loading={false}
    onChanged={async () => undefined}
  />);

  expect(screen.getByText("Propagating")).toBeTruthy();
  expect(screen.queryByText("Frozen · applied")).toBeNull();
  expect(screen.getByText(/already in-flight provider calls are not canceled/i)).toBeNull();
});
