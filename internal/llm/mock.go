package llm

import (
	"context"
	"time"
)

// MockClient implements Client with deterministic canned responses
// suitable for demo mode and testing.
type MockClient struct {
	delay time.Duration // per-chunk delay for simulating streaming
}

// NewMockClient returns a mock LLM client.
// chunkDelay controls how fast tokens stream (0 = instant).
func NewMockClient(chunkDelay time.Duration) *MockClient {
	return &MockClient{delay: chunkDelay}
}

const mockResponse = `[INFO] Cluster analysis complete.

**Applications overview:**
- **postgresql** (3 units): All units active and healthy. Leader is postgresql/0.
- **grafana** (1 unit): Status is **blocked** — missing required relation to a data source. This is the most urgent issue.
- **prometheus** (1 unit): Status is **waiting** — likely waiting for a relation to be established.
- **ubuntu-app** (2 units): Active and exposed on port 80.

**Issues detected:**

[WARNING] grafana/0 is blocked
  Root cause: Grafana requires a relation to a monitoring data source (e.g. prometheus) to function.
  Suggested fix:
    juju relate grafana:grafana-source prometheus:grafana-source

[WARNING] prometheus/0 is in waiting state
  Root cause: Prometheus may be waiting for a scrape-target relation or configuration.
  Suggested fix:
    juju relate prometheus:metrics-endpoint <target-app>:metrics-endpoint

[INFO] No machine-level issues detected. All 4 machines are running and healthy.`

// ChatStream implements Client. It ignores the conversation and returns
// a canned analysis response, streamed in small chunks to simulate real
// token-by-token delivery.
func (m *MockClient) ChatStream(ctx context.Context, _ []Message) (<-chan StreamEvent, error) {
	ch := make(chan StreamEvent, 16)

	go func() {
		defer close(ch)
		// Stream the response character-by-character in small bursts.
		const chunkSize = 3
		for i := 0; i < len(mockResponse); i += chunkSize {
			end := i + chunkSize
			if end > len(mockResponse) {
				end = len(mockResponse)
			}
			select {
			case ch <- StreamEvent{Delta: mockResponse[i:end]}:
			case <-ctx.Done():
				return
			}
			if m.delay > 0 {
				select {
				case <-time.After(m.delay):
				case <-ctx.Done():
					return
				}
			}
		}
		select {
		case ch <- StreamEvent{Done: true}:
		case <-ctx.Done():
		}
	}()

	return ch, nil
}

// Close implements Client.
func (m *MockClient) Close() error { return nil }
