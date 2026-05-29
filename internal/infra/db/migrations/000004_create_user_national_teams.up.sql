CREATE TABLE user_national_teams (
    user_id          TEXT NOT NULL REFERENCES users (id),
    national_team_id TEXT NOT NULL REFERENCES national_teams (id),
    PRIMARY KEY (user_id, national_team_id)
);

CREATE INDEX idx_user_national_teams_national_team_id ON user_national_teams (national_team_id);

DROP INDEX idx_users_national_team_id;

ALTER TABLE users DROP COLUMN national_team_id;
