-- name: ListNationalTeams :many
SELECT id, name, flag_url, code
FROM national_teams
ORDER BY name;

-- name: GetNationalTeamByID :one
SELECT id, name, flag_url, code
FROM national_teams
WHERE id = ?;
