import "./staticDemoBootstrap";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import { ExternalCaptureApp } from "./components/ExternalCaptureApp";
import { LifecycleTodayEvidencePage } from "./components/LifecycleTodayEvidencePage";
import { DisplayPreferencesRoot } from "./components/DisplayPreferences";
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

const root = document.getElementById("root");
if (!root) throw new Error("Application root is missing");
const params = new URLSearchParams(window.location.search);
const invitationToken = params.get("capture_invite");
const lifecycleEvidence = import.meta.env.VITE_STATIC_DEMO === "true" && params.get("fixture") === "today-lifecycle";
const application = invitationToken
  ? <ExternalCaptureApp invitationToken={invitationToken}/>
  : lifecycleEvidence
    ? <LifecycleTodayEvidencePage/>
    : <App/>;
createRoot(root).render(<StrictMode><DisplayPreferencesRoot>{application}</DisplayPreferencesRoot></StrictMode>);
