CREATE TABLE users
(
    id              UUID PRIMARY KEY NOT NULL,
    username        TEXT UNIQUE      NOT NULL,
    email           TEXT UNIQUE      NOT NULL,
    password        TEXT             NOT NULL,
    email_confirmed BOOL,
    created_at      TIMESTAMPTZ      NOT NULL,
    updated_at      TIMESTAMPTZ,
    deleted_at      TIMESTAMPTZ,
    cleaned_at      TIMESTAMPTZ
);

CREATE TABLE email_confirmations
(
    id         UUID PRIMARY KEY NOT NULL,
    user_id    UUID             NOT NULL REFERENCES users (id),
    code       TEXT UNIQUE      NOT NULL,
    created_at TIMESTAMPTZ      NOT NULL,
    expires_at TIMESTAMPTZ      NOT NULL
);

CREATE TABLE user_sessions
(
    id         UUID PRIMARY KEY NOT NULL,
    user_id    UUID             NOT NULL REFERENCES users (id),
    created_at TIMESTAMPTZ      NOT NULL,
    updated_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ      NOT NULL
);

CREATE TABLE password_reset_tokens
(
    id         UUID PRIMARY KEY NOT NULL,
    user_id    UUID             NOT NULL REFERENCES users (id),
    token      TEXT             NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ      NOT NULL,
    created_at TIMESTAMPTZ      NOT NULL
);

CREATE TABLE change_email_requests
(
    id         UUID PRIMARY KEY NOT NULL,
    user_id    UUID             NOT NULL REFERENCES users (id),
    new_email  VARCHAR(255)     NOT NULL,
    token      TEXT             NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ      NOT NULL,
    created_at TIMESTAMPTZ      NOT NULL
);

CREATE TABLE oauth_user_accounts
(
    id               uuid PRIMARY KEY NOT NULL,
    user_id          uuid             NOT NULL REFERENCES users (id),
    -- see /oauth/provider.go for providers enum
    provider         TEXT             NOT NULL,
    provider_user_id text             NOT NULL,
    created_at       TIMESTAMPTZ      NOT NULL,

    UNIQUE (provider, provider_user_id)
);

CREATE TABLE entities
(
    id              uuid PRIMARY KEY NOT NULL,
    public_id       TEXT             NOT NULL,
    owner_id        UUID             NOT NULL REFERENCES users (id),
    title           TEXT,
    summary_raw     TEXT,
    summary         TEXT,
    type            TEXT             NOT NULL,
    visibility      TEXT             NOT NULL,
    status          TEXT             NOT NULL,
    created_at      TIMESTAMPTZ      NOT NULL,
    updated_at      TIMESTAMPTZ,
    deleted_at      TIMESTAMPTZ
);

CREATE UNIQUE INDEX entities_public_id_type_uidx
    ON entities (public_id, type)
    WHERE deleted_at IS NULL;

CREATE TABLE pages
(
    id              uuid PRIMARY KEY NOT NULL REFERENCES entities (id),
    content_raw TEXT,
    content     TEXT
);

