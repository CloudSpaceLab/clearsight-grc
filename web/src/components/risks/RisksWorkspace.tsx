import { RiskRecord } from "./RiskRecord";
import { RiskRegister } from "./RiskRegister";
import "./risk.css";

type Props = {
  organizationName?: string;
  legalEntityName?: string;
  targetID?: string;
  onTarget: (id?: string) => void;
};

export function RisksWorkspace({ organizationName, legalEntityName, targetID, onTarget }: Props) {
  if (targetID) return <RiskRecord riskID={targetID} onBack={() => onTarget(undefined)}/>;
  return <RiskRegister organizationName={organizationName} legalEntityName={legalEntityName} onOpenRisk={(id) => onTarget(id)}/>;
}
