import { lazy, Suspense, useEffect } from "react";
import { ResponsesView } from "./ResponsesView";
import { SentFormsView } from "./SentFormsView";
import { DocumentBrowser } from "../documents/DocumentBrowser";

const CommunicationsView = lazy(() => import("./CommunicationsView"));
const FormPoliciesView = lazy(() => import("./FormPoliciesView"));

export type GovernedFormsTab = "Sent forms" | "Responses" | "Documents" | "Policies" | "Imports" | "Communications";

export function FormsTabContent({ tab, canConfigureCommunications = true }: { tab: GovernedFormsTab; canConfigureCommunications?: boolean }) {
  if (tab === "Sent forms") return <SentFormsView/>;
  if (tab === "Responses") return <ResponsesView/>;
  if (tab === "Documents") return <DocumentBrowser scopeLabel="Documents submitted through forms"/>;
  if (tab === "Policies") return <Suspense fallback={<div className="forms-loading" aria-live="polite" aria-busy="true">Loading response policies…</div>}><FormPoliciesView/></Suspense>;
  if (tab === "Communications") return <Suspense fallback={<div className="forms-loading" aria-live="polite" aria-busy="true">Loading communications editor…</div>}><CommunicationsView canConfigure={canConfigureCommunications}/></Suspense>;
  return <ImportsHandoff/>;
}

function ImportsHandoff() {
  useEffect(() => { window.location.replace("#imports"); }, []);
  return <p role="status">Opening Imports…</p>;
}
