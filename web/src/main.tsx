import { lazy, StrictMode, Suspense } from "react";
import { createRoot } from "react-dom/client";
import { consumeCaptureInvitation, purgeLegacyCaptureSession } from "./captureInvitationBrowser";
import { ExternalCaptureApp } from "./components/ExternalCaptureApp";
import { LifecycleTodayEvidencePage } from "./components/LifecycleTodayEvidencePage";
import { OperatingMutationsEvidencePage } from "./components/OperatingMutationsEvidencePage";
import { OversightEvidencePage } from "./components/oversight/OversightEvidencePage";
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

const UIComponentGallery = lazy(() => import("./components/ui-gallery/UIComponentGallery").then((module) => ({ default: module.UIComponentGallery })));
const App = lazy(() => import("./App"));

void bootstrapApplication();

async function bootstrapApplication() {
  const staticDemo = import.meta.env.VITE_STATIC_DEMO === "true" && import.meta.env.VITE_UI_EVIDENCE === "true";
  if (staticDemo) await import("./staticDemoBootstrap");

  purgeLegacyCaptureSession(sessionStorage);
  const invitationToken = consumeCaptureInvitation(window);
  const root = document.getElementById("root");
  if (!root) throw new Error("Application root is missing");

  const params = new URLSearchParams(window.location.search);
  const presentation = staticDemo ? "demo" : runtimePresentation(window.location.search);
  const fixture = params.get("fixture");
  const uiGalleryEvidence = staticDemo && fixture === "ui-component-gallery";
  const lifecycleEvidence = staticDemo && fixture === "today-lifecycle";
  const operatingEvidence = staticDemo && fixture === "operating-mutations";
  const oversightEvidence = staticDemo && fixture === "oversight";
  const application = invitationToken !== null
    ? <ExternalCaptureApp invitationToken={invitationToken}/>
    : oversightEvidence
      ? <OversightEvidencePage/>
      : uiGalleryEvidence
        ? <Suspense fallback={<p role="status">Loading the sample component gallery…</p>}><UIComponentGallery/></Suspense>
        : lifecycleEvidence
          ? <LifecycleTodayEvidencePage/>
          : operatingEvidence
            ? <OperatingMutationsEvidencePage/>
            : <SessionGate presentation={presentation}><Suspense fallback={<p role="status">Loading the ClearSight workspace…</p>}><App presentation={presentation}/></Suspense></SessionGate>;

  createRoot(root).render(<StrictMode><DisplayPreferencesRoot>{application}</DisplayPreferencesRoot></StrictMode>);
}
