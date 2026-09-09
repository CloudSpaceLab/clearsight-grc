import { lazy, StrictMode, Suspense } from "react";
import { createRoot } from "react-dom/client";
import "./staticDemoBootstrap";
import { PolicyAssessmentEvidence } from "./PolicyAssessmentEvidence";
import { FieldAssessmentEvidencePage, installVendorAssessmentEvidence } from "./vendorAssessmentEvidence";
import { installVendorCollectionEvidence } from "./vendorCollectionEvidence";
import { installUITruthEvidence, UITruthEvidencePage } from "./uiTruthEvidence";
import { VendorCaptureEvidencePage, type VendorCaptureEvidenceState } from "./vendorCaptureEvidence";
import { VendorReleaseEvidencePage } from "./vendorReleaseEvidence";
import { consumeCaptureInvitation } from "./captureInvitationBrowser";
import { ExternalCaptureApp } from "./components/ExternalCaptureApp";
import { LifecycleTodayEvidencePage } from "./components/LifecycleTodayEvidencePage";
import { OperatingMutationsEvidencePage } from "./components/OperatingMutationsEvidencePage";
import { OversightEvidencePage } from "./components/oversight/OversightEvidencePage";
import { DisplayPreferencesRoot } from "./components/DisplayPreferences";
import { SessionGate } from "./components/SessionGate";
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

const invitationToken = consumeCaptureInvitation(window);
const root = document.getElementById("root");
if (!root) throw new Error("Application root is missing");

const fixture = new URLSearchParams(window.location.search).get("fixture");
installVendorAssessmentEvidence();
installVendorCollectionEvidence();
installUITruthEvidence();
const application = invitationToken !== null
  ? <ExternalCaptureApp invitationToken={invitationToken}/>
  : fixture?.startsWith("vendor-release-") ? <VendorReleaseEvidencePage state={fixture.replace("vendor-release-", "")}/>
  : fixture?.startsWith("ui-truth-") ? <UITruthEvidencePage state={fixture.replace("ui-truth-", "")}/>
  : fixture?.startsWith("vendor-capture-") ? <VendorCaptureEvidencePage state={fixture.replace("vendor-capture-", "") as VendorCaptureEvidenceState}/>
  : fixture === "field-assessment-policy" ? <PolicyAssessmentEvidence/>
  : fixture === "field-assessment-builder" || fixture === "field-assessment-review"
    ? <FieldAssessmentEvidencePage review={fixture === "field-assessment-review"}/>
  : fixture === "oversight"
    ? <OversightEvidencePage/>
    : fixture === "ui-component-gallery"
      ? <Suspense fallback={<p role="status">Loading the sample component gallery…</p>}><UIComponentGallery/></Suspense>
      : fixture === "today-lifecycle"
        ? <LifecycleTodayEvidencePage/>
        : fixture === "operating-mutations"
          ? <OperatingMutationsEvidencePage/>
          : <SessionGate presentation="demo"><Suspense fallback={<p role="status">Loading the sample ClearSight workspace…</p>}><App presentation="demo"/></Suspense></SessionGate>;

createRoot(root).render(<StrictMode><DisplayPreferencesRoot>{application}</DisplayPreferencesRoot></StrictMode>);
