import type { FrameworkEditionOption } from "../programOperationsApi";

type Props = { edition: FrameworkEditionOption };

export function trustedFrameworkSourceURL(value: string): string | null {
  try {
    const url = new URL(value);
    return url.protocol === "https:" && url.hostname && !url.username && !url.password ? url.href : null;
  } catch {
    return null;
  }
}

export function frameworkEditionRightsLabel(basis: string): string {
  return basis === "CITATIONS_ONLY" ? "Citations only"
    : basis === "SUMMARIES_PERMITTED" ? "Summaries permitted" : "Not verified";
}

// Proposal and approval share one source/rights gate. A published metadata row
// alone must never become an actionable Program standard.
export function frameworkEditionHasVerifiableSource(edition: FrameworkEditionOption): boolean {
  return Boolean(trustedFrameworkSourceURL(edition.source_url)) &&
    (edition.license_basis === "CITATIONS_ONLY" || edition.license_basis === "SUMMARIES_PERMITTED");
}

/** Source provenance only. A published reference is not an adopted obligation. */
export function FrameworkEditionProvenance({ edition }: Props) {
  const sourceURL = trustedFrameworkSourceURL(edition.source_url);
  const rights = frameworkEditionRightsLabel(edition.license_basis);

  return <>
    <dl className="cs-program-standards__facts">
      <div><dt>Publisher</dt><dd>{edition.publisher_code}</dd></div>
      <div><dt>Standard</dt><dd>{edition.framework_code}</dd></div>
      <div><dt>Edition</dt><dd>{edition.edition_code}</dd></div>
      <div><dt>Jurisdiction</dt><dd>{edition.jurisdiction}</dd></div>
      <div><dt>Published</dt><dd>{edition.source_published_on}</dd></div>
      <div><dt>Source rights</dt><dd>{rights}</dd></div>
      {edition.effective_on && <div><dt>Effective</dt><dd>{edition.effective_on}</dd></div>}
      {edition.license_reference && <div><dt>Rights reference</dt><dd>{edition.license_reference}</dd></div>}
      {edition.approval_reference && <div><dt>Release approval</dt><dd>{edition.approval_reference}</dd></div>}
    </dl>
    {(edition.source_sha256 || edition.release_sha256) && <details className="cs-framework-provenance__checksums">
      <summary>Source checksums</summary>
      <dl className="cs-program-standards__facts">
        {edition.source_sha256 && <div><dt>Source SHA-256</dt><dd><code>{edition.source_sha256}</code></dd></div>}
        {edition.release_sha256 && <div><dt>Release SHA-256</dt><dd><code>{edition.release_sha256}</code></dd></div>}
      </dl>
    </details>}
    {sourceURL
      ? <a className="cs-program-standards__source" href={sourceURL} target="_blank" rel="noopener noreferrer">Show published source ↗</a>
      : <p className="cs-program-standards__meta">Source reference unavailable</p>}
  </>;
}
