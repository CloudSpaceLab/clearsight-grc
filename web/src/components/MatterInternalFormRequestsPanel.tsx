import { useState } from "react";
import { FocusedSheet } from "./FocusedSheet";
import { DistributionComposer } from "./forms/DistributionComposer";
import { Button, Notice } from "./ui";

type Props = {
  matterID: string;
  matterReference: string;
};

export function MatterInternalFormRequestsPanel({ matterID, matterReference }: Props) {
  const [open, setOpen] = useState(false);
  const [notice, setNotice] = useState(false);

  return <section className="matter-record-panel" aria-labelledby="matter-internal-form-requests-title">
    <div className="section-heading-row">
      <div>
        <span className="eyebrow">Internal collection</span>
        <h2 id="matter-internal-form-requests-title">Employee form request</h2>
        <p>Collect information or evidence from an employee for {matterReference}.</p>
      </div>
      <Button type="button" variant="secondary" onPress={() => { setNotice(false); setOpen(true); }}>Request employee form</Button>
    </div>
    {notice && <Notice tone="success">Employee form request created.</Notice>}
    {open && <FocusedSheet label="Request employee form" closeLabel="Close employee form request" size="wide" onClose={() => setOpen(false)}>
      <div className="cs-sheet-heading">
        <span className="eyebrow">Employee form request</span>
        <h2>Request employee form</h2>
        <p>Choose the approved form, employee and deadline for {matterReference}.</p>
      </div>
      <DistributionComposer
        subject={{ type: "MATTER", id: matterID, label: matterReference }}
        internalOnly
        onCancel={() => setOpen(false)}
        onCreated={() => {
          setOpen(false);
          setNotice(true);
        }}
      />
    </FocusedSheet>}
  </section>;
}
