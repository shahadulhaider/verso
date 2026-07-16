package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sony/gobreaker/v2"

	versoerrors "github.com/shahadulhaider/verso/libs/go/errors"
	versojwt "github.com/shahadulhaider/verso/libs/go/jwt"

	"github.com/shahadulhaider/verso/services/verso-review-service/internal/repository"
	"github.com/shahadulhaider/verso/services/verso-review-service/internal/service"
)

type ReviewHandler struct {
	svc  *service.ReviewService
	repo *repository.Repo
	cb   *gobreaker.CircuitBreaker[any]
}

func New(svc *service.ReviewService, repo *repository.Repo, cb *gobreaker.CircuitBreaker[any]) *ReviewHandler {
	return &ReviewHandler{svc: svc, repo: repo, cb: cb}
}

func (h *ReviewHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *ReviewHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Pool().Ping(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *ReviewHandler) CreateReview(w http.ResponseWriter, r *http.Request) {
	claims, ok := versojwt.ClaimsFromContext(r.Context())
	if !ok {
		versoerrors.Unauthorized("missing claims").WriteJSON(w)
		return
	}

	var req service.CreateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		versoerrors.BadRequest("invalid request body").WriteJSON(w)
		return
	}

	result, err := h.cb.Execute(func() (any, error) {
		return h.svc.CreateReview(r.Context(), claims.UserID, req)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	rev := result.(*repository.Review)
	writeJSON(w, http.StatusCreated, toReviewResponse(rev))
}

func (h *ReviewHandler) GetReview(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	result, err := h.cb.Execute(func() (any, error) {
		return h.svc.GetReview(r.Context(), id)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	rev := result.(*repository.Review)
	comments, _ := h.svc.ListComments(r.Context(), id)

	resp := toReviewResponse(rev)
	resp["comments"] = toCommentResponses(comments)
	writeJSON(w, http.StatusOK, resp)
}

func (h *ReviewHandler) UpdateReview(w http.ResponseWriter, r *http.Request) {
	claims, ok := versojwt.ClaimsFromContext(r.Context())
	if !ok {
		versoerrors.Unauthorized("missing claims").WriteJSON(w)
		return
	}

	id := chi.URLParam(r, "id")
	var req service.UpdateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		versoerrors.BadRequest("invalid request body").WriteJSON(w)
		return
	}

	result, err := h.cb.Execute(func() (any, error) {
		return h.svc.UpdateReview(r.Context(), claims.UserID, id, req)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	rev := result.(*repository.Review)
	writeJSON(w, http.StatusOK, toReviewResponse(rev))
}

func (h *ReviewHandler) DeleteReview(w http.ResponseWriter, r *http.Request) {
	claims, ok := versojwt.ClaimsFromContext(r.Context())
	if !ok {
		versoerrors.Unauthorized("missing claims").WriteJSON(w)
		return
	}

	id := chi.URLParam(r, "id")
	_, err := h.cb.Execute(func() (any, error) {
		return nil, h.svc.DeleteReview(r.Context(), claims.UserID, id)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ReviewHandler) ListReviews(w http.ResponseWriter, r *http.Request) {
	workID := chi.URLParam(r, "workId")
	cursor := r.URL.Query().Get("cursor")
	sort := r.URL.Query().Get("sort")
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	result, err := h.cb.Execute(func() (any, error) {
		return h.svc.ListReviews(r.Context(), workID, cursor, sort, limit)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	reviews := result.([]*repository.Review)
	resp := make([]map[string]any, len(reviews))
	for i, rev := range reviews {
		resp[i] = toReviewResponse(rev)
	}

	out := map[string]any{"items": resp}
	if len(reviews) > 0 {
		out["nextCursor"] = reviews[len(reviews)-1].CreatedAt.Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *ReviewHandler) CastVote(w http.ResponseWriter, r *http.Request) {
	claims, ok := versojwt.ClaimsFromContext(r.Context())
	if !ok {
		versoerrors.Unauthorized("missing claims").WriteJSON(w)
		return
	}

	id := chi.URLParam(r, "id")
	var req service.CastVoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		versoerrors.BadRequest("invalid request body").WriteJSON(w)
		return
	}

	result, err := h.cb.Execute(func() (any, error) {
		return h.svc.CastVote(r.Context(), claims.UserID, id, req)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *ReviewHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	claims, ok := versojwt.ClaimsFromContext(r.Context())
	if !ok {
		versoerrors.Unauthorized("missing claims").WriteJSON(w)
		return
	}

	id := chi.URLParam(r, "id")
	var req service.AddCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		versoerrors.BadRequest("invalid request body").WriteJSON(w)
		return
	}

	result, err := h.cb.Execute(func() (any, error) {
		return h.svc.AddComment(r.Context(), claims.UserID, id, req)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	comment := result.(*repository.ReviewComment)
	writeJSON(w, http.StatusCreated, toCommentResponse(comment))
}

func (h *ReviewHandler) GetAggregateRating(w http.ResponseWriter, r *http.Request) {
	workID := chi.URLParam(r, "workId")

	result, err := h.cb.Execute(func() (any, error) {
		return h.svc.GetAggregateRating(r.Context(), workID)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *ReviewHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		versoerrors.NotFound("resource not found").WriteJSON(w)
	case errors.Is(err, repository.ErrDuplicateReview):
		versoerrors.Conflict("you have already reviewed this work").WriteJSON(w)
	case errors.Is(err, repository.ErrForbidden):
		versoerrors.Forbidden("you can only modify your own reviews").WriteJSON(w)
	case errors.Is(err, gobreaker.ErrOpenState):
		versoerrors.New(http.StatusServiceUnavailable, "Service Unavailable", "circuit breaker open").WriteJSON(w)
	default:
		msg := err.Error()
		if msg == "workId is required" ||
			msg == "body is required" ||
			msg == "rating must be between 0.5 and 5.0" ||
			msg == "rating must be in 0.5 increments" ||
			msg == "voteType must be 'like' or 'helpful'" {
			versoerrors.BadRequest(msg).WriteJSON(w)
			return
		}
		versoerrors.InternalError("an unexpected error occurred").WriteJSON(w)
	}
}

func toReviewResponse(rev *repository.Review) map[string]any {
	resp := map[string]any{
		"id":               strings.TrimSpace(rev.ID),
		"userId":           strings.TrimSpace(rev.UserID),
		"workId":           strings.TrimSpace(rev.WorkID),
		"ratingOverall":    rev.RatingOverall,
		"containsSpoilers": rev.ContainsSpoilers,
		"likeCount":        rev.LikeCount,
		"commentCount":     rev.CommentCount,
		"helpfulCount":     rev.HelpfulCount,
		"isFeatured":       rev.IsFeatured,
		"version":          rev.Version,
		"createdAt":        rev.CreatedAt.Format("2006-01-02T15:04:05Z"),
		"updatedAt":        rev.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if rev.EditionID != nil {
		resp["editionId"] = strings.TrimSpace(*rev.EditionID)
	}
	if rev.RatingPlot != nil {
		resp["ratingPlot"] = *rev.RatingPlot
	}
	if rev.RatingCharacters != nil {
		resp["ratingCharacters"] = *rev.RatingCharacters
	}
	if rev.RatingPacing != nil {
		resp["ratingPacing"] = *rev.RatingPacing
	}
	if rev.RatingProse != nil {
		resp["ratingProse"] = *rev.RatingProse
	}
	if rev.Title != nil {
		resp["title"] = *rev.Title
	}
	if rev.Body != nil {
		resp["body"] = *rev.Body
	}
	return resp
}

func toCommentResponse(c *repository.ReviewComment) map[string]any {
	resp := map[string]any{
		"id":        strings.TrimSpace(c.ID),
		"reviewId":  strings.TrimSpace(c.ReviewID),
		"userId":    strings.TrimSpace(c.UserID),
		"body":      c.Body,
		"createdAt": c.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if c.ParentCommentID != nil {
		resp["parentCommentId"] = strings.TrimSpace(*c.ParentCommentID)
	}
	return resp
}

func toCommentResponses(comments []*repository.ReviewComment) []map[string]any {
	if comments == nil {
		return []map[string]any{}
	}
	result := make([]map[string]any, len(comments))
	for i, c := range comments {
		result[i] = toCommentResponse(c)
	}
	return result
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
