-- +goose Up
CREATE TABLE IF NOT EXISTS review (
    id                  CHAR(26)        PRIMARY KEY,
    user_id             CHAR(26)        NOT NULL,
    work_id             CHAR(26)        NOT NULL,
    edition_id          CHAR(26)        NULL,
    rating_overall      NUMERIC(2,1)    NOT NULL
                                        CHECK (rating_overall >= 0.5
                                           AND rating_overall <= 5.0
                                           AND rating_overall * 2 = FLOOR(rating_overall * 2)),
    rating_plot         NUMERIC(2,1)    NULL
                                        CHECK (rating_plot IS NULL
                                            OR (rating_plot >= 0.5
                                                AND rating_plot <= 5.0
                                                AND rating_plot * 2 = FLOOR(rating_plot * 2))),
    rating_characters   NUMERIC(2,1)    NULL
                                        CHECK (rating_characters IS NULL
                                            OR (rating_characters >= 0.5
                                                AND rating_characters <= 5.0
                                                AND rating_characters * 2 = FLOOR(rating_characters * 2))),
    rating_pacing       NUMERIC(2,1)    NULL
                                        CHECK (rating_pacing IS NULL
                                            OR (rating_pacing >= 0.5
                                                AND rating_pacing <= 5.0
                                                AND rating_pacing * 2 = FLOOR(rating_pacing * 2))),
    rating_prose        NUMERIC(2,1)    NULL
                                        CHECK (rating_prose IS NULL
                                            OR (rating_prose >= 0.5
                                                AND rating_prose <= 5.0
                                                AND rating_prose * 2 = FLOOR(rating_prose * 2))),
    title               VARCHAR(255)    NULL,
    body                TEXT            NULL,
    contains_spoilers   BOOLEAN         NOT NULL DEFAULT FALSE,
    like_count          INT             NOT NULL DEFAULT 0,
    comment_count       INT             NOT NULL DEFAULT 0,
    helpful_count       INT             NOT NULL DEFAULT 0,
    is_featured         BOOLEAN         NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ     NULL,
    version             INT             NOT NULL DEFAULT 1
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_review_user_work ON review (user_id, work_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS ix_review_work_rating ON review (work_id, rating_overall);
CREATE INDEX IF NOT EXISTS ix_review_user ON review (user_id);
CREATE INDEX IF NOT EXISTS ix_review_created ON review (created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS review;
