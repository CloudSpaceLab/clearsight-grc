import { useEffect, useRef, useState, type FormEvent } from "react";
import { loadResponseAssessment, recordResponseAssessment, type ResponseAssessmentDetail } from "../formAssessmentApi";
import { apiErrorKind } from "../http";
import { timestampLabel } from "./recordedTime";

type Scope = {tenant:string; entity:string; actor:string};
type Attempt = {key:string; version:number; field:string; outcome:string; reason?:string};
type Stored = Omit<Attempt,"reason">;
function validFields(value: ResponseAssessmentDetail): boolean {
 if (!Array.isArray(value.fields)) return false;
 const ids = new Set<string>();
 return value.fields.every(item => {
  const f=item?.field; if (!f || typeof f.id!=="string" || !f.id || ids.has(f.id) || typeof f.label!=="string") return false;
  ids.add(f.id); const a=f.assessment;
  if (a && (!["NONE","MANUAL","AUTOMATIC","AUTOMATIC_REVIEW"].includes(a.mode) || (a.rubric!==undefined && (!Array.isArray(a.rubric) || a.rubric.some(r=>!r || typeof r.id!=="string" || !r.id || typeof r.label!=="string") || new Set(a.rubric.map(r=>r.id)).size!==a.rubric.length)))) return false;
  const d=item.decision; return !d || (typeof d.outcome_id==="string" && typeof d.rationale==="string" && (d.reviewer_display_name===undefined || typeof d.reviewer_display_name==="string"));
 });
}
function assessedResult(value: ResponseAssessmentDetail): {label:string; asOf?:string} | undefined {
 if (value.state==="NOT_REQUIRED") return undefined;
 if (value.version===0) return {label:"Assessment result pending."};
 const score=value.assessed_score;
 const counts=[value.required_count,value.reviewed_required_count,value.reviewed_count];
 if (counts.some(n=>!Number.isSafeInteger(n) || n<0) || value.reviewed_required_count>value.required_count || value.reviewed_required_count>value.reviewed_count || !score || (score.mode!=="RISK" && score.mode!=="COMPLIANCE") || typeof score.raw_score!=="number" || !Number.isFinite(score.raw_score) || score.raw_score<0 || score.raw_score>100 || (score.state!=="FINAL" && score.state!=="PROVISIONAL") || (score.state==="FINAL" && (score.final!==true || value.state!=="ASSESSED" || value.required_count!==value.reviewed_required_count || (value.required_count===0 && value.reviewed_count===0))) || (score.state==="PROVISIONAL" && (score.final!==false || (value.state!=="AWAITING_REVIEW" && value.state!=="IN_REVIEW"))) || !score.calculated_at) return {label:"Assessed score unavailable."};
 const time=timestampLabel(score.calculated_at); if(time==="time unavailable") return {label:"Assessed score unavailable."};
 return {label:`Assessed ${score.mode==="RISK" ? "risk" : "compliance"} score: ${score.raw_score} / 100 · ${score.state==="FINAL" ? "Final" : "Provisional"}`,asOf:time};
}
function readAttempt(key?:string): Attempt | null | undefined {
 if (!key) return undefined;
 try { const raw=sessionStorage.getItem(key); if (!raw) return undefined; const p=JSON.parse(raw) as Stored;
  if (!p || !/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(p.key) || !Number.isSafeInteger(p.version) || p.version<0 || typeof p.field!=="string" || !p.field || typeof p.outcome!=="string" || !p.outcome || Object.keys(p).some(k=>!['key','version','field','outcome'].includes(k))) return null;
  return p;
 } catch { return null }
}
export function ResponseAssessment({id,scope,current,formID,formVersion,revision}:{id:string;scope?:Scope;current:boolean;formID:string;formVersion:number;revision:number}) {
 const storageKey=scope?.tenant && scope.entity && scope.actor ? "clearsight:assessment:"+JSON.stringify([scope.tenant,scope.entity,scope.actor,id]) : undefined;
 const [data,setData]=useState<ResponseAssessmentDetail>(); const [state,setState]=useState<"loading"|"ready"|"error">("loading");
 const [pending,setPending]=useState<Attempt|null|undefined>(()=>readAttempt(storageKey));
 const [outcomes,setOutcomes]=useState<Record<string,string>>({}); const [reasons,setReasons]=useState<Record<string,string>>({});
 const [busy,setBusy]=useState(false); const [error,setError]=useState<""|"validation"|"unknown"|"forbidden"|"storage">(""); const [saved,setSaved]=useState(false); const lock=useRef(false); const generation=useRef(0);
 useEffect(()=>{const g=++generation.current; setState("loading");loadResponseAssessment(id).then(v=>{if(g!==generation.current)return;if(v.response_id!==id || !formID || v.form_template_id!==formID || !Number.isSafeInteger(formVersion) || formVersion<1 || v.form_template_version!==formVersion || !Number.isSafeInteger(v.version) || v.version<0 || !validFields(v)) {setState("error");return}setData(v);setState("ready")}).catch(()=>{if(g===generation.current)setState("error")});return()=>{generation.current++}},[id,formID,formVersion]);
 async function submit(attempt:Attempt, retry=false) {
  const field=data?.fields.find(x=>x.field.id===attempt.field); const reason=(attempt.reason ?? reasons[attempt.field] ?? "").trim();
  if (!storageKey || !scope || !data || !Number.isSafeInteger(revision) || revision<1 || !current || !data.current || !data.may_review || !field?.may_review || !field.field.assessment?.rubric?.some(r=>r.id===attempt.outcome) || !reason || new TextEncoder().encode(reason).length>4000 || busy || lock.current || error==="forbidden") return;
  lock.current=true;setBusy(true);setError("");const g=generation.current;
  try { const value=await recordResponseAssessment(id,{expected_version:attempt.version,decisions:[{field_id:attempt.field,outcome_id:attempt.outcome,rationale:reason}]},attempt.key);
   const receipt=value.fields?.find(x=>x.field.id===attempt.field)?.decision;
   if(value.response_id!==id || value.form_template_id!==formID || value.form_template_version!==formVersion || value.version!==attempt.version+1 || !validFields(value) || !Number.isSafeInteger(revision) || revision<1 || !receipt?.id || receipt.response_id!==id || receipt.response_revision!==revision || receipt.form_template_id!==formID || receipt.form_template_version!==formVersion || receipt.field_id!==attempt.field || receipt.outcome_id!==attempt.outcome || receipt.rationale!==reason || receipt.reviewer_id!==scope.actor) throw new Error("Assessment receipt unavailable");
   if(g!==generation.current)return;sessionStorage.removeItem(storageKey);setPending(undefined);setData(value);setSaved(true);setReasons({});setOutcomes({});
  } catch(e) { if(g!==generation.current)return;const kind=apiErrorKind(e);if(kind==="validation" && !retry) {try{sessionStorage.removeItem(storageKey);setPending(undefined);setError("validation")}catch{setError("storage")}} else if(kind==="forbidden" || kind==="unauthorized" || kind==="not_found" || kind==="conflict") setError("forbidden"); else setError("unknown") }
  finally {lock.current=false;if(g===generation.current)setBusy(false)}
 }
 function save(event:FormEvent,field:string) {event.preventDefault();if(!storageKey || !data || pending!==undefined || !outcomes[field] || !reasons[field]?.trim())return;try{const attempt:Attempt={key:crypto.randomUUID(),version:data.version,field,outcome:outcomes[field],reason:reasons[field].trim()};sessionStorage.setItem(storageKey,JSON.stringify({key:attempt.key,version:attempt.version,field:attempt.field,outcome:attempt.outcome}));setPending(attempt);setSaved(false);void submit(attempt)}catch{setError("storage")}}
 if(state==="loading") return <section className="v-panel v-field-assessment"><h2>Field assessments</h2><p>Loading assessments…</p></section>;
 if(state==="error" || !data) return <section className="v-panel v-field-assessment"><h2>Field assessments</h2><p>Assessments unavailable. Reload the response.</p></section>;
 const pendingField=pending && data.fields.find(x=>x.field.id===pending.field); const canAct=Boolean(storageKey && scope && Number.isSafeInteger(revision) && revision>0 && current && data.current && data.may_review); const result=assessedResult(data);
 return <section className="v-panel v-field-assessment"><h2>Field assessments</h2>{saved && <p role="status">Assessment saved.</p>}{result && pending===undefined && <div><h3>Assessed result</h3><p>{result.label}</p>{result.asOf && <p>Calculated {result.asOf}{!data.current && " · Earlier response"}</p>}</div>}{data.fields.filter(x=>x.field.assessment?.mode==="MANUAL" || x.field.assessment?.mode==="AUTOMATIC_REVIEW").length===0 && <p>No fields require assessment.</p>}
 {data.fields.filter(x=>x.field.assessment?.mode==="MANUAL" || x.field.assessment?.mode==="AUTOMATIC_REVIEW").map(({field,decision,may_review})=><div className="v-assessment-field" key={field.id}><h3>{field.label}</h3>{decision && <p>{field.assessment?.rubric?.find(x=>x.id===decision.outcome_id)?.label ?? "Outcome unavailable"} · {decision.reviewer_display_name || "Reviewer name unavailable"}</p>}{decision?.rationale && <p>{decision.rationale}</p>}
 {canAct && may_review && !field.assessment?.rubric?.length && <p>Assessment outcomes unavailable.</p>}{canAct && may_review && Boolean(field.assessment?.rubric?.length) && pending===undefined && <form onSubmit={e=>save(e,field.id)}><label>Outcome for {field.label}<select required value={outcomes[field.id]??""} onChange={e=>setOutcomes({...outcomes,[field.id]:e.target.value})}><option value="">Select outcome</option>{field.assessment?.rubric?.map(r=><option key={r.id} value={r.id}>{r.label}</option>)}</select></label><label>Reason for {field.label}<textarea required value={reasons[field.id]??""} onChange={e=>setReasons({...reasons,[field.id]:e.target.value})}/></label>{new TextEncoder().encode(reasons[field.id]??"").length>4000 && <p role="alert">Reason is too long. Shorten it.</p>}<button className="v-primary" disabled={busy || new TextEncoder().encode(reasons[field.id]??"").length>4000}>Save assessment</button></form>}</div>)}
 {!canAct && <p>Assessment decisions unavailable.</p>}{error==="validation" && <p role="alert">Assessment not saved. Check the outcome and reason.</p>}{error==="storage" && <p role="alert">Assessment not saved. Browser storage is unavailable.</p>}{pending===null && <p role="alert">Assessment status unavailable. Check assessment history before saving.</p>}
 {pending && <div>{busy ? <p role="status">Saving assessment…</p> : <p role="alert">Assessment status unknown. Retrying will submit the same assessment.</p>}{pendingField ? <><p>{pendingField.field.label} · {pendingField.field.assessment?.rubric?.find(r=>r.id===pending.outcome)?.label ?? "Outcome unavailable"}</p>{pending.reason ? <label>Reason for {pendingField.field.label}<textarea readOnly value={pending.reason}/></label> : <label>Original reason for {pendingField.field.label}<textarea value={reasons[pending.field]??""} onChange={e=>setReasons({...reasons,[pending.field]:e.target.value})}/></label>}{new TextEncoder().encode(pending.reason ?? reasons[pending.field]??"").length>4000 && <p role="alert">Reason is too long. Shorten it.</p>}<button className="v-primary" disabled={busy || !canAct || !pendingField.may_review || error==="forbidden" || !(pending.reason ?? reasons[pending.field]??"").trim() || new TextEncoder().encode(pending.reason ?? reasons[pending.field]??"").length>4000} onClick={()=>void submit(pending,true)}>Retry assessment</button></> : <p>Assessment unavailable. Reload the response.</p>}{error==="forbidden" && <p role="alert">Assessment unavailable. Reload to check access.</p>}</div>}</section>;
}









