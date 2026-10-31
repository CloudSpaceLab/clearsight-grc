import type { IdentityPerson, OrganizationPosition } from "../../identityAccessApi";

export function personName(person: IdentityPerson) {
  return person.display_name || person.user_name || "Unnamed person";
}

export function initials(name: string) {
  return name.trim().split(/\s+/).slice(0, 2).map((part) => part[0]).join("").toUpperCase() || "?";
}

export function humanAccessLabel(value: string) {
  const words = value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
  return words.replace(/\b(Cro|Cco|Ciso|Grc|Dpo|Scim|Oidc|Api)\b/g, (word) => word.toUpperCase());
}

export function positionsFor(personID: string, positions: OrganizationPosition[]) {
  return positions.filter((position) => position.occupant_principal_id === personID);
}
