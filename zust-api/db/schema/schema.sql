-- Create enum
CREATE TYPE account_status AS ENUM ('inactive', 'active', 'banned', 'locked');
CREATE TYPE video_status AS ENUM ('published', 'deleted');

-- Create table account
CREATE TABLE IF NOT EXISTS account (
    account_id UUID PRIMARY KEY DEFAULT gen_random_UUID(),
    email VARCHAR(40) NOT NULL UNIQUE,
    username VARCHAR(20) NOT NULL UNIQUE,
    password VARCHAR(60), -- BCrypt hashing generates 60 characters
    description VARCHAR(100),
    status account_status NOT NULL DEFAULT account_status('inactive'),
    -- OAuth2-specific fields
    oauth_provider VARCHAR(10), -- 'google', 'github'
    oauth_provider_id VARCHAR(25), -- the user ID from provider
    -- JWT token version: used for ban/logout everywhere
    token_version INT NOT NULL DEFAULT 1
);

CREATE UNIQUE INDEX idx_unique_email ON account (email);
CREATE UNIQUE INDEX idx_unique_username ON account (username);

-- Create table subscribe
CREATE TABLE IF NOT EXISTS subscribe (
    subscriber_id UUID NOT NULL REFERENCES account(account_id),
    subscribe_to_id UUID NOT NULL REFERENCES account(account_id),
    PRIMARY KEY(subscriber_id, subscribe_to_id),
    subscribe_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Create table video
CREATE TABLE IF NOT EXISTS video (
    video_id UUID PRIMARY KEY DEFAULT gen_random_UUID(),
    title VARCHAR(50) NOT NULL UNIQUE,
    duration INT NOT NULL DEFAULT 0,
    description VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    publisher_id UUID NOT NULL REFERENCES account(account_id),
    status video_status NOT NULL DEFAULT video_status('published')
);

-- Create table like_video
CREATE TABLE IF NOT EXISTS like_video (
    video_id UUID NOT NULL REFERENCES video(video_id),
    account_id UUID NOT NULL REFERENCES account(account_id),
    PRIMARY KEY(video_id, account_id),
    like_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Create table watch_video
CREATE TABLE IF NOT EXISTS watch_video (
    video_id UUID NOT NULL REFERENCES video(video_id),
    account_id UUID NOT NULL REFERENCES account(account_id),
    PRIMARY KEY(video_id, account_id),
    watch_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Create table favorite
CREATE TABLE IF NOT EXISTS favorite (
    video_id UUID NOT NULL REFERENCES video(video_id),
    account_id UUID NOT NULL REFERENCES account(account_id),
    PRIMARY KEY(video_id, account_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);