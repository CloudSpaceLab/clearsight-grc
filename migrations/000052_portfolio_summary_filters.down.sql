BEGIN;
DROP INDEX IF EXISTS matters_due_portfolio_idx;
DROP INDEX IF EXISTS matters_owner_portfolio_idx;
DROP INDEX IF EXISTS matters_portfolio_filter_idx;
DROP INDEX IF EXISTS programs_portfolio_filter_idx;
COMMIT;
