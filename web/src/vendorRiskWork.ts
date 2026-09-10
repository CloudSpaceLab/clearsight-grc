import { loadMatter } from "./api";
import { loadVendorRelationshipLinks } from "./vendorLinkApi";
import type { MatterAggregate, MatterAction } from "./types";

export type VendorRiskFinding = { relationshipIDs: string[]; record: MatterAggregate };
export type VendorRiskWork = { items: VendorRiskFinding[]; complete: boolean; checkedAt: string };
const findingTypes = new Set(["VENDOR_DEFICIENCY", "AUDIT_FINDING", "SUPERVISORY_FINDING", "CONTROL_GAP", "FAILED_VERIFICATION"]);
export function openFinding(item: VendorRiskFinding) { return !["CLOSED", "CANCELLED"].includes(item.record.matter.status); }
export function openRiskAction(action: MatterAction) { return ["PLANNED", "IN_PROGRESS", "BLOCKED"].includes(action.status); }
export function overdueRiskAction(action: MatterAction, now = Date.now()) { return openRiskAction(action) && !!action.due_at && Date.parse(action.due_at) < now; }

async function boundedReads<T, R>(values: T[], read: (value: T) => Promise<R>): Promise<PromiseSettledResult<R>[]> {
  const results: PromiseSettledResult<R>[] = new Array(values.length);
  let next = 0;
  await Promise.all(Array.from({ length: Math.min(4, values.length) }, async () => {
    while (next < values.length) {
      const index = next++;
      try { results[index] = { status: "fulfilled", value: await read(values[index]!) }; }
      catch (reason) { results[index] = { status: "rejected", reason }; }
    }
  }));
  return results;
}

export async function loadVendorRiskWork(relationshipIDs: string[]): Promise<VendorRiskWork> {
  const ids = [...new Set(relationshipIDs)];
  let complete = ids.length <= 50;
  const targets = new Map<string, string[]>();
  const links = await boundedReads(ids.slice(0, 50), (relationship_id) => loadVendorRelationshipLinks({ relationship_id, limit: 50 }));
  links.forEach((result, index) => {
    if (result.status === "rejected") { complete = false; return; }
    if (result.value.next_cursor) complete = false;
    for (const link of result.value.items.slice(0, 50)) {
      if (link.state !== "ACTIVE" || link.target_type !== "MATTER") continue;
      const relationships = targets.get(link.target_id) ?? [];
      if (!relationships.includes(ids[index]!)) relationships.push(ids[index]!);
      targets.set(link.target_id, relationships);
    }
  });
  if (targets.size > 50) complete = false;
  const targetIDs = [...targets.keys()].slice(0, 50);
  const records = await boundedReads(targetIDs, loadMatter);
  const items: VendorRiskFinding[] = [];
  records.forEach((result, index) => {
    if (result.status === "rejected") { complete = false; return; }
    if (result.value.matter.id !== targetIDs[index]) { complete = false; return; }
    if (findingTypes.has(result.value.matter.type)) items.push({ record: result.value, relationshipIDs: targets.get(targetIDs[index])! });
  });
  return { items, complete, checkedAt: new Date().toISOString() };
}

export function summarizeVendorRiskWork(work: VendorRiskWork, now = Date.now()) {
  if (!work.complete) return { findings: null, openActions: null, overdueActions: null, implementedActions: null };
  const findings = [...new Map(work.items.filter(openFinding).map(item => [item.record.matter.id, item])).values()];
  const actions = [...new Map(findings.flatMap(item => item.record.actions).map(action => [action.id, action])).values()];
  return { findings: findings.length, openActions: actions.filter(openRiskAction).length, overdueActions: actions.filter(action => overdueRiskAction(action, now)).length, implementedActions: actions.filter(action => action.status === "IMPLEMENTED").length };
}
