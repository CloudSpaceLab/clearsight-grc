import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ProgramAggregate } from "../types";
import { ProgramWorkspace } from "./ProgramWorkspace";

vi.mock("./ProgramCoverage", () => ({ ProgramCoverage: () => <section><h2>Requirement coverage</h2></section> }));
vi.mock("./ProgramRequirements", () => ({ ProgramRequirements: () => <section><h2>Requirements</h2></section> }));

const aggregate = { program: { id: "program/1", code: "P-1", name: "Access assurance", type: "COMPLIANCE", status: "ACTIVE", version: 2, owning_function: "Risk", jurisdiction: "Nigeria" }, requirements: [] } as unknown as ProgramAggregate;

describe("Program workspace", () => {
  it("shows the exact Program, recorded facts and the existing issue route", () => {
    render(<ProgramWorkspace aggregate={aggregate} backHref="#/programs?q=access" issuesHref="#/programs/program%2F1/issues?q=access" onUpdated={() => undefined}/>);
    expect(screen.getByRole("heading", { name: "Access assurance" })).toBeTruthy();
    expect(screen.getByText("Risk")).toBeTruthy();
    expect(screen.getByText("Nigeria")).toBeTruthy();
    expect(screen.getByRole("link", { name: "Open issues" }).getAttribute("href")).toBe("#/programs/program%2F1/issues?q=access");
    expect(screen.getByRole("heading", { name: "Requirement coverage" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Requirements" })).toBeTruthy();
  });

  it("moves focus to requirements without changing the current route", () => {
    render(<ProgramWorkspace aggregate={aggregate} backHref="#/programs" issuesHref="#/programs/program%2F1/issues" onUpdated={() => undefined}/>);
    const section = screen.getByRole("region", { name: "Program requirements" });
    const scroll = vi.mocked(section.scrollIntoView);
    scroll.mockClear();
    fireEvent.click(screen.getByRole("button", { name: "Requirements" }));
    expect(scroll).toHaveBeenCalledWith({ block: "start" });
    expect(document.activeElement).toBe(section);
  });

  it("does not expose raw owner or authority identifiers", () => {
    render(<ProgramWorkspace aggregate={{ ...aggregate, program: { ...aggregate.program, owning_function: "", jurisdiction: "", owner_principal_id: "private-owner", authority_principal_id: "private-reviewer" } }} backHref="#/programs" issuesHref="#/programs/program%2F1/issues" onUpdated={() => undefined}/>);
    expect(screen.queryByText("private-owner")).toBeNull();
    expect(screen.queryByText("private-reviewer")).toBeNull();
    expect(screen.getAllByText("Not recorded")).toHaveLength(2);
  });
});
