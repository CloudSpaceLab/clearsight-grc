ALTER TABLE third_party_work_requests
    DROP CONSTRAINT third_party_work_requests_request_kind_check,
    ADD CONSTRAINT third_party_work_requests_request_kind_check
        CHECK (request_kind IN ('GENERAL','CERTIFICATION_REFRESH'));
