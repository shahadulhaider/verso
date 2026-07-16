package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/shahadulhaider/verso/libs/go/envelope"
	"github.com/shahadulhaider/verso/services/verso-library-service/internal/service"
)

const (
	consumerGroup         = "verso-library-service"
	topicUserRegistered   = "verso.identity.user-registered.v1"
	topicEntitlementGrant = "verso.commerce.entitlement-granted.v1"
)

type userRegisteredPayload struct {
	UserID      string `json:"userId"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

type Consumer struct {
	client     *kgo.Client
	librarySvc *service.LibraryService
	log        *slog.Logger
}

func New(brokers []string, librarySvc *service.LibraryService, log *slog.Logger) (*Consumer, error) {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(consumerGroup),
		kgo.ConsumeTopics(topicUserRegistered, topicEntitlementGrant),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		client:     cl,
		librarySvc: librarySvc,
		log:        log,
	}, nil
}

func (c *Consumer) Run(ctx context.Context) {
	c.log.Info("consumer started",
		slog.String("topics", topicUserRegistered+","+topicEntitlementGrant),
		slog.String("group", consumerGroup))

	for {
		fetches := c.client.PollFetches(ctx)
		if ctx.Err() != nil {
			return
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			for _, e := range errs {
				c.log.Error("fetch error",
					slog.String("topic", e.Topic),
					slog.Int("partition", int(e.Partition)),
					slog.String("error", e.Err.Error()))
			}
		}

		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()
			c.handleRecord(ctx, record)
		}
	}
}

func (c *Consumer) handleRecord(ctx context.Context, record *kgo.Record) {
	switch record.Topic {
	case topicUserRegistered:
		c.handleUserRegistered(ctx, record)
	case topicEntitlementGrant:
		c.log.Info("entitlement-granted received (stub — skipping)",
			slog.Int64("offset", record.Offset))
	default:
		c.log.Warn("unknown topic", slog.String("topic", record.Topic))
	}
}

func (c *Consumer) handleUserRegistered(ctx context.Context, record *kgo.Record) {
	env, err := envelope.Unmarshal(record.Value)
	if err != nil {
		c.log.Error("unmarshal envelope",
			slog.String("error", err.Error()),
			slog.String("topic", record.Topic),
			slog.Int64("offset", record.Offset))
		return
	}

	var payload userRegisteredPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		c.log.Error("unmarshal payload",
			slog.String("error", err.Error()),
			slog.String("event_id", env.EventID))
		return
	}

	if err := c.librarySvc.CreateDefaultShelves(ctx, payload.UserID); err != nil {
		c.log.Error("create default shelves",
			slog.String("error", err.Error()),
			slog.String("user_id", payload.UserID),
			slog.String("event_id", env.EventID))
		return
	}

	c.log.Info("created default shelves",
		slog.String("user_id", payload.UserID),
		slog.String("event_id", env.EventID))
}

func (c *Consumer) Close() {
	c.client.Close()
}
