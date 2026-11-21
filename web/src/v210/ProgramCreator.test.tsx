import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, expect, it, vi } from 'vitest';
import { ProgramCreator } from './ProgramCreator';
import { ApiError } from '../http';
const api = vi.hoisted(() => ({ createProgram: vi.fn(), loadProgramSetupCandidates: vi.fn() }));
vi.mock('../continuityCommands', () => api);
const context = { tenant: { id: 'bank', name: 'Bank' }, legal_entity: { id: 'entity', name: 'Bank' }, actor: { id: 'maker', name: 'Ada', kind: 'PERSON' }, mode: 'live' };
const candidates = { owner_candidates: [{ id: 'owner', display_name: 'Owner', kind: 'PERSON', role: 'Owner' }], approval_authority_candidates: [{ id: 'checker', display_name: 'Risk officer', kind: 'PERSON', role: 'CRO' }], has_more: false, generated_at: '2026-11-22T00:00:00Z' };
const program = { id: 'p1', tenant_id: 'bank', legal_entity_id: 'entity', code: 'DELIVERY', name: 'Delivery records', type: 'COMPLIANCE', owning_function: 'Logistics', owner_principal_id: 'owner', authority_principal_id: 'checker', status: 'DRAFT', version: 1, scope: { description: 'Delivery records' } };
beforeEach(() => { vi.resetAllMocks(); api.loadProgramSetupCandidates.mockResolvedValue(candidates); });
async function fill() {
  fireEvent.change(await screen.findByLabelText('Program name'), { target: { value: 'Delivery records' } });
  fireEvent.change(screen.getByLabelText('Program code'), { target: { value: 'DELIVERY' } });
  fireEvent.change(screen.getByLabelText('Program type'), { target: { value: 'COMPLIANCE' } });
  fireEvent.change(screen.getByLabelText('Owning function'), { target: { value: 'Logistics' } });
  fireEvent.change(screen.getByLabelText('Scope'), { target: { value: 'Delivery records' } });
  fireEvent.change(screen.getByLabelText('Program owner'), { target: { value: 'owner' } });
  fireEvent.change(screen.getByLabelText('Approval authority'), { target: { value: 'checker' } });
}
it('creates a scoped draft with explicitly selected owner and approver', async () => {
  api.createProgram.mockResolvedValue({ program, requirements: [], applicability: [] });
  render(<ProgramCreator context={context}/>); await fill();
  fireEvent.click(screen.getByRole('button', { name: 'Create program' }));
  expect(await screen.findByText('Program created.')).toBeTruthy();
  expect(api.createProgram).toHaveBeenCalledWith(expect.objectContaining({ code: 'DELIVERY', name: 'Delivery records', type: 'COMPLIANCE', owningFunction: 'Logistics', scopeDescription: 'Delivery records', ownerCandidateID: 'owner', approvalAuthorityCandidateID: 'checker' }), { tenantID: 'bank', legalEntityID: 'entity', actorID: 'maker' });
  expect(screen.getByRole('link', { name: 'Open program' }).getAttribute('href')).toBe('#/programs/p1');
});
it('does not offer creation when responsibility cannot be verified', async () => {
  api.loadProgramSetupCandidates.mockRejectedValue(new ApiError(403, 'denied'));
  render(<ProgramCreator context={context}/>);
  expect(await screen.findByText('Program creation unavailable.')).toBeTruthy();
  expect(screen.queryByRole('button', { name: 'Create program' })).toBeNull();
});
it('blocks another creation after an uncertain response and retains entered values', async () => {
  api.createProgram.mockRejectedValue(new Error('network'));
  render(<ProgramCreator context={context}/>); await fill();
  fireEvent.click(screen.getByRole('button', { name: 'Create program' }));
  expect(await screen.findByText(/Program creation status unknown/)).toBeTruthy();
  expect(screen.queryByRole('button', { name: 'Create program' })).toBeNull();
  expect((screen.getByLabelText('Program name') as HTMLInputElement).value).toBe('Delivery records');
  expect(api.createProgram).toHaveBeenCalledTimes(1);
});
it('does not confirm a receipt from another legal entity', async () => {
  api.createProgram.mockResolvedValue({ program: { ...program, legal_entity_id: 'other' }, requirements: [], applicability: [] });
  render(<ProgramCreator context={context}/>); await fill();
  fireEvent.click(screen.getByRole('button', { name: 'Create program' }));
  expect(await screen.findByText(/Program creation status unknown/)).toBeTruthy();
  expect(screen.queryByRole('link', { name: 'Open program' })).toBeNull();
});

it('requires separate owner and approval authority', async () => {
  api.loadProgramSetupCandidates.mockResolvedValue({ ...candidates, approval_authority_candidates: [{ id: 'owner', display_name: 'Owner', kind: 'PERSON' }] });
  render(<ProgramCreator context={context}/>);
  fireEvent.change(await screen.findByLabelText('Program owner'), { target: { value: 'owner' } });
  fireEvent.change(screen.getByLabelText('Approval authority'), { target: { value: 'owner' } });
  expect(screen.getByText('Choose different people for ownership and approval.')).toBeTruthy();
  expect((screen.getByRole('button', { name: 'Create program' }) as HTMLButtonElement).disabled).toBe(true);
  expect(api.createProgram).not.toHaveBeenCalled();
});

it('ignores a creation receipt after a scope change', async () => {
  let finish!: (value: unknown) => void;
  api.createProgram.mockReturnValue(new Promise((resolve) => { finish = resolve; }));
  const view = render(<ProgramCreator context={context}/>); await fill();
  fireEvent.click(screen.getByRole('button', { name: 'Create program' }));
  view.rerender(<ProgramCreator context={{ ...context, legal_entity: { id: 'other', name: 'Other bank' } }}/>);
  finish({ program, requirements: [], applicability: [] });
  expect(await screen.findByLabelText('Program name')).toBeTruthy();
  expect((screen.getByLabelText('Program name') as HTMLInputElement).value).toBe('');
  expect(screen.queryByText('Program created.')).toBeNull();
});

it('does not send duplicate creation commands', async () => {
  api.createProgram.mockReturnValue(new Promise(() => {}));
  render(<ProgramCreator context={context}/>); await fill();
  const form = screen.getByRole('button', { name: 'Create program' }).closest('form')!;
  fireEvent.submit(form); fireEvent.submit(form);
  expect(api.createProgram).toHaveBeenCalledTimes(1);
});
