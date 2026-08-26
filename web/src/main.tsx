import "./staticDemoBootstrap";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import { DemoAuthGate } from "./components/DemoAuthGate";
import { bootstrapExternalCapture, ExternalCaptureApp } from "./components/ExternalCaptureApp";
import { LifecycleTodayEvidencePage } from "./components/LifecycleTodayEvidencePage";
import { OperatingMutationsEvidencePage } from "./components/OperatingMutationsEvidencePage";
import { DisplayPreferencesRoot } from "./components/DisplayPreferences";
import { runtimePresentation } from "./runtimePresentation";
import "./styles.css";
import "./evidence.css";
import "./continuity.css";
import "./journeys.css";
import "./document-import.css";
import "./interventions.css";
import "./automation-policies.css";
import "./product-finish.css";
import "./ui-preferences.css";
import "./visual-review-fixes.css";
import "./capture-inputs.css";
import "./operating-mutations.css";
import "./demo-login.css";
import "./defect-review-fixes.css";
import "./monitoring.css";
import "./vendors.css";

const root = document.getElementById("root");
if (!root) throw new Error("Application root is missing");
const externalCapture = bootstrapExternalCapture(window);
const params = new URLSearchParams(window.location.search);
const presentation = runtimePresentation(window.location.search);
const fixture = params.get("fixture");
const lifecycleEvidence = import.meta.env.VITE_STATIC_DEMO === "true" && fixture === "today-lifecycle";
const operatingEvidence = import.meta.env.VITE_STATIC_DEMO === "true" && fixture === "operating-mutations";
const application = externalCapture.isExternalCapture
  ? <ExternalCaptureApp invitationToken={externalCapture.invitationToken} resumedSessionID={externalCapture.resumedSessionID}/>
  : lifecycleEvidence
    ? <LifecycleTodayEvidencePage/>
    : operatingEvidence
      ? <OperatingMutationsEvidencePage/>
      : <DemoAuthGate presentation={presentation}><App presentation={presentation}/></DemoAuthGate>;
createRoot(root).render(<StrictMode><DisplayPreferencesRoot>{application}</DisplayPreferencesRoot></StrictMode>);
