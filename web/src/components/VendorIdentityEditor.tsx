import { useEffect, useId, useRef, useState } from "react";
import { apiErrorKind } from "../http";
import { isCommittedCommandReceipt, loadVendorIdentity, newVendorBrandIdempotencyKey, removeApprovedVendorLogo, updateVendorIdentity, uploadApprovedVendorLogo } from "../vendorApi";
import { normalizeWebsiteDomain, validateWebsiteDomain, vendorIdentityLimits } from "../vendorIdentity";
import type { VendorIdentityPresentation, VendorRelationshipAggregate } from "../vendorTypes";
import { FileDropzone } from "./FileDropzone";
import { VendorBrandIcon, vendorBrandLabel } from "./VendorBrandIcon";

const allowedLogoTypes = new Set(["image/png", "image/jpeg", "image/webp", "image/x-icon", "image/vnd.microsoft.icon"]);
const maxLogoBytes = 512 * 1024;

type IdentityValues = { legalName: string; tradingName: string; registrationRef: string; jurisdiction: string; websiteDomain: string };

export function VendorIdentityEditor({ record, onCancel, onIdentitySaved, onBrandSaved, onPresentationReloaded }: {
  record: VendorRelationshipAggregate;
  onCancel: () => void;
  onIdentitySaved: (presentation: VendorIdentityPresentation) => void;
  onBrandSaved: (presentation: VendorIdentityPresentation) => void;
  onPresentationReloaded?: (presentation: VendorIdentityPresentation) => void;
}) {
  const initialPresentation: VendorIdentityPresentation = { vendor: record.vendor, brand: record.brand ?? { state: "UNAVAILABLE", version: 0, event_version: 0 } };
  const [presentation, setPresentation] = useState<VendorIdentityPresentation>(initialPresentation);
  const [values, setValues] = useState<IdentityValues>(() => identityValues(record.vendor));
  const [state, setState] = useState<"loading" | "ready" | "unavailable">("loading");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [formError, setFormError] = useState("");
  const [notice, setNotice] = useState("");
  const [saving, setSaving] = useState(false);
  const [file, setFile] = useState<File>();
  const [fileError, setFileError] = useState("");
  const [brandError, setBrandError] = useState("");
  const [brandOperation, setBrandOperation] = useState<"upload" | "remove" | "">("");
  const [reloading, setReloading] = useState(false);
  const [conflict, setConflict] = useState<"identity" | "brand">();
  const legalNameInput = useRef<HTMLInputElement>(null);
  const presentationRef = useRef(initialPresentation);
  const mutationRef = useRef(false);
  const uploadKey = useRef("");
  const removeKey = useRef("");
  const descriptionID = useId();
  const savingBrand = brandOperation !== "";
  const mutating = saving || savingBrand || reloading;

  useEffect(() => {
    let current = true;
    setState("loading");
    void loadVendorIdentity(record.vendor.id).then((loaded) => {
      if (!current) return;
      replacePresentation(loaded);
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

  function replacePresentation(next: VendorIdentityPresentation) {
    presentationRef.current = next;
    setPresentation(next);
  }

  function setValue(key: keyof IdentityValues, value: string) {
    setValues((current) => ({ ...current, [key]: value }));
    setErrors((current) => ({ ...current, [key]: "" }));
  }

  async function saveIdentity(event: React.FormEvent) {
    event.preventDefault();
    if (mutationRef.current) return;
    const nextErrors: Record<string, string> = {};
    if (!values.legalName.trim()) nextErrors.legalName = "Enter the vendor's legal name.";
    const websiteError = validateWebsiteDomain(values.websiteDomain);
    if (websiteError) nextErrors.websiteDomain = websiteError;
    setErrors(nextErrors);
    if (Object.values(nextErrors).some(Boolean)) return;
    mutationRef.current = true;
    setSaving(true); setFormError(""); setNotice("");
    try {
      const saved = await updateVendorIdentity(record.vendor.id, {
        expected_version: presentationRef.current.vendor.version,
        legal_name: values.legalName.trim(),
        trading_name: optional(values.tradingName),
        registration_ref: optional(values.registrationRef),
        jurisdiction: optional(values.jurisdiction),
        website_domain: normalizeWebsiteDomain(values.websiteDomain),
      });
      if (isCommittedCommandReceipt(saved)) {
        setFormError("Vendor details were saved, but the updated vendor could not be loaded. Reload the current vendor to confirm the saved version.");
        setConflict("identity");
        return;
      }
      const merged = {
        vendor: saved.vendor,
        brand: saved.brand.version > presentationRef.current.brand.version ? saved.brand : presentationRef.current.brand,
      };
      replacePresentation(merged);
      setConflict(undefined);
      onIdentitySaved(merged);
    } catch (error) {
      const kind = apiErrorKind(error);
      setFormError(identityErrorText(kind));
      setConflict(kind === "conflict" ? "identity" : undefined);
    } finally {
      mutationRef.current = false;
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
    if (!file || mutationRef.current) return;
    mutationRef.current = true;
    setBrandOperation("upload"); setBrandError(""); setNotice("");
    try {
      const saved = await uploadApprovedVendorLogo(record.vendor.id, file, presentationRef.current.brand.version, uploadKey.current || newVendorBrandIdempotencyKey());
      if (isCommittedCommandReceipt(saved)) {
        setBrandError("The approved logo was saved, but the updated vendor could not be loaded. Reload the current vendor to confirm the saved icon.");
        setConflict("brand");
        return;
      }
      const merged = {
        vendor: saved.vendor.version > presentationRef.current.vendor.version ? saved.vendor : presentationRef.current.vendor,
        brand: saved.brand,
      };
      replacePresentation(merged);
      setFile(undefined);
      uploadKey.current = "";
      setConflict(undefined);
      setNotice("Approved logo saved.");
      onBrandSaved(merged);
    } catch (error) {
      const kind = apiErrorKind(error);
      setBrandError(brandErrorText(kind));
      setConflict(kind === "conflict" ? "brand" : undefined);
    } finally {
      mutationRef.current = false;
      setBrandOperation("");
    }
  }

  async function removeApprovedLogo() {
    if (mutationRef.current) return;
    mutationRef.current = true;
    setBrandOperation("remove"); setBrandError(""); setNotice("");
    try {
      removeKey.current ||= newVendorBrandIdempotencyKey();
      const saved = await removeApprovedVendorLogo(record.vendor.id, presentationRef.current.brand.version, removeKey.current);
      if (isCommittedCommandReceipt(saved)) {
        setBrandError("The approved logo was removed, but the updated vendor could not be loaded. Reload the current vendor to confirm which icon is now shown.");
        setConflict("brand");
        return;
      }
      const merged = {
        vendor: saved.vendor.version > presentationRef.current.vendor.version ? saved.vendor : presentationRef.current.vendor,
        brand: saved.brand,
      };
      replacePresentation(merged);
      removeKey.current = "";
      setConflict(undefined);
      setNotice(saved.brand.state === "WEBSITE_ICON" ? "Website icon restored." : "Approved logo removed. The vendor monogram is shown until a website icon is available.");
      onBrandSaved(merged);
    } catch (error) {
      const kind = apiErrorKind(error);
      setBrandError(kind === "conflict" ? "The vendor logo changed. Reload the current vendor before removing the approved logo." : kind === "forbidden" || kind === "unauthorized" ? "Your current role cannot remove the approved vendor logo." : "The approved logo could not be removed. Try again.");
      setConflict(kind === "conflict" ? "brand" : undefined);
    } finally {
      mutationRef.current = false;
      setBrandOperation("");
    }
  }

  async function reloadCurrentVendor() {
    if (mutationRef.current) return;
    mutationRef.current = true;
    setReloading(true); setNotice("");
    try {
      const loaded = await loadVendorIdentity(record.vendor.id);
      replacePresentation(loaded);
      onPresentationReloaded?.(loaded);
      if (file) uploadKey.current = newVendorBrandIdempotencyKey();
      removeKey.current = "";
      setFormError(""); setBrandError(""); setConflict(undefined);
      setNotice(file ? "Current vendor details reloaded. Your entries and selected file are unchanged." : "Current vendor details reloaded. Your entries are unchanged.");
    } catch {
      const message = file ? "Current vendor details could not be reloaded. Your entries and selected file are unchanged; try again." : "Current vendor details could not be reloaded. Your entries are unchanged; try again.";
      if (conflict === "brand") setBrandError(message); else setFormError(message);
    } finally {
      mutationRef.current = false;
      setReloading(false);
    }
  }

  function retryInitialLoad() {
    setState("loading");
    void loadVendorIdentity(record.vendor.id).then((loaded) => {
      replacePresentation(loaded);
      setValues(identityValues(loaded.vendor));
      setState("ready");
    }).catch(() => setState("unavailable"));
  }

  if (state === "loading") return <div className="vendor-form vendor-identity-loading" aria-live="polite" aria-busy="true">Loading vendor details…</div>;
  if (state === "unavailable") return <section className="vendor-state" role="alert"><h2>Vendor details are unavailable</h2><p>The current vendor identity could not be loaded. Return to the relationship or try again.</p><div className="vendor-form-actions"><button type="button" className="secondary-button" onClick={onCancel}>Return to relationship</button><button type="button" className="primary-button" onClick={retryInitialLoad}>Try again</button></div></section>;

  return <form className="vendor-form vendor-identity-form" onSubmit={saveIdentity} noValidate aria-busy={mutating || undefined}>
    <div className="vendor-identity-form-heading"><div><span className="eyebrow">Shared vendor identity</span><h2>Edit vendor details</h2><p>Update the organization details used across this vendor&apos;s relationships.</p></div><div className="vendor-identity-heading-actions"><VendorBrandIcon vendorID={presentation.vendor.id} legalName={presentation.vendor.legal_name} brand={presentation.brand} size="detail"/><button type="button" className="secondary-button" onClick={onCancel} disabled={mutating}>Return to relationship</button></div></div>
    {notice && <p className="vendor-notice" role="status">{notice}</p>}
    {formError && <div className="vendor-form-error" role="alert">{formError}</div>}
    {conflict && <div className="vendor-conflict-recovery"><button type="button" className="secondary-button" disabled={mutating} onClick={() => void reloadCurrentVendor()}>{reloading ? "Reloading…" : "Reload current vendor"}</button></div>}
    <div className="vendor-form-grid">
      <IdentityField label="Legal name" required error={errors.legalName}><input ref={legalNameInput} id="vendor-identity-legal-name" value={values.legalName} maxLength={vendorIdentityLimits.legalName} disabled={mutating} onChange={(event) => setValue("legalName", event.target.value)} aria-invalid={Boolean(errors.legalName)}/></IdentityField>
      <IdentityField label="Trading name"><input id="vendor-identity-trading-name" value={values.tradingName} maxLength={vendorIdentityLimits.tradingName} disabled={mutating} onChange={(event) => setValue("tradingName", event.target.value)}/></IdentityField>
      <IdentityField label="Registration reference"><input id="vendor-identity-registration" value={values.registrationRef} maxLength={vendorIdentityLimits.registrationRef} disabled={mutating} onChange={(event) => setValue("registrationRef", event.target.value)}/></IdentityField>
      <IdentityField label="Jurisdiction"><input id="vendor-identity-jurisdiction" value={values.jurisdiction} maxLength={vendorIdentityLimits.jurisdiction} disabled={mutating} onChange={(event) => setValue("jurisdiction", event.target.value)}/></IdentityField>
      <IdentityField label="Website domain" error={errors.websiteDomain} wide><input id="vendor-identity-website" type="text" inputMode="url" autoComplete="url" value={values.websiteDomain} maxLength={vendorIdentityLimits.websiteDomain} placeholder="vendor.example" disabled={mutating} onChange={(event) => setValue("websiteDomain", event.target.value)} aria-invalid={Boolean(errors.websiteDomain)} aria-describedby={descriptionID}/></IdentityField>
      <p id={descriptionID} className="vendor-field-help">Enter the hostname only, such as vendor.example. The saved hostname is used to retrieve a website icon.</p>
    </div>
    <section className="vendor-brand-controls" aria-labelledby="vendor-brand-heading">
      <div className="vendor-brand-status"><div><h3 id="vendor-brand-heading">Vendor icon</h3><strong>{vendorBrandLabel(presentation.brand)}</strong></div><p>{brandStatusDetail(presentation.brand.state)}</p></div>
      <FileDropzone compact label="Approved logo file" description="PNG, JPEG, WebP or ICO · 512 KiB maximum. Selecting a file does not save it." accept=".png,.jpg,.jpeg,.webp,.ico,image/png,image/jpeg,image/webp,image/x-icon,image/vnd.microsoft.icon" disabled={mutating} busy={brandOperation === "upload"} actionLabel="Choose logo" replaceLabel="Replace logo" fileName={file?.name} fileSize={file?.size} onSelect={selectFile}/>
      {file && <div className="vendor-logo-staged"><span>Selected file is ready to save.</span><button type="button" className="text-button" disabled={mutating} onClick={() => { setFile(undefined); uploadKey.current = ""; }}>Remove selected file</button></div>}
      {fileError && <p className="inline-error" role="alert">{fileError}</p>}
      {brandError && <p className="vendor-form-error" role="alert">{brandError}</p>}
      <div className="vendor-brand-actions">
        {presentation.brand.state === "APPROVED_LOGO" && <div><p>Removing the approved logo restores the website icon when one is available; otherwise the vendor monogram is shown.</p><button type="button" className="secondary-button" disabled={mutating} onClick={() => void removeApprovedLogo()}>{brandOperation === "remove" ? "Removing…" : "Remove approved logo"}</button></div>}
        {file && <button type="button" className="secondary-button" disabled={mutating} onClick={() => void useApprovedLogo()}>{brandOperation === "upload" ? "Saving logo…" : "Use approved logo"}</button>}
      </div>
    </section>
    <div className="vendor-form-actions"><button type="button" className="secondary-button" onClick={onCancel} disabled={mutating}>Cancel</button><button type="submit" className="primary-button" disabled={mutating}>{saving ? "Saving…" : "Save vendor details"}</button></div>
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
  if (kind === "conflict") return "Vendor details changed. Reload the current vendor, then save your entries again.";
  if (kind === "forbidden" || kind === "unauthorized") return "Your current role cannot change these vendor details. Your entries are still here.";
  if (kind === "validation") return "Check the vendor details. Your entries are still here.";
  return "The vendor details could not be saved. Your entries are still here; try again.";
}
function brandErrorText(kind: ReturnType<typeof apiErrorKind>) {
  if (kind === "conflict") return "The vendor logo changed. Reload the current vendor, then try the selected file again.";
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
