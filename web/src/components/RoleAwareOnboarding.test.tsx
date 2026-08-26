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
const runtime = { tenant: { id: "bank-demo" }, actor: { id: "role-cro", role_codes: ["CRO"] } };

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((resolvePromise, rejectPromise) => { resolve = resolvePromise; reject = rejectPromise; });
  return { promise, resolve, reject };
}

describe("RoleAwareOnboarding", () => {
  it("resolves the guide from verified role codes and executes the live step action", async () => {
    vi.mocked(loadRoleGuide).mockResolvedValue(guide);
    vi.mocked(loadGuideState).mockResolvedValue(initial);
    vi.mocked(saveGuideState).mockImplementation(async (_code, value) => ({ ...initial, ...value, version: value.version + 1 }));
    const onStep = vi.fn();
    render(<RoleAwareOnboarding surface="TODAY" runtime={{ tenant: { id: "bank-demo" }, actor: { id: "role-cro", role_codes: ["CRO"] } }} onStep={onStep}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Start guide" }));
    expect(saveGuideState).not.toHaveBeenCalled();
    fireEvent.click(await screen.findByRole("button", { name: "Open Today" }));
    await waitFor(() => expect(onStep).toHaveBeenCalledWith(expect.objectContaining({ id: "today", view: "today" })));
    expect(saveGuideState).toHaveBeenCalledWith(guide.code, expect.objectContaining({ current_step: 1, completed: false }));
    expect(loadRoleGuide).toHaveBeenCalledWith("TODAY");
  });

  it("keeps the guide retryable and advances only after a recovered workspace action", async () => {
    vi.mocked(loadRoleGuide).mockResolvedValue(guide);
    vi.mocked(loadGuideState).mockResolvedValue(initial);
    vi.mocked(saveGuideState).mockImplementation(async (_code, value) => ({ ...initial, ...value, version: value.version + 1 }));
    const onStep = vi.fn().mockRejectedValueOnce(new Error("Vendor records are unavailable")).mockResolvedValueOnce(undefined);
    render(<RoleAwareOnboarding surface="TODAY" runtime={{ tenant: { id: "bank-demo" }, actor: { id: "role-cro", role_codes: ["CRO"] } }} onStep={onStep}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Start guide" }));
    const action = await screen.findByRole("button", { name: "Open Today" });
    fireEvent.click(action);
    await waitFor(() => expect((action as HTMLButtonElement).disabled).toBe(false));
    expect(saveGuideState).not.toHaveBeenCalled();
    expect(screen.getByRole("alert").textContent).toContain("This guide step could not be opened. Try again.");

    fireEvent.click(screen.getByRole("button", { name: "Open Today" }));
    await waitFor(() => expect(saveGuideState).toHaveBeenCalledWith(guide.code, expect.objectContaining({ current_step: 1, completed: false })));
    expect(screen.getByRole("heading", { name: "Inspect a Program" })).toBeTruthy();
  });

  it("resumes a dismissed incomplete guide at its saved numbered step without persisting", async () => {
    vi.mocked(loadRoleGuide).mockResolvedValue(guide);
    vi.mocked(loadGuideState).mockResolvedValue({ ...initial, current_step: 1, dismissed: true, version: 3 });
    render(<RoleAwareOnboarding surface="TODAY" runtime={{ tenant: { id: "bank-demo" }, actor: { id: "role-cro", role_codes: ["CRO"] } }} onStep={vi.fn()}/>);

    const launcher = await screen.findByRole("button", { name: /Resume Executive risk or compliance leader guide/ });
    expect(launcher.textContent).toContain("Resume guide");
    fireEvent.click(launcher);
    expect(await screen.findByRole("heading", { name: "Inspect a Program" })).toBeTruthy();
    expect(screen.queryByRole("complementary", { name: /Today guide/i })).toBeNull();
    expect(saveGuideState).not.toHaveBeenCalled();
  });

  it("restarts a completed guide at step zero and shows the cinematic introduction", async () => {
    vi.mocked(loadRoleGuide).mockResolvedValue(guide);
    vi.mocked(loadGuideState).mockResolvedValue({ ...initial, current_step: 2, completed: true, version: 3 });
    vi.mocked(saveGuideState).mockResolvedValue({ ...initial, version: 4 });
    render(<RoleAwareOnboarding surface="TODAY" runtime={runtime} onStep={vi.fn()}/>);

    const launcher = await screen.findByRole("button", { name: /Restart Executive risk or compliance leader guide/ });
    expect(launcher.textContent).toContain("Restart guide");
    fireEvent.click(launcher);
    expect(await screen.findByRole("complementary", { name: /Today guide/i })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Start guide" })).toBeTruthy();
    expect(screen.queryByRole("complementary", { name: "Getting started" })).toBeNull();
    expect(saveGuideState).toHaveBeenCalledWith(guide.code, { current_step: 0, completed: false, dismissed: false, version: 3 });
  });

  it("persists Skip for now through the existing dismissal state", async () => {
    vi.mocked(loadRoleGuide).mockResolvedValue(guide);
    vi.mocked(loadGuideState).mockResolvedValue(initial);
    vi.mocked(saveGuideState).mockResolvedValue({ ...initial, dismissed: true, version: 1 });
    render(<RoleAwareOnboarding surface="TODAY" runtime={{ tenant: { id: "bank-demo" }, actor: { id: "role-cro", role_codes: ["CRO"] } }} onStep={vi.fn()}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Skip for now" }));

    await waitFor(() => expect(saveGuideState).toHaveBeenCalledWith(guide.code, expect.objectContaining({ completed: false, dismissed: true })));
    expect(screen.getByRole("button", { name: /Resume Executive risk or compliance leader guide/ })).toBeTruthy();
  });

  it("loads and renders the Vendors introduction for the Vendors surface", async () => {
    const vendorGuide = {
      ...guide,
      code: "vendor-operations-first-run",
      surface: "VENDORS" as const,
      role: "Vendor relationship owner",
      title: "Manage vendor relationships",
      description: "Record the service, collect missing facts and route exceptions for review.",
    };
    vi.mocked(loadRoleGuide).mockResolvedValue(vendorGuide);
    vi.mocked(loadGuideState).mockResolvedValue({ ...initial, guide_code: vendorGuide.code });
    render(<RoleAwareOnboarding surface="VENDORS" runtime={{ tenant: { id: "bank-demo" }, actor: { id: "owner-1", role_codes: ["BUSINESS_OWNER"] } }} onStep={vi.fn()}/>);

    expect(await screen.findByRole("complementary", { name: /Vendor guide/i })).toBeTruthy();
    expect(screen.getByRole("img", { name: /Vendor relationship path/i })).toBeTruthy();
    expect(loadRoleGuide).toHaveBeenCalledWith("VENDORS");
  });

  it("discards a stale Today success after Vendors resolves", async () => {
    const todayLoad = deferred<typeof guide>();
    const vendorGuide = { ...guide, code: "vendor-operations-first-run", surface: "VENDORS" as const, role: "Vendor relationship owner", title: "Manage vendor relationships", steps: [{ ...guide.steps[0]!, id: "register", title: "Review the vendor register", action: "Review vendors", view: "vendors" as const }] };
    const vendorsLoad = deferred<typeof vendorGuide>();
    vi.mocked(loadRoleGuide).mockImplementation((surface) => surface === "TODAY" ? todayLoad.promise : vendorsLoad.promise);
    vi.mocked(loadGuideState).mockImplementation(async (code) => ({ ...initial, guide_code: code }));
    const onStep = vi.fn();
    const view = render(<RoleAwareOnboarding surface="TODAY" runtime={runtime} onStep={onStep}/>);

    view.rerender(<RoleAwareOnboarding surface="VENDORS" runtime={runtime} onStep={onStep}/>);
    vendorsLoad.resolve(vendorGuide);
    expect(await screen.findByText("Guide for Vendor relationship owner")).toBeTruthy();
    todayLoad.resolve(guide);
    await waitFor(() => expect(screen.queryByText("Guide for Executive risk or compliance leader")).toBeNull());
    fireEvent.click(screen.getByRole("button", { name: "Start guide" }));
    expect(screen.getByRole("heading", { name: "Review the vendor register" })).toBeTruthy();
  });

  it("keeps Vendors pending when the stale Today request resolves first", async () => {
    const todayLoad = deferred<typeof guide>();
    const vendorGuide = { ...guide, code: "vendor-operations-first-run", surface: "VENDORS" as const, role: "Vendor relationship owner", title: "Manage vendor relationships" };
    const vendorsLoad = deferred<typeof vendorGuide>();
    vi.mocked(loadRoleGuide).mockImplementation((surface) => surface === "TODAY" ? todayLoad.promise : vendorsLoad.promise);
    vi.mocked(loadGuideState).mockImplementation(async (code) => ({ ...initial, guide_code: code }));
    const view = render(<RoleAwareOnboarding surface="TODAY" runtime={runtime} onStep={vi.fn()}/>);

    view.rerender(<RoleAwareOnboarding surface="VENDORS" runtime={runtime} onStep={vi.fn()}/>);
    todayLoad.resolve(guide);
    await waitFor(() => expect(loadRoleGuide).toHaveBeenCalledWith("VENDORS"));
    expect(screen.queryByText("Guide for Executive risk or compliance leader")).toBeNull();
    vendorsLoad.resolve(vendorGuide);
    expect(await screen.findByText("Guide for Vendor relationship owner")).toBeTruthy();
  });

  it("discards a stale Today failure after Vendors resolves and clears Today immediately", async () => {
    vi.mocked(loadRoleGuide).mockResolvedValueOnce(guide);
    vi.mocked(loadGuideState).mockResolvedValue(initial);
    const view = render(<RoleAwareOnboarding surface="TODAY" runtime={runtime} onStep={vi.fn()}/>);
    expect(await screen.findByText("Guide for Executive risk or compliance leader")).toBeTruthy();

    const todayFailure = deferred<typeof guide>();
    const vendorGuide = { ...guide, code: "vendor-operations-first-run", surface: "VENDORS" as const, role: "Vendor relationship owner", title: "Manage vendor relationships" };
    vi.mocked(loadRoleGuide).mockReset().mockImplementation((surface) => surface === "TODAY" ? todayFailure.promise : Promise.resolve(vendorGuide));
    view.rerender(<RoleAwareOnboarding surface="TODAY" runtime={{ ...runtime, actor: { ...runtime.actor, id: "role-cro-2" } }} onStep={vi.fn()}/>);
    await waitFor(() => expect(loadRoleGuide).toHaveBeenCalledWith("TODAY"));
    view.rerender(<RoleAwareOnboarding surface="VENDORS" runtime={{ ...runtime, actor: { ...runtime.actor, id: "owner-1" } }} onStep={vi.fn()}/>);
    expect(screen.queryByText("Guide for Executive risk or compliance leader")).toBeNull();
    expect(await screen.findByText("Guide for Vendor relationship owner")).toBeTruthy();
    todayFailure.reject(new Error("stale Today failure"));
    await waitFor(() => expect(screen.getByText("Guide for Vendor relationship owner")).toBeTruthy());
  });

  it("ignores a Today save that resolves after the Vendors guide loads", async () => {
    const save = deferred<typeof initial>();
    const vendorGuide = { ...guide, code: "vendor-operations-first-run", surface: "VENDORS" as const, role: "Vendor relationship owner", title: "Manage vendor relationships" };
    vi.mocked(loadRoleGuide).mockImplementation(async (requestedSurface) => requestedSurface === "VENDORS" ? vendorGuide : guide);
    vi.mocked(loadGuideState).mockImplementation(async (code) => ({ ...initial, guide_code: code }));
    vi.mocked(saveGuideState).mockReturnValue(save.promise);
    const view = render(<RoleAwareOnboarding surface="TODAY" runtime={runtime} onStep={vi.fn()}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Start guide" }));
    fireEvent.click(screen.getByRole("button", { name: "Open Today" }));
    await waitFor(() => expect(saveGuideState).toHaveBeenCalled());

    view.rerender(<RoleAwareOnboarding surface="VENDORS" runtime={{ ...runtime, actor: { ...runtime.actor, id: "owner-1" } }} onStep={vi.fn()}/>);
    expect(await screen.findByText("Guide for Vendor relationship owner")).toBeTruthy();
    save.resolve({ ...initial, current_step: 1, version: 1 });
    await waitFor(() => expect(screen.getByText("Guide for Vendor relationship owner")).toBeTruthy());
    expect(screen.queryByRole("heading", { name: "Inspect a Program" })).toBeNull();
  });

  it("reports a progress-save failure separately without repeating the workspace action", async () => {
    vi.mocked(loadRoleGuide).mockResolvedValue(guide);
    vi.mocked(loadGuideState).mockResolvedValue(initial);
    vi.mocked(saveGuideState).mockRejectedValue(new Error("State store unavailable"));
    const onStep = vi.fn().mockResolvedValue(undefined);
    render(<RoleAwareOnboarding surface="TODAY" runtime={runtime} onStep={onStep}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Start guide" }));
    fireEvent.click(screen.getByRole("button", { name: "Open Today" }));

    expect((await screen.findByRole("alert")).textContent).toBe("Guide progress could not be saved. Your workspace remains available; try again.");
    expect(onStep).toHaveBeenCalledOnce();
    expect(screen.getByRole("heading", { name: "Review Today" })).toBeTruthy();
    expect((screen.getByRole("button", { name: "Open Today" }) as HTMLButtonElement).disabled).toBe(false);
  });

  it("shows first-run guidance without creating a modal or moving focus", async () => {
    vi.mocked(loadRoleGuide).mockResolvedValue(guide);
    vi.mocked(loadGuideState).mockResolvedValue(initial);
    const workspaceAction = document.createElement("button");
    workspaceAction.textContent = "Review priority item";
    document.body.append(workspaceAction);
    workspaceAction.focus();

    render(<RoleAwareOnboarding surface="TODAY" runtime={{ tenant: { id: "bank-demo" }, actor: { id: "role-cro", role_codes: ["CRO"] } }} onStep={vi.fn()}/>);

    const panel = await screen.findByRole("complementary", { name: /Today guide/i });
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

    render(<RoleAwareOnboarding surface="TODAY" runtime={{ tenant: { id: "bank-demo" }, actor: { id: "role-cro", role_codes: ["CRO"] } }} onStep={vi.fn()}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Skip for now" }));

    await waitFor(() => expect(saveGuideState).toHaveBeenCalledTimes(2));
    expect(screen.queryByRole("complementary", { name: /Today guide/i })).toBeNull();
    expect(screen.getByRole("button", { name: /Resume Executive risk or compliance leader guide/ })).toBeTruthy();
    expect(screen.getByRole("alert").textContent).toBe("Guide dismissal could not be saved. The guide is closed for this session; resume it to try again.");
  });
});
