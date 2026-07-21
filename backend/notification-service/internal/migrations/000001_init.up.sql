CREATE TABLE notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT,
    payload TEXT,
    read BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ
);

CREATE INDEX idx_notifications_user_id ON notifications (user_id);
