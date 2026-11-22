import { useEffect, useRef, useState, type FormEvent } from 'react';
import { recordMatterOutcomeResult } from '../matterOperationsApi';
import { apiErrorKind } from '../http';

type Issue = { id:string; tenant_id:string; legal_entity_id?:string; version:number };
type Check = { id:string; expected_outcome:string; status:string; failure_response?:string };
type Props = { matter:Issue; contract:Check; principalID:string; existingResultIDs:string[]; onRefresh:()=>void; onAccessRevoked:()=>void };
const consequences:Record<string,string> = { BLOCK_CLOSE:'The issue cannot close until the outcome is achieved.', REOPEN:'The issue will reopen.', CREATE_MATTER:'A follow-up issue will be created.', ESCALATE:'The issue will be escalated.' };
export function MatterOutcomeResult({matter,contract,principalID,existingResultIDs,onRefresh,onAccessRevoked}:Props) {
 const [open,setOpen]=useState(false),[outcome,setOutcome]=useState(''),[notes,setNotes]=useState(''),[busy,setBusy]=useState(false),[unknown,setUnknown]=useState(false),[refreshFailed,setRefreshFailed]=useState(false);
 const locked=useRef(false),mounted=useRef(true);
 useEffect(()=>{mounted.current=true;return ()=>{mounted.current=false}},[]);
 async function submit(event:FormEvent) {
  event.preventDefault();if(locked.current||busy||!principalID||!matter.legal_entity_id||contract.status!=='ACTIVE'||!['PASS','FAIL','INCONCLUSIVE'].includes(outcome)||!notes.trim()||(outcome==='FAIL' && !Object.hasOwn(consequences,contract.failure_response ?? '')))return;
  locked.current=true;setBusy(true);
  try {const receipt=await recordMatterOutcomeResult(matter.id,matter.version,{contractID:contract.id,result:outcome,rationale:notes.trim()});if(!mounted.current)return;
   const added=Array.isArray(receipt.verification_results)?receipt.verification_results.filter((entry)=>!existingResultIDs.includes(entry.id)):[];const value=added[0];
   if(receipt.matter?.id!==matter.id||receipt.matter.tenant_id!==matter.tenant_id||receipt.matter.legal_entity_id!==matter.legal_entity_id||!Number.isSafeInteger(receipt.matter.version)||receipt.matter.version<=matter.version||receipt.matter.version>matter.version+3||added.length!==1||!value?.id||value.tenant_id!==matter.tenant_id||value.matter_id!==matter.id||value.contract_id!==contract.id||value.result!==outcome||value.rationale!==notes.trim()||value.reviewer_principal_id!==principalID)throw new Error('Unconfirmed');
   try {onRefresh();setOpen(false)} catch {setRefreshFailed(true)}
  } catch (error) {if(!mounted.current)return;if(['unauthorized','forbidden','not_found'].includes(apiErrorKind(error)))onAccessRevoked();else setUnknown(true)} finally {if(mounted.current)setBusy(false)}
 }
 if(refreshFailed)return <p role="alert">Result saved. Refresh the issue to see it.</p>;
 if(unknown)return <><p role="alert">Result status unknown. Review the issue before trying again.</p><button type="button" onClick={onRefresh}>Review issue</button></>;
 if(!open)return <button type="button" onClick={()=>setOpen(true)}>Record result</button>;
 return <form className="v-intake" onSubmit={(event)=>void submit(event)}><h4>Record result</h4><label>Outcome<select value={outcome} onChange={(event)=>setOutcome(event.target.value)} required><option value="">Select outcome</option><option value="PASS">Outcome achieved</option><option value="FAIL">Outcome not achieved</option><option value="INCONCLUSIVE">More review needed</option></select></label>{outcome==='FAIL' && <p>{consequences[contract.failure_response ?? ""] ?? 'Failure response unavailable.'}</p>}<label>Review notes<textarea value={notes} onChange={(event)=>setNotes(event.target.value)} required /></label><div className="v-intake-actions"><button type="button" disabled={busy} onClick={()=>setOpen(false)}>Cancel</button><button className="v-primary" disabled={busy || (outcome==='FAIL' && !Object.hasOwn(consequences,contract.failure_response ?? ''))}>{busy?'Saving…':'Save result'}</button></div></form>;
}
