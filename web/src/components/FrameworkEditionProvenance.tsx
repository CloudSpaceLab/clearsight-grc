import type { FrameworkEditionOption } from "../programOperationsApi";

type Props = { edition: FrameworkEditionOption };

function trustedSourceURL(value: string): string | null {
  try {
    const url = new URL(value);
    return url.protocol === "https:" && url.hostname && !url.username && !url.password ? url.href : null;
  } catch {
    return null;
  }
}

/** Source provenance only. A published reference is not an adopted obligation. */
export function FrameworkEditionProvenance({ edition }: Props) {
  const sourceURL = trustedSourceURL(edition.source_url);
  const rights = edition.license_basis === "CITATIONS_ONLY" ? "Citations only"
    : edition.license_basis === "SUMMARIES_PERMITTED" ? "Summaries permitted" : "Not verified";

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
      {edition.source_sha256 && <div><dt>Source SHA-256</dt><dd><code>{edition.source_sha256}</code></dd></div>}
      {edition.release_sha256 && <div><dt>Release SHA-256</dt><dd><code>{edition.release_sha256}</code></dd></div>}
    </dl>
    {sourceURL
      ? <a className="cs-program-standards__source" href={sourceURL} target="_blank" rel="noopener noreferrer">Show published source ↗</a>
      : <p className="cs-program-standards__meta">Source reference unavailable</p>}
  </>;
}
