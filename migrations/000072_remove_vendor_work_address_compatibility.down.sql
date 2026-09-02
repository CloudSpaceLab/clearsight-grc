BEGIN;

DROP TRIGGER reject_retired_vendor_address_work_request_write ON third_party_work_requests;
DROP FUNCTION reject_retired_vendor_address_work_request();

COMMIT;
