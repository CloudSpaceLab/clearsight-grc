import { render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { MatterWorkDetail } from "./MatterWorkDetail";

const api = vi.hoisted(() => ({ loadMatter: vi.fn(), loadEvidenceSources: vi.fn() }));
const operations = vi.hoisted(() => ({ loadMatterOperations: vi.fn(), defineMatterOutcomeCheck: vi.fn() }));
vi.mock("../matterOperationsApi", () => operations);
const responses = vi.hoisted(() => ({ loadCompletedResponses: vi.fn() }));
vi.mock("../api", () => api);
vi.mock("../formsDistributionApi", () => responses);
const context = { tenant: { id: "bank", name: "Bank" }, legal_entity: { id: "entity", name: "Bank" }, actor: { id: "person", name: "Person" }, mode: "LIVE" };
const matter = { matter: { id: "issue-1", tenant_id: "bank", legal_entity_id: "entity", title: "Review access exception", summary: "Access needs review.", reference: "ISS-1", status: "DECISION_REQUIRED", version: 2 }, scope: { tenant_id: "bank", legal_entity_id: "entity", principal_id: "person", record_tenant_id: "bank", record_legal_entity_id: "entity" }, status_label: "Decision needed", owner_display_name: "Amina Bello", decisions: [] };
beforeEach(() => { vi.resetAllMocks(); api.loadEvidenceSources.mockResolvedValue([]); operations.loadMatterOperations.mockResolvedValue({matter_id:"issue-1",matter_version:2,authority_available:true,operations:[{command:"matter.outcome.define",can_act:true,candidates:[{id:"reviewer",display_name:"Ada Okafor"}]}]}); responses.loadCompletedResponses.mockResolvedValue({ items: [] }); });

it("shows the scoped issue and the unavailable decision state", async () => {
  api.loadMatter.mockResolvedValue(matter);
  render(<MatterWorkDetail id="issue-1" context={context}/>);
  expect(await screen.findByRole("heading", { name: "Review access exception" })).toBeTruthy();
  expect(screen.getByText("Access needs review.")).toBeTruthy();
  expect(screen.getByText("Amina Bello")).toBeTruthy();
  expect(screen.getByText("Decision unavailable.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: /approve|reject|review/i })).toBeNull();
});

it.each([
  { ...matter, matter: { ...matter.matter, id: "other" } },
  { ...matter, matter: { ...matter.matter, tenant_id: "other" } },
  { ...matter, matter: { ...matter.matter, legal_entity_id: "other" } },
  { ...matter, scope: { ...matter.scope, principal_id: "other" } },
  { ...matter, scope: { ...matter.scope, tenant_id: "other" } },
  { ...matter, scope: { ...matter.scope, legal_entity_id: "other" } },
  { ...matter, scope: undefined },
])("does not show a mismatched record", async (value) => {
  api.loadMatter.mockResolvedValue(value);
  render(<MatterWorkDetail id="issue-1" context={context}/>);
  expect(await screen.findByRole("alert")).toBeTruthy();
  expect(screen.queryByText("Access needs review.")).toBeNull();
  expect(responses.loadCompletedResponses).not.toHaveBeenCalled();
});

it("accepts a verified alias scope with canonical record identifiers", async () => {
  api.loadMatter.mockResolvedValue({ ...matter, matter: { ...matter.matter, tenant_id: "canonical-bank", legal_entity_id: "canonical-entity" }, scope: { ...matter.scope, record_tenant_id: "canonical-bank", record_legal_entity_id: "canonical-entity" } });
  render(<MatterWorkDetail id="issue-1" context={context}/>);
  expect(await screen.findByText("Access needs review.")).toBeTruthy();
});

it("clears the issue while a new actor's access is checked", async () => {
  api.loadMatter.mockResolvedValueOnce(matter).mockReturnValueOnce(new Promise(() => {}));
  const view = render(<MatterWorkDetail id="issue-1" context={context}/>);
  expect(await screen.findByText("Access needs review.")).toBeTruthy();
  view.rerender(<MatterWorkDetail id="issue-1" context={{ ...context, actor: { ...context.actor, id: "other" } }}/>);
  expect(screen.queryByText("Access needs review.")).toBeNull();
});

it("distinguishes an unnamed recorded owner from an unassigned issue", async () => {
  api.loadMatter.mockResolvedValue({ ...matter, owner_display_name: "", matter: { ...matter.matter, owner_principal_id: "person-2" } });
  render(<MatterWorkDetail id="issue-1" context={context}/>);
  expect(await screen.findByText("Owner name unavailable")).toBeTruthy();
  expect(screen.queryByText("Owner not recorded")).toBeNull();
});


it("shows outcome definition only for the scoped Matter and current authority",async()=>{api.loadMatter.mockResolvedValue({...matter,actions:[],verification_contracts:[],verification_results:[]});render(<MatterWorkDetail id="issue-1" context={context}/>);expect(await screen.findByText("No outcome checks recorded.")).toBeTruthy();expect(await screen.findByRole("button",{name:"Define outcome check"})).toBeTruthy();expect(operations.loadMatterOperations).toHaveBeenCalledWith("issue-1");});
