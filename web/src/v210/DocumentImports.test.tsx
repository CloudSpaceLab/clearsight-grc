import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { DocumentImports } from "./DocumentImports";
const api = vi.hoisted(() => ({ loadDocumentImports: vi.fn(), loadDocumentImport: vi.fn() }));
vi.mock("../documentApi", () => api);
beforeEach(() => vi.clearAllMocks());
it("opens the exact imported document and shows extracted sections and limitations", async () => {
 api.loadDocumentImports.mockResolvedValue([{ id:"doc/1", file_name:"Policy.pdf", extraction_status:"PARTIAL", updated_at:"2026-11-20T12:00:00Z" }]);
 api.loadDocumentImport.mockResolvedValue({id:"doc/1",file_name:"Policy.pdf", extraction_status:"PARTIAL", sections:[{id:"s1",title:"Access",text:"Review privileged access."}], limitations:["Some pages need OCR."], sections_omitted:1});
 render(<DocumentImports/>);
 fireEvent.click(await screen.findByRole("button", {name:"Open Policy.pdf"}));
 expect(await screen.findByText("Review privileged access.")).toBeTruthy();
 expect(screen.getByText("Some pages need OCR.")).toBeTruthy();
 expect(screen.getByText("1 section omitted.")).toBeTruthy();
 expect(api.loadDocumentImport).toHaveBeenCalledWith("doc/1");
});
it("shows a bounded empty result and recovers from a read failure", async () => {
 api.loadDocumentImports.mockRejectedValueOnce(new Error("denied")).mockResolvedValueOnce([]);
 render(<DocumentImports/>);
 expect(await screen.findByRole("alert")).toBeTruthy();
 fireEvent.click(screen.getByRole("button",{name:"Try again"}));
 expect(await screen.findByText("No imported documents in the current scope.")).toBeTruthy();
});
it("does not show stale detail after selecting another document", async () => {
 api.loadDocumentImports.mockResolvedValue([{id:"a",file_name:"A.txt",extraction_status:"EXTRACTED"},{id:"b",file_name:"B.txt",extraction_status:"EXTRACTED"}]);
 let resolveFirst!: (value: unknown) => void;
 api.loadDocumentImport.mockImplementation((id: string) => id === "a" ? new Promise((resolve)=>{resolveFirst=resolve;}) : Promise.resolve({id:"b",file_name:"B.txt",extraction_status:"EXTRACTED",sections:[],limitations:[]}));
 render(<DocumentImports/>);
 fireEvent.click(await screen.findByRole("button",{name:"Open A.txt"}));
 fireEvent.click(screen.getByRole("button",{name:"Open B.txt"}));
 expect(await screen.findByRole("heading",{name:"B.txt"})).toBeTruthy();
 resolveFirst({id:"a",file_name:"A.txt",sections:[],limitations:[]});
 await vi.waitFor(()=>expect(screen.queryByRole("heading",{name:"A.txt"})).toBeNull());
});
