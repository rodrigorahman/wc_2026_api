-- name: CreateUser :one
INSERT INTO users (id, full_name, email, password_hash)
VALUES (?, ?, ?, ?)
RETURNING id, full_name, email, password_hash, created_at, temp_password_hash, temp_password_expires_at;

-- name: GetUserByEmail :one
SELECT id, full_name, email, password_hash, created_at, temp_password_hash, temp_password_expires_at
FROM users
WHERE email = ?;

-- name: GetUserByID :one
SELECT id, full_name, email, password_hash, created_at, temp_password_hash, temp_password_expires_at
FROM users
WHERE id = ?;

-- name: SetTempPassword :execrows
UPDATE users
SET temp_password_hash = ?, temp_password_expires_at = ?
WHERE id = ?;

-- name: UpdatePassword :execrows
UPDATE users
SET password_hash = ?, temp_password_hash = NULL, temp_password_expires_at = NULL
WHERE id = ?;

-- name: AddUserNationalTeam :exec
INSERT INTO user_national_teams (user_id, national_team_id)
VALUES (?, ?);

-- name: ListUserNationalTeams :many
SELECT national_team_id
FROM user_national_teams
WHERE user_id = ?;
