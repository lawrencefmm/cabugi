ALTER TABLE submission_jobs
  ADD COLUMN lease_token UUID,
  ADD COLUMN lease_expires_at TIMESTAMPTZ;

UPDATE submission_jobs sj
SET claimed_at = NULL,
    available_at = 'infinity'::timestamptz
FROM submissions s
WHERE s.id = sj.submission_id
  AND s.status = 'judge_failed';

UPDATE submission_jobs
SET lease_expires_at = NOW()
WHERE claimed_at IS NOT NULL
  AND lease_expires_at IS NULL;

CREATE INDEX submission_jobs_available_at_created_at_idx ON submission_jobs(available_at, created_at);
CREATE INDEX submission_jobs_lease_expires_at_idx ON submission_jobs(lease_expires_at);
