import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { loadGuideState, loadRoleGuide, saveGuideState } from "../onboardingApi";
import { RoleAwareOnboarding } from "./RoleAwareOnboarding";

vi.mock("../onboardingApi", () => ({
  loadRoleGuide: vi.fn(),
  loadGuideState: vi.fn(),
  saveGuideState: vi.fn(),
}));

const guide = {
  code: "executive-first-run", surface: "TODAY" as const, profile: "executive", role: "Executive risk or compliance leader", version: 1,
  title: "Read the operating brief", description: "Understand what needs attention.", illustration: "guided-orbit",
  steps: [
    { id: "today", title: "Review Today", description: "Start with assigned work.", action: "Open Today", view: "today" as const, target: "today-brief" },
    { id: "program", title: "Inspect a Program", description: "Open the exact record.", action: "Open first Program", view: "programs" as const, intent: "open-first-program", target: "programs-workspace" },
  ],
};
const initial = { tenant_id: "bank-demo", principal_id: "role-cro", guide_code: guide.code, guide_version: 1, current_step: 0, completed: false, dismissed: false, version: 0 };

describe("RoleAwareOnboarding", () => {
  it("resolves the guide from verified role codes and executes the live step action", async () => {
    vi.mocked(loadRoleGuide).mockResolvedValue(guide);
    vi.mocked(loadGuideState).mockResolvedValue(initial);
    vi.mocked(saveGuideState).mockImplementation(async (_code, value) => ({ ...initial, ...value, version: value.version + 1 }));
    const onStep = vi.fn();
    render(<RoleAwareOnboarding runtime={{ tenant: { id: "bank-demo" }, actor: { id: "role-cro", role_codes: ["CRO"] } }} onStep={onStep}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Open Today" }));
    await waitFor(() => expect(onStep).toHaveBeenCalledWith(expect.objectContaining({ id: "today", view: "today" })));
    expect(saveGuideState).toHaveBeenCalledWith(guide.code, expect.objectContaining({ current_step: 1, completed: false }));
  });

  it("keeps the guide retryable and advances only after a recovered workspace action", async () => {
    vi.mocked(loadRoleGuide).mockResolvedValue(guide);
    vi.mocked(loadGuideState).mockResolvedValue(initial);
    vi.mocked(saveGuideState).mockImplementation(async (_code, value) => ({ ...initial, ...value, version: value.version + 1 }));
    const onStep = vi.fn().mockRejectedValueOnce(new Error("Vendor records are unavailable")).mockResolvedValueOnce(undefined);
    render(<RoleAwareOnboarding runtime={{ tenant: { id: "bank-demo" }, actor: { id: "role-cro", role_codes: ["CRO"] } }} onStep={onStep}/>);

    const action = await screen.findByRole("button", { name: "Open Today" });
    fireEvent.click(action);
    await waitFor(() => expect((action as HTMLButtonElement).disabled).toBe(false));
    expect(saveGuideState).not.toHaveBeenCalled();
    expect(screen.getByRole("alert").textContent).toContain("This guide step could not be opened. Try again.");

    fireEvent.click(screen.getByRole("button", { name: "Open Today" }));
    await waitFor(() => expect(saveGuideState).toHaveBeenCalledWith(guide.code, expect.objectContaining({ current_step: 1, completed: false })));
    expect(screen.getByRole("heading", { name: "Inspect a Program" })).toBeTruthy();
  });

  it("allows a dismissed guide to be restarted from the launcher", async () => {
    vi.mocked(loadRoleGuide).mockResolvedValue(guide);
    vi.mocked(loadGuideState).mockResolvedValue({ ...initial, dismissed: true, version: 3 });
    vi.mocked(saveGuideState).mockResolvedValue({ ...initial, version: 4 });
    render(<RoleAwareOnboarding runtime={{ tenant: { id: "bank-demo" }, actor: { id: "role-cro", role_codes: ["CRO"] } }} onStep={vi.fn()}/>);

    fireEvent.click(await screen.findByRole("button", { name: /Restart Executive risk or compliance leader guide/ }));
    expect(await screen.findByRole("complementary", { name: "Getting started" })).toBeTruthy();
    expect(saveGuideState).toHaveBeenCalledWith(guide.code, { current_step: 0, completed: false, dismissed: false, version: 3 });
  });

  it("shows first-run guidance without creating a modal or moving focus", async () => {
    vi.mocked(loadRoleGuide).mockResolvedValue(guide);
    vi.mocked(loadGuideState).mockResolvedValue(initial);
    const workspaceAction = document.createElement("button");
    workspaceAction.textContent = "Review priority item";
    document.body.append(workspaceAction);
    workspaceAction.focus();

    render(<RoleAwareOnboarding runtime={{ tenant: { id: "bank-demo" }, actor: { id: "role-cro", role_codes: ["CRO"] } }} onStep={vi.fn()}/>);

    const panel = await screen.findByRole("complementary", { name: "Getting started" });
    expect(panel.getAttribute("aria-modal")).toBeNull();
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(document.activeElement).toBe(workspaceAction);
    workspaceAction.remove();
  });

  it("closes the guide even when concurrent state changes exhaust the save retry", async () => {
    vi.mocked(loadRoleGuide).mockResolvedValue(guide);
    vi.mocked(loadGuideState)
      .mockResolvedValueOnce(initial)
      .mockResolvedValue({ ...initial, version: 2 });
    vi.mocked(saveGuideState).mockRejectedValue(new Error("Onboarding state changed"));

    render(<RoleAwareOnboarding runtime={{ tenant: { id: "bank-demo" }, actor: { id: "role-cro", role_codes: ["CRO"] } }} onStep={vi.fn()}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Dismiss" }));

    await waitFor(() => expect(saveGuideState).toHaveBeenCalledTimes(2));
    expect(screen.queryByRole("complementary", { name: "Getting started" })).toBeNull();
    expect(screen.getByRole("button", { name: /Restart Executive risk or compliance leader guide/ })).toBeTruthy();
  });
});
