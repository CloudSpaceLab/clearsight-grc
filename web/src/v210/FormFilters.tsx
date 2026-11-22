import { useEffect, useState, type FormEvent } from "react";
import type { FormLibraryFacets, FormTemplateQuery } from "../formsTypes";
import type { LifecycleStatus } from "../monitoringTypes";

const statuses: ReadonlyArray<{value: LifecycleStatus; label: string}> = [
  {value:"DRAFT",label:"Draft"}, {value:"PENDING_APPROVAL",label:"Awaiting approval"}, {value:"ACTIVE",label:"Active"}, {value:"PAUSED",label:"Paused"}, {value:"REJECTED",label:"Rejected"}, {value:"RETIRED",label:"Retired"},
];
export function formStatusCounts(facets?: FormLibraryFacets) {
  const values = facets?.status; if (!values || typeof values !== "object" || Array.isArray(values)) return undefined;
  if (Object.entries(values).some(([key, value]) => !statuses.some((status) => status.value === key) || !Number.isSafeInteger(value) || (value as number) < 0)) return undefined;
  const sum = Object.values(values).reduce((total, value) => total + (value ?? 0), 0);
  return Number.isSafeInteger(sum) ? values : undefined;
}
export function FormFilters({query, facets, onChange}:{query:FormTemplateQuery; facets?:FormLibraryFacets; onChange:(query:FormTemplateQuery)=>void}) {
  const [search,setSearch]=useState(query.search ?? ""); const [tag,setTag]=useState(query.tag ?? "");
  useEffect(()=>{setSearch(query.search ?? "");setTag(query.tag ?? "");},[query.search,query.tag]);
  const counts=formStatusCounts(facets); const total=counts ? Object.values(counts).reduce((sum,value)=>sum+(value ?? 0),0) : undefined;
  function submit(event:FormEvent) {event.preventDefault();onChange({...query,search:search.trim() || undefined,tag:tag.trim() || undefined,cursor:undefined});}
  return <><form className="v-program-search v-form-filters" onSubmit={submit}><label>Find a form<input maxLength={200} value={search} onChange={(event)=>setSearch(event.target.value)} placeholder="Form name"/></label><label>Tag<input maxLength={200} value={tag} onChange={(event)=>setTag(event.target.value)}/></label><button>Search</button></form>
    <nav className="v-local-nav v-form-statuses" aria-label="Form status"><button type="button" aria-current={!query.status ? "page" : undefined} onClick={()=>onChange({...query,status:undefined,cursor:undefined})}>All{total === undefined ? "" : ` ${total}`}</button>{statuses.filter(({value})=>!counts || Boolean(counts[value]) || query.status===value).map(({value,label})=><button type="button" key={value} aria-current={query.status===value ? "page" : undefined} onClick={()=>onChange({...query,status:value,cursor:undefined})}>{label}{counts ? ` ${counts[value] ?? 0}` : ""}</button>)}</nav>
  </>;
}
