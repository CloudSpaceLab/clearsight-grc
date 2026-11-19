BEGIN;
ALTER TABLE program_assessment_schedules
 DROP CONSTRAINT program_assessment_schedule_interval_check,
 ADD CONSTRAINT program_assessment_schedules_interval_months_check CHECK (interval_months IN (1,3,6,12));
ALTER TABLE program_assessment_schedules DROP COLUMN interval_days;
COMMIT;
