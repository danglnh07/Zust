-- name: CreateAccountWithPassword :one
 INSERT INTO account (email, username, password)
 VALUES ($1, $2, $3)
 RETURNING *;

 -- name: CreateAccountWithOAuth :one
 INSERT INTO account (email, username, avatar, oauth_provider, oauth_provider_id)
 VALUES ($1, $2, $3, $4, $5)
 RETURNING *;

 -- name: LoginWithPassword :one
SELECT account_id, email, username, avatar, cover, description, token_version FROM account
WHERE email = $1 AND password = $2;

-- name: LoginWithOAuth :one
SELECT account_id, email, username, avatar, cover, description, token_version FROM account
WHERE oauth_provider = $1 AND oauth_provider_id = $2;