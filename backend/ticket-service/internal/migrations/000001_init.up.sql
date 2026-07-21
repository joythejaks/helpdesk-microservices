CREATE TABLE tickets (
    id BIGSERIAL PRIMARY KEY,
    title TEXT,
    description TEXT,
    user_id BIGINT,
    status TEXT DEFAULT 'open',
    priority TEXT DEFAULT 'Medium',
    requester TEXT,
    department TEXT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    assigned_agent_id BIGINT,
    assigned_at TIMESTAMPTZ,
    resolved_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,
    due_at TIMESTAMPTZ
);

CREATE TABLE ticket_status_histories (
    id BIGSERIAL PRIMARY KEY,
    ticket_id BIGINT,
    from_status TEXT,
    to_status TEXT,
    changed_by BIGINT,
    changed_at TIMESTAMPTZ
);

CREATE TABLE ticket_comments (
    id BIGSERIAL PRIMARY KEY,
    ticket_id BIGINT,
    author_id BIGINT,
    author_role TEXT,
    body TEXT,
    created_at TIMESTAMPTZ,
    is_internal BOOLEAN DEFAULT false
);

CREATE TABLE ticket_attachments (
    id BIGSERIAL PRIMARY KEY,
    ticket_id BIGINT,
    uploader_id BIGINT,
    filename TEXT,
    content_type TEXT,
    size BIGINT,
    data BYTEA,
    created_at TIMESTAMPTZ
);
