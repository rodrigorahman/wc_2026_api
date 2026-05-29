CREATE TABLE users (
    id               TEXT NOT NULL PRIMARY KEY,
    full_name        TEXT NOT NULL,
    email            TEXT NOT NULL UNIQUE,
    password_hash    TEXT NOT NULL,
    national_team_id TEXT NOT NULL REFERENCES national_teams (id),
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_national_team_id ON users (national_team_id);
