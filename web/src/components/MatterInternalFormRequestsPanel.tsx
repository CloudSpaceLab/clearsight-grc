import { useState } from "react";
import { createLibraryFormDraft } from "../formsApi";
import type { FormTemplate } from "../monitoringTypes";
import { FocusedSheet } from "./FocusedSheet";
import { FormBuilder } from "./FormBuilder";
import { DistributionComposer } from "./forms/DistributionComposer";
import { SubjectFormActivity } from "./forms/SubjectFormActivity";
import { ActionLink, Button, Notice } from "./ui";

type Props = {
  matterID: string;
  matterReference: string;
};

export function MatterInternalFormRequestsPanel({ matterID, matterReference }: Props) {
  const [requestOpen, setRequestOpen] = useState(false);
  const [authorOpen, setAuthorOpen] = useState(false);
  const [notice, setNotice] = useState("");
  const [createdDraft, setCreatedDraft] = useState<FormTemplate>();

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
