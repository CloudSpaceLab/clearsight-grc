import { describe, expect, it } from "vitest";
import { vendorDueDiligenceStarterForm } from "./vendorDueDiligenceForm";
import { keepVisibleAnswers, validateCaptureFields, visibleCaptureFields } from "./components/capture/contract";
import type { CaptureAnswers, CaptureFormContract } from "./types";

const contract = vendorDueDiligenceStarterForm as CaptureFormContract;
const answers: CaptureAnswers = {
  contact_email: { text: "security@vendor.example" },
  service_description: { text: "Hosted customer service application." },
  data_classes: { values: ["Customer personal data"] },
  subprocessors: { text: "No" },
  security_framework: { text: "None" },
  authorized_attestation: { text: "yes" },
};

describe("vendor starter assurance collection", () => {
  it("accepts a truthful missing-document declaration with a required explanation", () => {
    const missing = { ...answers, assurance_available: { text: "No" } };
    expect(validateCaptureFields(visibleCaptureFields(contract, missing), missing)).toEqual([
      expect.objectContaining({ fieldID: "assurance_gap" }),
    ]);
    const explained = { ...missing, assurance_gap: { text: "Certification is scheduled for the next quarter." } };
    expect(validateCaptureFields(visibleCaptureFields(contract, explained), explained)).toEqual([]);
    expect(visibleCaptureFields(contract, explained).some((field) => field.id === "security_document")).toBe(false);
  });

  it("requires the document when available and drops a hidden prior gap answer", () => {
    const available = { ...answers, assurance_available: { text: "Yes" }, assurance_gap: { text: "Old declaration" } };
    expect(validateCaptureFields(visibleCaptureFields(contract, available), available)).toContainEqual(expect.objectContaining({ fieldID: "security_document" }));
    expect(keepVisibleAnswers(contract, available)).not.toHaveProperty("assurance_gap");
  });
});
