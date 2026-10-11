-- name: GetUser
SELECT * FROM users WHERE id = $1;

-- name: GetUser
SELECT * FROM users WHERE email = $1;
