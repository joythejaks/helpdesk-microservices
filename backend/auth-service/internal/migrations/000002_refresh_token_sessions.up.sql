-- A user can now hold several refresh-token sessions (one per device) instead
-- of exactly one. created_at orders sessions so the oldest can be evicted
-- when a user goes over the per-user cap; expires_at lets expired sessions be
-- pruned. Both are defaulted/nullable so existing rows stay valid (they keep
-- expires_at NULL, and the JWT's own exp still gates them) and pods still on
-- the previous version keep working during a rolling update.
ALTER TABLE refresh_tokens
    ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN expires_at TIMESTAMPTZ;

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);
