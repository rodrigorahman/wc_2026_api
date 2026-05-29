ALTER TABLE users ADD COLUMN national_team_id TEXT NOT NULL REFERENCES national_teams (id) DEFAULT '';

CREATE INDEX idx_users_national_team_id ON users (national_team_id);

DROP TABLE user_national_teams;
