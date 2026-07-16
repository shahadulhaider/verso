package indexer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/shahadulhaider/verso/libs/go/envelope"
	"github.com/shahadulhaider/verso/services/verso-search-service/internal/opensearch"
)

const (
	consumerGroup        = "verso-search-indexer"
	topicCreated         = "verso.catalog.work-created.v1"
	topicReviewPublished = "verso.review.review-published.v1"
	topicEmbeddingIndex  = "verso.ai.embedding-indexed.v1"
)

type workCreatedPayload struct {
	WorkID    string `json:"workId"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt"`
}

type reviewPublishedPayload struct {
	ReviewID string  `json:"reviewId"`
	WorkID   string  `json:"workId"`
	UserID   string  `json:"userId"`
	Rating   float64 `json:"rating"`
}

type embeddingIndexedPayload struct {
	WorkID  string `json:"workId"`
	ModelID string `json:"modelId"`
}

type Indexer struct {
	client *kgo.Client
	os     *opensearch.Client
	log    *slog.Logger
}

func New(brokers []string, osClient *opensearch.Client, log *slog.Logger) (*Indexer, error) {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(consumerGroup),
		kgo.ConsumeTopics(topicCreated, topicReviewPublished, topicEmbeddingIndex),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
	if err != nil {
		return nil, err
	}

	return &Indexer{
		client: cl,
		os:     osClient,
		log:    log,
	}, nil
}

func (ix *Indexer) Run(ctx context.Context) {
	ix.log.Info("indexer started",
		slog.String("topics", topicCreated+","+topicReviewPublished+","+topicEmbeddingIndex),
		slog.String("group", consumerGroup),
	)

	for {
		fetches := ix.client.PollFetches(ctx)
		if ctx.Err() != nil {
			return
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			for _, e := range errs {
				ix.log.Error("fetch error",
					slog.String("topic", e.Topic),
					slog.Int("partition", int(e.Partition)),
					slog.String("error", e.Err.Error()),
				)
			}
		}

		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()
			ix.handleRecord(ctx, record)
		}
	}
}

func (ix *Indexer) handleRecord(ctx context.Context, record *kgo.Record) {
	env, err := envelope.Unmarshal(record.Value)
	if err != nil {
		ix.log.Error("unmarshal envelope",
			slog.String("error", err.Error()),
			slog.String("topic", record.Topic),
			slog.Int64("offset", record.Offset),
		)
		return
	}

	switch record.Topic {
	case topicCreated:
		ix.handleWorkCreated(ctx, env)
	case topicReviewPublished:
		ix.handleReviewPublished(ctx, env)
	case topicEmbeddingIndex:
		ix.handleEmbeddingIndexed(env)
	default:
		ix.log.Warn("unknown topic", slog.String("topic", record.Topic))
	}
}

func (ix *Indexer) handleWorkCreated(ctx context.Context, env *envelope.EventEnvelope) {
	var payload workCreatedPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		ix.log.Error("unmarshal payload",
			slog.String("error", err.Error()),
			slog.String("event_id", env.EventID),
		)
		return
	}

	doc := &opensearch.WorkDocument{
		WorkID:    payload.WorkID,
		Title:     payload.Title,
		CreatedAt: payload.CreatedAt,
	}

	if doc.CreatedAt == "" {
		doc.CreatedAt = env.OccurredAt.Format(time.RFC3339)
	}

	if err := ix.os.IndexDocument(ctx, doc); err != nil {
		ix.log.Error("index document",
			slog.String("error", err.Error()),
			slog.String("work_id", payload.WorkID),
		)
		return
	}

	ix.log.Info("indexed work",
		slog.String("work_id", payload.WorkID),
		slog.String("title", payload.Title),
		slog.String("event_id", env.EventID),
	)
}

func (ix *Indexer) handleReviewPublished(ctx context.Context, env *envelope.EventEnvelope) {
	var payload reviewPublishedPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		ix.log.Error("unmarshal review payload",
			slog.String("error", err.Error()),
			slog.String("event_id", env.EventID),
		)
		return
	}

	if err := ix.os.UpdateWorkRating(ctx, payload.WorkID, payload.Rating); err != nil {
		ix.log.Error("update work rating",
			slog.String("error", err.Error()),
			slog.String("work_id", payload.WorkID),
		)
		return
	}

	ix.log.Info("updated work rating",
		slog.String("work_id", payload.WorkID),
		slog.Float64("rating", payload.Rating),
		slog.String("event_id", env.EventID),
	)
}

func (ix *Indexer) handleEmbeddingIndexed(env *envelope.EventEnvelope) {
	var payload embeddingIndexedPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		ix.log.Error("unmarshal embedding payload",
			slog.String("error", err.Error()),
			slog.String("event_id", env.EventID),
		)
		return
	}

	ix.log.Info("embedding indexed",
		slog.String("work_id", payload.WorkID),
		slog.String("model_id", payload.ModelID),
		slog.String("event_id", env.EventID),
	)
}

func (ix *Indexer) Close() {
	ix.client.Close()
}
