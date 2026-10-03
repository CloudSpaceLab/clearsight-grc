import { LossRecord } from "./LossRecord";
import { LossRegister } from "./LossRegister";
import "./loss.css";

type Props = {
  organizationName?: string;
  legalEntityName?: string;
  targetID?: string;
  onTarget: (id?: string) => void;
  onOpenRisk?: (riskID: string) => void;
  onOpenMatter?: (matterID: string) => void;
};

export function LossesWorkspace({ organizationName, legalEntityName, targetID, onTarget, onOpenRisk, onOpenMatter }: Props) {
  if (targetID) return <LossRecord lossID={targetID} onBack={() => onTarget(undefined)} onOpenRisk={onOpenRisk} onOpenMatter={onOpenMatter}/>;
  return <LossRegister organizationName={organizationName} legalEntityName={legalEntityName} onOpenLoss={(id) => onTarget(id)}/>;
}
