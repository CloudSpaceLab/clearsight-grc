import { useEffect, useState } from "react";
import { createLibraryFormDraft, loadFormTemplatePage } from "../formsApi";
import type { FormLibraryItem } from "../formsTypes";
import { FocusedSheet } from "./FocusedSheet";
import { FormBuilder } from "./FormBuilder";
import { DistributionComposer } from "./forms/DistributionComposer";
import { SubjectFormActivity } from "./forms/SubjectFormActivity";
import { StatusPill } from "./forms/dashboard/TemplateLibraryTable";
import { ActionLink, Button, Notice } from "./ui";

type Props = {
  matterID: string;
  matterReference: string;
};

type LinkedFormsState = "loading" | "live" | "unavailable";
const linkedFormsPageSize = 6;

export function MatterInternalFormRequestsPanel({ matterID, matterReference }: Props) {
  const [requestOpen, setRequestOpen] = useState(false);
  const [authorOpen, setAuthorOpen] = useState(false);
  const [notice, setNotice] = useState("");
  const [createdDraft, setCreatedDraft] = useState<{ id: string }>();
  const [linkedFormsState, setLinkedFormsState] = useState<LinkedFormsState>("loading");
  const [linkedForms, setLinkedForms] = useState<FormLibraryItem[]>([]);
  const [linkedFormsCursor, setLinkedFormsCursor] = useState<string>();
  const [linkedFormsReload, setLinkedFormsReload] = useState(0);
  const [linkedFormsPageError, setLinkedFormsPageError] = useState("");
  const [loadingMoreLinkedForms, setLoadingMoreLinkedForms] = useState(false);

  useEffect(() => {
    const controller = new AbortController();
    setLinkedFormsState("loading");
    setLinkedForms([]);
    setLinkedFormsCursor(undefined);
    setLinkedFormsPageError("");
    void loadFormTemplatePage(
      { origin_type: "MATTER", origin_id: matterID, limit: linkedFormsPageSize },
      controller.signal,
    ).then((page) => {
      if (controller.signal.aborted) return;
      setLinkedForms(page.items);
      setLinkedFormsCursor(page.next_cursor);
      setLinkedFormsState("live");
    }).catch(() => {
      if (!controller.signal.aborted) setLinkedFormsState("unavailable");
    });
    return () => controller.abort();
  }, [linkedFormsReload, matterID]);

  async function loadMoreLinkedForms() {
    if (!linkedFormsCursor || loadingMoreLinkedForms) return;
    const cursor = linkedFormsCursor;
    setLoadingMoreLinkedForms(true);
    setLinkedFormsPageError("");
    try {
      const page = await loadFormTemplatePage({
        origin_type: "MATTER",
        origin_id: matterID,
        cursor,
        limit: linkedFormsPageSize,
      });
      setLinkedForms((current) => appendUniqueFormItems(current, page.items));
      setLinkedFormsCursor(page.next_cursor);
    } catch {
      setLinkedFormsPageError("More linked forms could not be loaded.");
    } finally {
      setLoadingMoreLinkedForms(false);
    }
  }

  return <section className="matter-record-panel" aria-labelledby="matter-form-requests-title">
    <div className="section-heading-row">
      <div>
        <span className="eyebrow">Forms</span>
        <h2 id="matter-form-requests-title">Forms and requests</h2>
        <p>Create a form for {matterReference} or request an approved form from an employee.</p>
      </div>
      <div className="matter-panel-actions">
        <Button type="button" variant="secondary" onPress={() => { setNotice(""); setCreatedDraft(undefined); setAuthorOpen(true); }}>Create linked form</Button>
        <Button type="button" variant="secondary" onPress={() => { setNotice(""); setCreatedDraft(undefined); setRequestOpen(true); }}>Request employee form</Button>
      </div>
    </div>

    {notice && <Notice tone="success">{notice}{createdDraft && <> <ActionLink href={`#forms/${encodeURIComponent(createdDraft.id)}`}>Open form draft</ActionLink></>}</Notice>}

    <div className="subject-form-activity__group" aria-labelledby="linked-forms-title">
      <header>
        <h3 id="linked-forms-title">Linked forms</h3>
        {linkedFormsState === "live" && <span>{linkedForms.length} shown</span>}
      </header>
      {linkedFormsState === "loading" && <p role="status">Loading linked forms…</p>}
      {linkedFormsState === "unavailable" && <Notice tone="warning">
        Linked forms are unavailable. Other issue work remains available. <Button variant="secondary" size="compact" onPress={() => setLinkedFormsReload((value) => value + 1)}>Retry linked forms</Button>
      </Notice>}
      {linkedFormsState === "live" && linkedForms.length === 0 && <p>No linked forms recorded.</p>}
      {linkedForms.length > 0 && <ul>{linkedForms.map((item) => <li key={item.template.id}>
        <div>
          <strong>{item.template.name}</strong>
          <span>Revision {item.template.version}</span>
        </div>
        <div className="subject-form-activity__actions">
          <StatusPill status={item.template.status}/>
          <ActionLink href={`#forms/${encodeURIComponent(item.template.id)}`}>Open form</ActionLink>
        </div>
      </li>)}</ul>}
      {linkedFormsPageError && <Notice tone="warning">
        {linkedFormsPageError} <Button variant="secondary" size="compact" isLoading={loadingMoreLinkedForms} onPress={() => void loadMoreLinkedForms()}>Retry linked forms</Button>
      </Notice>}
      {linkedFormsCursor && !linkedFormsPageError && <Button variant="secondary" size="compact" isLoading={loadingMoreLinkedForms} onPress={() => void loadMoreLinkedForms()}>Load more linked forms</Button>}
    </div>

    <SubjectFormActivity subjectType="MATTER" subjectID={matterID} subjectLabel={matterReference}/>

    {authorOpen && <FocusedSheet label="Create linked form" closeLabel="Close form builder" size="wide" onClose={() => setAuthorOpen(false)}>
      <div className="cs-sheet-heading">
        <span className="eyebrow">Issue form</span>
        <h2>Create linked form</h2>
        <p>Create a form for {matterReference}. Approval is required before sending.</p>
      </div>
      <FormBuilder
        saveDraft={(input) => createLibraryFormDraft({ ...input, origin: { type: "MATTER", id: matterID } })}
        onSaved={(form) => {
          setAuthorOpen(false);
          setCreatedDraft(form);
          setNotice("Form draft created.");
          setLinkedFormsReload((value) => value + 1);
        }}
        onCancel={() => setAuthorOpen(false)}
        allowIncompleteComplianceDraft
      />
    </FocusedSheet>}

    {requestOpen && <FocusedSheet label="Request employee form" closeLabel="Close employee form request" size="wide" onClose={() => setRequestOpen(false)}>
      <div className="cs-sheet-heading">
        <span className="eyebrow">Employee form request</span>
        <h2>Request employee form</h2>
        <p>Choose the approved form, employee and deadline for {matterReference}.</p>
      </div>
      <DistributionComposer
        subject={{ type: "MATTER", id: matterID, label: matterReference }}
        internalOnly
        onCancel={() => setRequestOpen(false)}
        onCreated={() => {
          setRequestOpen(false);
          setNotice("Employee form request created.");
        }}
      />
    </FocusedSheet>}
  </section>;
}

function appendUniqueFormItems(current: FormLibraryItem[], incoming: FormLibraryItem[]) {
  if (incoming.length === 0) return current;
  const seen = new Set(current.map((item) => item.template.id));
  return [...current, ...incoming.filter((item) => !seen.has(item.template.id))];
}
