package indexer_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redpanda"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/shahadulhaider/verso/libs/go/envelope"
	"github.com/shahadulhaider/verso/services/verso-search-service/internal/indexer"
	"github.com/shahadulhaider/verso/services/verso-search-service/internal/opensearch"
)

func TestIndexerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	rpContainer, err := redpanda.Run(ctx, "redpandadata/redpanda:v24.3.7")
	if err != nil {
		t.Fatalf("start redpanda: %v", err)
	}
	t.Cleanup(func() { rpContainer.Terminate(ctx) })

	brokerAddr, err := rpContainer.KafkaSeedBroker(ctx)
	if err != nil {
		t.Fatalf("kafka seed broker: %v", err)
	}

	osContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "opensearchproject/opensearch:2.17.0",
			ExposedPorts: []string{"9200/tcp"},
			Env: map[string]string{
				"discovery.type":              "single-node",
				"DISABLE_SECURITY_PLUGIN":     "true",
				"DISABLE_INSTALL_DEMO_CONFIG": "true",
				"OPENSEARCH_JAVA_OPTS":        "-Xms128m -Xmx128m",
			},
			WaitingFor: wait.ForHTTP("/_cluster/health").WithPort("9200/tcp").WithStartupTimeout(90 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start opensearch: %v", err)
	}
	t.Cleanup(func() { osContainer.Terminate(ctx) })

	osHost, _ := osContainer.Host(ctx)
	osPort, _ := osContainer.MappedPort(ctx, "9200")
	osURL := fmt.Sprintf("http://%s:%s", osHost, osPort.Port())

	topic := "verso.catalog.work-created.v1"
	adminClient, err := kgo.NewClient(kgo.SeedBrokers(brokerAddr))
	if err != nil {
		t.Fatalf("admin client: %v", err)
	}
	adm := kadm.NewClient(adminClient)
	_, err = adm.CreateTopic(ctx, 1, 1, nil, topic)
	if err != nil {
		t.Fatalf("create topic: %v", err)
	}
	adminClient.Close()

	osClient := opensearch.New(osURL, log)
	if err := osClient.EnsureIndex(ctx); err != nil {
		t.Fatalf("ensure index: %v", err)
	}

	ix, err := indexer.New([]string{brokerAddr}, osClient, log)
	if err != nil {
		t.Fatalf("init indexer: %v", err)
	}

	ixCtx, ixCancel := context.WithCancel(ctx)
	defer ixCancel()
	go ix.Run(ixCtx)

	payload, _ := json.Marshal(map[string]string{
		"workId":    "01TESTWORK00000000000000001",
		"title":     "Integration Test Book",
		"createdAt": time.Now().UTC().Format(time.RFC3339),
	})
	env := envelope.New(ctx, topic, "test-producer", "01TESTWORK00000000000000001", payload)
	envBytes, _ := env.Marshal()

	producer, err := kgo.NewClient(kgo.SeedBrokers(brokerAddr))
	if err != nil {
		t.Fatalf("producer client: %v", err)
	}
	defer producer.Close()

	if err := producer.ProduceSync(ctx, &kgo.Record{
		Topic: topic,
		Value: envBytes,
	}).FirstErr(); err != nil {
		t.Fatalf("produce: %v", err)
	}

	var hits []opensearch.SearchHit
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)

		refreshReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, osURL+"/works/_refresh", nil)
		http.DefaultClient.Do(refreshReq)

		hits, err = osClient.Search(ctx, "Integration Test Book")
		if err == nil && len(hits) > 0 {
			break
		}
	}

	if len(hits) == 0 {
		t.Fatal("expected at least 1 search hit after indexing")
	}

	if hits[0].WorkID != "01TESTWORK00000000000000001" {
		t.Errorf("workId: got %q, want %q", hits[0].WorkID, "01TESTWORK00000000000000001")
	}
	if hits[0].Title != "Integration Test Book" {
		t.Errorf("title: got %q, want %q", hits[0].Title, "Integration Test Book")
	}

	ixCancel()
	ix.Close()
}
