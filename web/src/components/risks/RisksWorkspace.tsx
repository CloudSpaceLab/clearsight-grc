import { RiskRecord } from "./RiskRecord";
import { RiskRegister } from "./RiskRegister";
import "./risk.css";

type Props = {
  organizationName?: string;
  legalEntityName?: string;
  targetID?: string;
  onTarget: (id?: string) => void;
  onOpenProgramControl?: (programID: string, objectiveID: string) => void;
};

export function RisksWorkspace({ organizationName, legalEntityName, targetID, onTarget, onOpenProgramControl }: Props) {
  if (targetID) return <RiskRecord riskID={targetID} onBack={() => onTarget(undefined)} onOpenProgramControl={onOpenProgramControl}/>;
  return <RiskRegister organizationName={organizationName} legalEntityName={legalEntityName} onOpenRisk={(id) => onTarget(id)}/>;
}
