export type ConfigurationSection = "overview" | "access" | "authority" | "data" | "automation" | "operations";

export type ConfigurationArea = {
  id: Exclude<ConfigurationSection, "overview">;
  label: string;
  description: string;
};

export const configurationAreas: ConfigurationArea[] = [
  { id: "access", label: "People & access", description: "Sign-in, directory sources, people, groups and workspace access." },
  { id: "authority", label: "Authority & routing", description: "Responsibilities, approval routes, delegations and escalation." },
  { id: "data", label: "Data & integrations", description: "Document imports, connected sources, mappings and reconciliation." },
  { id: "automation", label: "Automation & AI", description: "Governed automation policies, AI workloads and AI policy rollout." },
  { id: "operations", label: "System operations", description: "Background processing and calculated Program status health." },
];

export function configurationSectionFromHash(hash: string): ConfigurationSection {
  const route = hash.replace(/^#\/?/, "").split("?", 1)[0] ?? "";
  const [, section] = route.split("/").filter(Boolean);
  return configurationAreas.some((area) => area.id === section) ? section as ConfigurationSection : "overview";
}

export function configurationHash(section: ConfigurationSection) {
  return section === "overview" ? "#configure" : `#configure/${section}`;
}
