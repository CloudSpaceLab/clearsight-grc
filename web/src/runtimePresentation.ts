export type RuntimePresentation = "demo" | "live-preview";

export function runtimePresentation(search: string): RuntimePresentation {
  return new URLSearchParams(search).get("demo") === "0" ? "live-preview" : "demo";
}
