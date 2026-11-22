import { createRoot } from "react-dom/client";
import { NotificationPreferences } from "../src/v210/NotificationPreferences";
import "../src/v210/fresh.css";
createRoot(document.getElementById("root")!).render(<main className="v-main"><p className="v-sample">Sample data</p><NotificationPreferences scope={{tenant:"bank",actor:"person"}}/></main>);
