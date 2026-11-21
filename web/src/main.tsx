import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { consumeCaptureInvitation } from "./captureInvitationBrowser";
import { FreshApp } from "./v210/FreshApp";
import { Capture } from "./v210/Capture";
import "./v210/fresh.css";

const invitation = consumeCaptureInvitation(window);
const root = document.getElementById("root");
if (!root) throw new Error("Application root is missing");

createRoot(root).render(<StrictMode>{invitation !== null ? <Capture invitationToken={invitation}/> : <FreshApp/>}</StrictMode>);
