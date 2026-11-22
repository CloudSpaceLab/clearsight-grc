import { expect,it,vi } from "vitest";
import { loadMatter } from "./api";
const http=vi.hoisted(()=>({requestJSON:vi.fn(),requestVoid:vi.fn()}));vi.mock("./http",()=>http);
it("preserves whether outcome arrays were actually returned",async()=>{
 http.requestJSON.mockImplementation((_base:string,path:string)=>path==="/api/v1/context"?Promise.resolve({tenant:{id:"bank"},legal_entity:{id:"entity"},actor:{id:"person"}}):Promise.resolve({matter:{id:"m1"},verification_contracts:null,verification_results:null}));
 expect((await loadMatter("m1")).outcome_data_complete).toBe(true);
 http.requestJSON.mockImplementation((_base:string,path:string)=>path==="/api/v1/context"?Promise.resolve({tenant:{id:"bank"}}):Promise.resolve({matter:{id:"m1"}}));
 expect((await loadMatter("m1")).outcome_data_complete).toBe(false);
});
