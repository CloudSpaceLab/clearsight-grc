import { render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { FreshApp } from "./FreshApp";

vi.mock("../api", () => ({ loadSessionStatus: vi.fn(async () => ({ authenticated: true })), loadContext: vi.fn(async () => ({ tenant: { id: "t", name: "Bank" }, legal_entity: { id: "e", name: "Bank" }, actor: { id: "a", name: "Ada" }, capabilities: {} })) }));
vi.mock("./Home", () => ({ Home: () => <h1>Home</h1> }));
vi.mock("./NotificationPreferences", () => ({ NotificationPreferences: ({scope}: {scope:{tenant:string;actor:string}}) => <p>Email preferences: {scope.tenant} {scope.actor}</p> }));
vi.mock("./Vendors", () => ({ Vendors: ({ context, relationshipId, assessmentId }: { context: { legal_entity: { id: string } }; relationshipId?: string; assessmentId?: string }) => <><p>Vendor route: {relationshipId} {assessmentId}</p><p>Vendor scope: {context.legal_entity.id}</p></> }));
vi.mock("./Responses", () => ({ Responses: ({ id }: { id?: string }) => <p>Response route: {id}</p> }));
vi.mock("./FormAssignment", () => ({ FormAssignment: ({ formId }: { formId: string }) => <p>Assignment route: {formId}</p> }));
vi.mock("./FormCreator", () => ({ FormCreator: () => <h1>Create form</h1> }));
vi.mock("./ProgramCreator", () => ({ ProgramCreator: () => <h1>Create program</h1>, ProgramCreateLink: () => null }));
vi.mock("./FormEditor", () => ({ FormEditor: ({ id }: { id: string }) => <p>Form editor: {id}</p> }));
vi.mock("./Frameworks", () => ({ Frameworks: () => <h1>Framework catalog</h1> }));
vi.mock("./RiskWorkspace", () => ({ RiskWorkspace: ({ id }: { id?: string }) => <p>Risk route: {id ?? "register"}</p> }));
vi.mock("./PrivacyWorkspace", () => ({ PrivacyWorkspace: ({ id }: { id?: string }) => <p>Privacy route: {id ?? "register"}</p> }));
vi.mock("./Reports", () => ({ Reports: () => <h1>Reports and insights</h1> }));
vi.mock("./RCSAWorkspace", () => ({ RCSAWorkspace: ({ id }: { id?: string }) => <p>RCSA route: {id ?? "register"}</p> }));
vi.mock("./KeyIndicators", () => ({ KeyIndicators: () => <h1>Key indicators</h1> }));
vi.mock("./LossWorkspace", () => ({ LossWorkspace: ({ id }: { id?: string }) => <p>Loss route: {id ?? "register"}</p> }));
vi.mock("./ActorWorkWorkspace", () => ({ ActorWorkWorkspace: ({ context }: { context: { legal_entity: { id: string } } }) => <p>Work scope: {context.legal_entity.id}</p> }));

beforeEach(() => { window.location.hash = ""; });

it("opens the framework catalog", async () => {
  window.location.hash = "#/frameworks";
  render(<FreshApp/>);
  expect(await screen.findByRole("heading", { name: "Framework catalog" })).toBeTruthy();
});
it("opens scoped work", async () => {
  window.location.hash = "#/work";
  render(<FreshApp/>);
  expect(await screen.findByText("Work scope: e")).toBeTruthy();
});
it("rejects unsupported work actions", async () => {
  window.location.hash = "#/work/request-1";
  render(<FreshApp/>);
  expect(await screen.findByRole("heading", { name: "Record unavailable" })).toBeTruthy();
  expect(screen.queryByText("Work scope: e")).toBeNull();
});
it("keeps reports unavailable until dataset visibility is verified", async () => {
  window.location.hash = "#/reports";
  render(<FreshApp/>);
  expect(await screen.findByRole("heading", { name: "Reports unavailable" })).toBeTruthy();
  expect(screen.queryByRole("link", { name: "Reports" })).toBeNull();
});

it("opens the risk register and exact risk", async () => {
  window.location.hash = "#/risks/risk%20one";
  render(<FreshApp/>);
  expect(await screen.findByText("Risk route: risk one")).toBeTruthy();
});
it("opens the exact RCSA cycle", async () => {
  window.location.hash = "#/risks/rcsa/cycle%20one";
  render(<FreshApp/>);
  expect(await screen.findByText("RCSA route: cycle one")).toBeTruthy();
});
it("opens key indicators", async () => {
  window.location.hash = "#/risks/indicators";
  render(<FreshApp/>);
  expect(await screen.findByRole("heading", { name: "Key indicators" })).toBeTruthy();
});
it("opens the exact loss event", async () => {
  window.location.hash = "#/risks/losses/loss%20one";
  render(<FreshApp/>);
  expect(await screen.findByText("Loss route: loss one")).toBeTruthy();
  expect(screen.queryByText("Risk route: losses")).toBeNull();
});
it("opens the loss register", async () => {
  window.location.hash = "#/risks/losses";
  render(<FreshApp/>);
  expect(await screen.findByText("Loss route: register")).toBeTruthy();
});

it("rejects extra risk path segments", async () => {
  window.location.hash = "#/risks/risk/extra";
  render(<FreshApp/>);
  expect(await screen.findByRole("heading", { name: "Record unavailable" })).toBeTruthy();
});

it("keeps privacy records unavailable until scoped reads are verified", async () => {
  window.location.hash = "#/privacy/activity%20one";
  render(<FreshApp/>);
  expect(await screen.findByRole("heading", { name: "Privacy unavailable" })).toBeTruthy();
  expect(screen.queryByText("Privacy route: activity one")).toBeNull();
  expect(screen.queryByRole("link", { name: "Privacy" })).toBeNull();
});

it("rejects extra privacy path segments", async () => {
  window.location.hash = "#/privacy/activity/extra";
  render(<FreshApp/>);
  expect(await screen.findByRole("heading", { name: "Record unavailable" })).toBeTruthy();
});

it("opens the exact vendor assessment from its document link", async () => {
  window.location.hash = "#/vendors/vendor%20one/assessments/assessment%20two";
  render(<FreshApp/>);
  expect(await screen.findByText("Vendor route: vendor one assessment two")).toBeTruthy();
  expect(screen.getByText("Vendor scope: e")).toBeTruthy();
});

it("opens the exact submitted response from its document link", async () => {
  window.location.hash = "#/forms/responses/revision%20one";
  render(<FreshApp/>);
  expect(await screen.findByText("Response route: revision one")).toBeTruthy();
});

it("does not replace a malformed record link with a vendor list", async () => {
  window.location.hash = "#/vendors/%E0%A4%A/assessments/assessment";
  render(<FreshApp/>);
  expect(await screen.findByRole("heading", { name: "Record unavailable" })).toBeTruthy();
  expect(screen.queryByText(/Vendor route:/)).toBeNull();
});

it("does not create an assignment while internal response capture is unavailable", async () => {
  window.location.hash = "#/forms/form%20one/assign";
  render(<FreshApp/>);
  expect(await screen.findByRole("heading", { name: "Form assignment unavailable" })).toBeTruthy();
  expect(screen.queryByText("Assignment route: form one")).toBeNull();
});

it("opens the new form builder without looking up a template", async () => {
  window.location.hash = "#/forms/new";
  render(<FreshApp/>);
  expect(await screen.findByRole("heading", { name: "Create form" })).toBeTruthy();
});

it("opens Program creation without looking up an existing record", async () => {
  window.location.hash = "#/programs/new";
  render(<FreshApp/>);
  expect(await screen.findByRole("heading", { name: "Create program" })).toBeTruthy();
});

it("rejects extra segments on Program creation links", async () => {
  window.location.hash = "#/programs/new/extra";
  render(<FreshApp/>);
  expect(await screen.findByRole("heading", { name: "Record unavailable" })).toBeTruthy();
});

it("opens the editor for the exact selected form", async () => {
  window.location.hash = "#/forms/form%20one/edit";
  render(<FreshApp/>);
  expect(await screen.findByText("Form editor: form one")).toBeTruthy();
});

it("opens personal email preferences without administrative settings access", async () => {
  window.location.hash = "#/preferences"; render(<FreshApp/>);
  expect(await screen.findByText("Email preferences: t a")).toBeTruthy();
  expect(screen.getByRole("link", {name:"Email preferences"}).getAttribute("href")).toBe("#/preferences");
  expect(screen.queryByRole("link", {name:"Settings"})).toBeNull();
});
