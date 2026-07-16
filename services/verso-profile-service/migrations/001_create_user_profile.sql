-- +goose Up
CREATE TABLE IF NOT EXISTS user_profile (
    id                  CHAR(26)        PRIMARY KEY,
    username            VARCHAR(30)     UNIQUE NOT NULL,
    display_name        VARCHAR(100)    NOT NULL,
    bio                 TEXT            NULL,
    avatar_url          VARCHAR(512)    NULL,
    location            VARCHAR(100)    NULL,
    website_url         VARCHAR(512)    NULL,
    is_author           BOOLEAN         NOT NULL DEFAULT FALSE,
    is_publisher        BOOLEAN         NOT NULL DEFAULT FALSE,
    is_verified_critic  BOOLEAN         NOT NULL DEFAULT FALSE,
    privacy_level       VARCHAR(20)     NOT NULL DEFAULT 'public'
                                        CHECK (privacy_level IN ('public', 'friends_only', 'private')),
    reading_goal_annual INT             NULL,
    preferred_language  VARCHAR(5)      NOT NULL DEFAULT 'en',
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_profile_username ON user_profile (username);
CREATE INDEX IF NOT EXISTS ix_profile_is_author ON user_profile (id) WHERE is_author = TRUE;

-- +goose Down
DROP TABLE IF EXISTS user_profile;
