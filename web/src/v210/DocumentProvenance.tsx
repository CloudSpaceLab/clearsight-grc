import type { DocumentOccurrence } from "../submittedDocumentApi";

function person(id?: string) {
  return id?.trim() ? `Person ID: ${id}` : "Not recorded";
}

function moment(value?: string) {
  if (!value || /^0001-01-01T00:00:00(?:\.0+)?Z$/.test(value)) return "Not recorded";
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return "Date unavailable";
  return <time dateTime={value}>{new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit", timeZone: "UTC", timeZoneName: "short" }).format(date)}</time>;
}

function reviewLabel(value: string) {
  const known: Record<string, string> = { SUBMITTED: "Awaiting review", VALIDATED: "Validated", REJECTED: "Rejected", EXPIRED: "Expired", SUPERSEDED: "Superseded" };
  return known[value] ?? "Status unavailable";
}

export function DocumentProvenance({ file }: { file: DocumentOccurrence }) {
  return <>
    <div><dt>Uploaded</dt><dd>{moment(file.uploaded_at)}</dd></div>
    <div><dt>Uploaded by</dt><dd>{person(file.uploaded_by)}</dd></div>
    <div><dt>Submitted</dt><dd>{moment(file.submitted_at)}</dd></div>
    <div><dt>Submitted by</dt><dd>{person(file.submitted_by)}</dd></div>
    {file.review && <>
      <div><dt>Assessment document review</dt><dd>{reviewLabel(file.review.status)}</dd></div>
      <div><dt>Reviewed</dt><dd>{moment(file.review.reviewed_at)}</dd></div>
      <div><dt>Reviewed by</dt><dd>{person(file.review.reviewed_by)}</dd></div>
    </>}
  </>;
}
