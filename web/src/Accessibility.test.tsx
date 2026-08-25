import axe from "axe-core";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CapturePanel } from "./components/CapturePanel";
import { DemoLoginPage } from "./components/DemoLoginPage";
import { MattersWorkspace } from "./components/MattersWorkspace";
import { ProgramsWorkspace } from "./components/ProgramsWorkspace";
import { TodayInterventions } from "./components/TodayInterventions";
import type { DemoAccount } from "./api";
import type { AttentionItem, CaptureRequest, Readiness } from "./types";

vi.mock("./api", () => ({
  loadProgramSummaries: vi.fn().mockResolvedValue({ items: [], next_cursor: "", generated_at: "2026-08-07T13:00:00Z" }),
  loadProgram: vi.fn(),
  loadMatterSummaries: vi.fn().mockResolvedValue({ items: [], next_cursor: "", generated_at: "2026-08-07T13:00:00Z" }),
  loadMatter: vi.fn(),
  submitCaptureRequest: vi.fn(),
  loginDemo: vi.fn().mockResolvedValue(undefined),
}));

const item: AttentionItem = {
  id: "intervention-1",
  type: "REGULATORY_CHANGE",
  title: "Review digital-channel obligations",
  why_now: "A governing source changed.",
  scope: "Digital Channels",
  state: "Applicability review",
  evidence: "Official source verified",
  owner: "Regulatory Compliance",
  due_at: "2026-08-09T12:00:00Z",
  primary_action: "Review proposed obligations",
  action_target_type: "MATTER",
  action_target_id: "matter-1",
  intervention_class: "REVIEW",
  material_conclusion: "Seven provisions may change current obligations.",
};

const readiness: Readiness = {
  tenant_id: "bank-demo",
  status: "AT_RISK",
  baseline_known: true,
  generated_at: "2026-08-07T13:00:00Z",
  dimensions: { current: 18, aging: 1, at_risk: 1, unknown: 0, blocked_routing: 0, pending_human: 1 },
  active_drifts: [],
  recommended_actions: [],
};

const request: CaptureRequest = {
  id: "request-1",
  title: "Confirm the current accountable owner",
  purpose: "Resolve one ownership gap for the current review period.",
  why_you: "You are the current business owner for this scope.",
  status: "READY",
  sensitivity: "INTERNAL",
  estimated_minutes: 2,
  deadline: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
  known_facts: { application: "Mobile Banking" },
  fields: [{ id: "owner", label: "Current accountable owner", type: "text", required: true }],
  version: 1,
};

const demoAccounts: DemoAccount[] = [
  { label: "Chief Risk Officer", username: "cro@demo.clearsight.local", password: "demo", role_codes: ["CRO", "EXECUTIVE"] },
  { label: "System Administrator", username: "system-admin@demo.clearsight.local", password: "demo", role_codes: ["SYSTEM_ADMIN"] },
];

async function expectNoSemanticViolations(container: HTMLElement) {
  const results = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  expect(results.violations.map((violation) => violation.id)).toEqual([]);
}

async function waitForEmptyState(container: HTMLElement) {
  await waitFor(() => expect(container.querySelector(".empty-state")).not.toBeNull());
}

describe("semantic accessibility gates", () => {
  it("passes axe for the Today intervention surface", async () => {
    const { container } = render(<TodayInterventions items={[item]} connection="live" readiness={readiness} readinessState="live" onOpenItem={vi.fn()}/>);
    await expectNoSemanticViolations(container);
  });

  it("passes axe for the final capture assertion review", async () => {
    const { container } = render(<CapturePanel request={request}/>);
    fireEvent.change(screen.getByRole("textbox", { name: /Current accountable owner/ }), { target: { value: "Ada Okafor" } });
    fireEvent.click(screen.getByRole("button", { name: "Review response" }));
    expect(screen.getByRole("heading", { name: "Check your response" })).toBeTruthy();
    await expectNoSemanticViolations(container);
  });

  it("passes axe for Wizard progress, typed fields, and the linked error summary", async () => {
    const typedRequest: CaptureRequest = {
      ...request,
      presentation: { default_mode: "WIZARD", allow_mode_switch: true },
      sections: [{ id: "contact", title: "Contact" }, { id: "authority", title: "Authority" }],
      fields: [
        { id: "email", section_id: "contact", label: "Security contact email", type: "email", required: true },
        { id: "regions", section_id: "contact", label: "Processing regions", type: "multi_select", required: true, options: ["Nigeria", "Ghana"], constraints: { min_selections: 1 } },
        { id: "attestation", section_id: "authority", label: "Authorized response", type: "attestation", required: true, attestation: "I confirm that I am authorized to submit this response." },
      ],
    };
    const { container } = render(<CapturePanel request={typedRequest} external/>);
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    expect(screen.getByRole("alert").textContent).toMatch(/Security contact email is required/);
    await expectNoSemanticViolations(container);
  });

  it("passes axe for the role-aware demo login without repeating shared credentials", async () => {
    const { container } = render(<DemoLoginPage accounts={demoAccounts} onAuthenticated={vi.fn().mockResolvedValue(undefined)}/>);
    expect(screen.getByText("cro@demo.clearsight.local")).toBeTruthy();
    expect(screen.queryByText("demo")).toBeNull();
    expect(screen.getByRole("button", { name: /System Administrator/ })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Continue as Chief Risk Officer" })).toBeTruthy();
    await expectNoSemanticViolations(container);
  });

  it("passes axe for bounded Program and Matter empty/recovery states", async () => {
    const program = render(<ProgramsWorkspace/>);
    await waitForEmptyState(program.container);
    await expectNoSemanticViolations(program.container);
    program.unmount();

    const matter = render(<MattersWorkspace/>);
    await waitForEmptyState(matter.container);
    await expectNoSemanticViolations(matter.container);
  });
});
