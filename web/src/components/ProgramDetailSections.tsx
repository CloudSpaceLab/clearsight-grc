import type { ReactNode } from "react";
import type { ProgramSection } from "../appRouting";
import { Tabs } from "./ui";

export const programSections: ReadonlyArray<{ id: ProgramSection; label: string }> = [
  { id: "overview", label: "Overview" },
  { id: "requirements-controls", label: "Requirements & controls" },
  { id: "monitoring", label: "Data collection" },
  { id: "evidence-results", label: "Evidence & results" },
  { id: "issues-actions", label: "Issues & actions" },
  { id: "history", label: "History" },
];

type Props = {
  section: ProgramSection;
  panels: Record<ProgramSection, ReactNode>;
  onSectionChange: (section: ProgramSection) => void;
};

// Program navigation shares the tab, keyboard and compact-selector contract
// used by other ClearSight workspaces. The Program record owns the panels.
export function ProgramDetailSections({ section, panels, onSectionChange }: Props) {
  return <section className="program-detail-sections">
    <Tabs ariaLabel="Program sections" compactLabel="Program section"
      items={programSections} selectedKey={section} onSelectionChange={onSectionChange}>
      {(key) => <div className="program-section-panel">{panels[key]}</div>}
    </Tabs>
  </section>;
}
