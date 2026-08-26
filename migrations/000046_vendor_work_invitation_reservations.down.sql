BEGIN;

-- Reservation rows and their event history are security records. A code rollback
-- may stop creating them, but must not remove the durable association needed to
-- identify and revoke an invitation that was issued before finalization.

COMMIT;
