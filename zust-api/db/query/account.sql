-- name: CreateAccountWithPassword :one
INSERT INTO account (email, username, password)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateAccountWithOAuth :one
INSERT INTO account (email, username, status, oauth_provider, oauth_provider_id)
VALUES ($1, $2, 'active', $3, $4)
RETURNING *;

-- name: GetAccountByUsername :one
SELECT account_id, email, username, password, description, status, token_version FROM account
WHERE username = $1;

-- name: GetAccountByEmail :one
SELECT account_id, email, username, password, description, status, token_version FROM account
WHERE email = $1;

-- name: ActivateAccount :exec
UPDATE account
SET status = 'active'
WHERE account_id = $1;

-- name: LoginWithOAuth :one
SELECT account_id, email, username, description, status, token_version FROM account
WHERE oauth_provider = $1 AND oauth_provider_id = $2;

-- name: GetTokenVersion :one
SELECT token_version FROM account
WHERE account_id = $1;

-- name: IsAccountRegistered :one
SELECT EXISTS (
    SELECT 1 FROM account WHERE oauth_provider = $1 AND oauth_provider_id = $2
);

-- name: IncrementTokenVersion :exec
UPDATE account
SET token_version = token_version + 1
WHERE account_id = $1;

-- name: GetProfile :one
SELECT account_id, email, username, description, status FROM account
WHERE account_id = $1;

-- name: EditProfile :one
UPDATE account
SET username = $2, description = $3
WHERE account_id = $1
RETURNING account_id, email, username, description, status;

-- name: LockAccount :exec
UPDATE account
SET status = 'locked'
WHERE account_id = $1;

-- name: Subscribe :one
INSERT INTO subscribe (subscriber_id, subscribe_to_id)
VALUES ($1, $2)
RETURNING *;

-- name: Unsubscribe :exec
DELETE FROM subscribe
WHERE subscriber_id = $1 AND subscribe_to_id = $2;