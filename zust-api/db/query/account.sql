-- name: CreateAccountWithPassword :one
INSERT INTO account (email, username, password)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateAccountWithOAuth :one
INSERT INTO account (email, username, avatar, status, oauth_provider, oauth_provider_id)
VALUES ($1, $2, $3, 'active', $4, $5)
RETURNING *;

-- name: GetAccountByUsername :one
SELECT account_id, email, username, password, avatar, cover, description, status, token_version FROM account
WHERE username = $1;

-- name: GetAccountByEmail :one
SELECT account_id, email, username, password, avatar, cover, description, status, token_version FROM account
WHERE email = $1;

-- name: ActivateAccount :exec
UPDATE account
SET status = 'active'
WHERE account_id = $1;

-- name: LoginWithOAuth :one
SELECT account_id, email, username, avatar, cover, description, status, token_version FROM account
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