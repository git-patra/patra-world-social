CREATE TABLE result_jobs (
 match_id uuid PRIMARY KEY REFERENCES matches(id),
 result_hash bytea NOT NULL CHECK(octet_length(result_hash)=32),
 payload jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 completed_at timestamptz,
 attempts integer NOT NULL DEFAULT 0,
 next_attempt_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX result_jobs_pending ON result_jobs(next_attempt_at) WHERE completed_at IS NULL;
ALTER TABLE matches ADD COLUMN result_payload jsonb;
ALTER TABLE match_participants ADD COLUMN result_payload jsonb;
CREATE INDEX personal_history ON match_participants(user_id,match_id) WHERE user_id IS NOT NULL;
CREATE INDEX matches_history_order ON matches(prepared_at DESC,id DESC);
-- Retry metadata is mutable. The accepted result identity and body are not.
CREATE FUNCTION immutable_result_job() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.match_id IS DISTINCT FROM OLD.match_id OR NEW.result_hash IS DISTINCT FROM OLD.result_hash OR NEW.payload IS DISTINCT FROM OLD.payload THEN
  RAISE EXCEPTION 'immutable result job';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER immutable_result_job BEFORE UPDATE ON result_jobs FOR EACH ROW EXECUTE FUNCTION immutable_result_job();
