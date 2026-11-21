import type { RuntimeContext } from "../api";

// Presentation only. Backend authorization remains independent of this label.
export function isAdministrativeHome(context: RuntimeContext): boolean {
  const roles = context.actor.role_codes;
  return Array.isArray(roles) && roles.some((role) => typeof role === "string" && ["SYSTEM_ADMIN", "SUPER_ADMIN", "GRC_ADMIN", "GRC_ADMINISTRATOR"].includes(role.trim().toUpperCase()));
}
