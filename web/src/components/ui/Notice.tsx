import type { ReactNode } from "react";
import type { StatusTone } from "./StatusBadge";

type NoticeTone = Exclude<StatusTone, "neutral" | "unknown">;

export function Notice({ tone = "info", children }: { tone?: NoticeTone; children: ReactNode }) {
  return <div className={`cs-notice cs-tone--${tone}`} role={tone === "error" ? "alert" : "status"}>{children}</div>;
}
