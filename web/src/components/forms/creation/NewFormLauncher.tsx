import { useRef, useState } from "react";
import type { StarterTemplate } from "../../../formsTypes";
import { FocusedSheet } from "../../FocusedSheet";

type Props = {
  starters: StarterTemplate[];
  busy: string | null;
  onBlank: () => void;
  onAI: () => void;
  onImport: () => void;
  onUseStarter: (starter: StarterTemplate) => void;
  onClose: () => void;
};

export function NewFormLauncher({ starters, busy, onBlank, onAI, onImport, onUseStarter, onClose }: Props) {
  const galleryRef = useRef<HTMLElement>(null);
  const [previewCode, setPreviewCode] = useState<string>();

  function openTemplates() {
    const gallery = galleryRef.current;
    if (!gallery) return;
    gallery.scrollIntoView({ block: "start" });
    gallery.querySelector<HTMLButtonElement>("[data-template-action]")?.focus();
  }

  return <FocusedSheet
    label="New form"
    closeLabel="Close new form"
    panelClassName="forms-new-form-sheet"
    backdropClassName="forms-new-form-backdrop"
    onClose={onClose}
  >
    <div className="forms-new-form-content">
      <header className="forms-new-form-heading">
        <span className="eyebrow">New form</span>
        <h2>Create a form</h2>
        <p>Start clean, use a proven pattern, describe what you need, or bring in an existing source.</p>
      </header>

      <div className="forms-creation-methods" aria-label="Form creation methods">
        <CreationMethod
          icon="blank"
          title="Blank form"
          detail="Start clean"
          onClick={onBlank}
        />
        <CreationMethod
          icon="ai"
          title="Draft with AI"
          detail="Describe what you need"
          onClick={onAI}
        />
        <CreationMethod
          icon="template"
          title="From template"
          detail="Proven patterns"
          onClick={openTemplates}
        />
        <CreationMethod
          icon="import"
          title="Import"
          detail="Use an existing source"
          onClick={onImport}
        />
      </div>

      <section ref={galleryRef} className="forms-starter-gallery" aria-labelledby="starter-gallery-title">
        <div className="forms-starter-gallery-heading">
          <div>
            <span className="eyebrow">Proven patterns</span>
            <h3 id="starter-gallery-title">Starter templates</h3>
          </div>
          <span>{starters.length ? `${starters.length} available` : "None available"}</span>
        </div>

        {starters.length ? <div className="forms-starter-grid">
          {starters.map((starter) => {
            const expanded = previewCode === starter.code;
            return <article className="forms-starter-card" key={`${starter.code}:${starter.catalog_version}`}>
              <div className="forms-starter-card-copy">
                <div className="forms-starter-card-title">
                  <strong>{starter.template.name}</strong>
                  <span>{starter.template.sections.length} sections · {starter.template.fields.length} questions</span>
                </div>
                <p>{starter.template.purpose || starter.reference_label}</p>
                {starter.template.tags?.length ? <div className="forms-starter-tags">
                  {starter.template.tags.slice(0, 3).map((tag) => <span key={tag}>{tag}</span>)}
                </div> : null}
              </div>

              <StarterPreview starter={starter} expanded={expanded}/>

              <div className="forms-starter-actions">
                <button
                  type="button"
                  className="text-button"
                  aria-expanded={expanded}
                  onClick={() => setPreviewCode((current) => current === starter.code ? undefined : starter.code)}
                >
                  {expanded ? "Hide preview" : "Preview"}
                </button>
                <button
                  type="button"
                  className="forms-primary"
                  data-template-action
                  disabled={busy !== null}
                  onClick={() => onUseStarter(starter)}
                >
                  {busy === `starter:${starter.code}` ? "Creating…" : "Use template"}
                </button>
              </div>
            </article>;
          })}
        </div> : <div className="forms-starter-empty">
          <strong>No starter templates are available in this workspace.</strong>
          <span>You can still start with a blank form, AI proposal, or import.</span>
        </div>}
      </section>
    </div>
  </FocusedSheet>;
}

function CreationMethod({ icon, title, detail, onClick }: { icon: CreationIcon; title: string; detail: string; onClick: () => void }) {
  return <button type="button" className="forms-creation-method" onClick={onClick}>
    <CreationIconGraphic type={icon}/>
    <span>
      <strong>{title}</strong>
      <small>{detail}</small>
    </span>
    <span className="forms-creation-arrow" aria-hidden="true">›</span>
  </button>;
}

type CreationIcon = "blank" | "ai" | "template" | "import";

function CreationIconGraphic({ type }: { type: CreationIcon }) {
  const common = { fill: "none", stroke: "currentColor", strokeWidth: 1.7, strokeLinecap: "round" as const, strokeLinejoin: "round" as const };
  return <span className="forms-creation-icon" aria-hidden="true">
    <svg viewBox="0 0 24 24">
      {type === "blank" && <><path {...common} d="M6 3.8h8.5L18 7.3v12.9H6z"/><path {...common} d="M14.5 3.8v3.5H18M9 12h6M12 9v6"/></>}
      {type === "ai" && <><path {...common} d="M5 5.5h14v10H9l-4 3z"/><path {...common} d="M8.5 9.1h7M8.5 12h4.5"/></>}
      {type === "template" && <><rect {...common} x="4" y="4" width="7" height="7" rx="1.2"/><rect {...common} x="13" y="4" width="7" height="7" rx="1.2"/><rect {...common} x="4" y="13" width="7" height="7" rx="1.2"/><rect {...common} x="13" y="13" width="7" height="7" rx="1.2"/></>}
      {type === "import" && <><path {...common} d="M12 4v11M8.5 11.5 12 15l3.5-3.5"/><path {...common} d="M5 18.5h14"/></>}
    </svg>
  </span>;
}

function StarterPreview({ starter, expanded }: { starter: StarterTemplate; expanded: boolean }) {
  const fields = starter.template.fields.slice(0, expanded ? 6 : 3);
  return <div className={expanded ? "forms-starter-preview expanded" : "forms-starter-preview"} aria-label={`${starter.template.name} preview`}>
    <div className="forms-starter-preview-header">
      <span>{starter.template.sections[0]?.title || "Form"}</span>
      <small>Respondent preview</small>
    </div>
    <div className="forms-starter-preview-fields">
      {fields.map((field) => <div className="forms-starter-preview-field" key={field.id}>
        <span>{field.label || "Untitled question"}</span>
        <small>{field.required ? "Required · " : ""}{fieldTypeLabel(field.type)}</small>
      </div>)}
      {starter.template.fields.length > fields.length && <span className="forms-starter-preview-more">+{starter.template.fields.length - fields.length} more questions</span>}
    </div>
  </div>;
}

function fieldTypeLabel(value: string) {
  return value.replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
