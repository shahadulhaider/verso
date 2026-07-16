package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/shahadulhaider/verso/libs/go/envelope"
	"github.com/shahadulhaider/verso/libs/go/outbox"

	"github.com/shahadulhaider/verso/services/verso-profile-service/internal/repository"
)

const (
	serviceName          = "verso-profile-service"
	profileUpdatedEvent  = "verso.profile.profile-updated.v1"
	aggregateTypeProfile = "user_profile"
	maxBioLength         = 2000
)

var allowedUpdateFields = map[string]string{
	"displayName":       "display_name",
	"bio":               "bio",
	"location":          "location",
	"websiteUrl":        "website_url",
	"privacyLevel":      "privacy_level",
	"readingGoalAnnual": "reading_goal_annual",
	"preferredLanguage": "preferred_language",
}

var validPrivacyLevels = map[string]bool{
	"public":       true,
	"friends_only": true,
	"private":      true,
}

type ProfileService struct {
	repo *repository.Repo
	log  *slog.Logger
}

func New(repo *repository.Repo, log *slog.Logger) *ProfileService {
	return &ProfileService{repo: repo, log: log}
}

type UpdateProfileRequest struct {
	DisplayName       *string `json:"displayName,omitempty"`
	Bio               *string `json:"bio,omitempty"`
	Location          *string `json:"location,omitempty"`
	WebsiteURL        *string `json:"websiteUrl,omitempty"`
	PrivacyLevel      *string `json:"privacyLevel,omitempty"`
	ReadingGoalAnnual *int    `json:"readingGoalAnnual,omitempty"`
	PreferredLanguage *string `json:"preferredLanguage,omitempty"`
}

func (s *ProfileService) GetProfile(ctx context.Context, userID string) (*repository.Profile, error) {
	return s.repo.GetByID(ctx, userID)
}

func (s *ProfileService) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (*repository.Profile, error) {
	fields, changedFields, err := s.buildUpdateFields(req)
	if err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		return s.repo.GetByID(ctx, userID)
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	profile, err := s.repo.UpdateFields(ctx, tx, userID, fields)
	if err != nil {
		return nil, err
	}

	if err := s.publishProfileUpdated(ctx, tx, profile, changedFields); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return profile, nil
}

func (s *ProfileService) CreateDefaultProfile(ctx context.Context, userID, email, displayName string) error {
	username := deriveUsername(email)
	now := time.Now().UTC()

	p := &repository.Profile{
		ID:                userID,
		Username:          username,
		DisplayName:       displayName,
		PrivacyLevel:      "public",
		PreferredLanguage: "en",
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.repo.Create(ctx, tx, p); err != nil {
		if errors.Is(err, repository.ErrDuplicateProfile) {
			s.log.Info("profile already exists, skipping", slog.String("user_id", userID))
			return nil
		}
		return err
	}

	return tx.Commit(ctx)
}

func (s *ProfileService) buildUpdateFields(req UpdateProfileRequest) (map[string]any, []string, error) {
	fields := map[string]any{}
	changedFields := []string{}

	if req.DisplayName != nil {
		if *req.DisplayName == "" {
			return nil, nil, errors.New("display_name cannot be empty")
		}
		fields["display_name"] = *req.DisplayName
		changedFields = append(changedFields, "displayName")
	}
	if req.Bio != nil {
		if len(*req.Bio) > maxBioLength {
			return nil, nil, errors.New("bio exceeds 2000 characters")
		}
		fields["bio"] = *req.Bio
		changedFields = append(changedFields, "bio")
	}
	if req.Location != nil {
		fields["location"] = *req.Location
		changedFields = append(changedFields, "location")
	}
	if req.WebsiteURL != nil {
		fields["website_url"] = *req.WebsiteURL
		changedFields = append(changedFields, "websiteUrl")
	}
	if req.PrivacyLevel != nil {
		if !validPrivacyLevels[*req.PrivacyLevel] {
			return nil, nil, errors.New("invalid privacy_level")
		}
		fields["privacy_level"] = *req.PrivacyLevel
		changedFields = append(changedFields, "privacyLevel")
	}
	if req.ReadingGoalAnnual != nil {
		fields["reading_goal_annual"] = *req.ReadingGoalAnnual
		changedFields = append(changedFields, "readingGoalAnnual")
	}
	if req.PreferredLanguage != nil {
		fields["preferred_language"] = *req.PreferredLanguage
		changedFields = append(changedFields, "preferredLanguage")
	}

	return fields, changedFields, nil
}

func (s *ProfileService) publishProfileUpdated(ctx context.Context, tx pgx.Tx, profile *repository.Profile, changedFields []string) error {
	payload, err := json.Marshal(map[string]any{
		"userId":        profile.ID,
		"changedFields": changedFields,
	})
	if err != nil {
		return err
	}

	env := envelope.New(ctx, profileUpdatedEvent, serviceName, profile.ID, payload)
	return outbox.InsertEvent(ctx, tx, aggregateTypeProfile, profile.ID, env)
}

func deriveUsername(email string) string {
	parts := strings.SplitN(email, "@", 2)
	username := parts[0]
	if len(username) > 30 {
		username = username[:30]
	}
	return username
}
