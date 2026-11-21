import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { Vendors } from "./Vendors";

vi.mock("../vendorApi", () => ({ loadVendorRelationship: vi.fn().mockResolvedValue({ vendor: { id: "v", tenant_id: "t" }, relationship: { id: "r", tenant_id: "t", legal_entity_id: "e", vendor_id: "v" } }), loadVendorRelationships: vi.fn(), createVendorRelationship: vi.fn() }));
vi.mock("../http", () => ({ requestJSON: vi.fn().mockResolvedValue({}), ApiError: class extends Error {} }));
vi.mock("./VendorReview", () => ({ VendorReview: ({ onBack }: { onBack: () => void }) => <button onClick={onBack}>Back to source</button> }));

it("returns from an exact vendor assessment to the filtered document view", async () => {
  window.history.replaceState(null, "", `#/vendors/r/assessments/a?return_to=${encodeURIComponent("#/documents?view=NEEDS_REFRESH&q=policy")}`);
  render(<Vendors context={{ tenant: { id: "t", name: "Bank" }, legal_entity: { id: "e", name: "Bank" }, actor: { id: "u", name: "User" }, mode: "LIVE" }} relationshipId="r" assessmentId="a"/>);
  fireEvent.click(await screen.findByRole("button", { name: "Back to source" }));
  expect(window.location.hash).toBe("#/documents?view=NEEDS_REFRESH&q=policy");
});
