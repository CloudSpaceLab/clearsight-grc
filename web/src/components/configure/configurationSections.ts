export type ConfigurationSection = "overview" | "access" | "authority" | "data" | "automation" | "ai" | "operations";

export type ConfigurationArea = {
  id: Exclude<ConfigurationSection, "overview">;
  label: string;
  description: string;
};

export const configurationAreas: ConfigurationArea[] = [
  { id: "access", label: "People & access", description: "People, roles, departments, provisioning and workspace access." },
  { id: "authority", label: "Approval routes", description: "Routing policies, delegations, integrity checks and affected work." },
  { id: "data", label: "Data & integrations", description: "Document imports, connected sources, mappings and reconciliation." },
  { id: "automation", label: "Automation", description: "Automation policies and action limits." },
  { id: "ai", label: "AI governance", description: "AI workloads, security controls and policy rollout." },
  { id: "operations", label: "System operations", description: "Background jobs and Program status checks." },
];

export function configurationSectionFromHash(hash: string): ConfigurationSection {
  const route = hash.replace(/^#\/?/, "").split("?", 1)[0] ?? "";
  const [, section] = route.split("/").filter(Boolean);
  return configurationAreas.some((area) => area.id === section) ? section as ConfigurationSection : "overview";
}

export function configurationHash(section: ConfigurationSection) {
  return section === "overview" ? "#configure" : `#configure/${section}`;
}
