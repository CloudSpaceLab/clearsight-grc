import { useId } from "react";

export type CinematicGuideVariant = "today" | "vendors";

type Props = {
  variant: CinematicGuideVariant;
  role: string;
  title: string;
  description: string;
  busy?: boolean;
  onStart: () => void;
  onSkip: () => void | Promise<void>;
};

const stages = {
  today: ["Source context", "Assigned work", "Review and authority", "Confirmed outcome"],
  vendors: ["Vendor register", "Collect missing facts", "Review exceptions", "Request vendor action", "Confirm the outcome"],
} satisfies Record<CinematicGuideVariant, string[]>;

export function CinematicGuidePanel({ variant, role, title, description, busy = false, onStart, onSkip }: Props) {
  const titleID = useId();
  const descriptionID = useId();
  const isVendors = variant === "vendors";
  const workspace = isVendors ? "Vendors" : "Today";

  return <aside className={`cinematic-guide cinematic-guide--${variant}`} aria-label={`${isVendors ? "Vendor" : "Today"} guide`}>
    <div className="cinematic-guide__visual">
      <svg
        className="cinematic-guide__scene"
        viewBox="0 0 720 320"
        role="img"
        aria-labelledby={`${titleID} ${descriptionID}`}
        preserveAspectRatio="xMidYMid meet"
      >
        <title id={titleID}>{isVendors ? "Vendor relationship path" : "Today work path"}</title>
        <desc id={descriptionID}>{isVendors
          ? "A vendor relationship moves from the register through missing-fact collection, exception review and requested action to outcome confirmation."
          : "Current source context leads to assigned work, the required review or authority and a confirmed outcome."}</desc>
        <rect className="cinematic-guide__scene-frame" x="1" y="1" width="718" height="318" rx="28"/>
        <g className="cinematic-guide__scene-layer">
          <path className="cinematic-guide__rail" d={isVendors ? "M72 220 C180 96 272 258 370 145 S555 86 650 190" : "M78 218 C204 92 332 244 452 134 S594 108 650 168"}/>
          <path className="cinematic-guide__rail cinematic-guide__rail--quiet" d={isVendors ? "M54 246 C178 140 280 280 390 180 S560 122 676 218" : "M58 244 C198 134 344 276 468 170 S598 148 676 196"}/>
          <circle className="cinematic-guide__orbit" cx="360" cy="160" r="116"/>
          <circle className="cinematic-guide__orbit cinematic-guide__orbit--inner" cx="360" cy="160" r="82"/>
        </g>
        {isVendors ? <VendorScene/> : <TodayScene/>}
      </svg>
    </div>

    <div className="cinematic-guide__content">
      <span className="eyebrow">Guide for {role}</span>
      <h2>{title}</h2>
      <p>{description}</p>
      <ol className="cinematic-guide__sequence" aria-label={`${workspace} guide sequence`}>
        {stages[variant].map((stage, index) => <li key={stage}>
          <span aria-hidden="true">{String(index + 1).padStart(2, "0")}</span>
          <strong>{stage}</strong>
        </li>)}
      </ol>
      <div className="cinematic-guide__actions">
        <button className="primary-button" type="button" onClick={onStart} disabled={busy}>Start guide</button>
        <button className="text-button" type="button" onClick={() => void onSkip()} disabled={busy}>Skip for now</button>
      </div>
    </div>
  </aside>;
}

function TodayScene() {
  return <g className="cinematic-guide__scene-focus">
    <g transform="translate(62 174)">
      <rect className="cinematic-guide__node" width="112" height="82" rx="16"/>
      <path className="cinematic-guide__line cinematic-guide__line--cyan" d="M22 26h68M22 42h48M22 58h58"/>
      <circle className="cinematic-guide__accent cinematic-guide__accent--cyan" cx="88" cy="58" r="7"/>
    </g>
    <g transform="translate(232 88)">
      <rect className="cinematic-guide__node" width="116" height="92" rx="18"/>
      <path className="cinematic-guide__line cinematic-guide__line--amber" d="M24 29h68M24 47h52M24 65h38"/>
      <path className="cinematic-guide__accent cinematic-guide__accent--amber" d="M88 58l10 10 18-24"/>
    </g>
    <g transform="translate(400 128)">
      <path className="cinematic-guide__node" d="M58 0 110 22v42c0 36-23 58-52 72C29 122 6 100 6 64V22Z"/>
      <circle className="cinematic-guide__accent cinematic-guide__accent--violet" cx="58" cy="52" r="22"/>
      <path className="cinematic-guide__line" d="M47 52h22M58 41v22"/>
    </g>
    <g transform="translate(572 82)">
      <circle className="cinematic-guide__node" cx="62" cy="62" r="58"/>
      <circle className="cinematic-guide__accent cinematic-guide__accent--green" cx="62" cy="62" r="34"/>
      <path className="cinematic-guide__line cinematic-guide__line--strong" d="m45 62 12 12 24-30"/>
    </g>
  </g>;
}

function VendorScene() {
  return <g className="cinematic-guide__scene-focus">
    <g transform="translate(42 184)">
      <rect className="cinematic-guide__node" width="104" height="76" rx="15"/>
      <path className="cinematic-guide__line cinematic-guide__line--cyan" d="M20 24h62M20 40h42M20 56h54"/>
    </g>
    <g transform="translate(168 96)">
      <rect className="cinematic-guide__node" width="108" height="86" rx="17"/>
      <circle className="cinematic-guide__accent cinematic-guide__accent--amber" cx="30" cy="30" r="8"/>
      <path className="cinematic-guide__line" d="M48 30h38M22 54h64M22 68h44"/>
    </g>
    <g transform="translate(302 152)">
      <path className="cinematic-guide__node" d="M54 0 104 20v38c0 34-22 55-50 68C26 113 4 92 4 58V20Z"/>
      <path className="cinematic-guide__accent cinematic-guide__accent--coral" d="M54 32v38M54 84v2"/>
    </g>
    <g transform="translate(444 76)">
      <rect className="cinematic-guide__node" width="110" height="92" rx="18"/>
      <path className="cinematic-guide__line cinematic-guide__line--violet" d="M22 28h66M22 46h50M22 64h60"/>
      <path className="cinematic-guide__accent cinematic-guide__accent--violet" d="m77 64 11 11 17-22"/>
    </g>
    <g transform="translate(584 142)">
      <circle className="cinematic-guide__node" cx="54" cy="54" r="50"/>
      <circle className="cinematic-guide__accent cinematic-guide__accent--green" cx="54" cy="54" r="29"/>
      <path className="cinematic-guide__line cinematic-guide__line--strong" d="m39 54 11 11 20-25"/>
    </g>
  </g>;
}
