-- migrate:up

-- River migration tables and job queue infrastructure
-- Based on River v0.30.x migrations

-- Migration tracking table
CREATE TABLE river_migration(
    line TEXT NOT NULL,
    version bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    CONSTRAINT line_length CHECK (char_length(line) > 0 AND char_length(line) < 128),
    CONSTRAINT version_gte_1 CHECK (version >= 1),
    PRIMARY KEY (line, version)
);

-- Job state enum
CREATE TYPE river_job_state AS ENUM(
    'available',
    'cancelled',
    'completed',
    'discarded',
    'pending',
    'retryable',
    'running',
    'scheduled'
);

-- Main job table
CREATE TABLE river_job(
    id bigserial PRIMARY KEY,
    state river_job_state NOT NULL DEFAULT 'available',
    attempt smallint NOT NULL DEFAULT 0,
    max_attempts smallint NOT NULL,
    attempted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    finalized_at timestamptz,
    scheduled_at timestamptz NOT NULL DEFAULT NOW(),
    priority smallint NOT NULL DEFAULT 1,
    args jsonb NOT NULL,
    attempted_by text[],
    errors jsonb[],
    kind text NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}',
    queue text NOT NULL DEFAULT 'default',
    tags varchar(255)[] NOT NULL DEFAULT '{}',
    unique_key bytea,
    unique_states BIT(8),

    CONSTRAINT finalized_or_finalized_at_null CHECK (
        (finalized_at IS NULL AND state NOT IN ('cancelled', 'completed', 'discarded')) OR
        (finalized_at IS NOT NULL AND state IN ('cancelled', 'completed', 'discarded'))
    ),
    CONSTRAINT max_attempts_is_positive CHECK (max_attempts > 0),
    CONSTRAINT priority_in_range CHECK (priority >= 1 AND priority <= 4),
    CONSTRAINT queue_length CHECK (char_length(queue) > 0 AND char_length(queue) < 128),
    CONSTRAINT kind_length CHECK (char_length(kind) > 0 AND char_length(kind) < 128)
);

CREATE INDEX river_job_kind ON river_job USING btree(kind);
CREATE INDEX river_job_state_and_finalized_at_index ON river_job USING btree(state, finalized_at) WHERE finalized_at IS NOT NULL;
CREATE INDEX river_job_prioritized_fetching_index ON river_job USING btree(state, queue, priority, scheduled_at, id);
CREATE INDEX river_job_args_index ON river_job USING GIN(args);
CREATE INDEX river_job_metadata_index ON river_job USING GIN(metadata);

-- Function for unique job state checking
CREATE OR REPLACE FUNCTION river_job_state_in_bitmask(bitmask BIT(8), state river_job_state)
RETURNS boolean
LANGUAGE SQL
IMMUTABLE
AS $$
    SELECT CASE state
        WHEN 'available' THEN get_bit(bitmask, 7)
        WHEN 'cancelled' THEN get_bit(bitmask, 6)
        WHEN 'completed' THEN get_bit(bitmask, 5)
        WHEN 'discarded' THEN get_bit(bitmask, 4)
        WHEN 'pending'   THEN get_bit(bitmask, 3)
        WHEN 'retryable' THEN get_bit(bitmask, 2)
        WHEN 'running'   THEN get_bit(bitmask, 1)
        WHEN 'scheduled' THEN get_bit(bitmask, 0)
        ELSE 0
    END = 1;
$$;

-- Unique job index
CREATE UNIQUE INDEX river_job_unique_idx ON river_job (unique_key)
    WHERE unique_key IS NOT NULL
      AND unique_states IS NOT NULL
      AND river_job_state_in_bitmask(unique_states, state);

-- Leader election table
CREATE UNLOGGED TABLE river_leader(
    elected_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    leader_id text NOT NULL,
    name text PRIMARY KEY DEFAULT 'default',
    CONSTRAINT name_length CHECK (name = 'default'),
    CONSTRAINT leader_id_length CHECK (char_length(leader_id) > 0 AND char_length(leader_id) < 128)
);

-- Queue configuration table
CREATE TABLE river_queue (
    name text PRIMARY KEY NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    metadata jsonb NOT NULL DEFAULT '{}' ::jsonb,
    paused_at timestamptz,
    updated_at timestamptz NOT NULL
);

-- Client tracking table
CREATE UNLOGGED TABLE river_client (
    id text PRIMARY KEY NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    metadata jsonb NOT NULL DEFAULT '{}',
    paused_at timestamptz,
    updated_at timestamptz NOT NULL,
    CONSTRAINT name_length CHECK (char_length(id) > 0 AND char_length(id) < 128)
);

-- Client queue tracking table
CREATE UNLOGGED TABLE river_client_queue (
    river_client_id text NOT NULL REFERENCES river_client (id) ON DELETE CASCADE,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    max_workers bigint NOT NULL DEFAULT 0,
    metadata jsonb NOT NULL DEFAULT '{}',
    num_jobs_completed bigint NOT NULL DEFAULT 0,
    num_jobs_running bigint NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (river_client_id, name),
    CONSTRAINT name_length CHECK (char_length(name) > 0 AND char_length(name) < 128),
    CONSTRAINT num_jobs_completed_zero_or_positive CHECK (num_jobs_completed >= 0),
    CONSTRAINT num_jobs_running_zero_or_positive CHECK (num_jobs_running >= 0)
);

-- Record that we've applied all River migrations up to version 6
INSERT INTO river_migration (line, version) VALUES
    ('main', 1),
    ('main', 2),
    ('main', 3),
    ('main', 4),
    ('main', 5),
    ('main', 6);

-- migrate:down

DROP TABLE IF EXISTS river_client_queue;
DROP TABLE IF EXISTS river_client;
DROP TABLE IF EXISTS river_queue;
DROP TABLE IF EXISTS river_leader;
DROP INDEX IF EXISTS river_job_unique_idx;
DROP FUNCTION IF EXISTS river_job_state_in_bitmask;
DROP INDEX IF EXISTS river_job_metadata_index;
DROP INDEX IF EXISTS river_job_args_index;
DROP INDEX IF EXISTS river_job_prioritized_fetching_index;
DROP INDEX IF EXISTS river_job_state_and_finalized_at_index;
DROP INDEX IF EXISTS river_job_kind;
DROP TABLE IF EXISTS river_job;
DROP TYPE IF EXISTS river_job_state;
DROP TABLE IF EXISTS river_migration;
