-- name: ListUpcomingMatchesByUser :many
SELECT
    m.id,
    m.kickoff,
    m.stadium,
    m.city,
    m.stage,
    home.id       AS home_id,
    home.name     AS home_name,
    home.flag_url AS home_flag_url,
    home.code     AS home_code,
    away.id       AS away_id,
    away.name     AS away_name,
    away.flag_url AS away_flag_url,
    away.code     AS away_code
FROM matches m
JOIN national_teams home ON home.id = m.home_team_id
JOIN national_teams away ON away.id = m.away_team_id
WHERE m.kickoff > sqlc.arg(cutoff)
  AND (
        m.home_team_id IN (SELECT national_team_id FROM user_national_teams WHERE user_national_teams.user_id = sqlc.arg(user_id))
     OR m.away_team_id IN (SELECT national_team_id FROM user_national_teams WHERE user_national_teams.user_id = sqlc.arg(user_id))
  )
ORDER BY m.kickoff ASC;
