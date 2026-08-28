import { lazy, Suspense } from "react";
import { FormsEmptyState } from "./FormsEmptyState";
import { ResponsesView } from "./ResponsesView";
import { SentFormsView } from "./SentFormsView";

const CommunicationsView = lazy(() => import("./CommunicationsView"));

export type GovernedFormsTab = "Sent forms" | "Responses" | "Imports" | "Communications";

export function FormsTabContent({ tab }: { tab: GovernedFormsTab }) {
  if (tab === "Sent forms") return <SentFormsView/>;
  if (tab === "Responses") return <ResponsesView/>;
  if (tab === "Communications") return <Suspense fallback={<div className="forms-loading" aria-live="polite" aria-busy="true">Loading communications editor…</div>}><CommunicationsView/></Suspense>;
  return <FormsEmptyState eyebrow="Imports" tone="future" title="Turn source documents into reviewable form proposals" detail="Imports remains the authoritative document intake path. Form proposals appear only after governed extraction; extracted content is never silently activated." actions={<button type="button" onClick={() => { window.location.hash = "#imports"; }}>Open Imports</button>}/>;
}
