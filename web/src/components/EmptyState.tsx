import { PremiumIllustration } from "./PremiumIllustration";

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
  const illustrated = kind === "empty" || kind === "no-results";
  return <section className={`empty-state state-${kind}`} role={kind === "unavailable" || kind === "conflict" ? "status" : undefined}>
    {illustrated && <PremiumIllustration variant="empty"/>}
    <div>
      {label && <span className="eyebrow">{label}</span>}
      <h2>{title}</h2>
      <p>{description}</p>
      {action && <button className="primary-button" type="button" onClick={onAction}>{action}</button>}
    </div>
  </section>;
}
