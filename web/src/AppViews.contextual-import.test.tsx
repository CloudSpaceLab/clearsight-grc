import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ProgramsView, WorkView } from "./AppViews";

vi.mock("./components/ProgramsWorkspace", () => ({ ProgramsWorkspace: () => <div>Programs workspace</div> }));
vi.mock("./components/MattersWorkspace", () => ({ MattersWorkspace: () => <div>Matters workspace</div> }));
vi.mock("./components/EvidenceWorkspace", () => ({ EvidenceWorkspace: () => <div>Evidence workspace</div> }));
vi.mock("./components/BankJourneysWorkspace", () => ({ BankJourneysWorkspace: () => <div>Reference journeys</div> }));
vi.mock("./components/TodayInterventions", () => ({ TodayInterventions: () => <div>Today interventions</div> }));
vi.mock("./components/WorkspaceErrorBoundary", () => ({ WorkspaceErrorBoundary: ({ children }: { children: ReactNode }) => <>{children}</> }));

afterEach(() => vi.restoreAllMocks());

describe("contextual document analysis entry", () => {
  it("offers document analysis from Programs when the capability is supplied", () => {
    const onAnalyzeDocument = vi.fn();
    render(<ProgramsView organizationName="Meridian Trust Bank" onAnalyzeDocument={onAnalyzeDocument}/>);

    fireEvent.click(screen.getByRole("button", { name: "Analyze document to create or update Programs" }));

    expect(onAnalyzeDocument).toHaveBeenCalledTimes(1);
  });

  it("offers document analysis for issues and changes but not evidence review", () => {
    const onAnalyzeDocument = vi.fn();
    const props = {
      organizationName: "Meridian Trust Bank",
      actorPrincipalID: "role-cro",
      evidenceScopeToken: 0,
      onTab: vi.fn(),
      onBackMatter: vi.fn(),
      sources: [],
      requests: [],
      evidenceSourceState: "live" as const,
      evidenceRequestState: "live" as const,
      onEvidenceRetry: vi.fn(),
      onEvidenceRequestUpdated: vi.fn(() => true),
      onOpenEvidence: vi.fn(),
      onAnalyzeDocument,
    };
    const { rerender } = render(<WorkView {...props} tab="matters"/>);

    fireEvent.click(screen.getByRole("button", { name: "Analyze document to create an issue or change" }));
    expect(onAnalyzeDocument).toHaveBeenCalledTimes(1);

    rerender(<WorkView {...props} tab="evidence"/>);
    expect(screen.queryByRole("button", { name: /Analyze document/ })).toBeNull();
  });
});
