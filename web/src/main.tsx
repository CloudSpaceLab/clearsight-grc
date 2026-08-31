import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import { consumeCaptureInvitation, purgeLegacyCaptureSession } from "./captureInvitationBrowser";
import { ExternalCaptureApp } from "./components/ExternalCaptureApp";
import { LifecycleTodayEvidencePage } from "./components/LifecycleTodayEvidencePage";
import { OperatingMutationsEvidencePage } from "./components/OperatingMutationsEvidencePage";
import { DisplayPreferencesRoot } from "./components/DisplayPreferences";
import { SessionGate } from "./components/SessionGate";
import { runtimePresentation } from "./runtimePresentation";
import "./styles.css";
import "./cinematic-guide.css";
import "./evidence.css";
import "./continuity.css";
import "./journeys.css";
import "./document-import.css";
import "./document-handoff.css";
import "./interventions.css";
import "./automation-policies.css";
import "./ai-governance.css";
import "./product-finish.css";
import "./ui-preferences.css";
import "./visual-review-fixes.css";
import "./capture-inputs.css";
import "./capture-access.css";
import "./operating-mutations.css";
import "./matter-record.css";
import "./program-record.css";
import "./demo-login.css";
import "./defect-review-fixes.css";
import "./monitoring.css";
import "./forms-foundation.css";
import "./vendors.css";
import "./configure-workspace.css";
import "./enterprise-shell.css";

void bootstrapApplication();

async function bootstrapApplication() {
  const staticDemo = import.meta.env.VITE_STATIC_DEMO === "true";
  if (staticDemo) await import("./staticDemoBootstrap");

  purgeLegacyCaptureSession(sessionStorage);
  const invitationToken = consumeCaptureInvitation(window);
  const root = document.getElementById("root");
  if (!root) throw new Error("Application root is missing");

  const params = new URLSearchParams(window.location.search);
  const presentation = staticDemo ? "demo" : runtimePresentation(window.location.search);
  const fixture = params.get("fixture");
  const lifecycleEvidence = staticDemo && fixture === "today-lifecycle";
  const operatingEvidence = staticDemo && fixture === "operating-mutations";
  const application = invitationToken !== null
    ? <ExternalCaptureApp invitationToken={invitationToken}/>
    : lifecycleEvidence
      ? <LifecycleTodayEvidencePage/>
      : operatingEvidence
        ? <OperatingMutationsEvidencePage/>
        : <SessionGate presentation={presentation}><App presentation={presentation}/></SessionGate>;

  createRoot(root).render(<StrictMode><DisplayPreferencesRoot>{application}</DisplayPreferencesRoot></StrictMode>);
}
