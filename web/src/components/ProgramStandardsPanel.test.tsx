import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "../http";
import { decideProgramFrameworkAdoption, loadFrameworkEditionPage, proposeProgramFrameworkAdoption } from "../programOperationsApi";
import type { FrameworkEditionOption } from "../programOperationsApi";
import type { ProgramAggregate, ProgramFrameworkAdoption } from "../types";
import { ProgramStandardsPanel } from "./ProgramStandardsPanel";

vi.mock("../programOperationsApi", () => ({
  loadFrameworkEditionPage: vi.fn(),
  proposeProgramFrameworkAdoption: vi.fn(),
  decideProgramFrameworkAdoption: vi.fn(),
}));

const program = {
  id: "program-1", version: 4, name: "Security", jurisdiction: "NG", scope: {},
} as ProgramAggregate["program"];
const edition = (id: string, frameworkCode = "SECURITY"): FrameworkEditionOption => ({
  id, title: "Published " + frameworkCode, framework_code: frameworkCode,
  publisher_code: "PUBLISHER", edition_code: "2026",
  jurisdiction: "GLOBAL", license_basis: "CITATIONS_ONLY",
  source_url: "https://example.org/" + id, source_published_on: "2026-01-01",
  approval_reference: "release-approved",
});
const adoption = (id: string, frameworkCode: string): ProgramFrameworkAdoption => ({
  id, edition_id: id, publisher_code: "PUBLISHER",
  framework_code: frameworkCode, edition_code: "2026", status: "PENDING",
  proposed_by: "maker", rationale: "Proposed for the Program",
  proposed_at: "2026-10-15T10:00:00Z", version: 1,
});
const aggregate = (adoptions: ProgramFrameworkAdoption[] = []): ProgramAggregate =>
  ({ program, framework_adoptions: adoptions } as unknown as ProgramAggregate);
const owner = { command: "program.framework.propose", label: "Add standard", responsibility: "OWNER", can_act: true, reason: "" };
const approve = { command: "program.framework.approve", label: "Approve standard", responsibility: "AUTHORIZER", can_act: true, reason: "" };
const reject = { command: "program.framework.reject", label: "Reject standard", responsibility: "AUTHORIZER", can_act: true, reason: "" };

describe("Program standards — approved design integration", () => {
  beforeEach(() => vi.resetAllMocks());

  it("isolates the decision rationale for each pending adoption", () => {
    render(<ProgramStandardsPanel aggregate={aggregate([adoption("a", "SECURITY"), adoption("b", "PRIVACY")])}
      operations={[approve, reject]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", {name: "Review SECURITY 2026"}));
    fireEvent.change(screen.getByLabelText("Decision rationale for SECURITY 2026"),
      {target: {value: "Only security evidence was reviewed"}});
    fireEvent.click(screen.getByRole("button", {name: "Review PRIVACY 2026"}));
    const other = screen.getByLabelText("Decision rationale for PRIVACY 2026") as HTMLTextAreaElement;
    expect(other.value).toBe("");
    expect(screen.getByRole("button", {name: "Approve standard"}).hasAttribute("disabled")).toBe(true);
    expect(decideProgramFrameworkAdoption).not.toHaveBeenCalled();
  });

  it("keeps already-active or pending editions out of the proposal selection", async () => {
    vi.mocked(loadFrameworkEditionPage).mockResolvedValue({
      items: [edition("a"), edition("b", "PRIVACY")], next_after_id: "",
    });
    const active = {...adoption("a", "SECURITY"), status: "ACTIVE" as const};
    render(<ProgramStandardsPanel aggregate={aggregate([active])}
      operations={[owner]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", {name: "Add standard"}));
    const radios = await screen.findAllByRole("radio");
    expect(radios).toHaveLength(2);
    expect(radios[0]!.hasAttribute("disabled")).toBe(true);
    fireEvent.click(radios[1]!);
    expect(screen.getByLabelText("Selected edition").textContent).toContain("PRIVACY");
    expect(screen.getByLabelText("Selected edition").textContent).toContain("release-approved");
    expect(screen.getByRole("button", {name: "Submit for approval"}).hasAttribute("disabled")).toBe(true);
  });

  it("searches only loaded editions and advances the server cursor", async () => {
    vi.mocked(loadFrameworkEditionPage)
      .mockResolvedValueOnce({items: [edition("a")], next_after_id: "a"})
      .mockResolvedValueOnce({items: [edition("b", "PRIVACY")], next_after_id: ""});
    render(<ProgramStandardsPanel aggregate={aggregate()}
      operations={[owner]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", {name: "Add standard"}));
    await screen.findByRole("radio");
    fireEvent.change(screen.getByRole("searchbox", {name: "Search loaded standards"}), {target: {value: "PRIVACY"}});
    expect(screen.getByText("No matches in the loaded standards.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", {name: "Load more standards"}));
    await screen.findByRole("radio", {name: /Published PRIVACY/});
    expect(loadFrameworkEditionPage).toHaveBeenNthCalledWith(2, "GLOBAL", "a");
  });

  it("discards a slow earlier jurisdiction when the user switches scope", async () => {
    let finishFirst!: (page: {items: FrameworkEditionOption[]; next_after_id: string}) => void;
    vi.mocked(loadFrameworkEditionPage).mockImplementationOnce(() =>
      new Promise((resolve) => { finishFirst = resolve; }));
    vi.mocked(loadFrameworkEditionPage).mockResolvedValueOnce({
      items: [{...edition("ng", "NIGERIA"), jurisdiction: "NG"}], next_after_id: "",
    });
    render(<ProgramStandardsPanel aggregate={aggregate()}
      operations={[owner]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", {name: "Add standard"}));
    await waitFor(() => expect(loadFrameworkEditionPage).toHaveBeenCalledTimes(1));
    fireEvent.change(screen.getByLabelText("Jurisdiction"), {target: {value: "NG"}});
    await screen.findByRole("radio", {name: /Published NIGERIA/});
    finishFirst({items: [edition("global", "GLOBAL")], next_after_id: ""});
    await waitFor(() => expect(screen.queryByRole("radio", {name: /Published GLOBAL/})).toBeNull());
  });

  it("fails closed on a Program version conflict and offers authoritative reload", async () => {
    const onReload = vi.fn();
    vi.mocked(loadFrameworkEditionPage).mockResolvedValue({
      items: [edition("a")], next_after_id: "",
    });
    vi.mocked(proposeProgramFrameworkAdoption).mockRejectedValue(
      new ApiError(409, "Program changed. Reload before submitting.", "version_conflict"),
    );
    render(<ProgramStandardsPanel aggregate={aggregate()}
      operations={[owner]} onUpdated={vi.fn()} onReload={onReload}/>);
    fireEvent.click(screen.getByRole("button", {name: "Add standard"}));
    fireEvent.click(await screen.findByRole("radio", {name: /Published SECURITY/}));
    fireEvent.change(screen.getByLabelText("Reason for adoption"), {target: {value: "Regulatory requirement"}});
    fireEvent.click(screen.getByRole("button", {name: "Submit for approval"}));
    await screen.findByRole("alert");
    expect(screen.getByRole("button", {name: "Submit for approval"}).hasAttribute("disabled")).toBe(true);
    fireEvent.click(screen.getByRole("button", {name: "Reload Program"}));
    expect(onReload).toHaveBeenCalledTimes(1);
    expect(proposeProgramFrameworkAdoption).toHaveBeenCalledTimes(1);
  });

  it("does not submit an adoption when searching the catalog", async () => {
    vi.mocked(loadFrameworkEditionPage).mockResolvedValue({
      items: [edition("a")], next_after_id: "",
    });
    render(<ProgramStandardsPanel aggregate={aggregate()}
      operations={[owner]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", {name: "Add standard"}));
    fireEvent.click(await screen.findByRole("radio", {name: /Published SECURITY/}));
    fireEvent.change(screen.getByLabelText("Reason for adoption"), {target: {value: "Applies to scope"}});
    const search = screen.getByRole("searchbox", {name: "Search loaded standards"});
    expect(search.closest("form")).toBeNull();
    expect(screen.getByLabelText("Reason for adoption").closest("form")).not.toBeNull();
    fireEvent.keyDown(search, {key: "Enter", code: "Enter"});
    expect(proposeProgramFrameworkAdoption).not.toHaveBeenCalled();
  });

  it("shows access denied separately from an empty published catalog", async () => {
    vi.mocked(loadFrameworkEditionPage).mockRejectedValue(
      new ApiError(403, "Select a legal entity.", "legal_entity_required"),
    );
    render(<ProgramStandardsPanel aggregate={aggregate()}
      operations={[owner]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", {name: "Add standard"}));
    await waitFor(() => expect(screen.getByRole("alert").textContent).toContain("unavailable in this scope"));
    expect(screen.queryByText("No published standards available for this jurisdiction.")).toBeNull();
    expect(screen.getByRole("button", {name: "Submit for approval"}).hasAttribute("disabled")).toBe(true);
  });

  it("does not expose proposal or approval actions without server authority", () => {
    render(<ProgramStandardsPanel aggregate={aggregate([adoption("a", "SECURITY")])}
      operations={[]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    expect(screen.queryByRole("button", {name: "Add standard"})).toBeNull();
    expect(screen.queryByRole("button", {name: "Review SECURITY 2026"})).toBeNull();
  });
});
