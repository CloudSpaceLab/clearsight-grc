import { PremiumIllustration } from "./PremiumIllustration";

type Props = {
  title: string;
  description: string;
  action?: string;
  label?: string;
  onAction?: () => void;
};

export function EmptyState({ title, description, action, label, onAction }: Props) {
  return <section className="empty-state">
    <PremiumIllustration variant="empty"/>
    <div>
      {label && <span className="eyebrow">{label}</span>}
      <h2>{title}</h2>
      <p>{description}</p>
      {action && <button className="primary-button" onClick={onAction}>{action}</button>}
    </div>
  </section>;
}
