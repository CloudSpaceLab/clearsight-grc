import { useEffect, useState } from "react";
import { createLibraryFormDraft, loadFormTemplatePage } from "../formsApi";
import type { FormLibraryItem } from "../formsTypes";
import { FocusedSheet } from "./FocusedSheet";
import { FormBuilder } from "./FormBuilder";
import { DistributionComposer } from "./forms/DistributionComposer";
import { SubjectFormActivity } from "./forms/SubjectFormActivity";
import { StatusPill } from "./forms/dashboard/TemplateLibraryTable";
import { ActionCard, ActionLink, Button, EmptyState, Notice } from "./ui";

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
  const [activityRefreshKey, setActivityRefreshKey] = useState(0);
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

  return <section className="matter-record-panel matter-forms" aria-labelledby="matter-form-requests-title">
    <div className="matter-forms__heading">
      <span className="eyebrow">Forms</span>
      <h2 id="matter-form-requests-title">Forms and requests</h2>
      <p>Collect information for <strong>{matterReference}</strong> and review submitted evidence.</p>
    </div>

    <div className="matter-forms__actions" role="group" aria-label="Form actions">
      <ActionCard title="Request employee form" description="Send an approved form to an employee."
        icon={<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="m3 12 18-9-7 18-3-8-8-1Z"/><path d="m11 13 10-10"/></svg>}
        onPress={() => { setNotice(""); setCreatedDraft(undefined); setRequestOpen(true); }}
      />
      <ActionCard title="Create linked form" description="Build an issue-specific form. Approval is required before sending."
        icon={<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="M12 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8"/><path d="M16 3v6m-3-3h6"/></svg>}
        onPress={() => { setNotice(""); setCreatedDraft(undefined); setAuthorOpen(true); }}
      />
    </div>

    {notice && <Notice tone="success">{notice}{createdDraft && <> <ActionLink href={`#forms/${encodeURIComponent(createdDraft.id)}`}>Open form draft</ActionLink></>}</Notice>}

    <div className="matter-forms__activity-heading">
      <h3>Activity</h3>
      <p>Forms, requests and responses for this issue</p>
    </div>

    <SubjectFormActivity subjectType="MATTER" subjectID={matterID} subjectLabel={matterReference} variant="cards" refreshKey={activityRefreshKey} leading={
    <div className="subject-form-activity__group subject-form-activity__card" aria-labelledby="linked-forms-title">
      <header>
        <h3 id="linked-forms-title">Linked forms</h3>
        {linkedFormsState === "live" && linkedForms.length > 0 && <span>{linkedForms.length} shown</span>}
      </header>
      {linkedFormsState === "loading" && <p role="status">Loading linked forms…</p>}
      {linkedFormsState === "unavailable" && <Notice tone="warning">
        Linked forms are unavailable. Other issue work remains available. <Button variant="secondary" size="compact" onPress={() => setLinkedFormsReload((value) => value + 1)}>Retry linked forms</Button>
      </Notice>}
      {linkedFormsState === "live" && linkedForms.length === 0 && <EmptyState compact population="No linked forms yet" title="No linked forms yet" description="Build an issue-specific form to collect information."/>}
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
    }/>

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
          setActivityRefreshKey((value) => value + 1);
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
