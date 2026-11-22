import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, expect, it, vi } from 'vitest';
import { RequirementReviews } from './RequirementReviews';
import { ApiError } from '../http';
import type { ProgramAggregate } from '../types';
const api = vi.hoisted(() => ({ loadContext: vi.fn(), loadProgram: vi.fn(), loadRequirementReviews: vi.fn(), proposeRequirementReview: vi.fn(), decideRequirementReview: vi.fn() }));
vi.mock('../api', () => ({ loadContext: api.loadContext, loadProgram: api.loadProgram }));
vi.mock('./requirementReviewApi', () => api);
const requirement = { id: 'r1', code: 'R1', title: 'Review access', statement: 'Review access.', status: 'DRAFT' };
const program = { id: 'p1', tenant_id: 'bank', legal_entity_id: 'entity', owner_principal_id: 'owner', authority_principal_id: 'checker', status: 'ACTIVE', version: 4 };
const aggregate = { program, requirements: [requirement], applicability: [] } as unknown as ProgramAggregate;
const review = { id: 'review1', tenant_id: 'bank', program_id: 'p1', requirement_id: 'r1', proposed_by: 'owner', proposal_rationale: 'Check this requirement', status: 'PENDING', version: 1 };
function historyPage(page: number) {
  const items = Array.from({ length: 100 }, (_, index) => ({ ...review, id: `review-${String(page * 100 + index).padStart(5, '0')}`, status: 'REJECTED', version: 2 }));
  return { program_id: 'p1', program_version: 4, items, next_after_id: items[items.length - 1]!.id };
}
const operation = (command: string) => ({ command, can_act: true, label: '', responsibility: '', reason: '' });
const ownerOps = [operation('program.requirement.review.propose')];
const checkerOps = [operation('program.requirement.review.approve'), operation('program.requirement.review.reject')];
beforeEach(() => { vi.resetAllMocks(); api.loadContext.mockResolvedValue({ tenant: { id: 'bank' }, legal_entity: { id: 'entity' }, actor: { id: 'owner', kind: 'PERSON' } }); api.loadRequirementReviews.mockResolvedValue({ program_id: 'p1', program_version: 4, items: [], next_after_id: '' }); });
it('sends the exact draft for independent review at the current version', async () => {
  api.proposeRequirementReview.mockResolvedValue({ review, program: { ...program, version: 5 } });
  api.loadProgram.mockResolvedValue({ ...aggregate, program: { ...program, version: 5 } });
  const updated = vi.fn(); render(<RequirementReviews aggregate={aggregate} operations={ownerOps} onUpdated={updated}/>);
  fireEvent.click(await screen.findByRole('button', { name: 'Send Review access for review' }));
  fireEvent.change(screen.getByLabelText('Reason for review'), { target: { value: 'Check this requirement' } });
  fireEvent.click(screen.getByRole('button', { name: 'Send for review' }));
  expect(await screen.findByText('Requirement sent for review.')).toBeTruthy();
  expect(api.proposeRequirementReview).toHaveBeenCalledWith('p1', 'r1', 4, 'Check this requirement');
  expect(updated).toHaveBeenCalledTimes(1);
});
it('allows the independent authorized reviewer to approve the exact pending review', async () => {
  api.loadContext.mockResolvedValue({ tenant: { id: 'bank' }, legal_entity: { id: 'entity' }, actor: { id: 'checker', kind: 'PERSON' } });
  api.loadRequirementReviews.mockResolvedValue({ program_id: 'p1', program_version: 4, items: [review], next_after_id: '' });
  api.decideRequirementReview.mockResolvedValue({ review: { ...review, status: 'APPROVED', reviewed_by: 'checker', review_rationale: 'Reviewed', version: 2 }, program: { ...program, version: 5 } });
  api.loadProgram.mockResolvedValue({ ...aggregate, program: { ...program, version: 5 } });
  render(<RequirementReviews aggregate={aggregate} operations={checkerOps} onUpdated={() => {}}/>);
  fireEvent.click(await screen.findByRole('button', { name: 'Review Review access' }));
  expect(screen.getByText('Reason for review: Check this requirement')).toBeTruthy();
  fireEvent.change(screen.getByLabelText('Review reason'), { target: { value: 'Reviewed' } });
  fireEvent.click(screen.getByRole('button', { name: 'Approve requirement' }));
  expect(await screen.findByText('Requirement approved.')).toBeTruthy();
  expect(api.decideRequirementReview).toHaveBeenCalledWith('p1', 'review1', 4, 'approve', 'Reviewed');
});
it('records a rejection only against the selected pending review', async () => {
  api.loadContext.mockResolvedValue({ tenant: { id: 'bank' }, legal_entity: { id: 'entity' }, actor: { id: 'checker', kind: 'PERSON' } });
  api.loadRequirementReviews.mockResolvedValue({ program_id: 'p1', program_version: 4, items: [review], next_after_id: '' });
  api.decideRequirementReview.mockResolvedValue({ review: { ...review, status: 'REJECTED', reviewed_by: 'checker', review_rationale: 'Needs correction', version: 2 }, program: { ...program, version: 5 } });
  api.loadProgram.mockResolvedValue({ ...aggregate, program: { ...program, version: 5 } });
  render(<RequirementReviews aggregate={aggregate} operations={checkerOps} onUpdated={() => {}}/>);
  fireEvent.click(await screen.findByRole('button', { name: 'Review Review access' }));
  fireEvent.change(screen.getByLabelText('Review reason'), { target: { value: 'Needs correction' } });
  fireEvent.click(screen.getByRole('button', { name: 'Reject requirement' }));
  expect(await screen.findByText('Requirement rejected.')).toBeTruthy();
  expect(api.decideRequirementReview).toHaveBeenCalledWith('p1', 'review1', 4, 'reject', 'Needs correction');
});
it('does not offer a decision to the maker even if an operation permits it', async () => {
  api.loadRequirementReviews.mockResolvedValue({ program_id: 'p1', program_version: 4, items: [review], next_after_id: '' });
  render(<RequirementReviews aggregate={{ ...aggregate, program: { ...aggregate.program, authority_principal_id: 'owner' } }} operations={checkerOps} onUpdated={() => {}}/>);
  expect(await screen.findByText('Review pending')).toBeTruthy();
  expect(screen.queryByRole('button', { name: 'Review Review access' })).toBeNull();
});
it('does not offer a proposal when review history is incomplete', async () => {
  api.loadRequirementReviews.mockResolvedValue(historyPage(0));
  render(<RequirementReviews aggregate={aggregate} operations={ownerOps} onUpdated={() => {}}/>);
  expect(await screen.findByText('Review history incomplete.')).toBeTruthy();
  expect(screen.queryByRole('button', { name: /Send .* for review/ })).toBeNull();
});
it('rejects a first page whose continuation skips past its last row', async () => {
  api.loadRequirementReviews.mockResolvedValue({ ...historyPage(0), next_after_id: 'review-99999' });
  render(<RequirementReviews aggregate={aggregate} operations={ownerOps} onUpdated={() => {}}/>);
  expect(await screen.findByText('Requirement reviews unavailable.')).toBeTruthy();
  expect(screen.queryByRole('button', { name: /Send .* for review/ })).toBeNull();
});
it('rejects a first page with disordered review IDs', async () => {
  const page = historyPage(0); [page.items[0], page.items[1]] = [page.items[1]!, page.items[0]!];
  api.loadRequirementReviews.mockResolvedValue(page);
  render(<RequirementReviews aggregate={aggregate} operations={ownerOps} onUpdated={() => {}}/>);
  expect(await screen.findByText('Requirement reviews unavailable.')).toBeTruthy();
});
it('rejects a subsequent page whose continuation skips past its last row', async () => {
  api.loadRequirementReviews.mockResolvedValueOnce(historyPage(0)).mockResolvedValueOnce({ ...historyPage(1), next_after_id: 'review-99999' });
  render(<RequirementReviews aggregate={aggregate} operations={ownerOps} onUpdated={() => {}}/>);
  fireEvent.click(await screen.findByRole('button', { name: 'Load more reviews' }));
  expect(await screen.findByText('More review history unavailable. Try again.')).toBeTruthy();
  expect(screen.queryByRole('button', { name: /Send .* for review/ })).toBeNull();
});
it('rejects a subsequent page with disordered review IDs', async () => {
  const page = historyPage(1); [page.items[0], page.items[1]] = [page.items[1]!, page.items[0]!];
  api.loadRequirementReviews.mockResolvedValueOnce(historyPage(0)).mockResolvedValueOnce(page);
  render(<RequirementReviews aggregate={aggregate} operations={ownerOps} onUpdated={() => {}}/>);
  fireEvent.click(await screen.findByRole('button', { name: 'Load more reviews' }));
  expect(await screen.findByText('More review history unavailable. Try again.')).toBeTruthy();
  expect(screen.queryByRole('button', { name: /Send .* for review/ })).toBeNull();
});
it('loads the next scoped history page before allowing a proposal', async () => {
  api.loadRequirementReviews.mockResolvedValueOnce(historyPage(0)).mockResolvedValueOnce({ program_id: 'p1', program_version: 4, items: [], next_after_id: '' });
  render(<RequirementReviews aggregate={aggregate} operations={ownerOps} onUpdated={() => {}}/>);
  fireEvent.click(await screen.findByRole('button', { name: 'Load more reviews' }));
  expect(await screen.findByRole('button', { name: 'Send Review access for review' })).toBeTruthy();
  expect(api.loadRequirementReviews).toHaveBeenLastCalledWith('bank', 'p1', historyPage(0).next_after_id);
});
it('does not expose actions if a later review page loses access', async () => {
  const page = historyPage(0); page.items[0] = { ...page.items[0]!, status: 'PENDING', version: 1 };
  api.loadRequirementReviews.mockResolvedValueOnce(page).mockRejectedValueOnce(new ApiError(403, 'denied'));
  render(<RequirementReviews aggregate={aggregate} operations={ownerOps} onUpdated={() => {}}/>);
  expect(await screen.findByText('Review history incomplete.')).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'Load more reviews' }));
  expect(await screen.findByText('Requirement reviews unavailable.')).toBeTruthy();
  expect(screen.queryByText('Review history incomplete.')).toBeNull();
  expect(screen.queryByRole('button', { name: /Send .* for review/ })).toBeNull();
});
it('does not advance when a history page repeats its cursor', async () => {
  api.loadRequirementReviews.mockResolvedValueOnce(historyPage(0)).mockResolvedValueOnce(historyPage(0));
  render(<RequirementReviews aggregate={aggregate} operations={ownerOps} onUpdated={() => {}}/>);
  fireEvent.click(await screen.findByRole('button', { name: 'Load more reviews' }));
  expect(await screen.findByText('More review history unavailable. Try again.')).toBeTruthy();
  expect(screen.queryByRole('button', { name: /Send .* for review/ })).toBeNull();
});
it('rejects a later history page with another Program version', async () => {
  api.loadRequirementReviews.mockResolvedValueOnce(historyPage(0)).mockResolvedValueOnce({ program_id: 'p1', program_version: 5, items: [], next_after_id: '' });
  render(<RequirementReviews aggregate={aggregate} operations={ownerOps} onUpdated={() => {}}/>);
  fireEvent.click(await screen.findByRole('button', { name: 'Load more reviews' }));
  expect(await screen.findByText('More review history unavailable. Try again.')).toBeTruthy();
  expect(screen.queryByRole('button', { name: /Send .* for review/ })).toBeNull();
});
it('rejects a history row at or before the previous cursor', async () => {
  api.loadRequirementReviews.mockResolvedValueOnce(historyPage(0)).mockResolvedValueOnce({ program_id: 'p1', program_version: 4, items: [{ ...review, id: 'before', status: 'REJECTED' }], next_after_id: '' });
  render(<RequirementReviews aggregate={aggregate} operations={ownerOps} onUpdated={() => {}}/>);
  fireEvent.click(await screen.findByRole('button', { name: 'Load more reviews' }));
  expect(await screen.findByText('More review history unavailable. Try again.')).toBeTruthy();
  expect(screen.queryByRole('button', { name: /Send .* for review/ })).toBeNull();
});
it('requests a history page only once during synchronous repeated activation', async () => {
  api.loadRequirementReviews.mockResolvedValueOnce(historyPage(0)).mockReturnValueOnce(new Promise(() => {}));
  render(<RequirementReviews aggregate={aggregate} operations={ownerOps} onUpdated={() => {}}/>);
  const button = await screen.findByRole('button', { name: 'Load more reviews' });
  fireEvent.click(button); fireEvent.click(button);
  await vi.waitFor(() => expect(api.loadRequirementReviews).toHaveBeenCalledTimes(2));
});
it('can reach the end of a long history without an arbitrary page cap', async () => {
  for (let index = 0; index < 11; index++) api.loadRequirementReviews.mockResolvedValueOnce(historyPage(index));
  api.loadRequirementReviews.mockResolvedValueOnce({ program_id: 'p1', program_version: 4, items: [], next_after_id: '' });
  render(<RequirementReviews aggregate={aggregate} operations={ownerOps} onUpdated={() => {}}/>);
  for (let index = 0; index < 11; index++) {
    fireEvent.click(await screen.findByRole('button', { name: 'Load more reviews' }));
    await vi.waitFor(() => expect(api.loadRequirementReviews).toHaveBeenCalledTimes(index + 2));
  }
  expect(await screen.findByRole('button', { name: 'Send Review access for review' })).toBeTruthy();
});

it('moves focus into the review and returns it on cancellation', async () => {
  render(<RequirementReviews aggregate={aggregate} operations={ownerOps} onUpdated={() => {}}/>);
  const trigger = await screen.findByRole('button', { name: 'Send Review access for review' });
  trigger.focus(); fireEvent.click(trigger);
  expect(document.activeElement).toBe(screen.getByLabelText('Reason for review'));
  fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
  expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Send Review access for review' }));
});
it('prevents another proposal when its outcome is unconfirmed', async () => {
  api.proposeRequirementReview.mockRejectedValue(new Error('network'));
  const pending = vi.fn(); render(<RequirementReviews aggregate={aggregate} operations={ownerOps} onUpdated={() => {}} onPendingChange={pending}/>);
  fireEvent.click(await screen.findByRole('button', { name: 'Send Review access for review' }));
  fireEvent.change(screen.getByLabelText('Reason for review'), { target: { value: 'Check this requirement' } });
  fireEvent.click(screen.getByRole('button', { name: 'Send for review' }));
  expect(await screen.findByText(/Review status unknown/)).toBeTruthy();
  expect(screen.queryByRole('button', { name: 'Send for review' })).toBeNull();
  expect(api.proposeRequirementReview).toHaveBeenCalledTimes(1);
  expect(pending).toHaveBeenLastCalledWith(true);
});

it('withholds actions when the review page belongs to another Program version', async () => {
  api.loadRequirementReviews.mockResolvedValue({ program_id: 'p1', program_version: 3, items: [], next_after_id: '' });
  render(<RequirementReviews aggregate={aggregate} operations={ownerOps} onUpdated={() => {}}/>);
  expect(await screen.findByText('Requirement reviews unavailable.')).toBeTruthy();
  expect(screen.queryByRole('button', { name: /Send .* for review/ })).toBeNull();
});
it('retries the review history read after a temporary failure', async () => {
  api.loadRequirementReviews.mockRejectedValueOnce(new Error('unavailable')).mockResolvedValueOnce({ program_id: 'p1', program_version: 4, items: [], next_after_id: '' });
  render(<RequirementReviews aggregate={aggregate} operations={ownerOps} onUpdated={() => {}}/>);
  fireEvent.click(await screen.findByRole('button', { name: 'Try again' }));
  expect(await screen.findByRole('button', { name: 'Send Review access for review' })).toBeTruthy();
});

it('does not confirm a receipt for a different requirement', async () => {
  api.proposeRequirementReview.mockResolvedValue({ review: { ...review, requirement_id: 'other' }, program: { ...program, version: 5 } });
  const updated = vi.fn(); render(<RequirementReviews aggregate={aggregate} operations={ownerOps} onUpdated={updated}/>);
  fireEvent.click(await screen.findByRole('button', { name: 'Send Review access for review' }));
  fireEvent.change(screen.getByLabelText('Reason for review'), { target: { value: 'Check this requirement' } });
  fireEvent.click(screen.getByRole('button', { name: 'Send for review' }));
  expect(await screen.findByText(/Review status unknown/)).toBeTruthy();
  expect(updated).not.toHaveBeenCalled();
});

it('retains a confirmed review when the Program refresh fails', async () => {
  api.proposeRequirementReview.mockResolvedValue({ review, program: { ...program, version: 5 } });
  api.loadProgram.mockRejectedValue(new Error('unavailable'));
  render(<RequirementReviews aggregate={aggregate} operations={ownerOps} onUpdated={() => {}}/>);
  fireEvent.click(await screen.findByRole('button', { name: 'Send Review access for review' }));
  fireEvent.change(screen.getByLabelText('Reason for review'), { target: { value: 'Check this requirement' } });
  fireEvent.click(screen.getByRole('button', { name: 'Send for review' }));
  expect(await screen.findByText('Requirement sent for review.')).toBeTruthy();
  expect(await screen.findByText('Review recorded. Reload the Program to see the latest status.')).toBeTruthy();
  expect(screen.queryByRole('button', { name: /Send .* for review/ })).toBeNull();
});
