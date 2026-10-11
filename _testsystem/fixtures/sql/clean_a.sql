-- name: GetUser
SELECT * FROM users WHERE id = $1;

-- name: ListUsers
SELECT * FROM users;

-- name: DeleteUser
DELETE FROM users WHERE id = $1;
