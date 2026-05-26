-- migrate:up

CREATE TABLE feature_flags (
    id uuid DEFAULT public.generate_ulid() NOT NULL PRIMARY KEY,
    device_token varchar,
    actor_id uuid REFERENCES actors(id),
    name varchar NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    -- Require at least one owner so a flag always belongs to a device or an actor.
    -- [Ja] フラグが必ずデバイスまたは actor のいずれかに属するよう、最低 1 つの所有者を必須にする。
    CONSTRAINT chk_feature_flags_identifier CHECK (device_token IS NOT NULL OR actor_id IS NOT NULL),
    UNIQUE (actor_id, name),
    UNIQUE (device_token, name)
);

CREATE INDEX idx_feature_flags_actor_id ON feature_flags(actor_id);
CREATE INDEX idx_feature_flags_name ON feature_flags(name);
CREATE INDEX idx_feature_flags_device_token ON feature_flags(device_token);

-- migrate:down

DROP INDEX IF EXISTS idx_feature_flags_device_token;
DROP INDEX IF EXISTS idx_feature_flags_name;
DROP INDEX IF EXISTS idx_feature_flags_actor_id;
DROP TABLE IF EXISTS feature_flags;
