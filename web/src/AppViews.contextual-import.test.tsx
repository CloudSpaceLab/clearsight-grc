import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ProgramsView, ReferenceJourneysView, WorkView } from "./AppViews";

vi.mock("./components/ProgramsWorkspace", () => ({ ProgramsWorkspace: () => <div>Programs workspace</div> }));
vi.mock("./components/MattersWorkspace", () => ({ MattersWorkspace: () => <div>Matters workspace</div> }));
vi.mock("./components/EvidenceWorkspace", () => ({ EvidenceWorkspace: () => <div>Evidence workspace</div> }));
vi.mock("./components/BankJourneysWorkspace", () => ({ BankJourneysWorkspace: () => <div>Reference journeys</div> }));
vi.mock("./components/TodayInterventions", () => ({ TodayInterventions: () => <div>Today interventions</div> }));
vi.mock("./components/AssignedWorkQueue", () => ({ AssignedWorkQueue: ({ items }: { items: unknown[] }) => <div>Assigned work · {items.length}</div> }));
vi.mock("./components/WorkspaceErrorBoundary", () => ({ WorkspaceErrorBoundary: ({ children }: { children: ReactNode }) => <>{children}</> }));

afterEach(() => vi.restoreAllMocks());

describe("contextual document analysis entry", () => {
  it("loads the non-production reference journeys on demand", async () => {
    render(<ReferenceJourneysView organizationName="Meridian Trust Bank"/>);
    expect(screen.getByRole("heading", { name: "Reference journeys" })).toBeTruthy();
    expect(await screen.findByText("Reference journeys", { selector: "div" })).toBeTruthy();
  });

  it("offers document analysis from Programs when the capability is supplied", () => {
    const onAnalyzeDocument = vi.fn();
    render(<ProgramsView organizationName="Meridian Trust Bank" onAnalyzeDocument={onAnalyzeDocument}/>);

    fireEvent.click(screen.getByRole("button", { name: "Analyze document to create or update Programs" }));

    expect(onAnalyzeDocument).toHaveBeenCalledTimes(1);
  });

  it("uses the Program record title without a redundant portfolio heading", () => {
    const onAnalyzeDocument = vi.fn();
    const { rerender } = render(<ProgramsView
      organizationName="Meridian Trust Bank"
      targetID="program-1"
      onAnalyzeDocument={onAnalyzeDocument}
    />);
    expect(screen.queryByRole("heading", { name: "Programs" })).toBeNull();
    expect(screen.queryByRole("button", { name: /Analyze document/ })).toBeNull();
    expect(screen.getByText("Programs workspace")).toBeTruthy();

    rerender(<ProgramsView organizationName="Meridian Trust Bank" programsPage="library"/>);
    // Framework Library owns its complete V2 heading; this wrapper must not
    // create a second generic Programs/Library hero above that layout.
    expect(screen.queryByRole("heading", { name: "Framework Library" })).toBeNull();
    expect(screen.getByText("Programs workspace")).toBeTruthy();
    expect(screen.queryByRole("button", { name: /Analyze document/ })).toBeNull();
  });

  it("keeps a governed return action above exact evidence work", async () => {
    const onReturn = vi.fn();
    render(<WorkView
      organizationName="Meridian Trust Bank"
      actorPrincipalID="role-cro"
      evidenceScopeToken={0}
      tab="evidence"
      onTab={vi.fn()}
      onBackMatter={vi.fn()}
      returnContext={{ label: "Back to RCSA cycle", onReturn }}
      assignedItems={[]}
      assignedState="live"
      onOpenAssignedItem={vi.fn()}
      sources={[]}
      requests={[]}
      evidenceSourceState="live"
      evidenceRequestState="live"
      onEvidenceRetry={vi.fn()}
      onEvidenceRequestUpdated={vi.fn(() => true)}
      onOpenEvidence={vi.fn()}
    />);

    fireEvent.click(await screen.findByRole("button", { name: "Back to RCSA cycle" }));
    expect(onReturn).toHaveBeenCalledTimes(1);
    expect(await screen.findByText("Evidence workspace")).toBeTruthy();
  });

  it("offers document analysis for issues and changes but not evidence review", () => {
    const onAnalyzeDocument = vi.fn();
    const props = {
      organizationName: "Meridian Trust Bank",
      actorPrincipalID: "role-cro",
      evidenceScopeToken: 0,
      onTab: vi.fn(),
      onBackMatter: vi.fn(),
      assignedItems: [],
      assignedState: "live" as const,
      onOpenAssignedItem: vi.fn(),
      sources: [],
      requests: [],
      evidenceSourceState: "live" as const,
      evidenceRequestState: "live" as const,
      onEvidenceRetry: vi.fn(),
      onEvidenceRequestUpdated: vi.fn(() => true),
      onOpenEvidence: vi.fn(),
      onAnalyzeDocument,
    };
    const assigned = render(<WorkView {...props} assignedItems={[{ id: "work-1" } as never]} tab="assigned"/>);
    expect(screen.getByText("Assigned work · 1")).toBeTruthy();
    expect(screen.getByRole("navigation", { name: "Breadcrumb" })).toBeTruthy();
    expect(screen.getByRole("link", { name: "Home" }).getAttribute("href")).toBe("#today");
    expect(screen.getByText("1 loaded · 0 overdue · 0 approvals")).toBeTruthy();
    assigned.rerender(<WorkView {...props} assignedItems={[{ id: "work-1" } as never]} assignedState="unavailable" tab="assigned"/>);
    expect(screen.getByText("Assigned work unavailable")).toBeTruthy();
    expect(screen.queryByText("1 loaded · 0 overdue · 0 approvals")).toBeNull();
    assigned.unmount();

    const { rerender } = render(<WorkView {...props} tab="matters"/>);

    fireEvent.click(screen.getByRole("button", { name: "Analyze document to create an issue or change" }));
    expect(onAnalyzeDocument).toHaveBeenCalledTimes(1);

    rerender(<WorkView {...props} tab="evidence"/>);
    expect(screen.queryByRole("button", { name: /Analyze document/ })).toBeNull();
  });
});
