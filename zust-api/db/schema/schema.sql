-- Create enum
CREATE TYPE account_status AS ENUM ('active', 'ban', 'locked');
CREATE TYPE account_role AS ENUM ('user', 'admin');

-- Create table account
CREATE TABLE IF NOT EXISTS account (
    account_id UUID PRIMARY KEY DEFAULT gen_random_UUID(),
    email VARCHAR(40) NOT NULL,
    username VARCHAR(20) NOT NULL,
    password VARCHAR(60), -- BCrypt hashing generates 60 characters
    avatar VARCHAR(50) NOT NULL DEFAULT 'avatar.png',
    cover VARCHAR(50) NOT NULL DEFAULT 'cover_image.png',
    description VARCHAR(100),
    status account_status NOT NULL DEFAULT account_status('active'),
    role account_role NOT NULL DEFAULT account_role('user'),
    -- OAuth2-specific fields
    oauth_provider VARCHAR(10), -- 'google', 'github'
    oauth_provider_id VARCHAR(25), -- the user ID from provider
    -- JWT token version: used for ban/logout everywhere
    token_version INT NOT NULL DEFAULT 1
);