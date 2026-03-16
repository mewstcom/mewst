-- migrate:up

DROP INDEX IF EXISTS idx_rate_limits_key_window_start;

-- migrate:down

CREATE INDEX idx_rate_limits_key_window_start ON rate_limits(key, window_start);
