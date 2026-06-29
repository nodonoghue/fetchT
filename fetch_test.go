package fetcht

// I'm not entirely sure how to fully test the http calls, and since the request options are built inside
// the doWith and doWithout funcs, this is a bit of a disconnect for me.
//
// The benchmarks below drive the real request path against an httptest.Server (loopback HTTP), which is
// how the http calls can be exercised end-to-end. They compare the cost of the buffered/pooled decode
// path against a hand-written streaming net/http baseline.
//
// Run with:
//   go test -bench=BenchmarkGet -benchmem -run=^$ .
//
// What to read in the output (B/op and allocs/op are the interesting columns):
//   - _Default       fetchT with RawBody off (pooled buffer, no retained body copy)
//   - _RawBody       fetchT with WithRawBody() (forces a per-response copy out of the pool)
//   - _StdStreaming  raw net/http with json.NewDecoder(resp.Body) — no full-body buffer at all
//
// Note: these are SERIAL benchmarks, so they isolate per-request allocation cost. They do NOT show the
// MaxIdleConnsPerHost transport default's benefit — that only appears under concurrency, where the std
// default of 2 forces connection re-dials. Connection reuse, not allocation, is the dominant high-traffic
// cost; this measures the second-order one.

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type benchItem struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func buildItems(n int) []benchItem {
	items := make([]benchItem, n)
	for i := range items {
		items[i] = benchItem{ID: i, Name: "user", Email: "user@example.com"}
	}
	return items
}

// newBenchServer returns a server that always responds 200 with the given
// pre-encoded JSON body, so server-side cost stays constant across iterations.
func newBenchServer(b *testing.B, body []byte) *httptest.Server {
	b.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	b.Cleanup(srv.Close)
	return srv
}

func mustJSON(b *testing.B, v any) []byte {
	b.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		b.Fatalf("marshal: %v", err)
	}
	return data
}

func benchmarkFetchT(b *testing.B, body []byte, rawBody bool) {
	srv := newBenchServer(b, body)
	client, err := NewClient(WithBaseURL(srv.URL))
	if err != nil {
		b.Fatalf("NewClient: %v", err)
	}
	ctx := context.Background()

	var opts []RequestOption
	if rawBody {
		opts = append(opts, WithRawBody())
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := Get[[]benchItem](ctx, client, opts...)
		if err != nil {
			b.Fatalf("Get: %v", err)
		}
		if len(resp.Data) == 0 {
			b.Fatal("empty response data")
		}
	}
}

// benchmarkStdStreaming is the baseline: idiomatic net/http with a streaming
// decoder straight off resp.Body — no full-body buffer is ever materialized.
func benchmarkStdStreaming(b *testing.B, body []byte) {
	srv := newBenchServer(b, body)
	c := srv.Client()
	url := srv.URL
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			b.Fatalf("NewRequest: %v", err)
		}
		resp, err := c.Do(req)
		if err != nil {
			b.Fatalf("Do: %v", err)
		}
		var data []benchItem
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			b.Fatalf("Decode: %v", err)
		}
		_ = resp.Body.Close()
		if len(data) == 0 {
			b.Fatal("empty response data")
		}
	}
}

func BenchmarkGet_SmallPayload_Default(b *testing.B) {
	benchmarkFetchT(b, mustJSON(b, buildItems(1)), false)
}

func BenchmarkGet_SmallPayload_RawBody(b *testing.B) {
	benchmarkFetchT(b, mustJSON(b, buildItems(1)), true)
}

func BenchmarkGet_SmallPayload_StdStreaming(b *testing.B) {
	benchmarkStdStreaming(b, mustJSON(b, buildItems(1)))
}

func BenchmarkGet_LargePayload_Default(b *testing.B) {
	benchmarkFetchT(b, mustJSON(b, buildItems(1000)), false)
}

func BenchmarkGet_LargePayload_RawBody(b *testing.B) {
	benchmarkFetchT(b, mustJSON(b, buildItems(1000)), true)
}

func BenchmarkGet_LargePayload_StdStreaming(b *testing.B) {
	benchmarkStdStreaming(b, mustJSON(b, buildItems(1000)))
}

// ---- Parallel benchmarks: connection reuse (the transport-default fix) ----
//
// These run a fixed number of concurrent workers (independent of GOMAXPROCS, so
// the load and port pressure stay bounded and reproducible) and report a custom
// dials/op metric: the number of new TCP connections actually opened per request.
//
// That metric is the point — not ns/op. A transport that can keep enough idle
// connections reuses them (dials/op -> ~0 after warmup); one that can't re-dials,
// paying a fresh TCP (and, in production, TLS) handshake each time.
//
//   go test -bench=BenchmarkParallel -benchmem -run=^$ .
//
// Caution: BareTransport deliberately churns connections, leaving sockets in
// TIME_WAIT. The default -benchtime=1s is fine; a long -benchtime can exhaust
// ephemeral ports.

const parallelWorkers = 20

// dialCounter wraps a net.Dialer to count how many connections are actually
// opened. Concurrency above the transport's idle limit shows up here as dials.
type dialCounter struct{ n int64 }

func (d *dialCounter) dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	atomic.AddInt64(&d.n, 1)
	var dialer net.Dialer
	return dialer.DialContext(ctx, network, addr)
}

// newDelayServer responds after a small delay so concurrent requests hold their
// connections open simultaneously, forcing the transport's idle limit to bind.
func newDelayServer(b *testing.B, body []byte, delay time.Duration) *httptest.Server {
	b.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(delay)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	b.Cleanup(srv.Close)
	return srv
}

func benchmarkParallel(b *testing.B, transport *http.Transport, counter *dialCounter) {
	body := mustJSON(b, buildItems(1))
	srv := newDelayServer(b, body, 2*time.Millisecond)
	client, err := NewClient(WithBaseURL(srv.URL), WithTransport(transport))
	if err != nil {
		b.Fatalf("NewClient: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	work := make(chan struct{})
	var wg sync.WaitGroup
	for w := 0; w < parallelWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			for range work {
				resp, err := Get[[]benchItem](ctx, client)
				if err != nil {
					b.Errorf("Get: %v", err)
					return
				}
				if len(resp.Data) == 0 {
					b.Error("empty response data")
					return
				}
			}
		}()
	}
	for i := 0; i < b.N; i++ {
		work <- struct{}{}
	}
	close(work)
	wg.Wait()

	b.StopTimer()
	b.ReportMetric(float64(atomic.LoadInt64(&counter.n))/float64(b.N), "dials/op")
}

// BenchmarkParallel_TunedTransport uses the library default (MaxIdleConnsPerHost
// = 100): idle connections are retained and reused across concurrent requests.
func BenchmarkParallel_TunedTransport(b *testing.B) {
	counter := &dialCounter{}
	tr := defaultTransport()
	tr.DialContext = counter.dialContext
	benchmarkParallel(b, tr, counter)
}

// BenchmarkParallel_BareTransport uses a bare &http.Transport{}, which inherits
// the standard-library default MaxIdleConnsPerHost of 2. Above that concurrency
// it cannot keep enough idle connections and re-dials — see the dials/op metric.
func BenchmarkParallel_BareTransport(b *testing.B) {
	counter := &dialCounter{}
	tr := &http.Transport{DialContext: counter.dialContext}
	benchmarkParallel(b, tr, counter)
}