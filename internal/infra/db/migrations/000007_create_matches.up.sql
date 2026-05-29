CREATE TABLE matches (
    id           TEXT      NOT NULL PRIMARY KEY,
    kickoff      TIMESTAMP NOT NULL,
    home_team_id TEXT      NOT NULL REFERENCES national_teams (id),
    away_team_id TEXT      NOT NULL REFERENCES national_teams (id),
    stadium      TEXT      NOT NULL,
    city         TEXT      NOT NULL,
    stage        TEXT      NOT NULL,
    CHECK (home_team_id <> away_team_id)
);
