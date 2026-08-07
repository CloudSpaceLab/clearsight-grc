import "./staticDemoBootstrap";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import "./styles.css";
import "./evidence.css";
import "./continuity.css";
import "./journeys.css";
import "./document-import.css";
import "./interventions.css";
import "./automation-policies.css";
import "./product-finish.css";

const root = document.getElementById("root");
if (!root) throw new Error("Application root is missing");
createRoot(root).render(<StrictMode><App /></StrictMode>);
