-- Create enum
CREATE TYPE account_status AS ENUM ('inactive', 'active', 'banned', 'locked');

-- Create table account
CREATE TABLE IF NOT EXISTS account (
    account_id UUID PRIMARY KEY DEFAULT gen_random_UUID(),
    email VARCHAR(40) NOT NULL UNIQUE,
    username VARCHAR(20) NOT NULL UNIQUE,
    password VARCHAR(60), -- BCrypt hashing generates 60 characters
    avatar VARCHAR(50) NOT NULL DEFAULT 'avatar.png',
    cover VARCHAR(50) NOT NULL DEFAULT 'cover_image.png',
    description VARCHAR(100),
    status account_status NOT NULL DEFAULT account_status('inactive'),
    -- OAuth2-specific fields
    oauth_provider VARCHAR(10), -- 'google', 'github'
    oauth_provider_id VARCHAR(25), -- the user ID from provider
    -- JWT token version: used for ban/logout everywhere
    token_version INT NOT NULL DEFAULT 1
);