package consumer

import (
	"context"
	"log/slog"

	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	consumerGroup       = "verso-review-service"
	topicContentFlagged = "verso.trustsafety.content-flagged.v1"
)

type Consumer struct {
	client *kgo.Client
	log    *slog.Logger
}

func New(brokers []string, log *slog.Logger) (*Consumer, error) {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(consumerGroup),
		kgo.ConsumeTopics(topicContentFlagged),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
	if err != nil {
		return nil, err
	}

	return &Consumer{client: cl, log: log}, nil
}

func (c *Consumer) Run(ctx context.Context) {
	c.log.Info("consumer started",
		slog.String("topics", topicContentFlagged),
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
			c.handleRecord(record)
		}
	}
}

func (c *Consumer) handleRecord(record *kgo.Record) {
	switch record.Topic {
	case topicContentFlagged:
		c.log.Info("content-flagged received (stub — skipping)",
			slog.Int64("offset", record.Offset))
	default:
		c.log.Warn("unknown topic", slog.String("topic", record.Topic))
	}
}

func (c *Consumer) Close() {
	c.client.Close()
}
