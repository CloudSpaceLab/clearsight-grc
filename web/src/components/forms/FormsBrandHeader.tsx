import { useEffect, useState } from "react";
import type { ReactNode } from "react";
import { Monogram } from "../Monogram";
import { contrastTextForAccent, defaultFormsAccent, normalizeLogoURL, type FormsAppearance } from "./formsAppearance";

type Props = {
  organizationName: string;
  legalEntityName: string;
  appearance: FormsAppearance;
  onAppearanceChange: (appearance: FormsAppearance) => void;
  action?: ReactNode;
};

export function FormsBrandHeader({ organizationName, legalEntityName, appearance, onAppearanceChange, action }: Props) {
  const [logoDraft, setLogoDraft] = useState(appearance.logoURL ?? "");
  const [logoError, setLogoError] = useState("");
  const [failedLogo, setFailedLogo] = useState(false);

  useEffect(() => {
    setLogoDraft(appearance.logoURL ?? "");
    setFailedLogo(false);
  }, [appearance.logoURL]);

  function applyLogo() {
    const value = logoDraft.trim();
    if (!value) {
      setLogoError("");
      onAppearanceChange({ ...appearance, logoURL: undefined });
      return;
    }
    const normalized = normalizeLogoURL(value, window.location.href);
    if (!normalized) {
      setLogoError("Use an HTTPS logo URL or a path hosted by this ClearSight deployment.");
      return;
    }
    setLogoError("");
    onAppearanceChange({ ...appearance, logoURL: normalized });
  }

  function resetAppearance() {
    setLogoDraft("");
    setLogoError("");
    onAppearanceChange({ accentColor: defaultFormsAccent });
  }

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
        <p>Build reusable governed templates, review exact revisions, and keep approval quality visible before a form is reused.</p>
      </div>
    </div>
    <div className="forms-header-actions">
      {action}
      <details className="forms-appearance-control">
        <summary>Style workspace</summary>
        <div className="forms-appearance-popover">
          <label><span>Accent color</span><input aria-label="Workspace accent color" type="color" value={appearance.accentColor} onChange={(event) => onAppearanceChange({ ...appearance, accentColor: event.target.value })}/></label>
          <label className="forms-logo-field"><span>Bank or organization logo</span><input type="url" value={logoDraft} placeholder="https://bank.example/logo.svg" onChange={(event) => setLogoDraft(event.target.value)}/></label>
          {logoError && <small className="forms-appearance-error" role="alert">{logoError}</small>}
          <div className="forms-appearance-actions"><button type="button" onClick={applyLogo}>Apply logo</button><button className="text-button" type="button" onClick={resetAppearance}>Reset</button></div>
          <small>Workspace styling is saved only in this browser. Governed invitation and recipient branding remains separate.</small>
        </div>
      </details>
    </div>
  </header>;
}
