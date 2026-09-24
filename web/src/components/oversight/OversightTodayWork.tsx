import { TodayInterventions } from "../TodayInterventions";
import type { AttentionItem, Readiness } from "../../types";

type ConnectionState = "loading" | "live" | "unavailable";
type ReadinessState = "loading" | "live" | "unavailable";

type Props = {
  items: AttentionItem[];
  connection: ConnectionState;
  readiness: Readiness | null;
  readinessState: ReadinessState;
  onOpenItem: (item: AttentionItem) => void;
  onInspectAuthority: (item: AttentionItem) => void;
};

export function OversightTodayWork(props: Props) {
  return <section aria-label="Assigned work"><TodayInterventions {...props}/></section>;
}
