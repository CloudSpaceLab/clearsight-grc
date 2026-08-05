import { PremiumIllustration } from "./PremiumIllustration";
export function EmptyState({ title, description, action, onAction }: { title: string; description: string; action?: string; onAction?: () => void }) {
  return <section className="empty-state"><PremiumIllustration variant="empty"/><div><span className="eyebrow">Continuously prepared</span><h2>{title}</h2><p>{description}</p>{action && <button className="primary-button" onClick={onAction}>{action}</button>}</div></section>;
}
