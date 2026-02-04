-- migrate:up

CREATE TABLE rate_limits (
    id uuid DEFAULT public.generate_ulid() NOT NULL PRIMARY KEY,
    key VARCHAR NOT NULL,
    window_start TIMESTAMP WITH TIME ZONE NOT NULL,
    count INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(key, window_start)
);

CREATE INDEX idx_rate_limits_key_window_start ON rate_limits(key, window_start);
CREATE INDEX idx_rate_limits_window_start ON rate_limits(window_start);

-- migrate:down

DROP INDEX IF EXISTS idx_rate_limits_window_start;
DROP INDEX IF EXISTS idx_rate_limits_key_window_start;
DROP TABLE IF EXISTS rate_limits;
