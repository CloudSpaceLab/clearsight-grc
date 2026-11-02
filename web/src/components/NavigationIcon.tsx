import type { View } from "../appRouting";

export function NavigationIcon({ view }: { view: View | "frameworks" }) {
  const common = { width: 20, height: 20, viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: 1.7, strokeLinecap: "round" as const, strokeLinejoin: "round" as const, "aria-hidden": true };
  if (view === "today") return <svg {...common}><path d="M4 5h16v14H4z"/><path d="M8 3v4M16 3v4M7 11h4M7 15h7"/></svg>;
  if (view === "oversight") return <svg {...common}><path d="m3 10 9-7 9 7M5 9v12h5v-7h4v7h5V9"/></svg>;
  if (view === "frameworks") return <svg {...common}><path d="m3 7 9-4 9 4-9 4-9-4Zm0 5 9 4 9-4M3 17l9 4 9-4"/></svg>;
  if (view === "insights") return <svg {...common}><path d="M4 19V5M4 19h16"/><path d="m7 15 4-5 3 2 4-6"/><circle cx="7" cy="15" r="1"/><circle cx="11" cy="10" r="1"/><circle cx="14" cy="12" r="1"/><circle cx="18" cy="6" r="1"/></svg>;
  if (view === "reports") return <svg {...common}><path d="M4 3v18h17M8 16v-5M13 16V7M18 16v-3"/></svg>;
  if (view === "programs") return <svg {...common}><rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/></svg>;
  if (view === "forms") return <svg {...common}><path d="M6 3h9l3 3v15H6z"/><path d="M14 3v4h4M9 11h6M9 15h6M9 19h4"/></svg>;
  if (view === "vendors") return <svg {...common}><path d="M4 20V7l8-4 8 4v13"/><path d="M8 10h2M14 10h2M8 14h2M14 14h2M10 20v-3h4v3"/></svg>;
  if (view === "risks") return <svg {...common}><path d="m12 3 8 3v6c0 5-8 9-8 9s-8-4-8-9V6l8-3Z"/><path d="M12 8v5m0 3v.01"/></svg>;
  if (view === "rcsa") return <svg {...common}><rect x="5" y="4" width="14" height="17" rx="2"/><path d="M9 4V2h6v2M8 10l2 2 4-4M8 16h8"/></svg>;
  if (view === "losses") return <svg {...common}><path d="M4 4v16h16M7 8l5 5 3-3 5 6M16 16h4v-4"/></svg>;
  if (view === "ropa") return <svg {...common}><rect x="5" y="10" width="14" height="11" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3M12 14v3"/></svg>;
  if (view === "people") return <svg {...common}><circle cx="12" cy="7" r="4"/><path d="M4 21v-2a8 8 0 0 1 16 0v2"/></svg>;
  if (view === "work") return <svg {...common}><path d="M9 5h6l1 3h4v11H4V8h4z"/><path d="M8 13h8"/></svg>;
  if (view === "imports") return <svg {...common}><path d="M12 3v12M8 7l4-4 4 4"/><path d="M5 14v6h14v-6"/></svg>;
  if (view === "explore") return <svg {...common}><circle cx="12" cy="12" r="9"/><path d="m15 9-2 5-5 2 2-5z"/></svg>;
  return <svg {...common}><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1-2.8 2.8-.1-.1a1.7 1.7 0 0 0-1.9-.3 1.7 1.7 0 0 0-1 1.6V21h-4v-.1a1.7 1.7 0 0 0-1-1.6 1.7 1.7 0 0 0-1.9.3l-.1.1L4.2 17l.1-.1a1.7 1.7 0 0 0 .3-1.9A1.7 1.7 0 0 0 3 14H3v-4h.1a1.7 1.7 0 0 0 1.6-1 1.7 1.7 0 0 0-.3-1.9L4.2 7 7 4.2l.1.1A1.7 1.7 0 0 0 9 4.6a1.7 1.7 0 0 0 1-1.6V3h4v.1a1.7 1.7 0 0 0 1 1.6 1.7 1.7 0 0 0 1.9-.3l.1-.1L19.8 7l-.1.1a1.7 1.7 0 0 0-.3 1.9 1.7 1.7 0 0 0 1.6 1h.1v4H21a1.7 1.7 0 0 0-1.6 1z"/></svg>;
}
