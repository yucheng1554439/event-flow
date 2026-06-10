package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	events := flag.Int("events", 1000, "number of ShipPurchased events to publish")
	failures := flag.Int("failures", 10, "percentage of events with simulateFailure (0-100)")
	replay := flag.Int("replay", 0, "percentage of events to replay from DLQ after generation (0-100)")
	throughput := flag.Int("throughput", 50, "max events per second")
	apiURL := flag.String("api", env("EVENTFLOW_API", "http://localhost:8080"), "EventFlow API base URL")
	topic := flag.String("topic", "ship-orders", "Kafka topic")
	flag.Parse()

	ctx := context.Background()
	client := &http.Client{Timeout: 10 * time.Second}
	ticker := time.NewTicker(time.Second / time.Duration(max(*throughput, 1)))
	defer ticker.Stop()

	var published, failed, replayed atomic.Int64
	var wg sync.WaitGroup
	failEvery := 100 / max(*failures, 1)
	if *failures == 0 {
		failEvery = 0
	}

	log.Printf("Galactic Commerce generator: events=%d failures=%d%% throughput=%d/s", *events, *failures, *throughput)

	for i := 0; i < *events; i++ {
		<-ticker.C
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			simFail := failEvery > 0 && n%failEvery == 0
			key := fmt.Sprintf("ship-gen-%d-%d", time.Now().UnixNano(), n)
			body, _ := json.Marshal(map[string]any{
				"topic":          *topic,
				"eventType":      "ShipPurchased",
				"idempotencyKey": key,
				"payload": map[string]any{
					"pilotId":         rand.Intn(1000),
					"shipId":          fmt.Sprintf("ship-%d", rand.Intn(500)),
					"credits":         rand.Intn(100000) + 1000,
					"simulateFailure": simFail,
				},
			})
			req, _ := http.NewRequestWithContext(ctx, http.MethodPost, *apiURL+"/api/v1/events", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := client.Do(req)
			if err != nil {
				failed.Add(1)
				return
			}
			resp.Body.Close()
			if resp.StatusCode >= 300 {
				failed.Add(1)
				return
			}
			published.Add(1)
		}(i)
	}
	wg.Wait()

	log.Printf("Published: %d | Failed: %d", published.Load(), failed.Load())

	if *replay > 0 {
		time.Sleep(15 * time.Second)
		replayBody, _ := json.Marshal(map[string]any{
			"topic":       *topic,
			"dlqOnly":     true,
			"targetTopic": *topic,
		})
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, *apiURL+"/api/v1/replay", bytes.NewReader(replayBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err == nil {
			var result struct {
				Replayed int `json:"replayed"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&result)
			resp.Body.Close()
			replayed.Store(int64(result.Replayed))
		}
		log.Printf("Replayed from DLQ: %d", replayed.Load())
	}

	log.Println("Generator complete — check Grafana EventFlow Demo Dashboard")
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
