import { useId } from "react";

type Props = { variant?: "guided" | "empty" | "readiness" | "routing"; className?: string };

export function PremiumIllustration({ variant = "guided", className = "" }: Props) {
  const uid = useId().replace(/:/g, "");
  const accent = variant === "readiness" ? "var(--green)" : variant === "routing" ? "var(--violet)" : "var(--cyan)";
  const title = variant === "empty" ? "No results illustration" : variant === "routing" ? "Approval route illustration" : variant === "readiness" ? "Readiness illustration" : "Workspace guide illustration";
  return (
    <svg className={`premium-illustration ${className}`} viewBox="0 0 520 320" role="img" aria-labelledby={`${uid}-title ${uid}-description`}>
      <title id={`${uid}-title`}>{title}</title>
      <desc id={`${uid}-description`}>Abstract diagram of related work and a completed review checkpoint.</desc>
      <defs>
        <linearGradient id={`${uid}-panel`} x1="0" y1="0" x2="1" y2="1"><stop stopColor="var(--illustration-panel-start)"/><stop offset="1" stopColor="var(--illustration-panel-end)"/></linearGradient>
        <radialGradient id={`${uid}-orb`}><stop stopColor={accent} stopOpacity=".72"/><stop offset="1" stopColor={accent} stopOpacity="0"/></radialGradient>
        <filter id={`${uid}-blur`}><feGaussianBlur stdDeviation="22"/></filter>
        <filter id={`${uid}-shadow`}><feDropShadow dx="0" dy="18" stdDeviation="20" floodColor="var(--illustration-shadow)" floodOpacity=".28"/></filter>
      </defs>
      <ellipse cx="290" cy="278" rx="175" ry="22" fill="var(--illustration-ground)" opacity=".55"/>
      <circle cx="394" cy="80" r="82" fill={`url(#${uid}-orb)`} filter={`url(#${uid}-blur)`}/>
      <g filter={`url(#${uid}-shadow)`}>
        <path d="M106 74c0-18 15-33 33-33h216c18 0 33 15 33 33v158c0 18-15 33-33 33H139c-18 0-33-15-33-33V74Z" fill={`url(#${uid}-panel)`} stroke="var(--illustration-border)"/>
        <path d="M132 88h150" stroke="var(--illustration-muted)" strokeWidth="7" strokeLinecap="round" opacity=".34"/>
        <path d="M132 115h92" stroke={accent} strokeWidth="5" strokeLinecap="round" opacity=".78"/>
        <rect x="132" y="145" width="230" height="84" rx="18" fill="var(--illustration-inner)" stroke="var(--illustration-border)"/>
        <circle cx="169" cy="187" r="18" fill={accent} opacity=".15" stroke={accent}/>
        <path d="m160 187 6 6 13-15" fill="none" stroke={accent} strokeWidth="4" strokeLinecap="round" strokeLinejoin="round"/>
        <path d="M201 173h117M201 195h83" stroke="var(--illustration-muted)" strokeWidth="6" strokeLinecap="round" opacity=".45"/>
      </g>
      <g>
        <circle cx="91" cy="222" r="29" fill="var(--illustration-node)" stroke="var(--illustration-node-border)"/>
        <circle cx="431" cy="181" r="29" fill="var(--illustration-node)" stroke="var(--illustration-node-border)"/>
        <circle cx="412" cy="274" r="24" fill="var(--illustration-node)" stroke="var(--illustration-node-border)"/>
        <path d="M118 211c64-55 182-62 285-37M118 233c88 28 197 41 270 42" fill="none" stroke={accent} strokeOpacity=".36" strokeWidth="2" strokeDasharray="5 7"/>
        <circle cx="91" cy="222" r="7" fill={accent}/><circle cx="431" cy="181" r="7" fill={accent}/><circle cx="412" cy="274" r="7" fill={accent}/>
      </g>
      {variant === "empty" && <path d="M224 258h58" stroke={accent} strokeWidth="5" strokeLinecap="round" opacity=".8"/>}
    </svg>
  );
}
