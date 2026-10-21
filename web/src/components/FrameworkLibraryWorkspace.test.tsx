import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "../http";
import { loadFrameworkEditionPage } from "../programOperationsApi";
import type { FrameworkEditionOption } from "../programOperationsApi";
import { FrameworkLibraryWorkspace } from "./FrameworkLibraryWorkspace";

vi.mock("../programOperationsApi", () => ({ loadFrameworkEditionPage: vi.fn() }));

const edition = (id: string, frameworkCode = "SECURITY"): FrameworkEditionOption => ({
  id, framework_code: frameworkCode, publisher_code: "REFERENCE",
  edition_code: "2026", title: "Published " + frameworkCode,
  jurisdiction: "GLOBAL", source_published_on: "2026-10-01",
  source_url: "https://example.org/" + id,
  license_basis: "CITATIONS_ONLY", approval_reference: "release-approved",
  source_sha256: "f".repeat(64),
});

describe("Framework Library workspace", () => {
  beforeEach(() => vi.resetAllMocks());

  it("inspects exact source provenance and passes only the selected edition into Program selection", async () => {
    const onChooseProgram = vi.fn();
    vi.mocked(loadFrameworkEditionPage).mockResolvedValue({
      items: [edition("edition-1")], next_after_id: "",
    });
    render(<FrameworkLibraryWorkspace onBack={vi.fn()} onChooseProgram={onChooseProgram}/>);
    expect(await screen.findByRole("button", {name: "Inspect edition for Published SECURITY 2026"})).toBeTruthy();
    expect(screen.getByText("1 matching loaded · 1 loaded editions · GLOBAL")).toBeTruthy();
    expect(screen.getByRole("heading", {name: "Framework Library", level: 1})).toBeTruthy();
    expect(screen.getByRole("region", {name: "Framework filters"})).toBeTruthy();
    expect(screen.getByRole("heading", {name: "Edition details"})).toBeTruthy();
    expect(screen.getByRole("heading", {name: "Use in a Program"})).toBeTruthy();
    fireEvent.click(screen.getByRole("button", {name: "Inspect edition for Published SECURITY 2026"}));
    const details = await screen.findByRole("complementary", {name: "Framework edition details"});
    expect(screen.getByRole("list", {name: "Published framework editions"})).toBeTruthy();
    expect(details.textContent).toContain("Citations only");
    expect(details.textContent).toContain("release-approved");
    const checksumDisclosure = screen.getByText("Source checksums").closest("details");
    expect(checksumDisclosure?.open).toBe(false);
    fireEvent.click(screen.getByText("Source checksums"));
    expect(checksumDisclosure?.open).toBe(true);
    expect(details.textContent).toContain("Publication does not establish Program adoption");
    expect(screen.getByRole("link", {name: /Show published source/}).getAttribute("href")).toBe("https://example.org/edition-1");
    expect(onChooseProgram).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", {name: "Choose Program"}));
    expect(onChooseProgram).toHaveBeenCalledWith(edition("edition-1"));
    expect(loadFrameworkEditionPage).toHaveBeenCalledWith("GLOBAL", "");
  });

  it("orders the approved Library menu ahead of search and edition content, without fake update actions", async () => {
    const onBack = vi.fn();
    vi.mocked(loadFrameworkEditionPage).mockResolvedValue({
      items: [edition("edition-1")], next_after_id: "",
    });
    render(<FrameworkLibraryWorkspace onBack={onBack} onChooseProgram={vi.fn()}/>);
    await screen.findByRole("button", { name: "Inspect edition for Published SECURITY 2026" });
    const heading = screen.getByRole("heading", { name: "Framework Library", level: 1 });
    const header = heading.closest(".cs-framework-library__page-heading");
    expect(header).toBeTruthy();
    const views = screen.getByRole("navigation", { name: "Library views" });
    expect(Boolean(header!.compareDocumentPosition(views) & Node.DOCUMENT_POSITION_FOLLOWING)).toBe(true);
    const rights = screen.getByRole("button", { name: /Content rights/ });
    expect(rights).toBeTruthy();
    expect(within(views).getByText("Explore").getAttribute("aria-current")).toBe("page");
    expect(within(views).getByRole("button", { name: "Edition updates" }).hasAttribute("disabled")).toBe(true);
    const filter = screen.getByRole("searchbox", { name: "Search loaded standards" });
    const list = screen.getByRole("list", { name: "Published framework editions" });
    expect(Boolean(views.compareDocumentPosition(filter) & Node.DOCUMENT_POSITION_FOLLOWING)).toBe(true);
    expect(Boolean(filter.compareDocumentPosition(list) & Node.DOCUMENT_POSITION_FOLLOWING)).toBe(true);
    const workspace = screen.getByRole("region", { name: "Framework Library workspace" });
    const filterBar = workspace.querySelector(".cs-filter-bar");
    const layout = workspace.querySelector(".cs-framework-library__layout");
    expect(filterBar?.parentElement).toBe(workspace);
    expect(filterBar?.nextElementSibling).toBe(layout);
    fireEvent.click(within(views).getByRole("button", { name: "In my Programs" }));
    expect(onBack).toHaveBeenCalledTimes(1);
  });

  it("switches content-rights filters over the loaded edition population without inventing catalog entries", async () => {
    vi.mocked(loadFrameworkEditionPage).mockResolvedValue({
      items: [
        edition("citations", "SECURITY"),
        {...edition("summary", "PRIVACY"), license_basis: "SUMMARIES_PERMITTED"},
        {...edition("unknown", "RESILIENCE"), license_basis: "UNVERIFIED"},
      ], next_after_id: "",
    });
    render(<FrameworkLibraryWorkspace onBack={vi.fn()} onChooseProgram={vi.fn()}/>);
    await screen.findByRole("button", { name: "Inspect edition for Published SECURITY 2026" });
    fireEvent.click(screen.getByRole("button", { name: /Content rights/ }));
    fireEvent.click(await screen.findByRole("option", { name: "Summaries permitted" }));
    expect(screen.getAllByRole("button", { name: /Inspect edition for/ })).toHaveLength(1);
    expect(screen.getByRole("button", { name: "Inspect edition for Published PRIVACY 2026" })).toBeTruthy();
    expect(screen.getByText("1 matching loaded · 3 loaded editions · GLOBAL")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(screen.getAllByRole("button", { name: /Inspect edition for/ })).toHaveLength(3);
    expect(loadFrameworkEditionPage).toHaveBeenCalledTimes(1);
  });

  it("uses an accessible detail sheet on compact viewports", async () => {
    const original = window.matchMedia;
    Object.defineProperty(window, "matchMedia", {
      configurable: true,
      value: () => ({matches: true, addEventListener: () => {}, removeEventListener: () => {}}),
    });
    try {
      vi.mocked(loadFrameworkEditionPage).mockResolvedValue({
        items: [edition("edition-1")], next_after_id: "",
      });
      render(<FrameworkLibraryWorkspace onBack={vi.fn()} onChooseProgram={vi.fn()}/>);
      const inspect = await screen.findByRole("button", {name: "Inspect edition for Published SECURITY 2026"});
      expect(screen.queryByRole("complementary", {name: "Framework edition details"})).toBeNull();
      fireEvent.click(inspect);
      expect(await screen.findByRole("dialog", {name: "Edition details: Published SECURITY"})).toBeTruthy();
      fireEvent.click(screen.getByRole("button", {name: "Close"}));
      await waitFor(() => expect(screen.queryByRole("dialog", {name: "Edition details: Published SECURITY"})).toBeNull());
    } finally {
      Object.defineProperty(window, "matchMedia", {configurable: true, value: original});
    }
  });

  it("keeps mobile jurisdiction editing collapsed until requested", async () => {
    const original = window.matchMedia;
    Object.defineProperty(window, "matchMedia", {
      configurable: true,
      value: () => ({matches: true, addEventListener: () => {}, removeEventListener: () => {}}),
    });
    try {
      vi.mocked(loadFrameworkEditionPage)
        .mockResolvedValueOnce({items: [edition("global")], next_after_id: ""})
        .mockResolvedValueOnce({items: [{...edition("local"), jurisdiction: "NG"}], next_after_id: ""});
      render(<FrameworkLibraryWorkspace onBack={vi.fn()} onChooseProgram={vi.fn()}/>);
      await screen.findByRole("button", {name: "Inspect edition for Published SECURITY 2026"});
      expect(screen.queryByLabelText("Jurisdiction code")).toBeNull();
      const toggle = screen.getByRole("button", {name: "Jurisdiction: GLOBAL"});
      expect(toggle.getAttribute("aria-expanded")).toBe("false");
      fireEvent.click(toggle);
      expect(screen.getByLabelText("Jurisdiction code")).toBeTruthy();
      fireEvent.change(screen.getByLabelText("Jurisdiction code"), {target: {value: "ng"}});
      fireEvent.click(screen.getByRole("button", {name: "Apply"}));
      await waitFor(() => expect(loadFrameworkEditionPage).toHaveBeenLastCalledWith("NG", ""));
      expect(screen.getByRole("button", {name: "Jurisdiction: NG"}).getAttribute("aria-expanded")).toBe("false");
      expect(screen.queryByLabelText("Jurisdiction code")).toBeNull();
    } finally {
      Object.defineProperty(window, "matchMedia", {configurable: true, value: original});
    }
  });

  it("searches only loaded editions and advances the authoritative cursor", async () => {
    vi.mocked(loadFrameworkEditionPage)
      .mockResolvedValueOnce({items: [edition("edition-1")], next_after_id: "edition-1"})
      .mockResolvedValueOnce({items: [edition("edition-2", "PRIVACY")], next_after_id: ""});
    render(<FrameworkLibraryWorkspace onBack={vi.fn()} onChooseProgram={vi.fn()}/>);
    await screen.findByRole("button", {name: "Inspect edition for Published SECURITY 2026"});
    fireEvent.change(screen.getByRole("searchbox", {name: "Search loaded standards"}), {target: {value: "privacy"}});
    expect(screen.getByText("No matching loaded editions")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", {name: "Load more editions"}));
    expect(await screen.findByRole("button", {name: "Inspect edition for Published PRIVACY 2026"})).toBeTruthy();
    expect(loadFrameworkEditionPage).toHaveBeenNthCalledWith(2, "GLOBAL", "edition-1");
  });

  it("distinguishes denied access from an empty catalog without exposing stale results", async () => {
    vi.mocked(loadFrameworkEditionPage).mockRejectedValue(new ApiError(403, "Access denied"));
    render(<FrameworkLibraryWorkspace onBack={vi.fn()} onChooseProgram={vi.fn()}/>);
    expect(await screen.findByText("Published editions unavailable")).toBeTruthy();
    expect(screen.queryByRole("list", {name: "Published framework editions"})).toBeNull();
    expect(screen.queryByRole("button", {name: /Inspect edition/})).toBeNull();
  });

  it("retains loaded sources but blocks choosing a Program when a later catalog page fails", async () => {
    const onChooseProgram = vi.fn();
    vi.mocked(loadFrameworkEditionPage)
      .mockResolvedValueOnce({items: [edition("edition-1")], next_after_id: "edition-1"})
      .mockRejectedValueOnce(new Error("Request failed"))
      .mockResolvedValueOnce({items: [edition("edition-2", "PRIVACY")], next_after_id: ""});
    render(<FrameworkLibraryWorkspace onBack={vi.fn()} onChooseProgram={onChooseProgram}/>);
    fireEvent.click(await screen.findByRole("button", {name: "Inspect edition for Published SECURITY 2026"}));
    expect(screen.getByRole("button", {name: "Choose Program"}).hasAttribute("disabled")).toBe(false);
    fireEvent.click(screen.getByRole("button", {name: "Load more editions"}));
    expect(await screen.findByText(/Additional editions unavailable/)).toBeTruthy();
    expect(screen.getByRole("list", {name: "Published framework editions"}).textContent).toContain("Published SECURITY");
    expect(screen.getByRole("button", {name: "Choose Program"}).hasAttribute("disabled")).toBe(true);
    fireEvent.click(screen.getByRole("button", {name: "Retry catalog"}));
    expect(await screen.findByRole("button", {name: "Inspect edition for Published PRIVACY 2026"})).toBeTruthy();
    expect(screen.getByRole("button", {name: "Choose Program"}).hasAttribute("disabled")).toBe(false);
    expect(onChooseProgram).not.toHaveBeenCalled();
    expect(loadFrameworkEditionPage).toHaveBeenNthCalledWith(3, "GLOBAL", "edition-1");
  });

  it("shows source-backed first-edition preview without silently selecting it for Program adoption", async () => {
    const onChooseProgram = vi.fn();
    vi.mocked(loadFrameworkEditionPage).mockResolvedValue({items: [edition("edition-1")], next_after_id: ""});
    render(<FrameworkLibraryWorkspace onBack={vi.fn()} onChooseProgram={onChooseProgram}/>);
    const first = await screen.findByRole("button", {name: "Inspect edition for Published SECURITY 2026"});
    const details = screen.getByRole("complementary", {name: "Framework edition details"});
    expect(details.textContent).toContain("Published SECURITY");
    expect(details.textContent).toContain("Citations only");
    expect(details.textContent).toContain("Select an edition in the list to continue.");
    const hero = details.querySelector(".cs-framework-library__detail-band");
    expect(hero?.getAttribute("data-selected")).toBe("false");
    expect(hero?.textContent).toContain("Edition preview");
    expect(hero?.textContent).toContain("Published · 2026-10-01");
    expect(first.getAttribute("aria-pressed")).not.toBe("true");
    expect(screen.getByRole("button", {name: "Choose Program"}).hasAttribute("disabled")).toBe(true);
    expect(onChooseProgram).not.toHaveBeenCalled();
    fireEvent.click(first);
    expect(hero?.getAttribute("data-selected")).toBe("true");
    expect(hero?.textContent).toContain("Selected edition");
    expect(screen.getByRole("button", {name: "Choose Program"}).hasAttribute("disabled")).toBe(false);
  });

  it("never carries preview selection between filtered edition identities", async () => {
    const onChooseProgram = vi.fn();
    vi.mocked(loadFrameworkEditionPage).mockResolvedValue({
      items: [edition("first", "SECURITY"), edition("second", "PRIVACY")], next_after_id: "",
    });
    render(<FrameworkLibraryWorkspace onBack={vi.fn()} onChooseProgram={onChooseProgram}/>);
    const first = await screen.findByRole("button", {name: "Inspect edition for Published SECURITY 2026"});
    fireEvent.click(first);
    expect(screen.getByRole("button", {name: "Choose Program"}).hasAttribute("disabled")).toBe(false);
    fireEvent.change(screen.getByRole("searchbox", {name: "Search loaded standards"}), {target: {value: "PRIVACY"}});
    const details = screen.getByRole("complementary", {name: "Framework edition details"});
    expect(details.textContent).toContain("Published PRIVACY");
    expect(details.textContent).not.toContain("Published SECURITY");
    expect(screen.getByRole("button", {name: "Choose Program"}).hasAttribute("disabled")).toBe(true);
    expect(onChooseProgram).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", {name: "Inspect edition for Published PRIVACY 2026"}));
    expect(screen.getByRole("button", {name: "Choose Program"}).hasAttribute("disabled")).toBe(false);
  });

  it("rejects invalid jurisdiction input before issuing another request", async () => {
    vi.mocked(loadFrameworkEditionPage).mockResolvedValue({items: [], next_after_id: ""});
    render(<FrameworkLibraryWorkspace onBack={vi.fn()} onChooseProgram={vi.fn()}/>);
    await screen.findByText("No published editions");
    fireEvent.change(screen.getByLabelText("Jurisdiction code"), {target: {value: "NG / GLOBAL"}});
    fireEvent.click(screen.getByRole("button", {name: "Apply"}));
    expect(screen.getByRole("alert").textContent).toContain("valid jurisdiction code");
    expect(loadFrameworkEditionPage).toHaveBeenCalledTimes(1);
  });

  it("discards an older jurisdiction response after the user changes the filter", async () => {
    let resolveFirst!: (page: {items: FrameworkEditionOption[]; next_after_id: string}) => void;
    vi.mocked(loadFrameworkEditionPage).mockImplementationOnce(() => new Promise((resolve) => { resolveFirst = resolve; }));
    vi.mocked(loadFrameworkEditionPage).mockResolvedValueOnce({
      items: [{...edition("edition-ng"), jurisdiction: "NG"}], next_after_id: "",
    });
    render(<FrameworkLibraryWorkspace onBack={vi.fn()} onChooseProgram={vi.fn()}/>);
    await waitFor(() => expect(loadFrameworkEditionPage).toHaveBeenCalledTimes(1));
    fireEvent.change(screen.getByLabelText("Jurisdiction code"), {target: {value: "ng"}});
    fireEvent.click(screen.getByRole("button", {name: "Apply"}));
    expect(await screen.findByRole("button", {name: "Inspect edition for Published SECURITY 2026"})).toBeTruthy();
    resolveFirst({items: [edition("edition-global", "OLD")], next_after_id: ""});
    await waitFor(() => expect(screen.queryByRole("button", {name: /Inspect edition for Published OLD/})).toBeNull());
    expect(loadFrameworkEditionPage).toHaveBeenLastCalledWith("NG", "");
  });

  it("returns to Programs without a material command", () => {
    vi.mocked(loadFrameworkEditionPage).mockResolvedValue({items: [], next_after_id: ""});
    const onBack = vi.fn();
    render(<FrameworkLibraryWorkspace onBack={onBack} onChooseProgram={vi.fn()}/>);
    fireEvent.click(within(screen.getByRole("navigation", {name: "Library views"})).getByRole("button", {name: "In my Programs"}));
    expect(onBack).toHaveBeenCalledTimes(1);
  });
  
  it("blocks Program handoff when an edition has no verifiable source or reuse rights", async () => {
    const onChooseProgram = vi.fn();
    vi.mocked(loadFrameworkEditionPage).mockResolvedValue({
      items: [
        {...edition("unknown"), license_basis: "UNVERIFIED"},
        {...edition("insecure", "PRIVACY"), source_url: "http://example.org/editions"},
      ],
      next_after_id: "",
    });
    render(<FrameworkLibraryWorkspace onBack={vi.fn()} onChooseProgram={onChooseProgram}/>);
    for (const label of [
      "Inspect edition for Published SECURITY 2026",
      "Inspect edition for Published PRIVACY 2026",
    ]) {
      fireEvent.click(await screen.findByRole("button", {name: label}));
      expect(screen.getByText(/Source or reuse rights are unverified/)).toBeTruthy();
      const action = screen.getByRole("button", {name: "Choose Program"});
      expect(action.hasAttribute("disabled")).toBe(true);
      fireEvent.click(action);
    }
    expect(onChooseProgram).not.toHaveBeenCalled();
  });

  
  it("keeps source rights readable in compact register rows without a separate badge row", async () => {
    const original = window.matchMedia;
    Object.defineProperty(window, "matchMedia", {
      configurable: true,
      value: () => ({matches: true, addEventListener: () => {}, removeEventListener: () => {}}),
    });
    try {
      vi.mocked(loadFrameworkEditionPage).mockResolvedValue({
        items: [edition("edition-1")], next_after_id: "",
      });
      render(<FrameworkLibraryWorkspace onBack={vi.fn()} onChooseProgram={vi.fn()}/>);
      const row = await screen.findByRole("button", {name: "Inspect edition for Published SECURITY 2026"});
      expect(row.textContent).toContain("GLOBAL · Citations only");
      expect(row.querySelector(".cs-selectable-record__end")).toBeNull();
      fireEvent.click(row);
      expect(await screen.findByRole("dialog", {name: "Edition details: Published SECURITY"})).toBeTruthy();
      expect(screen.getByText("2026-10-01")).toBeTruthy();
    } finally {
      Object.defineProperty(window, "matchMedia", {configurable: true, value: original});
    }
  });

  it("matches the approved desktop edition/publisher/right-hand record hierarchy without duplicating metadata", async () => {
    vi.mocked(loadFrameworkEditionPage).mockResolvedValue({
      items: [edition("standard-a"), {...edition("standard-b", "PRIVACY"), license_basis: "UNVERIFIED"}],
      next_after_id: "",
    });
    render(<FrameworkLibraryWorkspace onBack={vi.fn()} onChooseProgram={vi.fn()}/>);
    const item = await screen.findByRole("button", {name: "Inspect edition for Published SECURITY 2026"});
    const facts = item.querySelector(".cs-framework-library__record-facts");
    expect(facts).not.toBeNull();
    expect(facts?.textContent).toContain("REFERENCE");
    expect(facts?.textContent).toContain("Edition 2026");
    expect(facts?.textContent).toContain("Citations only");
    expect(item.querySelector(".cs-selectable-record__metadata")?.textContent).toBe("SECURITY");
    expect(item.querySelector(".cs-selectable-record__description")?.textContent).toContain("Published 2026-10-01");

    const unknown = screen.getByRole("button", {name: "Inspect edition for Published PRIVACY 2026"});
    expect(unknown.querySelector(".cs-framework-library__record-facts")?.textContent).toContain("Not verified");
    fireEvent.click(item);
    const detail = await screen.findByRole("complementary", {name: "Framework edition details"});
    expect(detail.querySelector(".cs-framework-library__detail-subtitle")?.textContent).toBe("REFERENCE · SECURITY");
    expect(screen.getByRole("button", {name: "Choose Program"}).hasAttribute("disabled")).toBe(false);
  });


});
