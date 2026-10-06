import type { MatterAggregate } from "../../types";
import { matterContextKind } from "../../matterPresentation";
import { IndicatorMatterContext } from "./IndicatorMatterContext";
import { OperationalLossMatterContext } from "./OperationalLossMatterContext";

type Props = {
  aggregate: MatterAggregate;
  onOpenLoss?: (lossID: string) => void;
  onOpenIndicator?: (indicatorID: string, kind?: "KRI" | "KCI") => void;
};

export function MatterDomainContext({ aggregate, onOpenLoss, onOpenIndicator }: Props) {
  const kind = matterContextKind(aggregate.matter);
  if (kind === "OPERATIONAL_LOSS") {
    return <OperationalLossMatterContext
      matterID={aggregate.matter.id}
      lossID={aggregate.matter.source_id!}
      onOpenLoss={onOpenLoss}
    />;
  }
  if (kind === "INDICATOR") {
    return <IndicatorMatterContext
      resultID={aggregate.matter.source_id!}
      onOpenIndicator={onOpenIndicator}
    />;
  }
  return null;
}
