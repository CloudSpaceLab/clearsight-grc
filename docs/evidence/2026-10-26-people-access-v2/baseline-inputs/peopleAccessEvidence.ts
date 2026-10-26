import type { IdentityAccessOverview, IdentityPerson } from "./identityAccessApi";
import type { GovernanceDelegationRecord, GovernancePolicyRecord } from "./governanceAdminApi";

const people: IdentityPerson[] = [
  { id: "sample-person-amina", display_name: "Amina Yusuf", user_name: "amina.yusuf", status: "ACTIVE", source_code: "Bank directory", source_state: "ACTIVE" },
  { id: "sample-person-chidi", display_name: "Chidi Okonkwo", user_name: "chidi.okonkwo", status: "ACTIVE", source_code: "Bank directory", source_state: "ACTIVE" },
  { id: "sample-person-funke", display_name: "Funke Adeyemi", user_name: "funke.adeyemi", status: "ACTIVE", source_code: "Bank directory", source_state: "DISABLED" },
  { id: "sample-person-ibrahim", display_name: "Ibrahim Musa", user_name: "ibrahim.musa", status: "INACTIVE", source_code: "Bank directory", source_state: "ACTIVE" },
  { id: "sample-person-kemi", display_name: "Kemi Adebayo", user_name: "kemi.adebayo", status: "ACTIVE", source_code: "Bank directory", source_state: "ACTIVE" },
  { id: "sample-person-morgan", display_name: "Morgan Ellis", status: "ACTIVE" },
  { id: "sample-person-nneka", display_name: "Nneka Okafor", user_name: "nneka.okafor", status: "ACTIVE", source_code: "Bank directory", source_state: "ACTIVE" },
  { id: "sample-person-olumide", display_name: "Olumide Balogun", user_name: "olumide.balogun", status: "ACTIVE", source_code: "Bank directory", source_state: "ACTIVE" },
  { id: "sample-person-sarah", display_name: "Sarah Ojo", user_name: "sarah.ojo", status: "ACTIVE", source_code: "Bank directory", source_state: "ACTIVE" },
  { id: "sample-person-tunde", display_name: "Tunde Bello", user_name: "tunde.bello", status: "ACTIVE", source_code: "Bank directory", source_state: "ACTIVE" },
  { id: "sample-person-zainab", display_name: "Zainab Abubakar", user_name: "zainab.abubakar", status: "ACTIVE", source_code: "Legacy directory", source_state: "REVOKED" },
  { id: "sample-person-zara", display_name: "Zara Nwosu", user_name: "zara.nwosu", status: "ACTIVE", source_code: "Bank directory", source_state: "ACTIVE" },
];
const policies: GovernancePolicyRecord[] = [
  { id: "sample-policy-risk", tenant_id: "bank-demo", legal_entity_id: "bank-ng", code: "RISK-APPROVAL", name: "Material risk approval", status: "ACTIVE", current_version: 3, maker_id: "sample-person-chidi", checker_id: "sample-person-amina", effective_from: "2026-09-01T00:00:00Z", version: 4 },
  { id: "sample-policy-evidence", tenant_id: "bank-demo", legal_entity_id: "bank-ng", code: "EVIDENCE-REVIEW", name: "Independent evidence review", status: "DRAFT", current_version: 1, maker_id: "sample-person-chidi", version: 1 },
];
const delegations: GovernanceDelegationRecord[] = [
  { id: "sample-delegation-risk", tenant_id: "bank-demo", legal_entity_id: "bank-ng", from_principal_id: "sample-person-amina", to_principal_id: "sample-person-chidi", responsibility: "REVIEWER", scope: { legal_entity_id: "bank-ng", object_type: "PROGRAM" }, starts_at: "2026-10-20T00:00:00Z", ends_at: "2026-11-03T00:00:00Z", status: "ACTIVE", reason: "Sample: planned annual leave cover.", maker_id: "sample-person-amina", approver_id: "sample-person-morgan", version: 2 },
  { id: "sample-delegation-ops", tenant_id: "bank-demo", legal_entity_id: "bank-ng", from_principal_id: "sample-person-nneka", to_principal_id: "sample-person-tunde", responsibility: "PERFORMER", scope: { legal_entity_id: "bank-ng" }, starts_at: "2026-11-01T00:00:00Z", ends_at: "2026-11-07T00:00:00Z", status: "PENDING_APPROVAL", reason: "Sample: operations cover request.", maker_id: "sample-person-nneka", version: 1 },
];
declare global {
  interface Window {
    peopleAccessEvidence?: {
      reads: Array<{ path: string; query: string; status: number; people?: IdentityPerson[]; roleIDs?: string[]; roleNames?: string[]; counts?: { people: number; groups: number; sources: number; roles: number }; policyIDs?: string[]; policyNames?: string[]; policyRecords?: Array<{ id: string; name: string; status: string }>; delegationIDs?: string[]; delegationRecords?: Array<{ id: string; status: string }> }>;
      commands: Array<{ method: string; path: string }>;
    };
  }
}
// Imported only by evidenceMain. No sample records enter customer builds.
export function installPeopleAccessEvidence() {
  const fixture = new URLSearchParams(window.location.search).get("fixture") ?? "";
  if (!fixture.startsWith("people-v2-")) return;
  const variant = fixture.slice("people-v2-".length);
  const previous = globalThis.fetch.bind(globalThis);
  let failuresRemaining = variant === "recovery" ? 1 : 0;
  window.peopleAccessEvidence = { reads: [], commands: [] };
  globalThis.fetch = async (input, init) => {
    const raw = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    const url = new URL(raw, window.location.origin), path = url.pathname;
    const method = (init?.method ?? (input instanceof Request ? input.method : "GET")).toUpperCase();
    const json = (value: unknown, status = 200) => new Response(JSON.stringify(value), { status, headers: { "Content-Type": "application/json" } });
    const managed = path.startsWith("/api/v1/access/") || path.startsWith("/api/v1/governance/");
    if (managed && method !== "GET") {
      window.peopleAccessEvidence!.commands.push({ method, path });
      return json({ error: { code: "fixture_read_only", message: "This sample supports viewing only." } }, 405);
    }
    if (path === "/api/v1/context" && variant === "readonly") {
      const context = await (await previous(input, init)).json();
      return json({ ...context, capabilities: { ...context.capabilities, config_write: false, identity_configure: false } });
    }
    if (path === "/api/v1/authority/integrity" && variant === "checks-partial") return json({ error: { code: "integrity_unavailable", message: "Routing checks are temporarily unavailable." } }, 503);
    if (!managed) return previous(input, init);
    const result = (value: unknown, status = 200, facts = {}) => {
      window.peopleAccessEvidence!.reads.push({ path, query: url.search, status, ...facts });
      return json(value, status);
    };
    if (variant === "denied") return result({ error: { code: "permission_required", message: "People and access records are unavailable for this account." } }, 403);
    if (path === "/api/v1/access/overview") {
      if (failuresRemaining > 0) {
        failuresRemaining--;
        return result({ error: { code: "access_unavailable", message: "The directory could not be loaded. Try again." } }, 503);
      }
      const base = await (await previous(input, init)).json() as IdentityAccessOverview;
      const samplePeople = variant === "empty" ? [] : variant === "limited" ? Array.from({ length: 50 }, (_, index) => index < people.length ? people[index]! : { id: `sample-person-${index + 1}`, display_name: `Sample colleague ${String(index + 1).padStart(2, "0")}`, user_name: `sample.colleague.${index + 1}`, status: "ACTIVE", source_code: "Bank directory", source_state: "ACTIVE" }) : structuredClone(people);
      const value: IdentityAccessOverview = {
        ...base,
        can_configure: variant !== "readonly", can_configure_organization: variant !== "readonly", can_configure_escalation: false,
        people: samplePeople,
        sources: variant === "empty" ? [] : [{ id: "sample-source-bank", code: "Bank directory", status: "ACTIVE", subject_attribute: "userName", active_users: 9, active_groups: 2 }, { id: "sample-source-legacy", code: "Legacy directory", status: "REVOKED", subject_attribute: "externalId", active_users: 0, active_groups: 0 }],
        groups: variant === "empty" ? [] : [{ id: "sample-group-risk", display_name: "Risk review team", source_code: "Bank directory", source_state: "ACTIVE", member_count: 4 }, { id: "sample-group-ops", display_name: "Operations team", source_code: "Bank directory", source_state: "ACTIVE", member_count: 5 }],
        roles: variant === "empty" ? [] : [{ id: "sample-role-reviewer", code: "RISK_REVIEWER", name: "Risk reviewer", capabilities: ["program.read", "evidence.read"], version: 2, organization_editable: true }, { id: "sample-role-admin", code: "ACCESS_ADMIN", name: "Access administrator", capabilities: ["identity.read", "identity.configure"], version: 3, organization_editable: false, organization_lock_reasons: ["Managed by the bank identity policy."] }],
        bindings: variant === "empty" ? [] : [{ id: "sample-binding-risk", group_id: "sample-group-risk", group_name: "Risk review team", role_template_id: "sample-role-reviewer", role_code: "RISK_REVIEWER", legal_entity_id: "bank-ng", legal_entity: "Meridian Trust Bank Nigeria", department_path: ["BANK", "RISK"], valid_from: "2026-09-01T00:00:00Z" }],
        positions: variant === "empty" ? [] : [{ id: "sample-position-risk", code: "RISK-REVIEW", title: "Senior risk reviewer", function_name: "Risk", department_path: ["BANK", "RISK"], occupant_principal_id: "sample-person-amina", occupant_name: "Amina Yusuf", occupant_status: "ACTIVE", role_codes: ["RISK_REVIEWER"], valid_from: "2026-09-01T00:00:00Z", version: 2 }, { id: "sample-position-deputy", code: "RISK-DEPUTY", title: "Deputy risk reviewer", function_name: "Risk", department_path: ["BANK", "RISK"], occupant_principal_id: "sample-person-chidi", occupant_name: "Chidi Okonkwo", occupant_status: "ACTIVE", role_codes: ["RISK_REVIEWER"], valid_from: "2026-09-01T00:00:00Z", version: 1 }],
      };
      return result(value, 200, { people: value.people, roleIDs: value.roles.map((role) => role.id), roleNames: value.roles.map((role) => role.name), counts: { people: value.people.length, groups: value.groups.length, sources: value.sources.length, roles: value.roles.length } });
    }
    if (path === "/api/v1/governance/policies") {
      if (variant === "partial") return result({ error: { code: "policy_unavailable", message: "Approval routes could not be loaded." } }, 503);
      const items = variant === "empty" ? [] : policies;
      return result({ items }, 200, { policyIDs: items.map((item) => item.id), policyNames: items.map((item) => item.name), policyRecords: items.map(({ id, name, status }) => ({ id, name, status })) });
    }
    if (path === "/api/v1/governance/delegations") {
      const items = variant === "empty" ? [] : delegations;
      return result({ items }, 200, { delegationIDs: items.map((item) => item.id), delegationRecords: items.map(({ id, status }) => ({ id, status })) });
    }
    return previous(input, init);
  };
}
