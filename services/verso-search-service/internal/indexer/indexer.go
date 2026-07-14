// Package indexer consumes Kafka events and indexes them into OpenSearch.
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
	consumerGroup = "verso-search-indexer"
	topicCreated  = "verso.catalog.work-created.v1"
)

// workCreatedPayload is the inner payload of a work-created event.
type workCreatedPayload struct {
	WorkID    string `json:"workId"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt"`
}

// Indexer consumes events from Redpanda and indexes them into OpenSearch.
type Indexer struct {
	client *kgo.Client
	os     *opensearch.Client
	log    *slog.Logger
}

// New creates a new Indexer with the given Kafka brokers and OpenSearch client.
func New(brokers []string, osClient *opensearch.Client, log *slog.Logger) (*Indexer, error) {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(consumerGroup),
		kgo.ConsumeTopics(topicCreated),
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

// Run starts the consumer loop. It blocks until ctx is cancelled.
func (ix *Indexer) Run(ctx context.Context) {
	ix.log.Info("indexer started", slog.String("topic", topicCreated), slog.String("group", consumerGroup))

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

	// If createdAt is empty, use envelope's occurredAt.
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

// Close shuts down the Kafka consumer.
func (ix *Indexer) Close() {
	ix.client.Close()
}
