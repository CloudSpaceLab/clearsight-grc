import { EmptyState as SharedEmptyState } from "./ui/EmptyState";
import { Button } from "./ui/Button";

export type StateKind = "empty" | "no-results" | "unavailable" | "forbidden" | "not-found" | "conflict";

type Props = {
  title: string;
  description: string;
  action?: string;
  label?: string;
  onAction?: () => void;
  kind?: StateKind;
};

export function EmptyState({ title, description, action, label, onAction, kind = "empty" }: Props) {
  return <SharedEmptyState population={label ?? title} title={title} description={description}
    role={kind === "unavailable" || kind === "conflict" ? "status" : undefined}
    action={action && onAction ? <Button variant="primary" onPress={onAction}>{action}</Button> : undefined}/>;
}
