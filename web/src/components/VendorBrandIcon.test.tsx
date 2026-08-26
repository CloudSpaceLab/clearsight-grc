import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { VendorBrandIcon } from "./VendorBrandIcon";

describe("VendorBrandIcon", () => {
  it("renders a same-origin versioned icon and never a remote source", () => {
    const { container, rerender } = render(<VendorBrandIcon vendorID="vendor-1" legalName="Acme Processing Limited" brand={{ state: "WEBSITE_ICON", source: "VENDOR_WEBSITE", asset_token: "asset-4", version: 4, event_version: 4 }}/>);
    expect(screen.getByRole("img", { name: "Acme Processing Limited icon" }).getAttribute("src")).toBe("/api/v1/vendor-identities/vendor-1/brand?version=asset-4");
    expect(container.querySelector('img[src^="http"]')).toBeNull();

    rerender(<VendorBrandIcon vendorID="vendor-1" legalName="Acme Processing Limited" brand={{ state: "WEBSITE_ICON", source: "VENDOR_WEBSITE", asset_token: "https://vendor.example/logo.png", version: 4, event_version: 4 }}/>);
    expect(screen.queryByRole("img")).toBeNull();
    expect(screen.getByText("AL")).toBeTruthy();
  });

  it("falls back silently to the legal-name monogram when image loading fails", () => {
    render(<VendorBrandIcon vendorID="vendor-1" legalName="Acme Processing Limited" brand={{ state: "APPROVED_LOGO", source: "APPROVED_UPLOAD", asset_token: "asset-5", version: 5, event_version: 5 }}/>);
    fireEvent.error(screen.getByRole("img", { name: "Acme Processing Limited icon" }));
    expect(screen.queryByRole("img")).toBeNull();
    expect(screen.getByText("AL")).toBeTruthy();
  });
});
