import { useEffect, useState } from "react";
import type { ReactNode } from "react";
import { Monogram } from "../Monogram";
import { contrastTextForAccent, type FormsAppearance } from "./formsAppearance";

type Props = {
  organizationName: string;
  legalEntityName: string;
  appearance: FormsAppearance;
  action?: ReactNode;
};

export function FormsBrandHeader({ organizationName, legalEntityName, appearance, action }: Props) {
  const [failedLogo, setFailedLogo] = useState(false);

  useEffect(() => setFailedLogo(false), [appearance.logoURL]);

  const showLogo = Boolean(appearance.logoURL && !failedLogo);
  return <header className="forms-header forms-brand-header">
    <div className="forms-brand-context">
      <div className="forms-brand-mark" style={{ color: contrastTextForAccent(appearance.accentColor) }}>
        {showLogo
          ? <img src={appearance.logoURL} alt={`${organizationName} logo`} onError={() => setFailedLogo(true)}/>
          : <Monogram name={organizationName} decorative className="forms-brand-monogram"/>}
      </div>
      <div>
        <span className="eyebrow">{organizationName} · {legalEntityName}</span>
        <h1 id="forms-title">Forms</h1>
        <p>Create, send and govern information requests.</p>
      </div>
    </div>
    {action && <div className="forms-header-actions">{action}</div>}
  </header>;
}
