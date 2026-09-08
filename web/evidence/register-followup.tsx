import { createRoot } from "react-dom/client";
import { FormProposalReview } from "../src/components/forms/FormProposalReview";
import type { FormTemplateProposal } from "../src/formsTypes";
import "../src/styles.css";
import "../src/design-system/index.css";
import "../src/forms-foundation.css";
import "../src/capture-inputs.css";
import "../src/ui-preferences.css";

const sections = Array.from({length:5},(_,i)=>({id:`finding-${i}`,title:`Assessment ${i<3?1:2} · Finding ${i<3?i+1:i-2}`,help:`Sample data. ${i<3?"Payment application, assessed 13 February 2026":"Terminal service, assessed 6 February 2026"}. Finding: Current independent security evidence has not been provided. Severity at assessment: Medium. Original deadline: 31 March 2026. Recorded status: Open.`}));
const fields = sections.flatMap((section)=>[
  {id:`${section.id}-response`,section_id:section.id,label:"Response to this finding",type:"long_text" as const,required:true,description:"Recorded risk or implication: Security weaknesses may remain undetected."},
  {id:`${section.id}-action`,section_id:section.id,label:"Remediation action or explanation",type:"long_text" as const,required:true,description:"Bank recommendation: Provide current independent security assessment evidence."},
  {id:`${section.id}-owner`,section_id:section.id,label:"Person responsible for the remediation",type:"short_text" as const,required:true},
  {id:`${section.id}-date`,section_id:section.id,label:"Proposed completion date",type:"date" as const,required:false},
  {id:`${section.id}-evidence`,section_id:section.id,label:"Supporting evidence",type:"file" as const,required:false},
]);
const proposal:FormTemplateProposal={id:"sample-followup",source_kind:"DOCUMENT",status:"REVIEW_REQUIRED",proposed_contract:{scoring_mode:"NONE",presentation:{default_mode:"CLASSIC",allow_mode_switch:true},sections,fields},field_changes:fields.map((field,i)=>({id:`change-${i}`,kind:"ADD_FIELD",field,anchor:{sheet:"Sample register",row_start:Math.floor(i/5)+2},confidence:1,group_id:i<15?"one":"two",group_label:i<15?"Assessment 1: Sample vendor — Payment application — 13 February 2026":"Assessment 2: Sample vendor — Terminal service — 6 February 2026"})),unresolved_items:[],provenance:{proposal_version:"FINDING_FOLLOW_UP_V1",source_document_id:"sample",source_sha256:"a".repeat(64),source_version:1,extraction_status:"EXTRACTED"},created_by:"sample",created_at:"2026-09-08T00:00:00Z",updated_at:"2026-09-08T00:00:00Z",version:2};
document.documentElement.dataset.theme=new URLSearchParams(location.search).get("theme")==="dark"?"dark":"light";
createRoot(document.getElementById("root")!).render(<main style={{maxWidth:1200,margin:"auto",padding:16}}><h1>Sample register follow-up preview</h1><p>Sample data for layout verification. Draft creation requires the application service.</p><FormProposalReview proposal={proposal} sourceTitle="Sample findings register" onProposalChange={()=>undefined}/></main>);
