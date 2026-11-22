import { useEffect, useRef, useState } from 'react';
import { loadContext, loadProgram, type RuntimeContext } from '../api';
import { ApiError } from '../http';
import type { ProgramOperation } from '../programOperationsApi';
import type { ProgramAggregate } from '../types';
import { decideRequirementReview, loadRequirementReviews, proposeRequirementReview, type RequirementReview, type RequirementReviewPage } from './requirementReviewApi';

type Mode = { kind: 'propose'; requirement: string } | { kind: 'decide'; requirement: string; review: RequirementReview };
function validReviewPage(result: RequirementReviewPage, aggregate: ProgramAggregate) {
  return result.program_id === aggregate.program.id && result.program_version === aggregate.program.version && Array.isArray(result.items) && result.items.length <= 100 && typeof result.next_after_id === 'string' && (!result.next_after_id || (result.items.length === 100 && result.next_after_id.toLowerCase() === result.items[99]?.id?.toLowerCase())) && result.items.every((item, index) => Boolean(item.id && (index === 0 || result.items[index - 1]!.id.toLowerCase() < item.id.toLowerCase()) && item.tenant_id === aggregate.program.tenant_id && item.program_id === aggregate.program.id && item.requirement_id && item.proposed_by && typeof item.proposal_rationale === 'string' && item.proposal_rationale.trim() && ['PENDING', 'APPROVED', 'REJECTED'].includes(item.status) && Number.isSafeInteger(item.version) && item.version >= 1));
}
export function RequirementReviews({ aggregate, operations, onUpdated, disabled = false, onPendingChange }: { aggregate: ProgramAggregate; operations: ProgramOperation[]; onUpdated: (value: ProgramAggregate) => void; disabled?: boolean; onPendingChange?: (pending: boolean) => void }) {
  const [context, setContext] = useState<RuntimeContext>(); const [page, setPage] = useState<RequirementReviewPage>();
  const [loading, setLoading] = useState(true); const [attempt, setAttempt] = useState(0); const [error, setError] = useState(''); const [reason, setReason] = useState('');
  const [mode, setMode] = useState<Mode>(); const [busy, setBusy] = useState(false); const [notice, setNotice] = useState(''); const [locked, setLocked] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false); const [moreError, setMoreError] = useState('');
  const generation = useRef(0); const inFlight = useRef(false); const pageLock = useRef(false); const reasonInput = useRef<HTMLTextAreaElement>(null); const restoreFocus = useRef<string | null>(null);
  useEffect(() => { if (mode) reasonInput.current?.focus(); else if (restoreFocus.current) { const id = restoreFocus.current; restoreFocus.current = null; document.getElementById(id)?.focus(); } }, [mode]);
  useEffect(() => {
    const token = ++generation.current; setLoading(true); setPage(undefined); setContext(undefined); setMode(undefined); setLocked(false); setError(''); setMoreError(''); setLoadingMore(false); pageLock.current = false; inFlight.current = false;
    loadContext().then(async (actor) => ({ actor, result: await loadRequirementReviews(actor.tenant.id, aggregate.program.id) })).then(({ actor, result }) => {
      if (generation.current !== token) return;
      if (!actor.actor.id || actor.actor.kind?.toUpperCase() === 'SERVICE' || !actor.legal_entity.id || actor.legal_entity.id !== aggregate.program.legal_entity_id || !validReviewPage(result, aggregate)) { setError('Requirement reviews unavailable.'); return; }
      const pending = result.items.filter((item) => item.status === 'PENDING');
      if (new Set(pending.map((item) => item.requirement_id)).size !== pending.length || pending.some((item) => !aggregate.requirements.some((requirement) => requirement.id === item.requirement_id && requirement.status === 'DRAFT'))) { setError('Requirement reviews unavailable.'); return; }
      setContext(actor); setPage({ ...result, items: pending });
    }).catch(() => { if (generation.current === token) setError('Requirement reviews unavailable.'); }).finally(() => { if (generation.current === token) setLoading(false); });
    return () => { generation.current++; onPendingChange?.(false); };
  }, [aggregate.program.id, aggregate.program.version, aggregate.program.tenant_id, attempt]);
  const can = (command: string) => operations.some((operation) => operation.command === command && operation.can_act);
  const actor = context?.actor.id;
  const reasonTooLong = new TextEncoder().encode(reason.trim()).length > 2000;
  async function more() {
    if (!page || !context || !page.next_after_id || pageLock.current || disabled || locked || busy) return;
    const cursor = page.next_after_id; const token = generation.current; pageLock.current = true; setLoadingMore(true); setMoreError('');
    try {
      const latestActor = await loadContext();
      if (latestActor.tenant.id !== context.tenant.id || latestActor.legal_entity.id !== context.legal_entity.id || latestActor.actor.id !== context.actor.id || latestActor.actor.kind?.toUpperCase() === 'SERVICE') throw new ApiError(403, 'Scope changed');
      const next = await loadRequirementReviews(context.tenant.id, aggregate.program.id, cursor);
      if (token !== generation.current) return;
      const pending = [...page.items, ...next.items.filter((item) => item.status === 'PENDING')];
      if (!validReviewPage(next, aggregate) || (next.next_after_id && next.next_after_id.toLowerCase() <= cursor.toLowerCase()) || next.items.some((item) => item.id.toLowerCase() <= cursor.toLowerCase()) || new Set(pending.map((item) => item.id)).size !== pending.length || new Set(pending.map((item) => item.requirement_id)).size !== pending.length || pending.some((item) => !aggregate.requirements.some((requirement) => requirement.id === item.requirement_id && requirement.status === 'DRAFT'))) throw new Error('invalid page');
      setPage({ ...next, items: pending });
    } catch (failure) {
      if (token !== generation.current) return;
      if (failure instanceof ApiError && [401, 403].includes(failure.status)) { setPage(undefined); setContext(undefined); setError('Requirement reviews unavailable.'); }
      else setMoreError('More review history unavailable. Try again.');
    } finally { if (token === generation.current) { pageLock.current = false; setLoadingMore(false); } }
  }
  function open(next: Mode) { if (disabled || locked || busy) return; restoreFocus.current = `v-requirement-review-${encodeURIComponent(next.requirement)}`; onPendingChange?.(true); setMode(next); setReason(''); setError(''); setNotice(''); }
  async function submit(action: 'propose' | 'approve' | 'reject') {
    if (!mode || !context || !page || page.next_after_id || disabled || locked || inFlight.current || !reason.trim() || new TextEncoder().encode(reason.trim()).length > 2000) return;
    const requirement = aggregate.requirements.find((item) => item.id === mode.requirement && item.status === 'DRAFT');
    if (!requirement || (action === 'propose' ? mode.kind !== 'propose' || !can('program.requirement.review.propose') || actor !== aggregate.program.owner_principal_id || page.items.some((item) => item.requirement_id === requirement.id && item.status === 'PENDING') : mode.kind !== 'decide' || !can(`program.requirement.review.${action}`) || actor !== aggregate.program.authority_principal_id || mode.review.proposed_by === actor || mode.review.status !== 'PENDING')) return;
    const token = generation.current; inFlight.current = true; setBusy(true); setError(''); onPendingChange?.(true);
    let confirmed = false; let blocked = false; let responseVersion = 0;
    try {
      const result = action === 'propose' ? await proposeRequirementReview(aggregate.program.id, requirement.id, aggregate.program.version, reason.trim()) : await decideRequirementReview(aggregate.program.id, mode.kind === 'decide' ? mode.review.id : '', aggregate.program.version, action, reason.trim());
      if (token !== generation.current) return;
      const review = result.review; responseVersion = result.program.version;
      const valid = result.program.id === aggregate.program.id && responseVersion === aggregate.program.version + 1 && review.tenant_id === aggregate.program.tenant_id && review.program_id === aggregate.program.id && review.requirement_id === requirement.id && review.id && (action === 'propose' ? review.status === 'PENDING' && review.version === 1 && review.proposed_by === actor && review.proposal_rationale === reason.trim() : mode.kind === 'decide' && review.id === mode.review.id && review.proposed_by === mode.review.proposed_by && review.version === mode.review.version + 1 && review.reviewed_by === actor && review.review_rationale === reason.trim() && review.status === (action === 'approve' ? 'APPROVED' : 'REJECTED'));
      if (!valid) throw new Error('unconfirmed');
      confirmed = true; setMode(undefined); setLocked(true); setNotice(action === 'propose' ? 'Requirement sent for review.' : action === 'approve' ? 'Requirement approved.' : 'Requirement rejected.');
    } catch (failure) {
      if (token !== generation.current) return;
      if (failure instanceof ApiError && [400, 422].includes(failure.status)) setError('Review not saved. Check the reason and try again.');
      else if (failure instanceof ApiError && [403, 409].includes(failure.status)) { blocked = true; setMode(undefined); setLocked(true); setError('Review changed or access ended. Reload the Program.'); }
      else { blocked = true; setMode(undefined); setLocked(true); setError('Review status unknown. Reload the Program before trying again.'); }
    } finally { if (token === generation.current) { setBusy(false); if (!confirmed) { inFlight.current = blocked; onPendingChange?.(true); } } }
    if (!confirmed || token !== generation.current) return;
    try {
      const refreshed = await loadProgram(aggregate.program.id);
      if (token !== generation.current) return;
      if (refreshed.program.id !== aggregate.program.id || refreshed.program.version < responseVersion) throw new Error('stale');
      onPendingChange?.(false); onUpdated(refreshed);
    } catch { if (token === generation.current) setError('Review recorded. Reload the Program to see the latest status.'); }
  }
  if (loading) return <p role="status">Loading requirement reviews…</p>;
  return <section className="v-requirement-reviews" aria-label="Requirement reviews">
    {notice && <p role="status">{notice}</p>}{error && <p role="alert">{error}{!locked && !mode && <button type="button" onClick={() => setAttempt((value) => value + 1)}>Try again</button>}</p>}
    {locked && <button type="button" onClick={() => window.location.reload()}>Reload Program</button>}
    {page?.next_after_id && <p>Review history incomplete.</p>}
    {moreError && <p role="alert">{moreError}</p>}
    {page?.next_after_id && <button type="button" disabled={loadingMore || disabled || locked || busy} onClick={() => void more()}>{loadingMore ? 'Loading…' : 'Load more reviews'}</button>}
    {page && !page.next_after_id && aggregate.requirements.filter((requirement) => requirement.status === 'DRAFT').map((requirement) => {
      const pending = page.items.find((item) => item.requirement_id === requirement.id && item.status === 'PENDING');
      const available = !disabled && !locked && !busy && !mode && !page.next_after_id;
      return <div key={requirement.id}><strong>{requirement.title}</strong>{pending && <><p>Review pending</p><p>Reason for review: {pending.proposal_rationale}</p></>}
        {available && !pending && actor === aggregate.program.owner_principal_id && can('program.requirement.review.propose') && <button id={`v-requirement-review-${encodeURIComponent(requirement.id)}`} type="button" onClick={() => open({ kind: 'propose', requirement: requirement.id })}>Send {requirement.title} for review</button>}
        {available && pending && actor === aggregate.program.authority_principal_id && actor !== pending.proposed_by && (can('program.requirement.review.approve') || can('program.requirement.review.reject')) && <button id={`v-requirement-review-${encodeURIComponent(requirement.id)}`} type="button" onClick={() => open({ kind: 'decide', requirement: requirement.id, review: pending })}>Review {requirement.title}</button>}
      </div>;
    })}
    {mode && !locked && <form onSubmit={(event) => { event.preventDefault(); if (mode.kind === 'propose') void submit('propose'); }}><label>{mode.kind === 'propose' ? 'Reason for review' : 'Review reason'}<textarea ref={reasonInput} required maxLength={2000} value={reason} disabled={busy} onChange={(event) => setReason(event.target.value)}/></label>{reasonTooLong && <p role="alert">Reason is too long.</p>}<div className="v-program-requirement-actions">{mode.kind === 'propose' ? <button type="submit" disabled={busy || disabled || !reason.trim() || reasonTooLong}>Send for review</button> : <>{can('program.requirement.review.approve') && <button type="button" disabled={busy || disabled || !reason.trim() || reasonTooLong} onClick={() => void submit('approve')}>Approve requirement</button>}{can('program.requirement.review.reject') && <button type="button" disabled={busy || disabled || !reason.trim() || reasonTooLong} onClick={() => void submit('reject')}>Reject requirement</button>}</>}<button type="button" disabled={busy} onClick={() => { setMode(undefined); onPendingChange?.(false); }}>Cancel</button></div></form>}
  </section>;
}
