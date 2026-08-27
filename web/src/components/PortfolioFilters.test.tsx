import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadMatterSummaries, loadProgramSummaries } from "../api";
import { MattersWorkspace } from "./MattersWorkspace";
import { ProgramsWorkspace } from "./ProgramsWorkspace";

vi.mock("../api", () => ({ loadMatter: vi.fn(), loadMatterSummaries: vi.fn(), loadProgram: vi.fn(), loadProgramSummaries: vi.fn() }));

describe("portfolio filters", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.history.replaceState(null, "", "/#today");
    vi.mocked(loadProgramSummaries).mockResolvedValue({ items: [], generated_at: "2026-08-27T10:00:00Z" });
    vi.mocked(loadMatterSummaries).mockResolvedValue({ items: [], generated_at: "2026-08-27T10:00:00Z" });
  });

  it("applies and restores Program state, jurisdiction and ownership filters", async () => {
    window.history.replaceState(null, "", "/#programs?overall_state=CURRENT&jurisdiction=Nigeria&assigned_to_me=true");
    render(<ProgramsWorkspace/>);
    expect((await screen.findByLabelText("Operating state") as HTMLSelectElement).value).toBe("CURRENT");
    expect((screen.getByLabelText("Jurisdiction") as HTMLInputElement).value).toBe("Nigeria");
    expect((screen.getByLabelText("Assigned to me") as HTMLInputElement).checked).toBe(true);
    fireEvent.change(screen.getByLabelText("Search programs"), { target: { value: "cyber" } });
    fireEvent.click(screen.getByRole("button", { name: "Apply filters" }));
    await waitFor(() => expect(loadProgramSummaries).toHaveBeenLastCalledWith(expect.objectContaining({ q: "cyber", overallState: "CURRENT", jurisdiction: "Nigeria", assignedToMe: true, limit: 20 })));
    expect(window.location.hash).toContain("overall_state=CURRENT");
  });

  it("applies issue type, priority, due and ownership filters together", async () => {
    window.history.replaceState(null, "", "/#work/matters");
    render(<MattersWorkspace/>);
    await screen.findByLabelText("Issue type");
    fireEvent.change(screen.getByLabelText("Issue type"), { target: { value: "CONTROL_GAP" } });
    fireEvent.change(screen.getByLabelText("Priority"), { target: { value: "4" } });
    fireEvent.change(screen.getByLabelText("Due"), { target: { value: "OVERDUE" } });
    fireEvent.click(screen.getByLabelText("Assigned to me"));
    fireEvent.click(screen.getByRole("button", { name: "Apply filters" }));
    await waitFor(() => expect(loadMatterSummaries).toHaveBeenLastCalledWith(expect.objectContaining({ matterType: "CONTROL_GAP", priority: 4, dueCondition: "OVERDUE", assignedToMe: true, limit: 20 })));
    const chips = screen.getByLabelText("Applied issue filters");
    expect(within(chips).getByText("Control gap")).toBeTruthy();
    expect(within(chips).getByText("High priority")).toBeTruthy();
    expect(within(chips).getByText("Overdue")).toBeTruthy();
  });
});
