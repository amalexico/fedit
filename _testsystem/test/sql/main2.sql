-- name: GetUser
SELECT id, name, email FROM users WHERE id = $1;

-- name: ListUsers
SELECT id, name, email FROM users ORDER BY created_at DESC LIMIT 50;

-- name: DeleteUser
DELETE FROM users WHERE id = $1;
