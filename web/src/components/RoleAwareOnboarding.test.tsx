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
  code: "executive-first-run", profile: "executive", role: "Executive risk or compliance leader", version: 1,
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

  it("allows a dismissed guide to be restarted from the launcher", async () => {
    vi.mocked(loadRoleGuide).mockResolvedValue(guide);
    vi.mocked(loadGuideState).mockResolvedValue({ ...initial, dismissed: true, version: 3 });
    vi.mocked(saveGuideState).mockResolvedValue({ ...initial, version: 4 });
    render(<RoleAwareOnboarding runtime={{ tenant: { id: "bank-demo" }, actor: { id: "role-cro", role_codes: ["CRO"] } }} onStep={vi.fn()}/>);

    fireEvent.click(await screen.findByRole("button", { name: /Restart Executive risk or compliance leader introduction/ }));
    expect(await screen.findByRole("dialog")).toBeTruthy();
    expect(saveGuideState).toHaveBeenCalledWith(guide.code, { current_step: 0, completed: false, dismissed: false, version: 3 });
  });
});
