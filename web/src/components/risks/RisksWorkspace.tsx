import { RiskRecord } from "./RiskRecord";
import { RiskRegister } from "./RiskRegister";
import "./risk.css";

type Props = {
  organizationName?: string;
  legalEntityName?: string;
  organizationScopeID?: string;
  organizationScopeName?: string;
  actorID?: string;
  targetID?: string;
  onTarget: (id?: string) => void;
  onOpenProgram?: (programID: string) => void;
  onOpenMatter?: (matterID: string) => void;
  onOpenProgramControl?: (programID: string, objectiveID: string) => void;
};

export function RisksWorkspace({ organizationName, legalEntityName, organizationScopeID, organizationScopeName, actorID, targetID, onTarget, onOpenProgram, onOpenMatter, onOpenProgramControl }: Props) {
  if (targetID) return <RiskRecord riskID={targetID} actorID={actorID} onBack={() => onTarget(undefined)} onOpenProgram={onOpenProgram} onOpenMatter={onOpenMatter} onOpenProgramControl={onOpenProgramControl}/>;
  return <RiskRegister organizationName={organizationName} legalEntityName={legalEntityName} organizationScopeID={organizationScopeID} organizationScopeName={organizationScopeName} onOpenRisk={(id) => onTarget(id)}/>;
}
