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
        <rect x="48" y="24" width="112" height="88" rx="12" fill="var(--surface-2)" stroke="var(--border)" opacity=".72"/>
        <rect x="66" y="38" width="126" height="92" rx="12" fill="var(--surface)" stroke="var(--border)"/>
        <rect x="84" y="57" width="42" height="8" rx="4" fill="var(--forms-accent)"/>
        <rect x="84" y="76" width="86" height="6" rx="3" fill="var(--border)"/>
        <rect x="84" y="91" width="66" height="6" rx="3" fill="var(--border)"/>
        <circle cx="178" cy="28" r="15" fill="var(--forms-accent)"/>
        <path d="m171 28 5 5 9-11" fill="none" stroke="white" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"/>
        <circle cx="52" cy="126" r="10" fill="var(--surface-2)" stroke="var(--forms-accent)"/>
        <path d="M62 123c27-4 45-1 59 8" fill="none" stroke="var(--forms-accent)" strokeWidth="2" strokeLinecap="round" opacity=".55"/>
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
