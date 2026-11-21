import { beforeEach, expect, it, vi } from 'vitest';
import { decideRequirementReview, loadRequirementReviews, proposeRequirementReview } from './requirementReviewApi';
const mocks = vi.hoisted(() => ({ requestJSON: vi.fn(), continuityCommand: vi.fn() }));
vi.mock('../http', () => ({ requestJSON: mocks.requestJSON }));
vi.mock('../continuityCommands', () => ({ continuityCommand: mocks.continuityCommand }));
beforeEach(() => vi.resetAllMocks());
it('loads reviews for the scoped tenant and exact Program', async () => {
  await loadRequirementReviews('bank', 'program / 1');
  expect(mocks.requestJSON).toHaveBeenCalledWith('', '/api/v1/programs/program%20%2F%201/requirement-reviews?tenant_id=bank');
});
it('sends the exact draft and current Program version without a browser actor', async () => {
  await proposeRequirementReview('program / 1', 'requirement / 1', 4, 'Review this');
  expect(mocks.continuityCommand).toHaveBeenCalledWith('/api/v1/programs/program%20%2F%201/requirements/requirement%20%2F%201/review', { expected_version: 4, rationale: 'Review this' });
});
it.each(['approve', 'reject'] as const)('sends the %s decision to the exact review without a browser actor', async (action) => {
  await decideRequirementReview('program / 1', 'review / 1', 5, action, 'Decision reason');
  expect(mocks.continuityCommand).toHaveBeenCalledWith(`/api/v1/programs/program%20%2F%201/requirement-reviews/review%20%2F%201/${action}`, { expected_version: 5, rationale: 'Decision reason' });
});
