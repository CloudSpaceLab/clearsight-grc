import { useEffect, useState } from "react";
import { loadVendorRelationshipLinks } from "../vendorLinkApi";
import { loadVendorRelationship } from "../vendorApi";
import { loadVendorForms, type VendorFormRow } from "../vendorFormsApi";
import { Button } from "./ui";

type AffectedVendor = { id: string; name: string; relationships: string[]; forms: number };
type Summary = { relationships: number; forms: number; documents: number; responseFields: number; bankFindings: number; currencyUnknown: number; vendors: AffectedVendor[]; checkedAt: string };

async function allLinkedRelationships(programID: string) {
  const ids = new Set<string>();
  let cursor: string | undefined;
  let scanned = 0;
  do {
    const page = await loadVendorRelationshipLinks({ target_type: "PROGRAM", target_id: programID, limit: 50, ...(cursor ? { cursor } : {}) });
    scanned += page.items.length;
    for (const link of page.items) if (link.state === "ACTIVE") ids.add(link.relationship_id);
    cursor = page.next_cursor;
    if (scanned > 500) throw new Error("More than 500 linked relationships require a narrower review.");
  } while (cursor);
  return [...ids];
}

async function currentForms(relationshipID: string) {
  const forms: VendorFormRow[] = [];
  let cursor: string | undefined;
  let scanned = 0;
  let observedAt = "";
  do {
    const page = await loadVendorForms(relationshipID, { limit: 100, ...(cursor ? { cursor } : {}) });
    scanned += page.items.length;
    if (!observedAt || page.observed_at < observedAt) observedAt = page.observed_at;
    forms.push(...page.items.filter(row => row.current && row.response_state === "SUBMITTED"));
    cursor = page.next_cursor;
    if (scanned > 1000) throw new Error("More than 1,000 forms for one relationship require a narrower review.");
  } while (cursor);
  return { forms, observedAt };
}

export async function loadProgramVendorFollowUp(programID: string): Promise<Summary> {
  const relationships = await allLinkedRelationships(programID);
  const result: Summary = { relationships: relationships.length, forms: 0, documents: 0, responseFields: 0, bankFindings: 0, currencyUnknown: 0, vendors: [], checkedAt: "" };
  const vendors = new Map<string, AffectedVendor>();
  // Four bounded reads at a time keep a large linked register usable without an unscoped population request.
  for (let offset = 0; offset < relationships.length; offset += 4) {
    const batch = await Promise.all(relationships.slice(offset, offset + 4).map(async id => ({ id, ...(await currentForms(id)) })));
    for (const { id, forms, observedAt } of batch) {
      if (!result.checkedAt || observedAt < result.checkedAt) result.checkedAt = observedAt;
      let affected = 0;
      for (const form of forms) {
        const items = form.attention_items ?? [];
        const docs = items.filter(item => item.kind === "VENDOR_DOCUMENT" && item.state === "EXPIRED");
        const fields = items.filter(item => item.kind === "VENDOR_RESPONSE_FIELD");
        const internal = items.filter(item => item.kind === "INTERNAL_REVIEW" || (item.kind === "VENDOR_DOCUMENT" && item.source === "REVIEW" && item.state === "GAP"));
        const unknown = Boolean(form.outdated && docs.length === 0 && fields.length === 0 && internal.length === 0);
        if (!docs.length && !fields.length && !internal.length && !unknown) continue;
        affected++;
        result.forms++;
        result.documents += new Set(docs.map(item => item.field_id || item.label)).size;
        result.responseFields += new Set(fields.map(item => item.field_id || item.label)).size;
        result.bankFindings += new Set(internal.map(item => `${item.rule_id || item.field_id || item.label}:${item.state}`)).size;
        if (unknown) result.currencyUnknown++;
      }
      if (!affected) continue;
      const record = await loadVendorRelationship(id);
      const vendor = vendors.get(record.vendor.id) ?? { id: record.vendor.id, name: record.vendor.legal_name, relationships: [], forms: 0 };
      vendor.relationships.push(id);
      vendor.forms += affected;
      vendors.set(vendor.id, vendor);
    }
  }
  result.vendors = [...vendors.values()].sort((a, b) => a.name.localeCompare(b.name));
  return result;
}

export function ProgramVendorFollowUp({ programID }: { programID: string }) {
  const [summary, setSummary] = useState<Summary>();
  const [error, setError] = useState("");
  const [generation, setGeneration] = useState(0);
  useEffect(() => {
    let active = true;
    setSummary(undefined);
    setError("");
    void loadProgramVendorFollowUp(programID).then(value => { if (active) setSummary(value); }, cause => { if (active) setError(cause instanceof Error ? cause.message : "Linked vendor forms could not be checked."); });
    return () => { active = false; };
  }, [programID, generation]);
  return <section className="program-vendor-follow-up" aria-label="Linked vendor follow-up">
    <div><span className="eyebrow">Linked vendors</span><h2>Vendor information to update</h2></div>
    {!summary && !error && <p role="status">Checking forms from linked vendors…</p>}
    {error && <div role="alert"><p>{error}</p><Button onPress={() => setGeneration(value => value + 1)}>Retry vendor follow-up</Button></div>}
    {summary && <>
      {summary.relationships === 0 ? <p>No active vendor relationships are linked to this Program. Link a relationship from the vendor register.</p>
        : summary.forms === 0 ? <p>No expired documents or recorded response gaps were identified in submitted forms across {summary.relationships} linked {summary.relationships === 1 ? "relationship" : "relationships"}. Missing expiry dates are not counted as current.</p>
        : <><p><strong>{summary.forms} {summary.forms === 1 ? "form" : "forms"} across {summary.vendors.length} {summary.vendors.length === 1 ? "vendor need" : "vendors need"} follow-up.</strong> {[
          summary.documents && `${summary.documents} expired ${summary.documents === 1 ? "document" : "documents"}`,
          summary.responseFields && `${summary.responseFields} response ${summary.responseFields === 1 ? "field" : "fields"} to update`,
          summary.bankFindings && `${summary.bankFindings} bank-assessed ${summary.bankFindings === 1 ? "finding" : "findings"}`,
          summary.currencyUnknown && `${summary.currencyUnknown} outdated ${summary.currencyUnknown === 1 ? "response" : "responses"} requiring review`,
        ].filter(Boolean).join(" · ")}</p>
        <details><summary>Review affected vendors</summary><ul>{summary.vendors.map(vendor => <li key={vendor.id}><strong>{vendor.name}</strong><span>{vendor.forms} {vendor.forms === 1 ? "form" : "forms"}</span>{vendor.relationships.map(id => <a key={id} href={`#vendors/register/${encodeURIComponent(id)}`}>Review vendor</a>)}</li>)}</ul></details></>}
      <small>{summary.checkedAt && <>Form checks observed <time dateTime={summary.checkedAt}>{new Date(summary.checkedAt).toLocaleString()}</time> · </>}{summary.relationships} linked {summary.relationships === 1 ? "relationship" : "relationships"}</small>
    </>}
  </section>;
}
