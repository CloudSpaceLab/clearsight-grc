import "./staticDemoBootstrap";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import { DemoAuthGate } from "./components/DemoAuthGate";
import { ExternalCaptureApp } from "./components/ExternalCaptureApp";
import { LifecycleTodayEvidencePage } from "./components/LifecycleTodayEvidencePage";
import { OperatingMutationsEvidencePage } from "./components/OperatingMutationsEvidencePage";
import { DisplayPreferencesRoot } from "./components/DisplayPreferences";
import { runtimePresentation } from "./runtimePresentation";
import "./styles.css";
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
import "./operating-mutations.css";
import "./demo-login.css";
import "./defect-review-fixes.css";
import "./monitoring.css";

const root = document.getElementById("root");
if (!root) throw new Error("Application root is missing");
const params = new URLSearchParams(window.location.search);
const presentation = runtimePresentation(window.location.search);
const invitationToken = params.get("capture_invite");
const fixture = params.get("fixture");
const lifecycleEvidence = import.meta.env.VITE_STATIC_DEMO === "true" && fixture === "today-lifecycle";
const operatingEvidence = import.meta.env.VITE_STATIC_DEMO === "true" && fixture === "operating-mutations";
const application = invitationToken
  ? <ExternalCaptureApp invitationToken={invitationToken}/>
  : lifecycleEvidence
    ? <LifecycleTodayEvidencePage/>
    : operatingEvidence
      ? <OperatingMutationsEvidencePage/>
      : <DemoAuthGate presentation={presentation}><App presentation={presentation}/></DemoAuthGate>;
createRoot(root).render(<StrictMode><DisplayPreferencesRoot>{application}</DisplayPreferencesRoot></StrictMode>);
