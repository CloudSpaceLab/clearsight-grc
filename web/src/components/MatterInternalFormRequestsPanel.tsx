import { useEffect, useState } from "react";
import { loadFormTemplatePage } from "../formsApi";
import type { FormLibraryItem } from "../formsTypes";
import { FocusedSheet } from "./FocusedSheet";
import { DistributionComposer } from "./forms/DistributionComposer";
import { SubjectFormActivity } from "./forms/SubjectFormActivity";
import { StatusPill } from "./forms/dashboard/TemplateLibraryTable";
import { ActionCard, ActionLink, Button, Notice } from "./ui";

type Props = {
  matterID: string;
  matterReference: string;
};

type LinkedFormsState = "loading" | "live" | "unavailable";
const linkedFormsPageSize = 6;

export function MatterInternalFormRequestsPanel({ matterID, matterReference }: Props) {
  const [requestOpen, setRequestOpen] = useState(false);
  const [notice, setNotice] = useState("");
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
      <h2 id="matter-form-requests-title">Form requests</h2>
      <p>Request an approved form for <strong>{matterReference}</strong> and review the responses.</p>
    </div>

    <div className="matter-forms__actions" role="group" aria-label="Form actions">
      <ActionCard title="Use existing form" description="Choose an approved form and send a response request."
        icon={<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="m3 12 18-9-7 18-3-8-8-1Z"/><path d="m11 13 10-10"/></svg>}
        onPress={() => { setNotice(""); setRequestOpen(true); }}
      />
    </div>

    {notice && <Notice tone="success">{notice}</Notice>}

    <div className="matter-forms__activity-heading">
      <h3>Activity</h3>
      <p>Issue activity</p>
    </div>

    <SubjectFormActivity subjectType="MATTER" subjectID={matterID} subjectLabel={matterReference} variant="cards" refreshKey={activityRefreshKey} leading={
    <div className="subject-form-activity__group subject-form-activity__card" aria-labelledby="linked-forms-title">
      <header>
        <h3 id="linked-forms-title">Forms created for this issue</h3>
        {linkedFormsState === "live" && linkedForms.length > 0 && <span>{linkedForms.length} shown</span>}
      </header>
      {linkedFormsState === "loading" && <p role="status">Loading linked forms…</p>}
      {linkedFormsState === "unavailable" && <Notice tone="warning">
        Linked forms are unavailable. Other issue work remains available. <Button variant="secondary" size="compact" onPress={() => setLinkedFormsReload((value) => value + 1)}>Retry linked forms</Button>
      </Notice>}
      
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

    {requestOpen && <FocusedSheet label="Use existing form" closeLabel="Close form request" size="wide" onClose={() => setRequestOpen(false)}>
      <div className="cs-sheet-heading">
        <span className="eyebrow">Form request</span>
        <h2>Send existing form</h2>
        <p>Choose the form, recipient and response deadline for {matterReference}.</p>
      </div>
      <DistributionComposer
        subject={{ type: "MATTER", id: matterID, label: matterReference }}
        internalOnly
        onCancel={() => setRequestOpen(false)}
        onCreated={() => {
          setRequestOpen(false);
          setNotice("Form request created.");
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
