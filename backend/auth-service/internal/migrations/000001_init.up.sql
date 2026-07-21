CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT UNIQUE,
    password TEXT,
    role TEXT,
    name TEXT,
    department TEXT,
    availability TEXT DEFAULT 'offline'
);

CREATE TABLE refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT,
    token TEXT UNIQUE
);
