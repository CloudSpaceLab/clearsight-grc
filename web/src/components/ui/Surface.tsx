import type { ReactNode } from "react";

export function Surface({ children }: { children: ReactNode }) {
  return <div className="cs-surface">{children}</div>;
}

export function Card({ children }: { children: ReactNode }) {
  return <article className="cs-card">{children}</article>;
}
