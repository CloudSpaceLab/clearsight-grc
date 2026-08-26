import { useEffect, useState } from "react";
import { vendorBrandURL } from "../vendorApi";
import type { VendorBrandPresentation } from "../vendorTypes";
import { Monogram } from "./Monogram";

export function VendorBrandIcon({ vendorID, legalName, brand, decorative = false, size = "row" }: { vendorID: string; legalName: string; brand?: VendorBrandPresentation; decorative?: boolean; size?: "row" | "detail" }) {
  const source = brand?.asset_token && (brand.state === "APPROVED_LOGO" || brand.state === "WEBSITE_ICON")
    ? vendorBrandURL(vendorID, brand.asset_token)
    : undefined;
  const [failedSource, setFailedSource] = useState<string>();
  useEffect(() => setFailedSource(undefined), [source]);
  const className = `vendor-brand-icon vendor-brand-icon-${size}`;
  if (!source || failedSource === source) return <Monogram name={legalName} decorative={decorative} className={`${className} vendor-brand-monogram`}/>;
  return <span className={className}><img src={source} alt={decorative ? "" : `${legalName} icon`} onError={() => setFailedSource(source)}/></span>;
}

export function vendorBrandLabel(brand?: VendorBrandPresentation) {
  if (brand?.state === "APPROVED_LOGO") return "Approved logo";
  if (brand?.state === "WEBSITE_ICON") return "Website icon available";
  if (brand?.state === "PENDING") return "Website icon pending";
  return "Vendor icon unavailable";
}
