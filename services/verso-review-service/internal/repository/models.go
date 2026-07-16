package repository

import (
	"errors"
	"time"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrDuplicateReview = errors.New("duplicate review for this work")
	ErrForbidden       = errors.New("forbidden")
)

type Review struct {
	ID               string
	UserID           string
	WorkID           string
	EditionID        *string
	RatingOverall    float64
	RatingPlot       *float64
	RatingCharacters *float64
	RatingPacing     *float64
	RatingProse      *float64
	Title            *string
	Body             *string
	ContainsSpoilers bool
	LikeCount        int
	CommentCount     int
	HelpfulCount     int
	IsFeatured       bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
	Version          int
}

type ReviewComment struct {
	ID              string
	ReviewID        string
	UserID          string
	ParentCommentID *string
	Body            string
	CreatedAt       time.Time
	DeletedAt       *time.Time
}

type ReviewVote struct {
	UserID    string
	ReviewID  string
	VoteType  string
	CreatedAt time.Time
}

type AggregateRating struct {
	AverageRating float64  `json:"averageRating"`
	RatingsCount  int      `json:"ratingsCount"`
	ReviewsCount  int      `json:"reviewsCount"`
	AxisRatings   AxisAvgs `json:"axisRatings"`
}

type AxisAvgs struct {
	Plot       *float64 `json:"plot"`
	Characters *float64 `json:"characters"`
	Pacing     *float64 `json:"pacing"`
	Prose      *float64 `json:"prose"`
}
