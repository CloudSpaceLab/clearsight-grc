import type { ReactNode } from "react";
import type { ProgramSection } from "../appRouting";
import { Tabs } from "./ui";

// Approved Program HTML: exactly six primary views. Data collection remains a
// separate authorized contextual workflow at the existing /monitoring route.
export const programSections: ReadonlyArray<{ id: Exclude<ProgramSection, "monitoring">; label: string }> = [
  { id: "overview", label: "Overview" },
  { id: "requirements-controls", label: "Requirements" },
  { id: "assessments", label: "Assessments" },
  { id: "evidence-results", label: "Evidence" },
  { id: "issues-actions", label: "Issues" },
  { id: "history", label: "Activity" },
];

type Props = {
  section: ProgramSection;
  panels: Record<ProgramSection, ReactNode>;
  onSectionChange: (section: ProgramSection) => void;
  lead?: ReactNode;
};

// Program navigation shares the tab, keyboard and compact-selector contract
// used by other ClearSight workspaces. The Program record owns the panels.
export function ProgramDetailSections({ section, panels, onSectionChange, lead }: Props) {
  if (section === "monitoring") {
    return <section className="program-detail-sections program-detail-sections--contextual" aria-label="Data collection workspace">
      <header className="program-contextual-header">
        <div><span className="eyebrow">Program / Data collection</span><h2>Data collection</h2></div>
        <button type="button" className="secondary-button" onClick={() => onSectionChange("overview")}>Back to Overview</button>
      </header>
      <div className="program-section-panel">{lead}{panels.monitoring}</div>
    </section>;
  }
  return <section className="program-detail-sections">
    <Tabs ariaLabel="Program sections" compactLabel="Program section"
      items={programSections} selectedKey={section} onSelectionChange={onSectionChange}>
      {(key) => <div className="program-section-panel">{lead}{panels[key]}</div>}
    </Tabs>
  </section>;
}
