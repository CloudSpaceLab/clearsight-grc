// Isolated visual evidence only; this entry is never loaded by the customer app.
import { createRoot } from "react-dom/client";
import { installFreshEvidence } from "../src/v210/evidence";
import { PrivacyWorkspace } from "../src/v210/PrivacyWorkspace";
import "../src/v210/fresh.css";

installFreshEvidence();
createRoot(document.getElementById("root")!).render(<main className="v-main"><p className="v-sample">Sample data</p><PrivacyWorkspace /></main>);
