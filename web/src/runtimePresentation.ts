export type RuntimePresentation = "enterprise" | "demo";

export function runtimePresentation(search: string): RuntimePresentation {
  return new URLSearchParams(search).get("demo") === "1" ? "demo" : "enterprise";
}
