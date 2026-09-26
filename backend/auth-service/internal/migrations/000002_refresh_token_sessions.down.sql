DROP INDEX IF EXISTS idx_refresh_tokens_user_id;

ALTER TABLE refresh_tokens
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS created_at;
