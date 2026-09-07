import { render, screen } from "@testing-library/react";
import { expect, it } from "vitest";
import type { DocumentOccurrence } from "../../submittedDocumentApi";
import { DocumentFacts } from "./DocumentFile";

it("does not display a review date before a reviewer has acted", () => {
  const review: NonNullable<DocumentOccurrence["review"]> = { id: "pending", status: "SUBMITTED", reviewed_by: "", source: "VENDOR_ASSESSMENT" };
  const file = { form_title: "Vendor review", field_label: "Insurance", current: true, review } as DocumentOccurrence;
  render(<DocumentFacts file={file}/>);
  expect(screen.getByText("Awaiting review")).toBeTruthy();
  expect(screen.getByText("Reviewed").nextElementSibling?.textContent).toBe("Not recorded");
  expect(document.querySelector("time")).toBeNull();
});

it("preserves upload, submission and review timestamps with their distinct actors", () => {
  const file = { form_title: "Vendor review", field_label: "Insurance", uploaded_at: "2026-09-01T10:12:00Z", uploaded_by: "contributor-1",
    submitted_at: "2026-09-02T14:32:00Z", submitted_by: "respondent-2", current: true,
    review: { status: "VALIDATED", reviewed_by: "reviewer-3", reviewed_at: "2026-09-03T16:24:00Z" } } as DocumentOccurrence;
  render(<DocumentFacts file={file}/>);
  expect(screen.getByText("reviewer-3")).toBeTruthy();
  expect(screen.getByText("Validated by reviewer")).toBeTruthy();
  for (const instant of [file.uploaded_at, file.submitted_at, file.review!.reviewed_at!]) {
    const value = screen.getByText(new Date(instant).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" }));
    expect(value.closest("time")?.getAttribute("dateTime")).toBe(instant);
  }
  expect(screen.getByText("contributor-1")).toBeTruthy();
  expect(screen.getByText("respondent-2")).toBeTruthy();
});
