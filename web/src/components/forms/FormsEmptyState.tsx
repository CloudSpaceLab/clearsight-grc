import type { ReactNode } from "react";

type Props = {
  eyebrow?: string;
  title: string;
  detail: string;
  tone?: "empty" | "search" | "unavailable" | "future";
  actions?: ReactNode;
};

export function FormsEmptyState({ eyebrow, title, detail, tone = "empty", actions }: Props) {
  return <section className={`forms-empty-state forms-empty-state-${tone}`}>
    <div className="forms-empty-graphic" aria-hidden="true">
      <svg viewBox="0 0 240 150" role="presentation">
        <rect className="forms-empty-sheet forms-empty-sheet-back" x="48" y="24" width="112" height="88" rx="12"/>
        <rect className="forms-empty-sheet" x="66" y="38" width="126" height="92" rx="12"/>
        <rect className="forms-empty-accent" x="84" y="57" width="42" height="8" rx="4"/>
        <rect className="forms-empty-line" x="84" y="76" width="86" height="6" rx="3"/>
        <rect className="forms-empty-line" x="84" y="91" width="66" height="6" rx="3"/>
        <circle className="forms-empty-node" cx="178" cy="28" r="15"/>
        <path className="forms-empty-check" d="m171 28 5 5 9-11"/>
        <circle className="forms-empty-node forms-empty-node-muted" cx="52" cy="126" r="10"/>
        <path className="forms-empty-link" d="M62 123c27-4 45-1 59 8"/>
      </svg>
    </div>
    <div className="forms-empty-copy">
      {eyebrow && <span className="forms-detail-kicker">{eyebrow}</span>}
      <h2>{title}</h2>
      <p>{detail}</p>
      {actions && <div className="forms-empty-actions">{actions}</div>}
    </div>
  </section>;
}
