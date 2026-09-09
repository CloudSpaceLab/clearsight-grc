import { describe, expect, it } from "vitest";
import { suggestPerson, suggestRelationship, isRiskRegister } from "./registerMigrationApi";
import type { VendorRelationshipAggregate } from "./vendorTypes";

describe("risk register suggestions", () => {
  it("suggests a unique internal name, never a choice between namesakes", () => {
    const people = [{ id: "a", name: "Hakeem Ade" }, { id: "b", name: "Joel Obi" }];
    expect(suggestPerson(" Hakeem ", people)?.id).toBe("a");
    expect(suggestPerson("Hakeem", [...people, { id: "c", name: "Hakeem Musa" }])).toBeUndefined();
    expect(suggestPerson("Vendor", people)).toBeUndefined();
  });
  it("requires the vendor and service to match, including across search pages", () => {
    const item = { vendor: { legal_name: "Example Ltd", status: "ACTIVE" }, relationship: { id: "r", service_name: "Payments", status: "ACTIVE" } } as VendorRelationshipAggregate;
    expect(suggestRelationship(" Example Ltd ", "payments", { items: [item] })?.relationship.id).toBe("r");
    expect(suggestRelationship("Example Ltd", "Hosting", { items: [item] })).toBeUndefined();
    expect(suggestRelationship("Example Ltd", "Payments", { items: [item], next_cursor: "more" })).toBeUndefined();
    expect(suggestRelationship("Example Ltd", "Payments", { items: [item, item] })).toBeUndefined();
  });
  it("detects populated risk registers without mistaking questionnaires for findings", () => {
    expect(isRiskRegister([{ kind: "TABLE", anchor: {}, values: [["SERVICE PROVIDER", "Services Offered", "Findings", "Recommendations"]] }])).toBe(true);
    expect(isRiskRegister([{ kind: "TABLE", anchor: {}, values: [["Question", "Response"]] }])).toBe(false);
  });
});
