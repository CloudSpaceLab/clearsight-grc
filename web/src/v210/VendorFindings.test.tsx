import { render, screen } from "@testing-library/react";
import { expect, it } from "vitest";
import { VendorFindings } from "./VendorFindings";

it("shows assessment findings without substituting the assessment review", () => {
  render(<VendorFindings matters={[{ matter_id: "finding", title: "Outdated report", type: "VENDOR_DEFICIENCY", status: "OPEN" }, { matter_id: "review", title: "Assessment review", type: "VENDOR_REVIEW", status: "OPEN" }]} mayCreate/>);
  expect(screen.getByText("Outdated report")).toBeTruthy();
  expect(screen.getByText("Open")).toBeTruthy();
  expect(screen.queryByText("Assessment review")).toBeNull();
  expect(screen.getByText("Finding creation unavailable.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Add finding" })).toBeNull();
});

it("shows an assessment-specific empty state", () => {
  render(<VendorFindings matters={[]} mayCreate={false}/>);
  expect(screen.getByText("No linked findings returned for this assessment.")).toBeTruthy();
  expect(screen.queryByText("Finding creation unavailable.")).toBeNull();
});

it("does not present missing findings data as an empty result", () => {
  render(<VendorFindings mayCreate={false}/>);
  expect(screen.getByText("Findings unavailable.")).toBeTruthy();
  expect(screen.queryByText("No linked findings returned for this assessment.")).toBeNull();
});
