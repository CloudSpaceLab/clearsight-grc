import type { ReactNode } from "react";

export type StatusTone = "neutral" | "info" | "success" | "warning" | "error" | "unknown";

export function StatusBadge({ tone = "neutral", children }: { tone?: StatusTone; children: ReactNode }) {
  return <span className={`cs-status-badge cs-tone--${tone}`}>
    <span className="cs-status-badge__marker" aria-hidden="true"/>
    {children}
  </span>;
}
