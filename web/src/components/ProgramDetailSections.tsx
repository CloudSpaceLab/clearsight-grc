import { useEffect, useRef, useState } from "react";
import type { KeyboardEvent, ReactNode } from "react";
import type { ProgramSection } from "../appRouting";

export const programSections: ReadonlyArray<{ id: ProgramSection; label: string }> = [
  { id: "overview", label: "Overview" },
  { id: "requirements-controls", label: "Requirements & controls" },
  { id: "monitoring", label: "Monitoring" },
  { id: "evidence-results", label: "Evidence & results" },
  { id: "issues-actions", label: "Issues & actions" },
  { id: "history", label: "History" },
];

type Props = {
  section: ProgramSection;
  panels: Record<ProgramSection, ReactNode>;
  onSectionChange: (section: ProgramSection) => void;
  compact?: boolean;
};

function useCompactProgramNavigation(override?: boolean) {
  const query = "(max-width: 720px)";
  const [matches, setMatches] = useState(() => override ?? (typeof window.matchMedia === "function" && window.matchMedia(query).matches));
  useEffect(() => {
    if (override !== undefined || typeof window.matchMedia !== "function") return;
    const media = window.matchMedia(query);
    const update = () => setMatches(media.matches);
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, [override]);
  return override ?? matches;
}

export function ProgramDetailSections({ section, panels, onSectionChange, compact }: Props) {
  const isCompact = useCompactProgramNavigation(compact);
  const tabRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const panelHeading = useRef<HTMLHeadingElement | null>(null);
  const focusPanelAfterChange = useRef(false);
  const selectedIndex = programSections.findIndex((item) => item.id === section);
  const selected = programSections[selectedIndex] ?? programSections[0]!;

  useEffect(() => {
    if (!focusPanelAfterChange.current) return;
    focusPanelAfterChange.current = false;
    panelHeading.current?.focus();
  }, [section]);

  function selectTab(index: number) {
    const next = programSections[(index + programSections.length) % programSections.length]!;
    onSectionChange(next.id);
    tabRefs.current[(index + programSections.length) % programSections.length]?.focus();
  }

  function onTabKeyDown(event: KeyboardEvent<HTMLButtonElement>, index: number) {
    let next: number | undefined;
    if (event.key === "ArrowRight") next = index + 1;
    if (event.key === "ArrowLeft") next = index - 1;
    if (event.key === "Home") next = 0;
    if (event.key === "End") next = programSections.length - 1;
    if (next === undefined) return;
    event.preventDefault();
    selectTab(next);
  }

  return <section className="program-detail-sections">
    {isCompact ? <label className="program-section-selector"><span>Program section</span><select value={section} onChange={(event) => { focusPanelAfterChange.current = true; onSectionChange(event.target.value as ProgramSection); }}>{programSections.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}</select></label> : <div className="program-section-tabs" role="tablist" aria-label="Program sections">{programSections.map((item, index) => <button
      ref={(node) => { tabRefs.current[index] = node; }} id={`program-tab-${item.id}`} key={item.id} type="button" role="tab"
      aria-selected={item.id === section} aria-controls={`program-panel-${item.id}`} tabIndex={item.id === section ? 0 : -1}
      onClick={() => onSectionChange(item.id)} onKeyDown={(event) => onTabKeyDown(event, index)}
    >{item.label}</button>)}</div>}
    <div className="program-section-panel" id={`program-panel-${selected.id}`} role="tabpanel" aria-label={isCompact ? selected.label : undefined} aria-labelledby={isCompact ? undefined : `program-tab-${selected.id}`}>
      <h3 ref={panelHeading} tabIndex={-1}>{selected.label}</h3>
      {panels[selected.id]}
    </div>
  </section>;
}
