import { useState } from "react";
import { LossEntryDialog } from "./LossAuthoringDialog";
import { LossRecord } from "./LossRecord";
import { LossRegister } from "./LossRegister";
import "./loss.css";

type Props = {
  organizationName?: string;
  legalEntityName?: string;
  organizationScopeID?: string;
  organizationScopeName?: string;
  targetID?: string;
  onTarget: (id?: string) => void;
  onOpenRisk?: (riskID: string) => void;
  onOpenMatter?: (matterID: string) => void;
};

export function LossesWorkspace({
  organizationName,
  legalEntityName,
  organizationScopeID,
  organizationScopeName,
  targetID,
  onTarget,
  onOpenRisk,
  onOpenMatter,
}: Props) {
  const [recording, setRecording] = useState(false);

  if (targetID) return <LossRecord
    lossID={targetID}
    organizationScopeID={organizationScopeID}
    organizationScopeName={organizationScopeName}
    onBack={() => onTarget(undefined)}
    onOpenRisk={onOpenRisk}
    onOpenMatter={onOpenMatter}
  />;

  return <>
    <LossRegister
      organizationName={organizationName}
      legalEntityName={legalEntityName}
      organizationScopeID={organizationScopeID}
      organizationScopeName={organizationScopeName}
      onOpenLoss={(id) => onTarget(id)}
      onRecordLoss={() => setRecording(true)}
    />
    {recording && <LossEntryDialog
      organizationScopeID={organizationScopeID}
      organizationScopeName={organizationScopeName}
      legalEntityName={legalEntityName}
      onClose={() => setRecording(false)}
      onCreated={(loss) => {
        setRecording(false);
        onTarget(loss.id);
      }}
    />}
  </>;
}
