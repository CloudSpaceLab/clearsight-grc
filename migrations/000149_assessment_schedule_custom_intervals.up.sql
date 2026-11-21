BEGIN;
ALTER TABLE program_assessment_schedules
 DROP CONSTRAINT program_assessment_schedules_interval_months_check,
 ADD COLUMN interval_days integer NOT NULL DEFAULT 0,
 ADD CONSTRAINT program_assessment_schedule_interval_check CHECK (
  (interval_months BETWEEN 1 AND 24 AND interval_days = 0)
  OR (interval_months = 0 AND interval_days BETWEEN 1 AND 365)
 );
COMMIT;
