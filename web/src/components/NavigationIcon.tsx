import type { View } from "../appRouting";

export function NavigationIcon({ view }: { view: View }) {
  const common = { width: 20, height: 20, viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: 1.7, strokeLinecap: "round" as const, strokeLinejoin: "round" as const, "aria-hidden": true };
  if (view === "today") return <svg {...common}><path d="M4 5h16v14H4z"/><path d="M8 3v4M16 3v4M7 11h4M7 15h7"/></svg>;
  if (view === "programs") return <svg {...common}><path d="M5 4h14v16H5z"/><path d="M8 8h8M8 12h8M8 16h5"/></svg>;
  if (view === "forms") return <svg {...common}><path d="M6 3h9l3 3v15H6z"/><path d="M14 3v4h4M9 11h6M9 15h6M9 19h4"/></svg>;
  if (view === "vendors") return <svg {...common}><path d="M4 20V7l8-4 8 4v13"/><path d="M8 10h2M14 10h2M8 14h2M14 14h2M10 20v-3h4v3"/></svg>;
  if (view === "work") return <svg {...common}><path d="M9 5h6l1 3h4v11H4V8h4z"/><path d="M8 13h8"/></svg>;
  if (view === "imports") return <svg {...common}><path d="M12 3v12M8 7l4-4 4 4"/><path d="M5 14v6h14v-6"/></svg>;
  if (view === "explore") return <svg {...common}><circle cx="12" cy="12" r="9"/><path d="m15 9-2 5-5 2 2-5z"/></svg>;
  return <svg {...common}><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1-2.8 2.8-.1-.1a1.7 1.7 0 0 0-1.9-.3 1.7 1.7 0 0 0-1 1.6V21h-4v-.1a1.7 1.7 0 0 0-1-1.6 1.7 1.7 0 0 0-1.9.3l-.1.1L4.2 17l.1-.1a1.7 1.7 0 0 0 .3-1.9A1.7 1.7 0 0 0 3 14H3v-4h.1a1.7 1.7 0 0 0 1.6-1 1.7 1.7 0 0 0-.3-1.9L4.2 7 7 4.2l.1.1A1.7 1.7 0 0 0 9 4.6a1.7 1.7 0 0 0 1-1.6V3h4v.1a1.7 1.7 0 0 0 1 1.6 1.7 1.7 0 0 0 1.9-.3l.1-.1L19.8 7l-.1.1a1.7 1.7 0 0 0-.3 1.9 1.7 1.7 0 0 0 1.6 1h.1v4H21a1.7 1.7 0 0 0-1.6 1z"/></svg>;
}
