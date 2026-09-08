import { useState } from "react";
import { FormPolicyEditor } from "./components/forms/FormPolicyEditor";
import { Notice } from "./components/ui";
import "./forms.css";
import "./components/forms/form-policies.css";

export function PolicyAssessmentEvidence(){
 const [saved,setSaved]=useState(false);
 return <main className="forms-workspace" style={{padding:"var(--space-5)"}}><Notice tone="info">Sample data · Vendor assessment policy configuration, 8 September 2026.</Notice>{saved&&<Notice tone="info">Sample policy draft saved. Simulation and separate approval are required before activation.</Notice>}<FormPolicyEditor forms={[{id:"sample-risk-form",name:"Vendor security review",code:"VENDOR-SECURITY",version:4,requiresBankAssessment:true}]} onCreate={()=>setSaved(true)} onCancel={()=>{window.location.href="/?tour=off&fixture=vendor-form-assessment#vendors"}}/></main>;
}
