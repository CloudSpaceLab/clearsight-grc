import { lazy, StrictMode, Suspense } from "react";
import { createRoot } from "react-dom/client";
import { consumeCaptureInvitation } from "./captureInvitationBrowser";
import { ExternalCaptureApp } from "./components/ExternalCaptureApp";
import { DisplayPreferencesRoot } from "./components/DisplayPreferences";
import { SessionGate } from "./components/SessionGate";
import { runtimePresentation } from "./runtimePresentation";
import "./styles.css";
import "./cinematic-guide.css";
import "./evidence.css";
import "./continuity.css";
import "./journeys.css";
import "./document-handoff.css";
import "./interventions.css";
import "./ai-governance.css";
import "./product-finish.css";
import "./design-system/index.css";
import "./ui-preferences.css";
import "./visual-review-fixes.css";
import "./capture-inputs.css";
import "./capture-access.css";
import "./operating-mutations.css";
import "./demo-login.css";
import "./defect-review-fixes.css";
import "./enterprise-shell.css";

const App = lazy(() => import("./App"));

bootstrapApplication();

function bootstrapApplication() {
  const invitationToken = consumeCaptureInvitation(window);
  const root = document.getElementById("root");
  if (!root) throw new Error("Application root is missing");

  const presentation = runtimePresentation(window.location.search);
  const application = invitationToken !== null
    ? <ExternalCaptureApp invitationToken={invitationToken}/>
    : <SessionGate presentation={presentation}><Suspense fallback={<p role="status">Loading the ClearSight workspace…</p>}><App presentation={presentation}/></Suspense></SessionGate>;

  createRoot(root).render(<StrictMode><DisplayPreferencesRoot>{application}</DisplayPreferencesRoot></StrictMode>);
}
