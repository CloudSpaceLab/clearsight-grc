import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "../http";
import { loadVendorRelationship, loadVendorRelationships } from "../vendorApi";
import { endVendorRelationshipLink, linkVendorRelationship, loadVendorRelationshipLinks } from "../vendorLinkApi";
import type { VendorRelationshipAggregate } from "../vendorTypes";
import { VendorRelationshipLinks } from "./VendorRelationshipLinks";

vi.mock("../vendorApi", () => ({ loadVendorRelationship: vi.fn(), loadVendorRelationships: vi.fn() }));
vi.mock("../vendorLinkApi", () => ({ endVendorRelationshipLink: vi.fn(), linkVendorRelationship: vi.fn(), loadVendorRelationshipLinks: vi.fn() }));

const acme: VendorRelationshipAggregate = {
  vendor: { id: "vendor-1", tenant_id: "bank", legal_name: "Acme Processing Limited", status: "ACTIVE", created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z", version: 1 },
  relationship: { id: "relationship-1", tenant_id: "bank", legal_entity_id: "entity", vendor_id: "vendor-1", service_name: "Card transaction processing", business_owner_principal_id: "owner", criticality: "IMPORTANT", privacy_role: "PROCESSOR", status: "ACTIVE", created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z", version: 3 },
};

const beta: VendorRelationshipAggregate = {
  vendor: { ...acme.vendor, id: "vendor-2", legal_name: "Beta Cloud Limited" },
  relationship: { ...acme.relationship, id: "relationship-2", vendor_id: "vendor-2", service_name: "Cloud hosting" },
};

const activeLink = {
  id: "link-1", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "relationship-1",
  target_type: "PROGRAM" as const, target_id: "program-1", purpose_code: "SERVICE_SUPPORT", purpose_label: "Supports the payments programme",
  state: "ACTIVE" as const, created_by: "owner", version: 1, created_at: "2026-08-26T12:00:00Z", updated_at: "2026-08-26T12:00:00Z",
};

describe("VendorRelationshipLinks", () => {
  beforeEach(() => {
    vi.mocked(loadVendorRelationshipLinks).mockReset().mockResolvedValue({ items: [] });
    vi.mocked(linkVendorRelationship).mockReset();
    vi.mocked(endVendorRelationshipLink).mockReset();
    vi.mocked(loadVendorRelationships).mockReset().mockResolvedValue({ items: [] });
    vi.mocked(loadVendorRelationship).mockReset();
  });

  it("lists the vendor relationships linked to the exact target", async () => {
    vi.mocked(loadVendorRelationshipLinks).mockResolvedValue({ items: [activeLink] });
    vi.mocked(loadVendorRelationship).mockResolvedValue(acme);

    render(<VendorRelationshipLinks targetType="PROGRAM" targetID="program-1"/>);

    expect(await screen.findByRole("heading", { name: "Related vendors" })).toBeTruthy();
    expect(await screen.findByText("Acme Processing Limited")).toBeTruthy();
    expect(screen.getByText("Card transaction processing")).toBeTruthy();
    expect(screen.getByText("Supports the payments programme")).toBeTruthy();
    expect(loadVendorRelationshipLinks).toHaveBeenCalledWith({ target_type: "PROGRAM", target_id: "program-1", limit: 50 });
  });

  it("searches existing relationships on the server and links one with a bank-defined purpose", async () => {
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [acme] });
    vi.mocked(linkVendorRelationship).mockResolvedValue(activeLink);

    render(<VendorRelationshipLinks targetType="PROGRAM" targetID="program-1"/>);
    await screen.findByText("No vendor relationships are linked to this Program.");
    fireEvent.click(screen.getByRole("button", { name: "Link vendor" }));
    fireEvent.change(screen.getByLabelText("Search vendor relationships"), { target: { value: "Acme payments" } });
    fireEvent.click(screen.getByRole("button", { name: "Search vendors" }));

    await waitFor(() => expect(loadVendorRelationships).toHaveBeenCalledWith({ search: "Acme payments", limit: 20 }));
    fireEvent.click(await screen.findByRole("radio", { name: /Acme Processing Limited.*Card transaction processing/ }));
    expect(screen.queryByLabelText("Purpose code")).toBeNull();
    fireEvent.change(screen.getByLabelText("Relationship purpose"), { target: { value: "SERVICE_SUPPORT" } });
    fireEvent.click(screen.getByRole("button", { name: "Link vendor" }));

    await waitFor(() => expect(linkVendorRelationship).toHaveBeenCalledWith("relationship-1", {
      target_type: "PROGRAM", target_id: "program-1", purpose_code: "SERVICE_SUPPORT", purpose_label: "Service support",
    }));
    expect(await screen.findByText("Vendor linked to this Program.")).toBeTruthy();
  });

  it("preserves search, selection and purpose inputs when linking fails", async () => {
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [acme] });
    vi.mocked(linkVendorRelationship).mockRejectedValue(new Error("unavailable"));

    render(<VendorRelationshipLinks targetType="MATTER" targetID="matter-1"/>);
    await screen.findByText("No vendor relationships are linked to this issue or change.");
    fireEvent.click(screen.getByRole("button", { name: "Link vendor" }));
    const search = screen.getByLabelText("Search vendor relationships") as HTMLInputElement;
    fireEvent.change(search, { target: { value: "Acme" } });
    fireEvent.click(screen.getByRole("button", { name: "Search vendors" }));
    fireEvent.click(await screen.findByRole("radio", { name: /Acme Processing Limited/ }));
    const purpose = screen.getByLabelText("Relationship purpose") as HTMLSelectElement;
    fireEvent.change(purpose, { target: { value: "OTHER" } });
    const customPurpose = screen.getByLabelText("Custom purpose") as HTMLInputElement;
    expect((screen.getByRole("button", { name: "Link vendor" }) as HTMLButtonElement).disabled).toBe(true);
    fireEvent.change(customPurpose, { target: { value: "Supports remediation evidence" } });
    fireEvent.click(screen.getByRole("button", { name: "Link vendor" }));

    const alert = await screen.findByRole("alert");
    expect(alert.textContent).toContain("The vendor could not be linked");
    expect(search.value).toBe("Acme");
    expect(purpose.value).toBe("OTHER");
    expect(customPurpose.value).toBe("Supports remediation evidence");
    expect((screen.getByRole("radio", { name: /Acme Processing Limited/ }) as HTMLInputElement).checked).toBe(true);
  });

  it("keeps a failed search query and offers a concise retry", async () => {
    vi.mocked(loadVendorRelationships).mockRejectedValue(new Error("unavailable"));
    render(<VendorRelationshipLinks targetType="PROGRAM" targetID="program-1"/>);
    await screen.findByText("No vendor relationships are linked to this Program.");
    fireEvent.click(screen.getByRole("button", { name: "Link vendor" }));
    const search = screen.getByLabelText("Search vendor relationships") as HTMLInputElement;
    fireEvent.change(search, { target: { value: "Acme" } });
    fireEvent.click(screen.getByRole("button", { name: "Search vendors" }));

    const alert = await screen.findByRole("alert");
    expect(alert.textContent).toContain("Vendor search is unavailable");
    expect(search.value).toBe("Acme");
    expect(within(alert).getByRole("button", { name: "Try again" })).toBeTruthy();
  });

  it("ignores a link completion after the target changes", async () => {
    let finishLink!: (link: typeof activeLink) => void;
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [acme] });
    vi.mocked(linkVendorRelationship).mockImplementation(() => new Promise((resolve) => { finishLink = resolve; }));
    const { rerender } = render(<VendorRelationshipLinks targetType="PROGRAM" targetID="program-1"/>);
    await screen.findByText("No vendor relationships are linked to this Program.");
    fireEvent.click(screen.getByRole("button", { name: "Link vendor" }));
    fireEvent.change(screen.getByLabelText("Search vendor relationships"), { target: { value: "Acme" } });
    fireEvent.click(screen.getByRole("button", { name: "Search vendors" }));
    fireEvent.click(await screen.findByRole("radio", { name: /Acme Processing Limited/ }));
    fireEvent.change(screen.getByLabelText("Relationship purpose"), { target: { value: "SERVICE_SUPPORT" } });
    fireEvent.click(screen.getByRole("button", { name: "Link vendor" }));

    rerender(<VendorRelationshipLinks targetType="MATTER" targetID="matter-1"/>);
    await screen.findByText("No vendor relationships are linked to this issue or change.");
    finishLink(activeLink);

    await waitFor(() => expect(screen.queryByText("Acme Processing Limited")).toBeNull());
    expect(screen.queryByText("Vendor linked to this Program.")).toBeNull();
  });

  it("uses Enter in the search field to run the current search instead of linking a prior result", async () => {
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [acme] });
    render(<VendorRelationshipLinks targetType="PROGRAM" targetID="program-1"/>);
    await screen.findByText("No vendor relationships are linked to this Program.");
    fireEvent.click(screen.getByRole("button", { name: "Link vendor" }));
    const search = screen.getByLabelText("Search vendor relationships");
    fireEvent.change(search, { target: { value: "Acme" } });
    fireEvent.click(screen.getByRole("button", { name: "Search vendors" }));
    fireEvent.click(await screen.findByRole("radio", { name: /Acme Processing Limited/ }));
    fireEvent.change(screen.getByLabelText("Relationship purpose"), { target: { value: "SERVICE_SUPPORT" } });
    fireEvent.change(search, { target: { value: "Beta" } });
    fireEvent.keyDown(search, { key: "Enter", code: "Enter" });

    await waitFor(() => expect(loadVendorRelationships).toHaveBeenLastCalledWith({ search: "Beta", limit: 20 }));
    expect(linkVendorRelationship).not.toHaveBeenCalled();
  });

  it("does not show results returned for a query that the user has changed", async () => {
    let finishSearch!: (page: { items: VendorRelationshipAggregate[] }) => void;
    vi.mocked(loadVendorRelationships).mockImplementation(() => new Promise((resolve) => { finishSearch = resolve; }));
    render(<VendorRelationshipLinks targetType="PROGRAM" targetID="program-1"/>);
    await screen.findByText("No vendor relationships are linked to this Program.");
    fireEvent.click(screen.getByRole("button", { name: "Link vendor" }));
    const search = screen.getByLabelText("Search vendor relationships");
    fireEvent.change(search, { target: { value: "Acme" } });
    fireEvent.click(screen.getByRole("button", { name: "Search vendors" }));
    await waitFor(() => expect(loadVendorRelationships).toHaveBeenCalledTimes(1));
    fireEvent.change(search, { target: { value: "Beta" } });
    await act(async () => finishSearch({ items: [acme] }));

    expect(screen.queryByRole("radio", { name: /Acme Processing Limited/ })).toBeNull();
  });

  it("loads the next bounded page of links on request", async () => {
    vi.mocked(loadVendorRelationshipLinks)
      .mockResolvedValueOnce({ items: [activeLink], next_cursor: "next-link" })
      .mockResolvedValueOnce({ items: [{ ...activeLink, id: "link-2", relationship_id: "relationship-2" }] });
    vi.mocked(loadVendorRelationship).mockImplementation(async (id) => id === "relationship-1" ? acme : beta);
    render(<VendorRelationshipLinks targetType="PROGRAM" targetID="program-1"/>);

    await screen.findByText("Acme Processing Limited");
    fireEvent.click(screen.getByRole("button", { name: "Load more related vendors" }));

    expect(await screen.findByText("Beta Cloud Limited")).toBeTruthy();
    expect(loadVendorRelationshipLinks).toHaveBeenLastCalledWith({ target_type: "PROGRAM", target_id: "program-1", cursor: "next-link", limit: 50 });
  });

  it("labels an ended link as history instead of presenting it as current", async () => {
    vi.mocked(loadVendorRelationshipLinks).mockResolvedValue({ items: [{ ...activeLink, state: "ENDED" }], next_cursor: "more-links" });
    vi.mocked(loadVendorRelationship).mockResolvedValue(acme);
    render(<VendorRelationshipLinks targetType="PROGRAM" targetID="program-1"/>);

    expect(await screen.findByText("Acme Processing Limited")).toBeTruthy();
    expect(screen.getByText("Link ended")).toBeTruthy();
    expect(screen.queryByText("No vendor relationships are linked to this Program.")).toBeNull();
    expect(screen.getByRole("button", { name: "Load more related vendors" })).toBeTruthy();
  });

  it("ends an active link with a required reason and retains it as history", async () => {
    vi.mocked(loadVendorRelationshipLinks).mockResolvedValue({ items: [activeLink] });
    vi.mocked(loadVendorRelationship).mockResolvedValue(acme);
    vi.mocked(endVendorRelationshipLink).mockResolvedValue({ ...activeLink, state: "ENDED", version: 2, end_reason: "The vendor no longer supports this Program." });
    render(<VendorRelationshipLinks targetType="PROGRAM" targetID="program-1"/>);

    await screen.findByText("Acme Processing Limited");
    fireEvent.click(screen.getByRole("button", { name: "End link for Acme Processing Limited" }));
    const confirm = screen.getByRole("button", { name: "End vendor link" });
    expect((confirm as HTMLButtonElement).disabled).toBe(true);
    fireEvent.change(screen.getByLabelText("Reason for ending this link"), { target: { value: "The vendor no longer supports this Program." } });
    fireEvent.click(confirm);

    await waitFor(() => expect(endVendorRelationshipLink).toHaveBeenCalledWith("relationship-1", "link-1", { expected_version: 1, reason: "The vendor no longer supports this Program." }));
    expect(await screen.findByText("Link ended")).toBeTruthy();
    expect(screen.getByText("Vendor link ended. Existing history remains available.")).toBeTruthy();
  });

  it("refreshes related vendors after a conflict without clearing the link form", async () => {
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [acme] });
    vi.mocked(linkVendorRelationship).mockRejectedValue(new ApiError(409, "This vendor link changed.", "vendor_link_conflict"));
    render(<VendorRelationshipLinks targetType="PROGRAM" targetID="program-1"/>);
    await screen.findByText("No vendor relationships are linked to this Program.");
    fireEvent.click(screen.getByRole("button", { name: "Link vendor" }));
    const search = screen.getByLabelText("Search vendor relationships") as HTMLInputElement;
    fireEvent.change(search, { target: { value: "Acme" } });
    fireEvent.click(screen.getByRole("button", { name: "Search vendors" }));
    fireEvent.click(await screen.findByRole("radio", { name: /Acme Processing Limited/ }));
    const purpose = screen.getByLabelText("Relationship purpose") as HTMLSelectElement;
    fireEvent.change(purpose, { target: { value: "SERVICE_SUPPORT" } });
    fireEvent.click(screen.getByRole("button", { name: "Link vendor" }));

    const refresh = await screen.findByRole("button", { name: "Refresh related vendors" });
    fireEvent.click(refresh);
    await waitFor(() => expect(loadVendorRelationshipLinks).toHaveBeenCalledTimes(2));
    expect(search.value).toBe("Acme");
    expect(purpose.value).toBe("SERVICE_SUPPORT");
    expect((screen.getByRole("radio", { name: /Acme Processing Limited/ }) as HTMLInputElement).checked).toBe(true);
  });

  it("moves focus into the link form and restores it when cancelled", async () => {
    render(<VendorRelationshipLinks targetType="PROGRAM" targetID="program-1"/>);
    await screen.findByText("No vendor relationships are linked to this Program.");
    const open = screen.getByRole("button", { name: "Link vendor" });
    fireEvent.click(open);

    await waitFor(() => expect(document.activeElement).toBe(screen.getByLabelText("Search vendor relationships")));
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(document.activeElement).toBe(screen.getByRole("button", { name: "Link vendor" })));
  });
});
