import assert from "node:assert/strict";
import test from "node:test";
import { peopleAccessEvidenceScenarios, peopleAccessViews } from "./people-access-evidence-scenarios.mjs";
import { peopleAccessCaptureFailures, peopleAccessComposerFailures, peopleAccessDefinitionFailures, peopleAccessFilterFailures, peopleAccessHistoryFailures, peopleAccessSourcePairFailures } from "./people-access-evidence-contract.mjs";
const rect=(x,y,width,height)=>({x,y,width,height,right:x+width,bottom:y+height});
function fixture(){
 const scenario=peopleAccessEvidenceScenarios.find((item)=>item.peopleAccessView==="people"&&item.peopleAccessCheck==="layout"&&item.viewport.width===1440&&item.theme==="light");
 const names=["Amina Yusuf","Chidi Okonkwo","Funke Adeyemi","Ibrahim Musa","Kemi Adebayo","Morgan Ellis","Nneka Okafor","Olumide Balogun","Sarah Ojo","Tunde Bello","Zainab Abubakar","Zara Nwosu"];
 const layout={evidenceKind:"SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION",kind:"people",route:scenario.route,headingText:"People & access",headingCount:1,heading:rect(260,100,1140,130),menu:rect(260,244,1140,50),localTabs:peopleAccessViews.map((view)=>view.label),selectedTab:"People",selectedBounds:rect(275,247,74,44),selectedVisible:true,visiblePanelIDs:["people-panel"],selectedPanelID:"people-panel",settingsCurrent:"page",legacyRail:false,documentWidth:1440,scrollWidth:1440,firstRow:rect(260,596,730,76),firstRowVisible:true,firstIdentity:rect(320,610,130,20),firstUsefulVisible:true,primary:rect(260,420,730,900),context:rect(1004,420,396,700),placement:"beside",contextScrollWidth:396,contextClientWidth:396,recordNames:names,recordCount:12,detailHeading:"Amina Yusuf",footerText:"12 of 12 loaded profiles",text:"People Up to 50 profiles across the organization",metrics:[12,2,2,2].map((value)=>({value:String(value),quality:"partial"})),targets:[],undersizedTargets:[],minTargetWidth:44,minTargetHeight:44};
 return {scenario,capture:{...scenario,peopleAccessLayout:layout,sticky:{scrollY:500,context:rect(1004,96,396,700),heading:rect(1024,151,320,25),bar:rect(240,0,1200,68)},observation:{reads:[{path:"/api/v1/access/overview",query:"?limit=50",status:200,counts:{people:12,groups:2,sources:2,roles:2},people:names.map((display_name,index)=>({id:String(index),display_name,status:"ACTIVE"}))}],commands:[]},errors:[]}};
}
test("People/access evidence covers all4source views at5widths and2themes",()=>{
 const matrix=peopleAccessEvidenceScenarios.filter((item)=>item.peopleAccessMatrix);assert.equal(matrix.length,40);assert.equal(new Set(peopleAccessEvidenceScenarios.map((item)=>item.name)).size,peopleAccessEvidenceScenarios.length);
 for(const view of peopleAccessViews)assert.equal(matrix.filter((item)=>item.peopleAccessView===view.id).length,10);
});
test("People/access contract rejects clipped firstidentity, wrongmenu, broadqueries and silently truncated directory",()=>{
 const {scenario,capture}=fixture();assert.deepEqual(peopleAccessCaptureFailures(capture,scenario),[]);
 for(const mutate of [(c)=>{c.peopleAccessLayout.firstUsefulVisible=false},(c)=>{c.peopleAccessLayout.firstRowVisible=false},(c)=>{c.sticky.context.y=20},(c)=>{c.peopleAccessLayout.localTabs.reverse()},(c)=>{c.peopleAccessLayout.selectedVisible=false},(c)=>{c.observation.reads[0].query="?limit=500"},(c)=>{c.peopleAccessLayout.recordNames=c.peopleAccessLayout.recordNames.slice(0,8)},(c)=>{c.peopleAccessLayout.metrics[0].value="500"},(c)=>{c.peopleAccessLayout.scrollWidth=1500},(c)=>{c.peopleAccessLayout.contextScrollWidth=500},(c)=>{c.observation.commands.push({method:"POST",path:"/api/v1/access/sources"})}]){const changed=structuredClone(capture);mutate(changed);assert.ok(peopleAccessCaptureFailures(changed,scenario).length);}
});
test("People/access source binding rejects mismatched view and responsive placement",()=>{
 const {scenario,capture}=fixture();const source={...capture.peopleAccessLayout};assert.deepEqual(peopleAccessSourcePairFailures(source,capture,scenario),[]);
 for(const mutate of [(s)=>{s.kind="roles"},(s)=>{s.localTabs=["People"]},(s)=>{s.placement="stacked"},(s)=>{s.primary=null}]){const changed=structuredClone(source);mutate(changed);assert.ok(peopleAccessSourcePairFailures(changed,capture,scenario).length);}
});
test("directory filters must search beyond old eighth record and preserve population scope",()=>{
 const {capture}=fixture();const names=capture.peopleAccessLayout.recordNames;const filters={search:"",snapshots:[{step:"loaded",names},{step:"search-twelfth",names:["Zara Nwosu"]},{step:"empty-search",names:[]},{step:"cleared",names},{step:"inactive",names:["Ibrahim Musa"]},{step:"source-not-returned",names:["Morgan Ellis"]},{step:"restored",names}].map((row)=>({...row,reads:1}))};assert.deepEqual(peopleAccessFilterFailures(filters),[]);filters.snapshots[1].names=[];assert.ok(peopleAccessFilterFailures(filters).length);
});
test("browser history cannot pass after replacing the document/menu or losing active tab containment",()=>{
 const steps=["delegations","back-routes","back-roles","back-people","forward-roles","selected-person","back-person","forward-person"];
 const history={queryless:peopleAccessViews.map(({id,route})=>({view:id,first:"First",second:"Second",restored:"First",querylessRoute:route,restoredRoute:route,selectedRoute:`${route}?record=second`})),snapshots:steps.map((step,index)=>({step,sameDocument:true,sameMenu:true,selectedVisible:true,...(index>=5?{search:"Zara",names:["Zara Nwosu"],detail:"Zara Nwosu"}:{})}))};assert.deepEqual(peopleAccessHistoryFailures(history),[]);
 for(const key of ["sameDocument","sameMenu","selectedVisible"]){const changed=structuredClone(history);changed.snapshots[2][key]=false;assert.ok(peopleAccessHistoryFailures(changed).length);}
 const stale=structuredClone(history);stale.queryless[2].restored="Second";assert.ok(peopleAccessHistoryFailures(stale).length);
});
test("composer evidence rejects lost focus, stale portals, query loss and unbounded candidate recovery",()=>{
 const scenario=peopleAccessEvidenceScenarios.find((item)=>item.peopleAccessView==="delegations"&&item.peopleAccessCheck==="composer"&&item.viewport.width===390);
 const route="#configure/authority/delegations?delegation=sample-delegation-risk";
 const composer={route,initialFocusInside:true,focusTrapped:true,escapeClosed:true,focusRestored:true,back:{route:"#configure/access?delegation=sample-delegation-risk",selected:"People",dialogCount:0},forward:{route,dialogCount:0},sheet:rect(12,0,378,844),heading:rect(36,88,310,48),headingText:"Create whole-entity delegation",clientWidth:378,scrollWidth:378,targets:[rect(36,164,310,44)],candidateRecovery:{disabledSubmit:true,text:"No eligible people were found for this responsibility in the current legal entity."}};
 const observation={reads:[{path:"/api/v1/governance/delegation-candidates",status:200,candidatesCount:0,query:"?responsibility=REVIEWER&limit=50"}]};
 assert.deepEqual(peopleAccessComposerFailures(composer,scenario,observation),[]);
 for(const mutate of [(c)=>{c.initialFocusInside=false},(c)=>{c.focusRestored=false},(c)=>{c.back.dialogCount=1},(c)=>{c.forward.route="#configure/authority/delegations"},(c)=>{c.candidateRecovery.disabledSubmit=false},(c)=>{c.scrollWidth=500}]){const changed=structuredClone(composer);mutate(changed);assert.ok(peopleAccessComposerFailures(changed,scenario,observation).length);}
 const changed=structuredClone(observation);changed.reads[0].query="?responsibility=REVIEWER&limit=500";assert.ok(peopleAccessComposerFailures(composer,scenario,changed).length);
});
test("policy disclosure preserves returned rule facts including zero and keeps independent review actions reachable",()=>{
 const scenario=peopleAccessEvidenceScenarios.find((item)=>item.peopleAccessCheck==="definition"&&item.viewport.width===390);
 const definition={rules:[{id:"review",responsibility:"REVIEWER",min_materiality:0,priority:10,selector:{kind:"ROLE",ref:"RISK_REVIEWER"}}],escalations:[{id:"overdue",trigger:"OVERDUE"}]};
 const records=[{id:"review-policy",definition,checksum:"f".repeat(64)}];
 const receipt={route:"#configure/authority?policy=review-policy",fullDefinition:definition,checksum:records[0].checksum,expandedAfterToggle:true,facts:[{responsibility:"Reviewer",fields:{"Rule reference":"review","Minimum materiality":"0",Priority:"10",Role:"RISK_REVIEWER"}}],region:{...rect(40,100,310,400),clientWidth:310,scrollWidth:310},visibleRule:rect(40,100,310,400),approvalEnabled:true,returnDisabledWithoutReason:true,actions:[{visible:true,width:300,height:44},{visible:true,width:300,height:44}]};
 assert.deepEqual(peopleAccessDefinitionFailures(receipt,scenario,records),[]);
 for(const mutate of [(r)=>{r.checksum="different"},(r)=>{r.fullDefinition={rules:[]}},(r)=>{r.facts[0].fields["Minimum materiality"]="Not returned"},(r)=>{r.actions[1].visible=false}]){const changed=structuredClone(receipt);mutate(changed);assert.ok(peopleAccessDefinitionFailures(changed,scenario,records).length);}
});
