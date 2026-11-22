import { useRef } from "react";
import type { ProgramAggregate } from "../types";
import { ProgramCoverage } from "./ProgramCoverage";
import { ProgramRequirements } from "./ProgramRequirements";
import "./program-workspace.css";

type Props = { aggregate: ProgramAggregate; backHref: string; issuesHref: string; onUpdated: (value: ProgramAggregate) => void };

function displayStatus(status?: string) {
  if (!status) return "Unknown";
  const words = status.toLowerCase().replaceAll("_", " ");
  return words.charAt(0).toUpperCase() + words.slice(1);
}

export function ProgramWorkspace({ aggregate, backHref, issuesHref, onUpdated }: Props) {
  const requirementRegion = useRef<HTMLElement>(null);
  const program = aggregate.program;
  function goToRequirements() {
    requirementRegion.current?.scrollIntoView({ block: "start" });
    requirementRegion.current?.focus({ preventScroll: true });
  }

  return <section className="v-programs v-program-workspace">
    <a className="v-back" href={backHref}>← Programs</a>
    <header className="v-page-heading v-program-workspace-heading"><div><p className="v-eyebrow">Program · {program.code}</p><h1>{program.name}</h1></div><span className="v-status">{displayStatus(program.status)}</span></header>
    <nav className="v-program-workspace-nav" aria-label="Program sections"><span aria-current="page">Overview</span><button type="button" onClick={goToRequirements}>Requirements</button><a href={issuesHref}>Open issues</a></nav>
    <div className="v-program-workspace-layout">
      <section ref={requirementRegion} className="v-program-workspace-main" tabIndex={-1} aria-label="Program requirements"><ProgramRequirements aggregate={aggregate} onUpdated={onUpdated}/></section>
      <aside className="v-program-workspace-aside" aria-label="Program summary"><ProgramCoverage aggregate={aggregate}/><section className="v-panel v-program-facts"><h2>Program details</h2><dl><div><dt>Function</dt><dd>{program.owning_function || "Not recorded"}</dd></div><div><dt>Jurisdiction</dt><dd>{program.jurisdiction || "Not recorded"}</dd></div><div><dt>Version</dt><dd>{Number.isSafeInteger(program.version) && program.version > 0 ? program.version : "Unknown"}</dd></div></dl></section></aside>
    </div>
  </section>;
}
