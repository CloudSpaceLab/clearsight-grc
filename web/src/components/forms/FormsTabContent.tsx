import { lazy, Suspense } from "react";
import { Button } from "../ui";
import { FormsEmptyState } from "./FormsEmptyState";
import { ResponsesView } from "./ResponsesView";
import { SentFormsView } from "./SentFormsView";

const CommunicationsView = lazy(() => import("./CommunicationsView"));
const FormPoliciesView = lazy(() => import("./FormPoliciesView"));

export type GovernedFormsTab = "Sent forms" | "Responses" | "Policies" | "Imports" | "Communications";

export function FormsTabContent({ tab, canConfigureCommunications = true }: { tab: GovernedFormsTab; canConfigureCommunications?: boolean }) {
  if (tab === "Sent forms") return <SentFormsView/>;
  if (tab === "Responses") return <ResponsesView/>;
  if (tab === "Policies") return <Suspense fallback={<div className="forms-loading" aria-live="polite" aria-busy="true">Loading response policies…</div>}><FormPoliciesView/></Suspense>;
  if (tab === "Communications") return <Suspense fallback={<div className="forms-loading" aria-live="polite" aria-busy="true">Loading communications editor…</div>}><CommunicationsView canConfigure={canConfigureCommunications}/></Suspense>;
  return <FormsEmptyState population="Imported form proposals in this workspace" title="Turn source documents into reviewable form proposals" detail="Open Imports to process source documents. Review every generated proposal before creating a form draft." actions={<Button onPress={() => { window.location.hash = "#imports"; }}>Open Imports</Button>}/>;
}
