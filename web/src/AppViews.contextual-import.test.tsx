import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ProgramsView, WorkView } from "./AppViews";

vi.mock("./components/ProgramsWorkspace", () => ({ ProgramsWorkspace: () => <div>Programs workspace</div> }));
vi.mock("./components/MattersWorkspace", () => ({ MattersWorkspace: () => <div>Matters workspace</div> }));
vi.mock("./components/EvidenceWorkspace", () => ({ EvidenceWorkspace: () => <div>Evidence workspace</div> }));
vi.mock("./components/BankJourneysWorkspace", () => ({ BankJourneysWorkspace: () => <div>Reference journeys</div> }));
vi.mock("./components/TodayInterventions", () => ({ TodayInterventions: () => <div>Today interventions</div> }));
vi.mock("./components/AssignedWorkQueue", () => ({ AssignedWorkQueue: ({ items }: { items: unknown[] }) => <div>Assigned work · {items.length}</div> }));
vi.mock("./components/WorkspaceErrorBoundary", () => ({ WorkspaceErrorBoundary: ({ children }: { children: ReactNode }) => <>{children}</> }));

afterEach(() => vi.restoreAllMocks());

describe("contextual document analysis entry", () => {
  it("offers document analysis from Programs when the capability is supplied", () => {
    const onAnalyzeDocument = vi.fn();
    render(<ProgramsView organizationName="Meridian Trust Bank" onAnalyzeDocument={onAnalyzeDocument}/>);

    fireEvent.click(screen.getByRole("button", { name: "Analyze document to create or update Programs" }));

    expect(onAnalyzeDocument).toHaveBeenCalledTimes(1);
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
    assigned.unmount();

    const { rerender } = render(<WorkView {...props} tab="matters"/>);

    fireEvent.click(screen.getByRole("button", { name: "Analyze document to create an issue or change" }));
    expect(onAnalyzeDocument).toHaveBeenCalledTimes(1);

    rerender(<WorkView {...props} tab="evidence"/>);
    expect(screen.queryByRole("button", { name: /Analyze document/ })).toBeNull();
  });
});
