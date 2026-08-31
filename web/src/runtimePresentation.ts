export type RuntimePresentation = "enterprise" | "demo" | "live-preview";

export function runtimePresentation(search: string): RuntimePresentation {
  return new URLSearchParams(search).get("demo") === "1" ? "demo" : "enterprise";
}
