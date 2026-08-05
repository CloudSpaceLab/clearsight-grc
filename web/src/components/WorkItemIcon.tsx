type Props = { type: string };

const common = {
  width: 20,
  height: 20,
  viewBox: "0 0 24 24",
  fill: "none",
  stroke: "currentColor",
  strokeWidth: 1.8,
  strokeLinecap: "round" as const,
  strokeLinejoin: "round" as const,
  "aria-hidden": true,
};

export function WorkItemIcon({ type }: Props) {
  if (type === "REGULATORY_CHANGE") {
    return <svg {...common}><path d="M6 3.8h9l3 3V20H6z"/><path d="M15 3.8V7h3M9 11h6M9 15h6"/></svg>;
  }
  if (type.includes("EVIDENCE")) {
    return <svg {...common}><path d="M5 4h14v16H5z"/><path d="m8 12 2.2 2.2L16 8.5"/></svg>;
  }
  if (type.includes("APPROVAL") || type.includes("AUTHORITY")) {
    return <svg {...common}><path d="M12 3 5.5 5.8v5.1c0 4.2 2.8 7.8 6.5 9.1 3.7-1.3 6.5-4.9 6.5-9.1V5.8z"/><path d="m9 11.5 2 2 4-4"/></svg>;
  }
  return <svg {...common}><path d="M8 4h8M9 3v3M15 3v3M6 5h12v15H6z"/><path d="M9 10h6M9 14h4"/></svg>;
}
