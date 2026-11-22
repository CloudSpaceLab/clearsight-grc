import { createRoot } from "react-dom/client";
import { PrivacyWorkspace } from "../src/v210/PrivacyWorkspace";
import "../src/v210/fresh.css";
createRoot(document.getElementById("root")!).render(<main className="v-main"><p className="v-sample">Sample data</p><PrivacyWorkspace id="activity-1"/></main>);
