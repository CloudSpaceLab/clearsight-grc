import type { RuntimeContext } from "../api";
import { AssignedRequests } from "./AssignedRequests";

/** Isolated until the underlying response commands are authorized and tested. */
export function Work({ context }: { context: RuntimeContext }) {
  return <section><h1>Work & Approvals</h1><AssignedRequests context={context}/><p>Approval work unavailable.</p></section>;
}
