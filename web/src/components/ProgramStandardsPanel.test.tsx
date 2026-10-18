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
  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(loadFrameworkEditionPage).mockResolvedValue({items: [], next_after_id: ""});
  });

  it("opens existing requirement and form work for approved active standards", () => {
    const onNavigate = vi.fn();
    render(<ProgramStandardsPanel
      aggregate={aggregate([{...adoption("a", "SECURITY"), status: "ACTIVE"}])}
      operations={[]} onUpdated={vi.fn()} onReload={vi.fn()} onNavigate={onNavigate}/>);
    fireEvent.click(screen.getByRole("button", {name: "Review requirement links"}));
    fireEvent.click(screen.getByRole("button", {name: "Collection forms"}));
    expect(onNavigate).toHaveBeenNthCalledWith(1, "requirements-controls");
    expect(onNavigate).toHaveBeenNthCalledWith(2, "monitoring");
    expect(loadFrameworkEditionPage).not.toHaveBeenCalled();
    expect(proposeProgramFrameworkAdoption).not.toHaveBeenCalled();
  });

  it("does not offer adoption follow-up for pending or rejected editions", () => {
    const { rerender } = render(<ProgramStandardsPanel
      aggregate={aggregate([adoption("a", "SECURITY")])}
      operations={[]} onUpdated={vi.fn()} onReload={vi.fn()} onNavigate={vi.fn()}/>);
    expect(screen.queryByRole("button", {name: "Review requirement links"})).toBeNull();
    expect(screen.queryByRole("button", {name: "Collection forms"})).toBeNull();
    rerender(<ProgramStandardsPanel
      aggregate={aggregate([{...adoption("a", "SECURITY"), status: "REJECTED"}])}
      operations={[]} onUpdated={vi.fn()} onReload={vi.fn()} onNavigate={vi.fn()}/>);
    expect(screen.queryByRole("button", {name: "Review requirement links"})).toBeNull();
    expect(screen.queryByRole("button", {name: "Collection forms"})).toBeNull();
  });

  it("isolates the decision rationale for each pending adoption", () => {
    render(<ProgramStandardsPanel aggregate={aggregate([adoption("a", "SECURITY"), adoption("b", "PRIVACY")])}
      operations={[approve, reject]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", {name: "Review SECURITY 2026"}));
    fireEvent.change(screen.getByRole("textbox", {name: /Decision rationale for SECURITY 2026/}),
      {target: {value: "Only security evidence was reviewed"}});
    expect(screen.getByRole("dialog", {name: "Review standard adoption: SECURITY 2026"})).toBeTruthy();
    fireEvent.click(screen.getByRole("button", {name: "Close standard review"}));
    expect(screen.queryByRole("dialog", {name: "Review standard adoption: SECURITY 2026"})).toBeNull();
    fireEvent.click(screen.getByRole("button", {name: "Review PRIVACY 2026"}));
    const other = screen.getByRole("textbox", {name: /Decision rationale for PRIVACY 2026/}) as HTMLTextAreaElement;
    expect(other.value).toBe("");
    expect(other.required).toBe(true);
    expect(screen.getByRole("button", {name: "Approve standard"}).hasAttribute("disabled")).toBe(true);
    expect(decideProgramFrameworkAdoption).not.toHaveBeenCalled();
  });

  it("retains a failed review decision in its focused sheet and blocks conflicting approval", async () => {
    vi.mocked(loadFrameworkEditionPage).mockResolvedValue({items: [edition("a")], next_after_id: ""});
    const onReload = vi.fn();
    vi.mocked(decideProgramFrameworkAdoption).mockRejectedValue(
      new ApiError(409, "Program version changed.", "version_conflict"),
    );
    render(<ProgramStandardsPanel aggregate={aggregate([adoption("a", "SECURITY")])}
      operations={[approve, reject]} onUpdated={vi.fn()} onReload={onReload}/>);
    fireEvent.click(screen.getByRole("button", {name: "Review SECURITY 2026"}));
    const dialog = screen.getByRole("dialog", {name: "Review standard adoption: SECURITY 2026"});
    expect(dialog).toBeTruthy();
    fireEvent.change(screen.getByRole("textbox", {name: /Decision rationale for SECURITY 2026/}),
      {target: {value: "Reviewed the pending submission"}});
    await waitFor(() => expect(screen.getByRole("button", {name: "Approve standard"}).hasAttribute("disabled")).toBe(false));
    fireEvent.click(screen.getByRole("button", {name: "Approve standard"}));
    expect(await screen.findByRole("alert")).toBeTruthy();
    expect((screen.getByRole("textbox", {name: /Decision rationale for SECURITY 2026/}) as HTMLTextAreaElement).value)
      .toBe("Reviewed the pending submission");
    expect(screen.getByRole("button", {name: "Approve standard"}).hasAttribute("disabled")).toBe(true);
    fireEvent.click(screen.getByRole("button", {name: "Reload Program"}));
    expect(onReload).toHaveBeenCalledOnce();
  });

  it("blocks approval without exact published source but permits rejection", async () => {
    const updated = vi.fn();
    vi.mocked(decideProgramFrameworkAdoption).mockResolvedValue(aggregate([]));
    render(<ProgramStandardsPanel aggregate={aggregate([adoption("a", "SECURITY")])}
      operations={[approve, reject]} onUpdated={updated} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", {name: "Review SECURITY 2026"}));
    expect(await screen.findByText("Pending edition not found in this catalog jurisdiction.")).toBeTruthy();
    fireEvent.change(screen.getByRole("textbox", {name: /Decision rationale for SECURITY 2026/}),
      {target: {value: "Source not available"}});
    expect(screen.getByRole("button", {name: "Approve standard"}).hasAttribute("disabled")).toBe(true);
    fireEvent.click(screen.getByRole("button", {name: "Reject standard"}));
    await waitFor(() => expect(decideProgramFrameworkAdoption).toHaveBeenCalledWith(
      "program-1", "a", 4, "reject", "Source not available",
    ));
    expect(updated).toHaveBeenCalledOnce();
  });

  it("never approves a matching edition ID with different publisher or standard", async () => {
    vi.mocked(loadFrameworkEditionPage).mockResolvedValue({items: [edition("a", "PRIVACY")], next_after_id: ""});
    render(<ProgramStandardsPanel aggregate={aggregate([adoption("a", "SECURITY")])}
      operations={[approve, reject]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", {name: "Review SECURITY 2026"}));
    expect(await screen.findByText("Published edition identity does not match this adoption.")).toBeTruthy();
    fireEvent.change(screen.getByRole("textbox", {name: /Decision rationale for SECURITY 2026/}),
      {target: {value: "Publisher identity conflict"}});
    expect(screen.getByRole("button", {name: "Approve standard"}).hasAttribute("disabled")).toBe(true);
    expect(decideProgramFrameworkAdoption).not.toHaveBeenCalled();
  });

  it("loads the next catalog page before allowing approval of the exact pinned edition", async () => {
    vi.mocked(loadFrameworkEditionPage)
      .mockResolvedValueOnce({items: [edition("b")], next_after_id: "b"})
      .mockResolvedValueOnce({items: [edition("a")], next_after_id: ""});
    render(<ProgramStandardsPanel aggregate={aggregate([adoption("a", "SECURITY")])}
      operations={[approve, reject]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", {name: "Review SECURITY 2026"}));
    const more = await screen.findByRole("button", {name: "Load more editions"});
    expect(screen.getByRole("button", {name: "Approve standard"}).hasAttribute("disabled")).toBe(true);
    fireEvent.click(more);
    expect(await screen.findByRole("link", {name: /Show published source/})).toBeTruthy();
    await waitFor(() => expect(screen.getByRole("button", {name: "Approve standard"}).hasAttribute("disabled")).toBe(true));
    fireEvent.change(screen.getByRole("textbox", {name: /Decision rationale for SECURITY 2026/}),
      {target: {value: "Edition source inspected"}});
    expect(screen.getByRole("button", {name: "Approve standard"}).hasAttribute("disabled")).toBe(false);
    expect(loadFrameworkEditionPage).toHaveBeenNthCalledWith(2, "GLOBAL", "b");
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
    fireEvent.change(screen.getByRole("textbox", {name: /Reason for adoption/}), {target: {value: "Regulatory requirement"}});
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
    fireEvent.change(screen.getByRole("textbox", {name: /Reason for adoption/}), {target: {value: "Applies to scope"}});
    const search = screen.getByRole("searchbox", {name: "Search loaded standards"});
    expect(search.closest("form")).toBeNull();
    expect(screen.getByRole("textbox", {name: /Reason for adoption/}).closest("form")).not.toBeNull();
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

  it("resolves a selected Library edition before offering the governed proposal", async () => {
    const onUpdated = vi.fn();
    vi.mocked(loadFrameworkEditionPage).mockResolvedValue({
      items: [edition("a")], next_after_id: "",
    });
    vi.mocked(proposeProgramFrameworkAdoption).mockResolvedValue({
      ...aggregate(), program: {...program, version: 5},
    });
    render(<ProgramStandardsPanel aggregate={aggregate()}
      frameworkEdition={{editionID: "a", jurisdiction: "GLOBAL"}}
      operations={[owner]} onUpdated={onUpdated} onReload={vi.fn()}/>);
    const radio = await screen.findByRole("radio", {name: /Published SECURITY/}) as HTMLInputElement;
    expect(radio.checked).toBe(true);
    expect(screen.getByLabelText("Selected edition").textContent).toContain("Citations only");
    expect(proposeProgramFrameworkAdoption).not.toHaveBeenCalled();
    fireEvent.change(screen.getByRole("textbox", {name: /Reason for adoption/}), {target: {value: "Governed adoption request"}});
    fireEvent.click(screen.getByRole("button", {name: "Submit for approval"}));
    await waitFor(() => expect(proposeProgramFrameworkAdoption).toHaveBeenCalledWith(
      "program-1", 4, "a", "Governed adoption request",
    ));
    expect(onUpdated).toHaveBeenCalledTimes(1);
  });

  it("never proposes an unverified deep-linked edition outside the loaded catalog page", async () => {
    vi.mocked(loadFrameworkEditionPage).mockResolvedValue({
      items: [edition("b")], next_after_id: "",
    });
    render(<ProgramStandardsPanel aggregate={aggregate()}
      frameworkEdition={{editionID: "a", jurisdiction: "GLOBAL"}}
      operations={[owner]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    expect(await screen.findByText("The selected edition is unavailable in this jurisdiction. Return to the Framework Library.")).toBeTruthy();
    fireEvent.change(screen.getByRole("textbox", {name: /Reason for adoption/}), {target: {value: "Proposed by deep link"}});
    expect(screen.getByRole("button", {name: "Submit for approval"}).hasAttribute("disabled")).toBe(true);
    expect(proposeProgramFrameworkAdoption).not.toHaveBeenCalled();
  });

  it("does not open adoption from a Library hint without verified Program authority", () => {
    render(<ProgramStandardsPanel aggregate={aggregate()}
      frameworkEdition={{editionID: "a", jurisdiction: "GLOBAL"}}
      operations={[]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    expect(screen.getByText(/Edition selected. Adoption is unavailable/)).toBeTruthy();
    expect(screen.queryByRole("button", {name: "Submit for approval"})).toBeNull();
    expect(loadFrameworkEditionPage).not.toHaveBeenCalled();
  });

  it("does not expose proposal or approval actions without server authority", () => {
    render(<ProgramStandardsPanel aggregate={aggregate([adoption("a", "SECURITY")])}
      operations={[]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    expect(screen.queryByRole("button", {name: "Add standard"})).toBeNull();
    expect(screen.queryByRole("button", {name: "Review SECURITY 2026"})).toBeNull();
  });
  
  it("fails closed on unverified rights and unsafe source before proposing a Program adoption", async () => {
    vi.mocked(loadFrameworkEditionPage).mockResolvedValue({
      items: [
        {...edition("unknown"), license_basis: "UNVERIFIED"},
        {...edition("unsafe", "PRIVACY"), source_url: "http://example.org/unsafe"},
      ],
      next_after_id: "",
    });
    render(<ProgramStandardsPanel aggregate={aggregate()}
      operations={[owner]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", {name: "Add standard"}));
    for (const label of [/Published SECURITY/, /Published PRIVACY/]) {
      fireEvent.click(await screen.findByRole("radio", {name: label}));
      expect(screen.getByText(/Source or reuse rights are unverified/)).toBeTruthy();
      fireEvent.change(screen.getByRole("textbox", {name: /Reason for adoption/}),
        {target: {value: "Required compliance coverage"}});
      const submit = screen.getByRole("button", {name: "Submit for approval"});
      expect(submit.hasAttribute("disabled")).toBe(true);
      fireEvent.submit(submit.closest("form")!);
    }
    expect(proposeProgramFrameworkAdoption).not.toHaveBeenCalled();
  });

});
