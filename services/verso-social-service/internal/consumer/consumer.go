package consumer

import (
	"context"
	"log/slog"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/shahadulhaider/verso/libs/go/envelope"
)

const (
	consumerGroup   = "verso-social-service"
	topicRegistered = "verso.identity.user-registered.v1"
)

type Consumer struct {
	client *kgo.Client
	log    *slog.Logger
}

func New(brokers []string, log *slog.Logger) (*Consumer, error) {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(consumerGroup),
		kgo.ConsumeTopics(topicRegistered),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		client: cl,
		log:    log,
	}, nil
}

func (c *Consumer) Run(ctx context.Context) {
	c.log.Info("consumer started",
		slog.String("topic", topicRegistered),
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

func (c *Consumer) handleRecord(_ context.Context, record *kgo.Record) {
	env, err := envelope.Unmarshal(record.Value)
	if err != nil {
		c.log.Error("unmarshal envelope",
			slog.String("error", err.Error()),
			slog.String("topic", record.Topic),
			slog.Int64("offset", record.Offset))
		return
	}

	c.log.Info("received user-registered event (no-op)",
		slog.String("event_id", env.EventID),
		slog.String("event_type", env.Type))
}

func (c *Consumer) Close() {
	c.client.Close()
}
