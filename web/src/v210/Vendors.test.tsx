import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { Vendors } from "./Vendors";

const vendors = vi.hoisted(() => ({ loadVendorRelationships: vi.fn(), loadVendorRelationship: vi.fn(), createVendorRelationship: vi.fn() }));
const authority = vi.hoisted(() => ({ requestJSON: vi.fn() }));
vi.mock("../vendorApi", () => vendors);
vi.mock("../http", async (original) => ({ ...await original<typeof import("../http")>(), requestJSON: authority.requestJSON }));
const context = { tenant: { id: "bank", name: "Bank" }, legal_entity: { id: "entity", name: "Bank" }, actor: { id: "actor", name: "Ada" }, mode: "LIVE" };

beforeEach(() => { vi.clearAllMocks(); vendors.loadVendorRelationships.mockResolvedValue({ items: [] }); authority.requestJSON.mockResolvedValue({ principal: { id: "actor" } }); });

it("shows loaded vendor posture and filters real relationships by lifecycle", async () => {
  const relationship = { tenant_id: "bank", legal_entity_id: "entity", criticality: "CRITICAL", service_name: "Hosting" };
  vendors.loadVendorRelationships.mockResolvedValue({ items: [
    { vendor: { id: "v1", tenant_id: "bank", legal_name: "Atlas" }, relationship: { ...relationship, id: "r1", vendor_id: "v1", status: "PROPOSED" } },
    { vendor: { id: "v2", tenant_id: "bank", legal_name: "Delta" }, relationship: { ...relationship, id: "r2", vendor_id: "v2", status: "ACTIVE", criticality: "STANDARD" } },
  ], next_cursor: "next" });
  render(<Vendors context={context}/>);
  expect(await screen.findByRole("link", { name: "Atlas" })).toBeTruthy();
  expect(screen.getByRole("heading", { name: "Service engagements" })).toBeTruthy();
  expect(screen.getByText("Loaded suppliers")).toBeTruthy();
  expect(screen.getByText("Counts cover loaded relationships.")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Onboarding" }));
  expect(screen.getByRole("link", { name: "Atlas" })).toBeTruthy();
  expect(screen.queryByRole("link", { name: "Delta" })).toBeNull();
  expect(screen.getByRole("button", { name: "Load more" })).toBeTruthy();
});

it("does not show a zero relationship count while the register is unavailable", async () => {
  vendors.loadVendorRelationships.mockRejectedValue(new Error("offline"));
  render(<Vendors context={context}/>);
  expect(await screen.findByRole("alert")).toBeTruthy();
  expect(screen.queryByText("0 relationships")).toBeNull();
  expect(screen.getByText("Loaded suppliers").parentElement?.textContent).toContain("—");
});

it("resets the lifecycle filter when the entity changes", async () => {
  const first = { vendor: { id: "v1", tenant_id: "bank", legal_name: "Atlas" }, relationship: { id: "r1", vendor_id: "v1", tenant_id: "bank", legal_entity_id: "entity", criticality: "STANDARD", service_name: "Hosting", status: "PROPOSED" } };
  const second = { vendor: { id: "v2", tenant_id: "bank", legal_name: "Delta" }, relationship: { id: "r2", vendor_id: "v2", tenant_id: "bank", legal_entity_id: "other", criticality: "STANDARD", service_name: "Payments", status: "ACTIVE" } };
  vendors.loadVendorRelationships.mockResolvedValueOnce({ items: [first] }).mockResolvedValueOnce({ items: [second] });
  const tree = render(<Vendors context={context}/>);
  await screen.findByRole("link", { name: "Atlas" });
  fireEvent.click(screen.getByRole("button", { name: "Onboarding" }));
  tree.rerender(<Vendors context={{ ...context, legal_entity: { id: "other", name: "Other bank" } }}/>);
  expect(await screen.findByRole("link", { name: "Delta" })).toBeTruthy();
  expect(screen.getByRole("button", { name: "Engagements" }).getAttribute("aria-pressed")).toBe("true");
});

async function enterVendorDetails() {
  fireEvent.click(await screen.findByRole("button", { name: "Add vendor" }));
  fireEvent.change(screen.getByLabelText("Legal name"), { target: { value: "Atlas Ltd" } });
  fireEvent.click(screen.getByRole("button", { name: "Continue" }));
  fireEvent.change(screen.getByLabelText("Service"), { target: { value: "Hosting" } });
  fireEvent.change(screen.getByLabelText("Criticality"), { target: { value: "IMPORTANT" } });
  fireEvent.change(screen.getByLabelText("Privacy role"), { target: { value: "PROCESSOR" } });
  fireEvent.click(screen.getByRole("button", { name: "Review relationship" }));
}

it("creates a proposed relationship using the signed-in scope", async () => {
  vendors.createVendorRelationship.mockResolvedValue({ vendor: { id: "vendor-1", tenant_id: "bank", version: 1, legal_name: "Atlas Ltd" }, relationship: { id: "relationship-1", tenant_id: "bank", legal_entity_id: "entity", business_owner_principal_id: "actor", vendor_id: "vendor-1", version: 1, service_name: "Hosting", criticality: "IMPORTANT", privacy_role: "PROCESSOR", status: "PROPOSED" } });
  render(<Vendors context={context}/>);
  await enterVendorDetails();
  expect(vendors.createVendorRelationship).not.toHaveBeenCalled();
  expect(screen.getByText("Atlas Ltd")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Create relationship" }));
  expect(await screen.findByText("Relationship created" )).toBeTruthy();
  expect(vendors.createVendorRelationship).toHaveBeenCalledWith({ legal_name: "Atlas Ltd", service_name: "Hosting", criticality: "IMPORTANT", privacy_role: "PROCESSOR" });
  expect(screen.getByText("Proposed")).toBeTruthy();
});

it("does not present an unconfirmed creation receipt as success", async () => {
  vendors.createVendorRelationship.mockResolvedValue({ vendor: { legal_name: "Other Ltd" }, relationship: { service_name: "Hosting", status: "ACTIVE" } });
  render(<Vendors context={context}/>);
  await enterVendorDetails();
  fireEvent.click(screen.getByRole("button", { name: "Create relationship" }));
  expect(await screen.findByRole("heading", { name: "Relationship status unknown" })).toBeTruthy();
  expect(screen.queryByText("Relationship created")).toBeNull();
});

it("retains entered service details when returning from review", async () => {
  render(<Vendors context={context}/>);
  await enterVendorDetails();
  expect(document.activeElement).toBe(screen.getByRole("heading", { name: "Review relationship" }));
  fireEvent.click(screen.getByRole("button", { name: "Back" }));
  expect((screen.getByLabelText("Service") as HTMLInputElement).value).toBe("Hosting");
  expect((screen.getByLabelText("Criticality") as HTMLSelectElement).value).toBe("IMPORTANT");
  expect(document.activeElement).toBe(screen.getByRole("heading", { name: "Service details" }));
  expect(vendors.createVendorRelationship).not.toHaveBeenCalled();
});

it("shows scoped matching relationships before creating another vendor", async () => {
  vendors.loadVendorRelationships.mockResolvedValueOnce({ items: [] }).mockResolvedValueOnce({ items: [{ vendor: { id: "v", tenant_id: "bank", legal_name: "Atlas Ltd", version: 1 }, relationship: { id: "r", tenant_id: "bank", legal_entity_id: "entity", vendor_id: "v", version: 1, service_name: "Hosting" } }] });
  render(<Vendors context={context}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Add vendor" }));
  fireEvent.change(screen.getByLabelText("Legal name"), { target: { value: "Atlas" } });
  fireEvent.click(screen.getByRole("button", { name: "Check existing vendors" }));
  expect((await screen.findByRole("link", { name: "Atlas Ltd · Hosting (opens in new tab)" })).getAttribute("href")).toBe("#/vendors/r");
  expect(vendors.createVendorRelationship).not.toHaveBeenCalled();
});

it("does not reveal a vendor search result outside the current entity", async () => {
  vendors.loadVendorRelationships.mockResolvedValueOnce({ items: [] }).mockResolvedValueOnce({ items: [{ vendor: { id: "v", tenant_id: "bank", legal_name: "Atlas Ltd", version: 1 }, relationship: { id: "r", tenant_id: "bank", legal_entity_id: "other", vendor_id: "v", version: 1, service_name: "Hosting" } }] });
  render(<Vendors context={context}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Add vendor" }));
  fireEvent.change(screen.getByLabelText("Legal name"), { target: { value: "Atlas" } });
  fireEvent.click(screen.getByRole("button", { name: "Check existing vendors" }));
  expect(await screen.findByText("Vendor search unavailable.")).toBeTruthy();
  expect(screen.queryByRole("link", { name: "Atlas Ltd · Hosting" })).toBeNull();
});

it("discards a vendor search result after the legal name changes", async () => {
  let resolve!: (value: unknown) => void;
  vendors.loadVendorRelationships.mockResolvedValueOnce({ items: [] }).mockReturnValueOnce(new Promise((done) => { resolve = done; }));
  render(<Vendors context={context}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Add vendor" }));
  fireEvent.change(screen.getByLabelText("Legal name"), { target: { value: "Atlas" } });
  fireEvent.click(screen.getByRole("button", { name: "Check existing vendors" }));
  fireEvent.change(screen.getByLabelText("Legal name"), { target: { value: "Acme" } });
  resolve({ items: [{ vendor: { id: "v", tenant_id: "bank", legal_name: "Atlas Ltd", version: 1 }, relationship: { id: "r", tenant_id: "bank", legal_entity_id: "entity", vendor_id: "v", version: 1, service_name: "Hosting" } }] });
  await vi.waitFor(() => expect(screen.queryByRole("link", { name: "Atlas Ltd · Hosting" })).toBeNull());
});

it("lets the owner correct vendor details from review without losing service details", async () => {
  render(<Vendors context={context}/>);
  await enterVendorDetails();
  fireEvent.click(screen.getByRole("button", { name: "Edit vendor details" }));
  expect((screen.getByLabelText("Legal name") as HTMLInputElement).value).toBe("Atlas Ltd");
  fireEvent.change(screen.getByLabelText("Legal name"), { target: { value: "Atlas Bank Ltd" } });
  fireEvent.click(screen.getByRole("button", { name: "Continue" }));
  expect((screen.getByLabelText("Service") as HTMLInputElement).value).toBe("Hosting");
  fireEvent.click(screen.getByRole("button", { name: "Review relationship" }));
  expect(screen.getByText("Atlas Bank Ltd")).toBeTruthy();
  expect(vendors.createVendorRelationship).not.toHaveBeenCalled();
});

it("does not retry a creation with an uncertain outcome", async () => {
  vendors.createVendorRelationship.mockRejectedValue(new Error("connection lost"));
  render(<Vendors context={context}/>);
  await enterVendorDetails();
  fireEvent.click(screen.getByRole("button", { name: "Create relationship" }));
  expect((await screen.findByRole("alert")).textContent).toContain("Check vendors before trying again.");
  expect(screen.queryByRole("button", { name: "Create relationship" })).toBeNull();
});

it("does not open a different vendor relationship than the one requested", async () => {
  vendors.loadVendorRelationship.mockResolvedValue({ vendor: { id: "vendor-1", legal_name: "Other" }, relationship: { id: "different", vendor_id: "vendor-1", service_name: "Hosting", status: "PROPOSED" } });
  render(<Vendors context={context} relationshipId="requested"/>);
  expect(await screen.findByRole("heading", { name: "Vendor unavailable" })).toBeTruthy();
  expect(screen.queryByText("Other")).toBeNull();
});

it("withholds intake when the current route cannot be verified", async () => {
  authority.requestJSON.mockResolvedValue({ principal: { id: "someone-else" } });
  render(<Vendors context={context}/>);
  expect(await screen.findByText("Vendor creation unavailable.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Add vendor" })).toBeNull();
});

it("checks the same scoped creation route as the command", async () => {
  render(<Vendors context={context}/>);
  await screen.findByRole("button", { name: "Add vendor" });
  const init = authority.requestJSON.mock.calls[0]![2];
  expect(JSON.parse(init.body)).toEqual({ tenant_id: "bank", legal_entity_id: "entity", object_type: "VENDOR_RELATIONSHIP", object_id: "entity", responsibility: "ACCOUNTABLE_OWNER", decision_type: "thirdparty.relationship.create", materiality: 3 });
});

it("ignores a creation receipt after scope changes", async () => {
  let resolve!: (value: unknown) => void;
  vendors.createVendorRelationship.mockReturnValue(new Promise((done) => { resolve = done; }));
  const view = render(<Vendors context={context}/>); await enterVendorDetails();
  fireEvent.click(screen.getByRole("button", { name: "Create relationship" }));
  view.rerender(<Vendors context={{ ...context, legal_entity: { ...context.legal_entity, id: "another" } }}/>);
  resolve({ vendor: { id: "vendor-1", tenant_id: "bank", legal_name: "Atlas Ltd" }, relationship: { id: "r", tenant_id: "bank", legal_entity_id: "entity", business_owner_principal_id: "actor", vendor_id: "vendor-1", version: 1, service_name: "Hosting", criticality: "IMPORTANT", privacy_role: "PROCESSOR", status: "PROPOSED" } });
  await vi.waitFor(() => expect(screen.queryByText("Relationship created")).toBeNull());
});

it("does not accept a receipt for a different entity", async () => {
  vendors.createVendorRelationship.mockResolvedValue({ vendor: { id: "v", tenant_id: "bank", legal_name: "Atlas Ltd" }, relationship: { id: "r", tenant_id: "bank", legal_entity_id: "other", business_owner_principal_id: "actor", vendor_id: "v", version: 1, service_name: "Hosting", criticality: "IMPORTANT", privacy_role: "PROCESSOR", status: "PROPOSED" } });
  render(<Vendors context={context}/>); await enterVendorDetails();
  fireEvent.click(screen.getByRole("button", { name: "Create relationship" }));
  expect(await screen.findByRole("heading", { name: "Relationship status unknown" })).toBeTruthy();
});

it("keeps fields after a definite validation rejection", async () => {
  const { ApiError } = await import("../http");
  vendors.createVendorRelationship.mockRejectedValue(new ApiError(422, "Check vendor details", "vendor_invalid"));
  render(<Vendors context={context}/>); await enterVendorDetails();
  fireEvent.click(screen.getByRole("button", { name: "Create relationship" }));
  expect((await screen.findByRole("alert")).textContent).toContain("Check vendor details");
  expect(screen.getByText("Atlas Ltd")).toBeTruthy();
  expect(screen.getByRole("button", { name: "Back" })).toBeTruthy();
});

it("searches for the submitted vendor after an uncertain result", async () => {
  vendors.createVendorRelationship.mockRejectedValue(new Error("connection lost"));
  render(<Vendors context={context}/>); await enterVendorDetails();
  fireEvent.click(screen.getByRole("button", { name: "Create relationship" }));
  fireEvent.click(await screen.findByRole("button", { name: "View vendors" }));
  await vi.waitFor(() => expect(vendors.loadVendorRelationships).toHaveBeenLastCalledWith({ search: "Atlas Ltd", limit: 20 }));
});

it.skip("reuses an explicitly selected scoped vendor identity after authority synchronization is available", async () => {
  const existing = { vendor: { id: "v", tenant_id: "bank", legal_name: "Atlas Ltd", version: 3 }, relationship: { id: "old", tenant_id: "bank", legal_entity_id: "entity", vendor_id: "v", version: 2, service_name: "Previous service" } };
  vendors.loadVendorRelationships.mockResolvedValueOnce({ items: [] }).mockResolvedValueOnce({ items: [existing] });
  vendors.createVendorRelationship.mockResolvedValue({ vendor: existing.vendor, relationship: { id: "new", tenant_id: "bank", legal_entity_id: "entity", business_owner_principal_id: "actor", vendor_id: "v", version: 1, service_name: "Hosting", criticality: "IMPORTANT", privacy_role: "PROCESSOR", status: "PROPOSED" } });
  render(<Vendors context={context}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Add vendor" }));
  fireEvent.change(screen.getByLabelText("Legal name"), { target: { value: "Atlas" } });
  fireEvent.click(screen.getByRole("button", { name: "Find existing vendors" }));
  fireEvent.click(await screen.findByRole("button", { name: "Select Atlas Ltd · Previous service" }));
  fireEvent.click(screen.getByRole("button", { name: "Continue" }));
  fireEvent.change(screen.getByLabelText("Service"), { target: { value: "Hosting" } });
  fireEvent.change(screen.getByLabelText("Criticality"), { target: { value: "IMPORTANT" } });
  fireEvent.change(screen.getByLabelText("Privacy role"), { target: { value: "PROCESSOR" } });
  fireEvent.click(screen.getByRole("button", { name: "Review relationship" }));
  fireEvent.click(screen.getByRole("button", { name: "Create relationship" }));
  expect(await screen.findByText("Relationship created")).toBeTruthy();
  expect(vendors.createVendorRelationship).toHaveBeenCalledWith(expect.objectContaining({ existing_relationship_id: "old", legal_name: "Atlas Ltd" }));
});

it.skip("rejects a candidate outside the signed-in entity after reuse is enabled", async () => {
  vendors.loadVendorRelationships.mockResolvedValueOnce({ items: [] }).mockResolvedValueOnce({ items: [{ vendor: { id: "v", tenant_id: "bank", legal_name: "Atlas Ltd", version: 1 }, relationship: { id: "r", tenant_id: "bank", legal_entity_id: "other", vendor_id: "v", version: 1, service_name: "Hosting" } }] });
  render(<Vendors context={context}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Add vendor" }));
  fireEvent.change(screen.getByLabelText("Legal name"), { target: { value: "Atlas" } });
  fireEvent.click(screen.getByRole("button", { name: "Find existing vendors" }));
  expect(await screen.findByText("Vendor search unavailable.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: /Select Atlas/ })).toBeNull();
});

it("does not expose vendor reuse before transaction authority synchronization", async () => {
  render(<Vendors context={context}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Add vendor" }));
  fireEvent.change(screen.getByLabelText("Legal name"), { target: { value: "Atlas" } });
  expect(screen.queryByRole("button", { name: "Find existing vendors" })).toBeNull();
});


it("retains the submitted relationship details when creation status is unknown", async () => {
  vendors.createVendorRelationship.mockRejectedValue(new Error("connection lost"));
  render(<Vendors context={context}/>); await enterVendorDetails();
  fireEvent.click(screen.getByRole("button", { name: "Create relationship" }));
  expect(await screen.findByRole("heading", { name: "Relationship status unknown" })).toBeTruthy();
  expect(screen.getByText("Atlas Ltd")).toBeTruthy();
  expect(screen.getByText("Hosting")).toBeTruthy();
  expect(screen.getByText("Important")).toBeTruthy();
  expect(screen.getByText("Processor")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Create relationship" })).toBeNull();
});

it("opens matching relationships separately to retain intake details", async () => {
  vendors.loadVendorRelationships.mockResolvedValueOnce({ items: [] }).mockResolvedValueOnce({ items: [{ vendor: { id: "v", tenant_id: "bank", legal_name: "Atlas Ltd", version: 1 }, relationship: { id: "r", tenant_id: "bank", legal_entity_id: "entity", vendor_id: "v", version: 1, service_name: "Hosting" } }] });
  render(<Vendors context={context}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Add vendor" }));
  fireEvent.change(screen.getByLabelText("Legal name"), { target: { value: "Atlas" } });
  fireEvent.click(screen.getByRole("button", { name: "Check existing vendors" }));
  const link = await screen.findByRole("link", { name: /Atlas Ltd · Hosting/ });
  expect(link.getAttribute("target")).toBe("_blank");
  expect(link.getAttribute("rel")).toContain("noopener");
  expect(link.textContent).toContain("opens in new tab");
});

it("retains uncertain creation details while checking the vendor list", async () => {
  vendors.createVendorRelationship.mockRejectedValue(new Error("connection lost"));
  render(<Vendors context={context}/>); await enterVendorDetails();
  fireEvent.click(screen.getByRole("button", { name: "Create relationship" }));
  fireEvent.click(await screen.findByRole("button", { name: "View vendors" }));
  expect(await screen.findByRole("heading", { name: "Relationship status unknown" })).toBeTruthy();
  expect(screen.getByText("Hosting")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Add vendor" })).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Dismiss details" }));
  expect(screen.queryByRole("heading", { name: "Relationship status unknown" })).toBeNull();
  expect(screen.getByRole("button", { name: "Add vendor" })).toBeTruthy();
});

it("does not expose uncertain relationship details after scope changes", async () => {
  vendors.createVendorRelationship.mockRejectedValue(new Error("connection lost"));
  const view = render(<Vendors context={context}/>); await enterVendorDetails();
  fireEvent.click(screen.getByRole("button", { name: "Create relationship" }));
  expect(await screen.findByRole("heading", { name: "Relationship status unknown" })).toBeTruthy();
  view.rerender(<Vendors context={{ ...context, legal_entity: { ...context.legal_entity, id: "another" } }}/>);
  expect(screen.queryByText("Atlas Ltd")).toBeNull();
  expect(screen.queryByText("Hosting")).toBeNull();
});

it("discards a vendor list from the previous legal entity during scope change", async () => {
  let finish!: (value: unknown) => void;
  vendors.loadVendorRelationships.mockResolvedValueOnce({ items: [{ vendor: { id: "v", tenant_id: "bank", legal_name: "Private vendor" }, relationship: { id: "r", tenant_id: "bank", legal_entity_id: "entity", vendor_id: "v", service_name: "Hosting", criticality: "IMPORTANT", status: "PROPOSED" } }] }).mockReturnValueOnce(new Promise((resolve) => { finish = resolve; }));
  const view = render(<Vendors context={context}/>);
  expect(await screen.findByText("Private vendor")).toBeTruthy();
  view.rerender(<Vendors context={{ ...context, legal_entity: { ...context.legal_entity, id: "other" } }}/>);
  expect(screen.queryByText("Private vendor")).toBeNull();
  finish({ items: [] });
  expect(await screen.findByText("No vendors match this search.")).toBeTruthy();
});

it("rejects an exact vendor response from a different legal entity", async () => {
  vendors.loadVendorRelationship.mockResolvedValue({ vendor: { id: "v", tenant_id: "bank", legal_name: "Private vendor" }, relationship: { id: "r", tenant_id: "bank", legal_entity_id: "other", vendor_id: "v", service_name: "Hosting", criticality: "IMPORTANT", status: "PROPOSED" } });
  render(<Vendors context={context} relationshipId="r"/>);
  expect(await screen.findByRole("heading", { name: "Vendor unavailable" })).toBeTruthy();
  expect(screen.queryByText("Private vendor")).toBeNull();
});

it("hides a created relationship when the current legal entity changes", async () => {
  vendors.createVendorRelationship.mockResolvedValue({ vendor: { id: "vendor-1", tenant_id: "bank", version: 1, legal_name: "Atlas Ltd" }, relationship: { id: "relationship-1", tenant_id: "bank", legal_entity_id: "entity", business_owner_principal_id: "actor", vendor_id: "vendor-1", version: 1, service_name: "Hosting", criticality: "IMPORTANT", privacy_role: "PROCESSOR", status: "PROPOSED" } });
  const view = render(<Vendors context={context}/>); await enterVendorDetails();
  fireEvent.click(screen.getByRole("button", { name: "Create relationship" }));
  expect(await screen.findByText("Relationship created")).toBeTruthy();
  view.rerender(<Vendors context={{ ...context, legal_entity: { ...context.legal_entity, id: "other" } }}/>);
  expect(screen.queryByText("Atlas Ltd")).toBeNull();
  expect(screen.queryByText("Hosting")).toBeNull();
});

it("rejects a vendor list containing a relationship from another legal entity", async () => {
  vendors.loadVendorRelationships.mockResolvedValue({ items: [{ vendor: { id: "v", tenant_id: "bank", legal_name: "Private vendor" }, relationship: { id: "r", tenant_id: "bank", legal_entity_id: "other", vendor_id: "v", service_name: "Hosting", criticality: "IMPORTANT", status: "PROPOSED" } }] });
  render(<Vendors context={context}/>);
  expect((await screen.findByRole("alert")).textContent).toContain("Vendors unavailable.");
  expect(screen.queryByText("Private vendor")).toBeNull();
});

it("accepts a vendor response using the verified tenant UUID", async () => {
  vendors.loadVendorRelationships.mockResolvedValue({ items: [{ vendor: { id: "v", tenant_id: "tenant-uuid", legal_name: "Atlas Ltd" }, relationship: { id: "r", tenant_id: "tenant-uuid", legal_entity_id: "entity", vendor_id: "v", service_name: "Hosting", criticality: "IMPORTANT", status: "PROPOSED" } }] });
  render(<Vendors context={{ ...context, tenant: { ...context.tenant, id: "tenant-uuid" } }}/>);
  expect(await screen.findByRole("link", { name: "Atlas Ltd" })).toBeTruthy();
});

it("accepts existing-vendor results in the verified tenant UUID scope", async () => {
  vendors.loadVendorRelationships.mockResolvedValueOnce({ items: [] }).mockResolvedValueOnce({ items: [{ vendor: { id: "v", tenant_id: "tenant-uuid", legal_name: "Atlas Ltd", version: 1 }, relationship: { id: "r", tenant_id: "tenant-uuid", legal_entity_id: "entity", vendor_id: "v", version: 1, service_name: "Hosting" } }] });
  render(<Vendors context={{ ...context, tenant: { ...context.tenant, id: "tenant-uuid" } }}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Add vendor" }));
  fireEvent.change(screen.getByLabelText("Legal name"), { target: { value: "Atlas" } });
  fireEvent.click(screen.getByRole("button", { name: "Check existing vendors" }));
  expect(await screen.findByRole("link", { name: "Atlas Ltd · Hosting (opens in new tab)" })).toBeTruthy();
});

it("shows the current onboarding step as the relationship is prepared", async () => {
  render(<Vendors context={context}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Add vendor" }));
  const progress = screen.getByRole("list", { name: "Relationship setup progress" });
  expect(progress.querySelector('[aria-current="step"]')?.textContent).toContain("Vendor details");
  fireEvent.change(screen.getByLabelText("Legal name"), { target: { value: "Atlas Ltd" } });
  fireEvent.click(screen.getByRole("button", { name: "Continue" }));
  expect(progress.querySelector('[aria-current="step"]')?.textContent).toContain("Service details");
  expect(vendors.createVendorRelationship).not.toHaveBeenCalled();
});
