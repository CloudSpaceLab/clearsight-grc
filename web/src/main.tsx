import { lazy, StrictMode, Suspense } from "react";
import { createRoot } from "react-dom/client";
import { consumeCaptureInvitation } from "./captureInvitationBrowser";
import { ExternalCaptureApp } from "./components/ExternalCaptureApp";
import { DisplayPreferencesRoot } from "./components/DisplayPreferences";
import { SessionGate } from "./components/SessionGate";
import { runtimePresentation } from "./runtimePresentation";
import "./appStyles";

const App = lazy(() => import("./App"));

bootstrapApplication();

function bootstrapApplication() {
  const invitationToken = consumeCaptureInvitation(window);
  const root = document.getElementById("root");
  if (!root) throw new Error("Application root is missing");

  const presentation = runtimePresentation(window.location.search);
  const application = invitationToken !== null
    ? <DisplayPreferencesRoot><ExternalCaptureApp invitationToken={invitationToken}/></DisplayPreferencesRoot>
    : <SessionGate presentation={presentation}><DisplayPreferencesRoot><Suspense fallback={<p role="status">Loading the ClearSight workspace…</p>}><App presentation={presentation}/></Suspense></DisplayPreferencesRoot></SessionGate>;

  createRoot(root).render(<StrictMode>{application}</StrictMode>);
}
