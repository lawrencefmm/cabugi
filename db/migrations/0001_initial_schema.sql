CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE user_role AS ENUM (
  'user',
  'moderator',
  'admin'
);

CREATE TYPE problem_version_status AS ENUM (
  'draft',
  'in_review',
  'published',
  'archived'
);

CREATE TYPE submission_language AS ENUM (
  'cpp17',
  'python'
);

CREATE TYPE submission_status AS ENUM (
  'queued',
  'running',
  'accepted',
  'wrong_answer',
  'compile_error',
  'runtime_error',
  'time_limit_exceeded',
  'judge_failed'
);

CREATE TYPE submission_case_status AS ENUM (
  'accepted',
  'wrong_answer',
  'runtime_error',
  'time_limit_exceeded'
);

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  auth_subject TEXT NOT NULL UNIQUE,
  handle TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_roles (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role user_role NOT NULL,
  granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, role)
);

CREATE TABLE problems (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT NOT NULL UNIQUE,
  created_by_user_id UUID NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE problem_versions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  problem_id UUID NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
  version_number INTEGER NOT NULL,
  lifecycle_status problem_version_status NOT NULL DEFAULT 'draft',
  title TEXT NOT NULL,
  statement_markdown TEXT NOT NULL DEFAULT '',
  input_markdown TEXT NOT NULL DEFAULT '',
  output_markdown TEXT NOT NULL DEFAULT '',
  constraints_markdown TEXT NOT NULL DEFAULT '',
  notes_markdown TEXT NOT NULL DEFAULT '',
  examples JSONB NOT NULL DEFAULT '[]'::jsonb,
  time_limit_ms INTEGER NOT NULL,
  memory_limit_mb INTEGER NOT NULL,
  hidden_test_bundle_key TEXT NOT NULL,
  hidden_test_bundle_sha256 TEXT NOT NULL,
  created_by_user_id UUID NOT NULL REFERENCES users(id),
  reviewer_user_id UUID REFERENCES users(id),
  moderation_notes TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  submitted_for_review_at TIMESTAMPTZ,
  published_at TIMESTAMPTZ,
  archived_at TIMESTAMPTZ,
  UNIQUE (problem_id, version_number),
  CHECK (version_number > 0),
  CHECK (time_limit_ms > 0),
  CHECK (memory_limit_mb > 0)
);

CREATE UNIQUE INDEX problem_versions_one_published_per_problem_idx
  ON problem_versions (problem_id)
  WHERE lifecycle_status = 'published';

CREATE TABLE tags (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE problem_version_tags (
  problem_version_id UUID NOT NULL REFERENCES problem_versions(id) ON DELETE CASCADE,
  tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (problem_version_id, tag_id)
);

CREATE TABLE submissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  problem_version_id UUID NOT NULL REFERENCES problem_versions(id),
  language submission_language NOT NULL,
  source_code TEXT NOT NULL,
  status submission_status NOT NULL DEFAULT 'queued',
  queued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  compile_log_object_key TEXT,
  artifact_object_key TEXT,
  total_tests INTEGER NOT NULL DEFAULT 0,
  passed_tests INTEGER NOT NULL DEFAULT 0,
  CHECK (total_tests >= 0),
  CHECK (passed_tests >= 0),
  CHECK (passed_tests <= total_tests)
);

CREATE INDEX submissions_problem_version_id_idx ON submissions(problem_version_id);
CREATE INDEX submissions_user_id_queued_at_idx ON submissions(user_id, queued_at DESC);

CREATE TABLE submission_jobs (
  submission_id UUID PRIMARY KEY REFERENCES submissions(id) ON DELETE CASCADE,
  available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  claimed_at TIMESTAMPTZ,
  attempts INTEGER NOT NULL DEFAULT 0,
  last_error TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (attempts >= 0)
);

CREATE TABLE submission_results (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
  test_index INTEGER NOT NULL,
  verdict submission_case_status NOT NULL,
  execution_time_ms INTEGER NOT NULL DEFAULT 0,
  memory_bytes BIGINT NOT NULL DEFAULT 0,
  stdout_excerpt TEXT NOT NULL DEFAULT '',
  stderr_excerpt TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (submission_id, test_index),
  CHECK (test_index >= 0),
  CHECK (execution_time_ms >= 0),
  CHECK (memory_bytes >= 0)
);

CREATE INDEX submission_results_submission_id_idx ON submission_results(submission_id);
