import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { Responses } from "../../v210/Responses";
import { ApiError } from "../../http";

const responses = vi.hoisted(() => ({ loadCompletedResponses: vi.fn(), loadCompletedResponse: vi.fn() }));
const review = vi.hoisted(() => ({ loadResponseReviews: vi.fn(), recordResponseReview: vi.fn() }));
vi.mock("../../formsDistributionApi", () => responses);
vi.mock("../../responseReviewApi", () => review);
vi.mock("../../submittedDocumentApi", () => ({ loadDocuments: vi.fn(), documentContentURL: vi.fn() }));

const summary = { id: "response-2", distribution_id: "distribution-1", title: "Security review", form_template_id: "form-1", form_template_version: 2, revision: 2, current: true, completed_at: "2026-11-20T10:00:00Z", submitted_by_display_name: "Ada Bello" };
const detail = { response: summary, revision: { id: "response-2", revision: 2, current: true, supersedes_revision_id: "response-1" }, answers: [{ field_id: "report", label: "Test coverage", value: { text: "Payment service omitted" } }], documents: [] };
const olderReview = { id: "review-2", response_id: "response-2", version: 2, decision: "REVIEWED", reviewer_id: "reviewer-1", reviewer_display_name: "Risk Officer", reviewed_at: "2026-11-20T10:00:00Z" };
beforeEach(() => {
  sessionStorage.clear();
  vi.resetAllMocks();
  window.history.replaceState(null, "", "#/forms/responses/response-2");
  responses.loadCompletedResponse.mockResolvedValue(detail);
  review.loadResponseReviews.mockResolvedValue({ version: 4, may_review: true, items: [], next_before_version: 3 });
});

it("preserves the review reason on conflict and does not retry the command", async () => {
  review.recordResponseReview.mockRejectedValueOnce(new ApiError(409, "Changed"));
  render(<Responses id="response-2" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "CHANGES_REQUESTED" } });
  fireEvent.change(screen.getByLabelText("Reason for changes"), { target: { value: "The report excludes payment processing." } });
  fireEvent.click(screen.getByRole("button", { name: "Save review" }));
  expect(await screen.findByText("The response or review changed. Reload before saving.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Reload response" }));
  expect((await screen.findByLabelText("Reason for changes") as HTMLTextAreaElement).value).toBe("The report excludes payment processing.");
  expect(review.recordResponseReview).toHaveBeenCalledTimes(1);
});

it("retries earlier review history without reloading the selected response or losing the reason", async () => {
  review.loadResponseReviews.mockResolvedValueOnce({ version: 4, may_review: true, items: [], next_before_version: 3 })
    .mockRejectedValueOnce(new Error("History unavailable"))
    .mockResolvedValueOnce({ version: 4, may_review: true, items: [olderReview] });
  render(<Responses id="response-2" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "CHANGES_REQUESTED" } });
  fireEvent.change(screen.getByLabelText("Reason for changes"), { target: { value: "Clarify the reporting period." } });
  fireEvent.click(screen.getByRole("button", { name: "Load earlier reviews" }));
  expect(await screen.findByText("Earlier reviews unavailable.")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load earlier reviews" }));
  expect(await screen.findByText(/Risk Officer/)).toBeTruthy();
  expect(review.loadResponseReviews).toHaveBeenLastCalledWith("response-2", 3);
  expect(responses.loadCompletedResponse).toHaveBeenCalledTimes(1);
  expect((screen.getByLabelText("Reason for changes") as HTMLTextAreaElement).value).toBe("Clarify the reporting period.");
});

it("carries the original distribution when opening the exact previous revision", async () => {
  render(<Responses id="response-2" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect((await screen.findByRole("link", { name: "Previous response" })).getAttribute("href")).toBe("#/forms/responses/response-1?distribution_id=distribution-1");
});

it.each([undefined, "", "*", " ", "bad\u0000scope"])("does not offer an earlier response with invalid distribution %s", async (distribution_id) => {
  responses.loadCompletedResponse.mockResolvedValue({ ...detail, response: { ...summary, distribution_id } });
  render(<Responses id="response-2" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect(await screen.findByText("Earlier response unavailable.")).toBeTruthy();
  expect(screen.queryByRole("link", { name: "Previous response" })).toBeNull();
});

it("retains the Documents return destination in the previous-response link", async () => {
  window.history.replaceState(null, "", `#/forms/responses/response-2?return_to=${encodeURIComponent("#/documents?view=NEEDS_REFRESH")}`);
  render(<Responses id="response-2" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  const href = (await screen.findByRole("link", { name: "Previous response" })).getAttribute("href") ?? "";
  const params = new URLSearchParams(href.split("?")[1]);
  expect(params.get("distribution_id")).toBe("distribution-1");
  expect(params.get("return_to")).toBe("#/documents?view=NEEDS_REFRESH");
});

it("rejects a previous revision returned from a different distribution, including on retry", async () => {
  window.history.replaceState(null, "", "#/forms/responses/response-1?distribution_id=distribution-1");
  responses.loadCompletedResponse.mockResolvedValue({ ...detail, response: { ...summary, id: "response-1", distribution_id: "other", current: false }, revision: { id: "response-1", current: false } });
  render(<Responses id="response-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect(await screen.findByRole("heading", { name: "Response unavailable" })).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Try again" }));
  await waitFor(() => expect(responses.loadCompletedResponse).toHaveBeenCalledTimes(2));
  expect(await screen.findByRole("heading", { name: "Response unavailable" })).toBeTruthy();
  expect(screen.queryByText("Payment service omitted")).toBeNull();
  expect(review.recordResponseReview).not.toHaveBeenCalled();
});

it("rejects a detail response for a different revision", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ ...detail, revision: { ...detail.revision, id: "other" } });
  render(<Responses id="response-2" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect(await screen.findByRole("heading", { name: "Response unavailable" })).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
});

it("rechecks the distribution receipt when the route changes for the same revision", async () => {
  const view = render(<Responses id="response-2" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect(await screen.findByText("Payment service omitted")).toBeTruthy();
  window.history.replaceState(null, "", "#/forms/responses/response-2?distribution_id=other");
  view.rerender(<Responses id="response-2" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect(await screen.findByRole("heading", { name: "Response unavailable" })).toBeTruthy();
  expect(screen.queryByText("Payment service omitted")).toBeNull();
});

it("does not offer a decision when the independent reviewer route is unavailable", async () => {
  review.loadResponseReviews.mockResolvedValue({ version: 4, may_review: false, items: [] });
  render(<Responses id="response-2" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect(await screen.findByText("Review unavailable.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
  expect(review.recordResponseReview).not.toHaveBeenCalled();
});

it("retries the same decision after a pre-commit connection failure and reload", async () => {
 sessionStorage.clear(); const scope={tenant:"tenant",entity:"entity",actor:"reviewer"};
 review.recordResponseReview.mockRejectedValueOnce(new Error("connection lost")).mockResolvedValueOnce({id:"review-5",response_id:"response-2",response_revision:2,version:5,decision:"REVIEWED"});
 const view=render(<Responses id="response-2" reviewScope={scope}/>);
 fireEvent.change(await screen.findByLabelText("Review decision"),{target:{value:"REVIEWED"}}); fireEvent.change(screen.getByLabelText("Comment (optional)"),{target:{value:"Confidential reviewer comment"}}); fireEvent.click(screen.getByRole("button",{name:"Save review"}));
 expect(await screen.findByText(/Review status unknown/)).toBeTruthy(); const first=review.recordResponseReview.mock.calls[0]; expect(first[2]).toBeTruthy();
 expect(JSON.stringify(sessionStorage)).not.toContain("Confidential reviewer comment");
 view.unmount(); const other=render(<Responses id="response-2" reviewScope={{...scope,actor:"another"}}/>); expect(await screen.findByLabelText("Review decision")).toBeTruthy(); expect(screen.queryByRole("button",{name:"Retry review"})).toBeNull(); other.unmount();
 render(<Responses id="response-2" reviewScope={scope}/>); fireEvent.change(await screen.findByLabelText("Original comment"),{target:{value:"Confidential reviewer comment"}}); fireEvent.click(await screen.findByRole("button",{name:"Retry review"}));
 await waitFor(()=>expect(review.recordResponseReview).toHaveBeenCalledTimes(2)); expect(review.recordResponseReview.mock.calls[1]).toEqual(first);
 expect(await screen.findByText("Review saved.")).toBeTruthy(); sessionStorage.clear();
});


it("allows correction after a definite validation rejection", async () => {
 sessionStorage.clear(); const scope={tenant:"tenant",entity:"entity",actor:"reviewer"};
 review.recordResponseReview.mockRejectedValueOnce(new ApiError(422,"Invalid"));
 render(<Responses id="response-2" reviewScope={scope}/>);
 fireEvent.change(await screen.findByLabelText("Review decision"),{target:{value:"REVIEWED"}}); fireEvent.click(screen.getByRole("button",{name:"Save review"}));
 expect(await screen.findByText("Review not saved. Check the decision and comment.")).toBeTruthy();
 expect(screen.getByLabelText("Review decision")).toBeTruthy(); expect(screen.queryByRole("button",{name:"Retry review"})).toBeNull(); expect(sessionStorage.length).toBe(0);
});


it("does not submit without verified account scope", async () => {
 render(<Responses id="response-2"/>); expect(await screen.findByText("Review unavailable.")).toBeTruthy(); expect(screen.queryByRole("button",{name:"Save review"})).toBeNull(); expect(review.recordResponseReview).not.toHaveBeenCalled();
});

it("loads an earlier response only from the same distribution",async()=>{
 responses.loadCompletedResponse.mockResolvedValueOnce(detail).mockResolvedValueOnce({response:{...summary,id:"response-1",revision:1,current:false},revision:{id:"response-1",revision:1,current:false},answers:[],documents:[]});
 render(<Responses id="response-2" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>); fireEvent.click(await screen.findByRole("button",{name:"Load earlier responses"}));
 const link=await screen.findByRole("link",{name:"Response version 1"}); expect(link.getAttribute("href")).toBe("#/forms/responses/response-1?distribution_id=distribution-1"); expect(responses.loadCompletedResponse).toHaveBeenLastCalledWith("response-1");
});
it("does not display or follow an earlier response from another distribution",async()=>{
 responses.loadCompletedResponse.mockResolvedValueOnce(detail).mockResolvedValueOnce({response:{...summary,id:"response-1",revision:1,current:false,distribution_id:"other"},revision:{id:"response-1",revision:1,current:false},answers:[],documents:[]});
 render(<Responses id="response-2" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>); fireEvent.click(await screen.findByRole("button",{name:"Load earlier responses"})); expect(await screen.findByText("Earlier responses unavailable.")).toBeTruthy(); expect(screen.queryByRole("link",{name:"Response version 1"})).toBeNull(); expect(screen.getByRole("button",{name:"Load earlier responses"})).toBeTruthy();
});

it("loads the revision chain one authorized response at a time",async()=>{
 const first={...detail,response:{...summary,revision:3},revision:{id:"response-2",revision:3,current:true,supersedes_revision_id:"response-1"}};
 const middle={response:{...summary,id:"response-1",revision:2,current:false},revision:{id:"response-1",revision:2,current:false,supersedes_revision_id:"response-0"},answers:[],documents:[]};
 const oldest={response:{...summary,id:"response-0",revision:1,current:false},revision:{id:"response-0",revision:1,current:false},answers:[],documents:[]};
 responses.loadCompletedResponse.mockResolvedValueOnce(first).mockResolvedValueOnce(middle).mockResolvedValueOnce(oldest);
 render(<Responses id="response-2" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>); fireEvent.click(await screen.findByRole("button",{name:"Load earlier responses"})); expect(await screen.findByRole("link",{name:"Response version 2"})).toBeTruthy(); fireEvent.click(screen.getByRole("button",{name:"Load earlier responses"})); expect(await screen.findByRole("link",{name:"Response version 1"})).toBeTruthy(); expect(screen.queryByRole("button",{name:"Load earlier responses"})).toBeNull(); expect(responses.loadCompletedResponse.mock.calls.map((c)=>c[0])).toEqual(["response-2","response-1","response-0"]);
});
it("rejects an inconsistent historical revision",async()=>{
 responses.loadCompletedResponse.mockResolvedValueOnce(detail).mockResolvedValueOnce({response:{...summary,id:"response-1",revision:2,current:false},revision:{id:"response-1",revision:2,current:false},answers:[],documents:[]}); render(<Responses id="response-2" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>); fireEvent.click(await screen.findByRole("button",{name:"Load earlier responses"})); expect(await screen.findByText("Earlier responses unavailable.")).toBeTruthy(); expect(screen.queryByRole("link",{name:"Response version 2"})).toBeNull();
});

it("clears previously loaded versions when earlier access is denied",async()=>{
 const first={...detail,response:{...summary,revision:3},revision:{id:"response-2",revision:3,current:true,supersedes_revision_id:"response-1"}};const middle={response:{...summary,id:"response-1",revision:2,current:false},revision:{id:"response-1",revision:2,current:false,supersedes_revision_id:"response-0"},answers:[],documents:[]};
 responses.loadCompletedResponse.mockResolvedValueOnce(first).mockResolvedValueOnce(middle).mockRejectedValueOnce(new ApiError(403,"Forbidden")); render(<Responses id="response-2" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>); fireEvent.click(await screen.findByRole("button",{name:"Load earlier responses"})); expect(await screen.findByRole("link",{name:"Response version 2"})).toBeTruthy(); fireEvent.click(screen.getByRole("button",{name:"Load earlier responses"})); expect(await screen.findByText("Earlier responses unavailable.")).toBeTruthy(); expect(screen.queryByRole("link",{name:"Response version 2"})).toBeNull();
});
it("discards an earlier response returned after changing selection",async()=>{
 let finish!: (value:unknown)=>void; responses.loadCompletedResponse.mockResolvedValueOnce(detail).mockReturnValueOnce(new Promise(resolve=>{finish=resolve})).mockResolvedValueOnce({response:{...summary,id:"new",revision:1},revision:{id:"new",revision:1,current:true},answers:[],documents:[]});
 const view=render(<Responses id="response-2" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>); fireEvent.click(await screen.findByRole("button",{name:"Load earlier responses"})); view.rerender(<Responses id="new" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>); await screen.findByRole("heading",{name:"Security review"}); await waitFor(()=>expect(responses.loadCompletedResponse).toHaveBeenCalledWith("new")); finish({response:{...summary,id:"response-1",revision:1,current:false},revision:{id:"response-1",revision:1,current:false},answers:[],documents:[]}); await waitFor(()=>expect(screen.queryByRole("link",{name:"Response version 1"})).toBeNull());
});

it("continues through a long history with a bounded moving window",async()=>{
 responses.loadCompletedResponse.mockImplementation(async(id:string)=>{const version=Number(id.split("-").at(-1));return {response:{...summary,id,revision:version,current:version===28},revision:{id,revision:version,current:version===28,...(version>1?{supersedes_revision_id:`response-${version-1}`}:{})},answers:[],documents:[]};});
 render(<Responses id="response-28" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
 for(let version=27;version>=2;version--){fireEvent.click(await screen.findByRole("button",{name:"Load earlier responses"}));await screen.findByRole("link",{name:`Response version ${version}`});}
 expect(screen.queryByRole("link",{name:"Response version 27"})).toBeNull(); expect(screen.getByRole("link",{name:"Response version 2"})).toBeTruthy();
 fireEvent.click(screen.getByRole("button",{name:"Load earlier responses"}));expect(await screen.findByRole("link",{name:"Response version 1"})).toBeTruthy();expect(screen.queryByRole("button",{name:"Load earlier responses"})).toBeNull();
});
