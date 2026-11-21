import { render, screen } from "@testing-library/react";
import { expect, it } from "vitest";
import type { DocumentOccurrence } from "../../submittedDocumentApi";
import { DocumentProvenance } from "../../v210/DocumentProvenance";

const field = (name: string) => screen.getByText(name).nextElementSibling as HTMLElement;

it("does not display a review date before a reviewer has acted", () => {
  const review: NonNullable<DocumentOccurrence["review"]> = { id: "pending", status: "SUBMITTED", reviewed_by: "", source: "VENDOR_ASSESSMENT" };
  render(<dl><DocumentProvenance file={{ review } as DocumentOccurrence}/></dl>);
  expect(screen.getByText("Awaiting review")).toBeTruthy();
  expect(field("Reviewed").textContent).toBe("Not recorded");
  expect(field("Reviewed").querySelector("time")).toBeNull();
});

it("preserves upload, submission and review timestamps with their distinct actors", () => {
  const file = { uploaded_at: "2026-09-01T10:12:00Z", uploaded_by: "contributor-1", submitted_at: "2026-09-02T14:32:00Z", submitted_by: "respondent-2", review: { status: "VALIDATED", reviewed_by: "reviewer-3", reviewed_at: "2026-09-03T16:24:00Z" } } as DocumentOccurrence;
  render(<dl><DocumentProvenance file={file}/></dl>);
  expect(field("Uploaded by").textContent).toBe("Person ID: contributor-1");
  expect(field("Submitted by").textContent).toBe("Person ID: respondent-2");
  expect(field("Reviewed by").textContent).toBe("Person ID: reviewer-3");
  expect(screen.getByText("Validated")).toBeTruthy();
  for (const [name, instant] of [["Uploaded", file.uploaded_at], ["Submitted", file.submitted_at], ["Reviewed", file.review!.reviewed_at!]]) {
    expect(field(name).querySelector("time")?.getAttribute("datetime")).toBe(instant);
  }
});

it("distinguishes missing timestamps and actors from recorded values", () => {
  render(<dl><DocumentProvenance file={{ review: { status: "VALIDATED", reviewed_by: "" } } as DocumentOccurrence}/></dl>);
  for (const name of ["Uploaded", "Uploaded by", "Submitted", "Submitted by", "Reviewed", "Reviewed by"]) expect(field(name).textContent).toBe("Not recorded");
});

it("does not present a zero timestamp as a real event", () => {
  render(<dl><DocumentProvenance file={{ uploaded_at: "0001-01-01T00:00:00Z" } as DocumentOccurrence}/></dl>);
  expect(field("Uploaded").textContent).toBe("Not recorded");
  expect(field("Uploaded").querySelector("time")).toBeNull();
});
