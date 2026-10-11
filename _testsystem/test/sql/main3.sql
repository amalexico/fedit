-- name: GetUser1
SELECT id, name, email FROM users WHERE id = $1;

-- name: ListUsers1
SELECT id, name, email FROM users ORDER BY created_at DESC LIMIT 50;

-- name: DeleteUser1
DELETE FROM users WHERE id = $1;
