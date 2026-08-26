import { useEffect, useId, useRef, useState } from "react";
import { apiErrorKind } from "../http";
import { loadVendorIdentity, newVendorBrandIdempotencyKey, removeApprovedVendorLogo, updateVendorIdentity, uploadApprovedVendorLogo } from "../vendorApi";
import { validateWebsiteDomain, vendorIdentityLimits } from "../vendorIdentity";
import type { VendorIdentityPresentation, VendorRelationshipAggregate } from "../vendorTypes";
import { FileDropzone } from "./FileDropzone";
import { VendorBrandIcon, vendorBrandLabel } from "./VendorBrandIcon";

const allowedLogoTypes = new Set(["image/png", "image/jpeg", "image/webp", "image/x-icon", "image/vnd.microsoft.icon"]);
const maxLogoBytes = 512 * 1024;

type IdentityValues = { legalName: string; tradingName: string; registrationRef: string; jurisdiction: string; websiteDomain: string };

export function VendorIdentityEditor({ record, onCancel, onIdentitySaved, onBrandSaved }: {
  record: VendorRelationshipAggregate;
  onCancel: () => void;
  onIdentitySaved: (presentation: VendorIdentityPresentation) => void;
  onBrandSaved: (presentation: VendorIdentityPresentation) => void;
}) {
  const [presentation, setPresentation] = useState<VendorIdentityPresentation>({ vendor: record.vendor, brand: record.brand ?? { state: "UNAVAILABLE", version: 0, event_version: 0 } });
  const [values, setValues] = useState<IdentityValues>(() => identityValues(record.vendor));
  const [state, setState] = useState<"loading" | "ready" | "unavailable">("loading");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [formError, setFormError] = useState("");
  const [notice, setNotice] = useState("");
  const [saving, setSaving] = useState(false);
  const [file, setFile] = useState<File>();
  const [fileError, setFileError] = useState("");
  const [brandError, setBrandError] = useState("");
  const [savingBrand, setSavingBrand] = useState(false);
  const legalNameInput = useRef<HTMLInputElement>(null);
  const uploadKey = useRef("");
  const removeKey = useRef("");
  const descriptionID = useId();

  useEffect(() => {
    let current = true;
    setState("loading");
    void loadVendorIdentity(record.vendor.id).then((loaded) => {
      if (!current) return;
      setPresentation(loaded);
      setValues(identityValues(loaded.vendor));
      setState("ready");
    }).catch(() => {
      if (!current) return;
      setState("unavailable");
    });
    return () => { current = false; };
  }, [record.vendor.id]);

  useEffect(() => {
    if (state === "ready") legalNameInput.current?.focus();
  }, [state]);

  function setValue(key: keyof IdentityValues, value: string) {
    setValues((current) => ({ ...current, [key]: value }));
    setErrors((current) => ({ ...current, [key]: "" }));
  }

  async function saveIdentity(event: React.FormEvent) {
    event.preventDefault();
    const nextErrors: Record<string, string> = {};
    if (!values.legalName.trim()) nextErrors.legalName = "Enter the vendor's legal name.";
    const websiteError = validateWebsiteDomain(values.websiteDomain);
    if (websiteError) nextErrors.websiteDomain = websiteError;
    setErrors(nextErrors);
    if (Object.values(nextErrors).some(Boolean)) return;
    setSaving(true); setFormError(""); setNotice("");
    try {
      const saved = await updateVendorIdentity(record.vendor.id, {
        expected_version: presentation.vendor.version,
        legal_name: values.legalName.trim(),
        trading_name: optional(values.tradingName),
        registration_ref: optional(values.registrationRef),
        jurisdiction: optional(values.jurisdiction),
        website_domain: optional(values.websiteDomain)?.toLowerCase(),
      });
      setPresentation(saved);
      onIdentitySaved(saved);
    } catch (error) {
      setFormError(identityErrorText(apiErrorKind(error)));
    } finally {
      setSaving(false);
    }
  }

  function selectFile(selected?: File) {
    setFileError(""); setBrandError(""); setNotice("");
    if (!selected) { setFile(undefined); uploadKey.current = ""; return; }
    const extensionAllowed = /\.(png|jpe?g|webp|ico)$/i.test(selected.name);
    if (!allowedLogoTypes.has(selected.type) && !(selected.type === "" && extensionAllowed)) {
      setFile(undefined); uploadKey.current = "";
      setFileError("Select a PNG, JPEG, WebP or ICO file.");
      return;
    }
    if (selected.size > maxLogoBytes) {
      setFile(undefined); uploadKey.current = "";
      setFileError("Select a logo no larger than 512 KiB.");
      return;
    }
    setFile(selected); uploadKey.current = newVendorBrandIdempotencyKey();
  }

  async function useApprovedLogo() {
    if (!file) return;
    setSavingBrand(true); setBrandError(""); setNotice("");
    try {
      const saved = await uploadApprovedVendorLogo(record.vendor.id, file, presentation.brand.version, uploadKey.current || newVendorBrandIdempotencyKey());
      setPresentation(saved);
      setFile(undefined);
      uploadKey.current = "";
      setNotice("Approved logo saved.");
      onBrandSaved(saved);
    } catch (error) {
      setBrandError(brandErrorText(apiErrorKind(error)));
    } finally {
      setSavingBrand(false);
    }
  }

  async function useWebsiteIcon() {
    setSavingBrand(true); setBrandError(""); setNotice("");
    try {
      removeKey.current ||= newVendorBrandIdempotencyKey();
      const saved = await removeApprovedVendorLogo(record.vendor.id, presentation.brand.version, removeKey.current);
      setPresentation(saved);
      removeKey.current = "";
      setNotice(saved.brand.state === "WEBSITE_ICON" ? "Website icon restored." : "Approved logo removed. The vendor monogram is shown until a website icon is available.");
      onBrandSaved(saved);
    } catch (error) {
      const kind = apiErrorKind(error);
      setBrandError(kind === "conflict" ? "The vendor logo changed. Reload the vendor before removing the approved logo." : kind === "forbidden" || kind === "unauthorized" ? "Your current role cannot remove the approved vendor logo." : "The approved logo could not be removed. Try again.");
    } finally {
      setSavingBrand(false);
    }
  }

  if (state === "loading") return <div className="vendor-form vendor-identity-loading" aria-live="polite" aria-busy="true">Loading vendor details…</div>;
  if (state === "unavailable") return <section className="vendor-state" role="alert"><h2>Vendor details are unavailable</h2><p>The current vendor identity could not be loaded. Return to the relationship or try again.</p><div className="vendor-form-actions"><button type="button" className="secondary-button" onClick={onCancel}>Return to relationship</button><button type="button" className="primary-button" onClick={() => { setState("loading"); void loadVendorIdentity(record.vendor.id).then((loaded) => { setPresentation(loaded); setValues(identityValues(loaded.vendor)); setState("ready"); }).catch(() => setState("unavailable")); }}>Try again</button></div></section>;

  return <form className="vendor-form vendor-identity-form" onSubmit={saveIdentity} noValidate>
    <div className="vendor-identity-form-heading"><div><span className="eyebrow">Shared vendor identity</span><h2>Edit vendor details</h2><p>Update the organization details used across this vendor&apos;s relationships.</p></div><VendorBrandIcon vendorID={presentation.vendor.id} legalName={presentation.vendor.legal_name} brand={presentation.brand} size="detail"/></div>
    {notice && <p className="vendor-notice" role="status">{notice}</p>}
    {formError && <div className="vendor-form-error" role="alert">{formError}</div>}
    <div className="vendor-form-grid">
      <IdentityField label="Legal name" required error={errors.legalName}><input ref={legalNameInput} id="vendor-identity-legal-name" value={values.legalName} maxLength={vendorIdentityLimits.legalName} onChange={(event) => setValue("legalName", event.target.value)} aria-invalid={Boolean(errors.legalName)}/></IdentityField>
      <IdentityField label="Trading name"><input id="vendor-identity-trading-name" value={values.tradingName} maxLength={vendorIdentityLimits.tradingName} onChange={(event) => setValue("tradingName", event.target.value)}/></IdentityField>
      <IdentityField label="Registration reference"><input id="vendor-identity-registration" value={values.registrationRef} maxLength={vendorIdentityLimits.registrationRef} onChange={(event) => setValue("registrationRef", event.target.value)}/></IdentityField>
      <IdentityField label="Jurisdiction"><input id="vendor-identity-jurisdiction" value={values.jurisdiction} maxLength={vendorIdentityLimits.jurisdiction} onChange={(event) => setValue("jurisdiction", event.target.value)}/></IdentityField>
      <IdentityField label="Website domain" error={errors.websiteDomain} wide><input id="vendor-identity-website" type="text" inputMode="url" autoComplete="url" value={values.websiteDomain} maxLength={vendorIdentityLimits.websiteDomain} placeholder="vendor.example" onChange={(event) => setValue("websiteDomain", event.target.value)} aria-invalid={Boolean(errors.websiteDomain)} aria-describedby={descriptionID}/></IdentityField>
      <p id={descriptionID} className="vendor-field-help">Enter the hostname only, such as vendor.example. The saved hostname is used to retrieve a website icon.</p>
    </div>
    <section className="vendor-brand-controls" aria-labelledby="vendor-brand-heading">
      <div className="vendor-brand-status"><div><h3 id="vendor-brand-heading">Vendor icon</h3><strong>{vendorBrandLabel(presentation.brand)}</strong></div><p>{brandStatusDetail(presentation.brand.state)}</p></div>
      <FileDropzone
        compact
        label="Approved logo file"
        description="PNG, JPEG, WebP or ICO · 512 KiB maximum. Selecting a file does not save it."
        accept=".png,.jpg,.jpeg,.webp,.ico,image/png,image/jpeg,image/webp,image/x-icon,image/vnd.microsoft.icon"
        disabled={savingBrand}
        busy={savingBrand && Boolean(file)}
        actionLabel="Choose logo"
        replaceLabel="Replace logo"
        fileName={file?.name}
        fileSize={file?.size}
        onSelect={selectFile}
      />
      {file && <div className="vendor-logo-staged"><span>Selected file is ready to save.</span><button type="button" className="text-button" disabled={savingBrand} onClick={() => { setFile(undefined); uploadKey.current = ""; }}>Remove selected file</button></div>}
      {fileError && <p className="inline-error" role="alert">{fileError}</p>}
      {brandError && <p className="vendor-form-error" role="alert">{brandError}</p>}
      <div className="vendor-brand-actions">
        {presentation.brand.state === "APPROVED_LOGO" && <div><p>Removing the approved logo restores the website icon when one is available; otherwise the vendor monogram is shown.</p><button type="button" className="secondary-button" disabled={savingBrand} onClick={() => void useWebsiteIcon()}>{savingBrand ? "Updating…" : "Use website icon"}</button></div>}
        {file && <button type="button" className="secondary-button" disabled={savingBrand} onClick={() => void useApprovedLogo()}>{savingBrand ? "Saving logo…" : "Use approved logo"}</button>}
      </div>
    </section>
    <div className="vendor-form-actions"><button type="button" className="secondary-button" onClick={onCancel} disabled={saving || savingBrand}>Cancel</button><button type="submit" className="primary-button" disabled={saving || savingBrand}>{saving ? "Saving…" : "Save vendor details"}</button></div>
  </form>;
}

function IdentityField({ label, required, error, wide, children }: { label: string; required?: boolean; error?: string; wide?: boolean; children: React.ReactElement<{ id?: string }> }) {
  return <label className={wide ? "vendor-field wide" : "vendor-field"} htmlFor={children.props.id}><span className={required ? "required" : undefined}>{label}</span>{children}{error && <small role="alert">{error}</small>}</label>;
}

function identityValues(vendor: VendorIdentityPresentation["vendor"]): IdentityValues {
  return { legalName: vendor.legal_name, tradingName: vendor.trading_name ?? "", registrationRef: vendor.registration_ref ?? "", jurisdiction: vendor.jurisdiction ?? "", websiteDomain: vendor.website_domain ?? "" };
}
function optional(value: string) { return value.trim() || undefined; }
function identityErrorText(kind: ReturnType<typeof apiErrorKind>) {
  if (kind === "conflict") return "The vendor details changed. Your entries are still here; reload the vendor before saving again.";
  if (kind === "forbidden" || kind === "unauthorized") return "Your current role cannot change these vendor details. Your entries are still here.";
  if (kind === "validation") return "Check the vendor details. Your entries are still here.";
  return "The vendor details could not be saved. Your entries are still here; try again.";
}
function brandErrorText(kind: ReturnType<typeof apiErrorKind>) {
  if (kind === "conflict") return "The vendor logo changed. The selected file is still here; reload the vendor before trying again.";
  if (kind === "forbidden" || kind === "unauthorized") return "Your current role cannot change the approved vendor logo. The selected file is still here.";
  if (kind === "validation") return "The selected image could not be used. Select a PNG, JPEG, WebP or ICO file of 512 KiB or less.";
  return "The approved logo could not be saved. The selected file is still here; try again.";
}
function brandStatusDetail(state: VendorIdentityPresentation["brand"]["state"]) {
  if (state === "APPROVED_LOGO") return "The approved image is used in vendor records.";
  if (state === "WEBSITE_ICON") return "The stored website icon is used in vendor records.";
  if (state === "PENDING") return "Website icon retrieval is in progress. The vendor monogram remains available.";
  return "Check the website domain or upload an approved logo. The vendor monogram remains available.";
}
