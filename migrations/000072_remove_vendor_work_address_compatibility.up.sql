BEGIN;

-- ADDRESS_VERIFICATION rows may exist as immutable history from the retired
-- Vendor Work journey. Keep those records reconstructable, but fail every new
-- write through that retired path. The canonical journey is now a Matter,
-- Action and internal Evidence Request.
CREATE FUNCTION reject_retired_vendor_address_work_request()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.request_kind = 'ADDRESS_VERIFICATION' THEN
        IF TG_OP = 'UPDATE'
           AND OLD.request_kind = 'ADDRESS_VERIFICATION'
           AND OLD.state NOT IN ('ACCEPTED','CANCELLED')
           AND NEW.state = 'CANCELLED' THEN
            RETURN NEW;
        END IF;
        RAISE EXCEPTION 'ADDRESS_VERIFICATION vendor work requests are retired; use the canonical Matter evidence request journey'
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER reject_retired_vendor_address_work_request_write
BEFORE INSERT OR UPDATE ON third_party_work_requests
FOR EACH ROW EXECUTE FUNCTION reject_retired_vendor_address_work_request();

COMMIT;
