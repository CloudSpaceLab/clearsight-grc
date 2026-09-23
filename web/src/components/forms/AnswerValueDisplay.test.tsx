import axe from "axe-core";
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AnswerValueDisplay, formatDate } from "./AnswerValueDisplay";

describe("AnswerValueDisplay", () => {
  it("renders short text answers plainly", () => {
    render(<AnswerValueDisplay type="short_text" answer={{ text: "Lagos Island" }}/>);
    expect(screen.getByText("Lagos Island")).toBeTruthy();
  });

  it("keeps long text line structure", () => {
    const { container } = render(<AnswerValueDisplay type="long_text" answer={{ text: "First line\nSecond line" }}/>);
    expect(container.querySelectorAll("[style*='pre-wrap']").length).toBe(1);
  });

  it("formats integers with thousands separators", () => {
    render(<AnswerValueDisplay type="integer" answer={{ text: "1250000" }}/>);
    expect(screen.getByText(/\b1,250,000\b/)).toBeTruthy();
  });

  it("formats decimals and percentages with one fraction digit", () => {
    render(<AnswerValueDisplay type="decimal" answer={{ text: "12.345" }}/>);
    expect(screen.getByText("12.3")).toBeTruthy();
  });

  it("falls back to the raw text for non-numeric values", () => {
    render(<AnswerValueDisplay type="integer" answer={{ text: "unavailable" }}/>);
    expect(screen.getByText("unavailable")).toBeTruthy();
  });

  it("formats currency in NGN", () => {
    render(<AnswerValueDisplay type="currency" answer={{ text: "1250000" }}/>);
    expect(screen.getByText(/1,250,000/)).toBeTruthy();
  });

  it("formats dates in medium style", () => {
    render(<AnswerValueDisplay type="date" answer={{ text: "2025-11-28" }}/>);
    expect(screen.getByText(formatDate("2025-11-28"))).toBeTruthy();
  });

  it("renders yes_no as a neutral badge", () => {
    const { container } = render(<AnswerValueDisplay type="yes_no" answer={{ values: ["Yes"] }}/>);
    expect(screen.getByText("Yes").className).toMatch(/cs-status-badge/);
    expect(container.querySelector(".cs-tone--neutral")).toBeTruthy();
  });

  it("renders checkbox and attestation confirmation states", () => {
    render(<AnswerValueDisplay type="checkbox" answer={{ text: "true" }}/>);
    expect(screen.getByText("Confirmed")).toBeTruthy();
    render(<AnswerValueDisplay type="attestation" answer={{ text: "false" }}/>);
    expect(screen.getByText("Not confirmed")).toBeTruthy();
  });

  it("renders single_select labels", () => {
    render(<AnswerValueDisplay type="single_select" answer={{ text: "Retail Banking" }}/>);
    expect(screen.getByText("Retail Banking")).toBeTruthy();
  });

  it("renders multi_select as a chip list", () => {
    render(<AnswerValueDisplay type="multi_select" answer={{ values: ["Fraud attempt", "Operational outage"] }}/>);
    expect(screen.getByText("Fraud attempt")).toBeTruthy();
    expect(screen.getByText("Operational outage")).toBeTruthy();
    expect(screen.getAllByText(/.*/).filter((node) => node.className?.includes?.("cs-status-badge")).length).toBeGreaterThanOrEqual(2);
  });

  it("counts submitted files and names attachments", () => {
    render(<AnswerValueDisplay type="file" answer={{ artifact_ids: ["a1", "a2"] }} attachments={[{ file_name: "KRI evidence.pdf" }]}/>);
    expect(screen.getByText(/2 submitted documents/)).toBeTruthy();
    expect(screen.getByText(/KRI evidence\.pdf/)).toBeTruthy();
  });

  it("shows a launcher when documents exist and onOpenDocuments is provided", () => {
    const onOpen = vi.fn();
    render(<AnswerValueDisplay type="file" answer={{ artifact_ids: ["a1"] }} onOpenDocuments={onOpen}/>);
    screen.getByRole("button", { name: "Open submitted documents" }).click();
    expect(onOpen).toHaveBeenCalledTimes(1);
  });

  it("declares 'Signed' for signature answers", () => {
    render(<AnswerValueDisplay type="signature" answer={{ artifact_ids: ["s1"] }}/>);
    expect(screen.getByText("Signed")).toBeTruthy();
  });

  it("renders vendor_document metadata including formatted expiry", () => {
    render(<AnswerValueDisplay type="vendor_document" answer={{ document: { artifact_id: "d1", document_type: "Certificate", reference: "REF-42", expires_on: "2026-03-01" } }}/>);
    expect(screen.getByText(/Certificate/)).toBeTruthy();
    expect(screen.getByText(/REF-42/)).toBeTruthy();
    expect(screen.getByText(new RegExp(`expires ${formatDate("2026-03-01")}`))).toBeTruthy();
  });

  it("uses the evidence empty label for evidence families", () => {
    render(<AnswerValueDisplay type="file" answer={undefined} evidenceEmptyLabel="No evidence submitted for this field." emptyLabel="No answer submitted for this field."/>);
    expect(screen.getByText("No evidence submitted for this field.")).toBeTruthy();
  });

  it("uses the empty label for non-evidence families and honours custom labels", () => {
    render(<AnswerValueDisplay type="short_text" answer={undefined} emptyLabel="Not provided"/>);
    expect(screen.getByText("Not provided")).toBeTruthy();
    render(<AnswerValueDisplay type="short_text" answer={undefined} emptyLabel="No answer submitted for this field."/>);
    expect(screen.getByText("No answer submitted for this field.")).toBeTruthy();
  });

  it("normalizes uppercase vendor types", () => {
    render(<AnswerValueDisplay type="LONG_TEXT" answer={{ text: "Vendor narrative" }}/>);
    expect(screen.getByText("Vendor narrative")).toBeTruthy();
    render(<AnswerValueDisplay type="NUMBER" answer={{ text: "42.5" }}/>);
    expect(screen.getByText("42.5")).toBeTruthy();
  });

  it("has no axe violations", async () => {
    const view = render(
      <AnswerValueDisplay type="multi_select" answer={{ values: ["Option A", "Option B"] }} className="response-assessment__answer"/>
    );
    const results = await axe.run(view.container, { rules: { "color-contrast": { enabled: false } } });
    expect(results.violations.map((violation) => violation.id)).toEqual([]);
  });
});