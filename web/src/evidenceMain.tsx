// Isolated sample data for rendered evidence; never loaded by the application entry.
import { installFreshEvidence } from "./v210/evidence";
installFreshEvidence();
await import("./main");
