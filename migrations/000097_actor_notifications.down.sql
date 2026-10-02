BEGIN;
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM actor_notifications) THEN
    RAISE EXCEPTION 'Actor notification delivery history must be retained';
  END IF;
END $$;
DROP TABLE actor_notifications;
COMMIT;
