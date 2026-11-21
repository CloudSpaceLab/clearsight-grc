import { useEffect, useRef, useState, type FormEvent } from 'react';
import type { RuntimeContext } from '../api';
import { createProgram, loadProgramSetupCandidates, type ProgramSetupCandidates } from '../continuityCommands';
import { ApiError } from '../http';
import type { ProgramAggregate } from '../types';
import { programFilterQuery } from './programFilters';

function eligibleContext(context: RuntimeContext) { return Boolean(context.tenant.id && context.legal_entity.id && context.legal_entity.id !== '*' && context.actor.id && context.actor.kind?.toUpperCase() !== 'SERVICE'); }
function validCandidates(value: ProgramSetupCandidates) {
  return Array.isArray(value.owner_candidates) && Array.isArray(value.approval_authority_candidates) && typeof value.has_more === 'boolean' && [...value.owner_candidates, ...value.approval_authority_candidates].every((item) => Boolean(item.id && item.display_name));
}
export function ProgramCreateLink({ context }: { context: RuntimeContext }) {
  const [allowed, setAllowed] = useState(false);
  useEffect(() => { let active = true; setAllowed(false); if (eligibleContext(context)) void loadProgramSetupCandidates().then((value) => { if (active && validCandidates(value) && value.owner_candidates.length && value.approval_authority_candidates.length) setAllowed(true); }).catch(() => {}); return () => { active = false; }; }, [context.tenant.id, context.legal_entity.id, context.actor.id, context.actor.kind]);
  return allowed ? <a className="v-primary" href="#/programs/new">Create program</a> : null;
}
export function ProgramCreator({ context }: { context: RuntimeContext }) {
  const [candidates, setCandidates] = useState<ProgramSetupCandidates>(); const [load, setLoad] = useState<'loading' | 'ready' | 'error'>('loading'); const [attempt, setAttempt] = useState(0);
  const [name, setName] = useState(''); const [code, setCode] = useState(''); const [type, setType] = useState(''); const [owningFunction, setOwningFunction] = useState(''); const [scope, setScope] = useState(''); const [owner, setOwner] = useState(''); const [approver, setApprover] = useState('');
  const [busy, setBusy] = useState(false); const [error, setError] = useState(''); const [outcome, setOutcome] = useState<'' | 'unknown' | 'created'>(''); const [created, setCreated] = useState<ProgramAggregate>();
  const generation = useRef(0); const lock = useRef(false);
  useEffect(() => {
    const token = ++generation.current; setLoad('loading'); setCandidates(undefined); setName(''); setCode(''); setType(''); setOwningFunction(''); setScope(''); setOwner(''); setApprover(''); setOutcome(''); setCreated(undefined); setError(''); setBusy(false); lock.current = false;
    if (!eligibleContext(context)) { setLoad('error'); return () => { generation.current++; }; }
    loadProgramSetupCandidates().then((value) => { if (token !== generation.current) return; if (!validCandidates(value) || !value.owner_candidates.length || !value.approval_authority_candidates.length) { setLoad('error'); return; } setCandidates(value); setLoad('ready'); }).catch(() => { if (token === generation.current) setLoad('error'); });
    return () => { generation.current++; };
  }, [context.tenant.id, context.legal_entity.id, context.actor.id, context.actor.kind, attempt]);
  async function submit(event: FormEvent) {
    event.preventDefault();
    if (lock.current || load !== 'ready' || !candidates || outcome || ![name, code, type, owningFunction, scope, owner, approver].every((value) => value.trim()) || owner === approver || !candidates.owner_candidates.some((candidate) => candidate.id === owner) || !candidates.approval_authority_candidates.some((candidate) => candidate.id === approver)) return;
    const token = generation.current; lock.current = true; setBusy(true); setError('');
    try {
      const result = await createProgram({ name: name.trim(), code: code.trim(), type, owningFunction: owningFunction.trim(), scopeDescription: scope.trim(), ownerCandidateID: owner, approvalAuthorityCandidateID: approver }, { tenantID: context.tenant.id, legalEntityID: context.legal_entity.id, actorID: context.actor.id });
      if (token !== generation.current) return;
      const value = result.program;
      if (!value?.id || value.tenant_id !== context.tenant.id || value.legal_entity_id !== context.legal_entity.id || value.code !== code.trim().toUpperCase() || value.name !== name.trim() || value.type !== type || value.owning_function !== owningFunction.trim() || value.owner_principal_id !== owner || value.authority_principal_id !== approver || value.status !== 'DRAFT' || value.version !== 1 || value.scope?.description !== scope.trim()) throw new Error('unconfirmed');
      setCreated(result); setOutcome('created');
    } catch (failure) {
      if (token !== generation.current) return;
      if (failure instanceof ApiError && [400, 422].includes(failure.status)) { setError('Program not created. Check the details and try again.'); lock.current = false; }
      else { setOutcome('unknown'); setError('Program creation status unknown. Check the Program register before trying again.'); }
    } finally { if (token === generation.current) setBusy(false); }
  }
  return <section className="v-program-create"><a className="v-back" href="#/programs">← Programs</a><header className="v-page-heading"><div><p className="v-eyebrow">Programs</p><h1>Create program</h1></div></header>
    {load === 'loading' && <p role="status">Checking Program responsibilities…</p>}
    {load === 'error' && <div className="v-panel"><p role="alert">Program creation unavailable.</p><button type="button" onClick={() => setAttempt((value) => value + 1)}>Try again</button></div>}
    {outcome === 'created' && created && <div className="v-panel"><p role="status">Program created.</p><a className="v-primary" href={`#/programs/${encodeURIComponent(created.program.id)}`}>Open program</a></div>}
    {load === 'ready' && outcome !== 'created' && <form className="v-panel v-intake" onSubmit={(event) => void submit(event)}><p>All fields required.</p><fieldset disabled={busy || outcome === 'unknown'} className="v-program-create-fields">
      <label>Program name<input required value={name} onChange={(event) => setName(event.target.value)}/></label><label>Program code<input required value={code} onChange={(event) => setCode(event.target.value)}/></label>
      <label>Program type<select required value={type} onChange={(event) => setType(event.target.value)}><option value="">Select type</option><option value="COMPLIANCE">Compliance</option><option value="REGULATORY">Regulatory</option><option value="ASSURANCE">Assurance</option><option value="PRIVACY">Privacy</option><option value="THIRD_PARTY">Third party</option></select></label>
      <label>Owning function<input required value={owningFunction} onChange={(event) => setOwningFunction(event.target.value)}/></label><label>Scope<input required value={scope} onChange={(event) => setScope(event.target.value)}/></label>
      <label>Program owner<select required value={owner} onChange={(event) => setOwner(event.target.value)}><option value="">Select owner</option>{candidates?.owner_candidates.map((candidate) => <option key={candidate.id} value={candidate.id}>{candidate.display_name}{candidate.role ? ` · ${candidate.role}` : ''}</option>)}</select></label>
      <label>Approval authority<select required value={approver} onChange={(event) => setApprover(event.target.value)}><option value="">Select approval authority</option>{candidates?.approval_authority_candidates.map((candidate) => <option key={candidate.id} value={candidate.id}>{candidate.display_name}{candidate.role ? ` · ${candidate.role}` : ''}</option>)}</select></label>
    </fieldset>{candidates?.has_more && <p>Additional eligible people are not shown.</p>}{owner && owner === approver && <p role="alert">Choose different people for ownership and approval.</p>}{error && <p role="alert">{error}</p>}
    {outcome === 'unknown' ? <a href={`#/programs${programFilterQuery(code.trim(), '')}`}>Check Program register</a> : <div className="v-intake-actions"><button className="v-primary" disabled={busy || ![name, code, type, owningFunction, scope, owner, approver].every((value) => value.trim()) || owner === approver}>{busy ? 'Creating…' : 'Create program'}</button></div>}</form>}
  </section>;
}
