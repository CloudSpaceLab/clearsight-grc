import { render, screen } from "@testing-library/react";
import { expect, it } from "vitest";
import { AIGatewayProxyCard } from "./AIGatewayProxyCard";

const ingress = [
  { method: "GET", path: "/v1/models" },
  { method: "POST", path: "/v1/chat/completions" },
  { method: "POST", path: "/v1/responses" },
];

function routeText(value: string) {
  return (_: string, element: Element | null) => element?.tagName === "CODE" && element.textContent === value;
}

it("shows the published stable proxy and executable workload ingress", () => {
  render(<AIGatewayProxyCard
    proxy={{ configured: true, base_url: "https://ai.bank.example/proxy", ingress }}
    runtimeStatus={{ configured: true, available: true, tenant_id: "bank", environment: "PRODUCTION", desired_revision: 4, applied_revision: 4, degraded: false }}
    loading={false}
  />);

  expect(screen.getByText("https://ai.bank.example/proxy")).toBeTruthy();
  expect(screen.getByText(routeText("POST /v1/chat/completions"))).toBeTruthy();
  expect(screen.getByText(routeText("POST /v1/responses"))).toBeTruthy();
  expect(screen.queryByText(/metrics/)).toBeNull();
  expect(screen.getByText("Published")).toBeTruthy();
});

it("does not imply runtime health when no public proxy URL is published", () => {
  render(<AIGatewayProxyCard
    proxy={{ configured: false, ingress }}
    runtimeStatus={{ configured: false, available: false, tenant_id: "bank", environment: "PRODUCTION", desired_revision: 0, applied_revision: 0, degraded: false }}
    loading={false}
  />);

  expect(screen.getByText("Not published")).toBeTruthy();
  expect(screen.getByText(/No public proxy URL is configured/)).toBeTruthy();
  expect(screen.getByText(/Publishing a base URL does not imply/)).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Copy base URL" })).toBeNull();
});
