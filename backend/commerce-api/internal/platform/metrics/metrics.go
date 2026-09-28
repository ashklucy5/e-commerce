package metrics

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"project.local/commerce-api/internal/platform/queue"
)

const (
	maxHTTPSeries = 4096

	durationBucketCount = 10

	responseSizeBucketCount = 9
)

var durationBuckets = [durationBucketCount]time.Duration{
	5 * time.Millisecond,
	10 * time.Millisecond,
	25 * time.Millisecond,
	50 * time.Millisecond,
	100 * time.Millisecond,
	250 * time.Millisecond,
	500 * time.Millisecond,
	time.Second,
	2500 * time.Millisecond,
	5 * time.Second,
}

var responseSizeBuckets = [responseSizeBucketCount]uint64{
	512,
	1024,
	4 << 10,
	16 << 10,
	64 << 10,
	256 << 10,
	1 << 20,
	4 << 20,
	16 << 20,
}

type Registry struct {
	db *pgxpool.Pool

	redis *redis.Client

	startedAt time.Time

	inflight atomic.Int64

	series sync.Map

	seriesCount atomic.Uint64

	overflow *httpSeries
}

type httpSeries struct {
	method string

	route string

	statusClass string

	requests atomic.Uint64

	errors atomic.Uint64

	durationSumNanoseconds atomic.Uint64

	durationBuckets [durationBucketCount]atomic.Uint64

	responseBytesSum atomic.Uint64

	responseBuckets [responseSizeBucketCount]atomic.Uint64
}

type httpSeriesSnapshot struct {
	method string

	route string

	statusClass string

	requests uint64

	errors uint64

	durationSumNanoseconds uint64

	durationBuckets [durationBucketCount]uint64

	responseBytesSum uint64

	responseBuckets [responseSizeBucketCount]uint64
}

func NewRegistry(
	db *pgxpool.Pool,
	redisClient *redis.Client,
) *Registry {
	overflow :=
		&httpSeries{
			method: "OTHER",

			route: "overflow",

			statusClass: "other",
		}

	registry :=
		&Registry{
			db: db,

			redis: redisClient,

			startedAt: time.Now().
				UTC(),

			overflow: overflow,
		}

	return registry
}

func (r *Registry) HTTPRequestStarted() {
	if r == nil {
		return
	}

	r.inflight.Add(
		1,
	)
}

func (r *Registry) HTTPRequestFinished() {
	if r == nil {
		return
	}

	value :=
		r.inflight.Add(
			-1,
		)

	if value < 0 {
		r.inflight.Store(
			0,
		)
	}
}

func (r *Registry) ObserveHTTP(
	method string,
	route string,
	status int,
	duration time.Duration,
	responseBytes int,
) {
	if r == nil {
		return
	}

	method =
		normalizeMethod(
			method,
		)

	route =
		normalizeRoute(
			route,
		)

	statusClass :=
		normalizeStatusClass(
			status,
		)

	series :=
		r.httpSeries(
			method,
			route,
			statusClass,
		)

	series.requests.Add(
		1,
	)

	if status >= 400 {
		series.errors.Add(
			1,
		)
	}

	if duration < 0 {
		duration = 0
	}

	durationNanoseconds :=
		uint64(
			duration.Nanoseconds(),
		)

	series.durationSumNanoseconds.Add(
		durationNanoseconds,
	)

	for index, upperBound := range durationBuckets {

		if duration <=
			upperBound {

			series.
				durationBuckets[index].
				Add(
					1,
				)
		}
	}

	if responseBytes < 0 {
		responseBytes = 0
	}

	size :=
		uint64(
			responseBytes,
		)

	series.responseBytesSum.Add(
		size,
	)

	for index, upperBound := range responseSizeBuckets {

		if size <=
			upperBound {

			series.
				responseBuckets[index].
				Add(
					1,
				)
		}
	}
}

func (r *Registry) RenderPrometheus(
	ctx context.Context,
) []byte {
	if r == nil {
		return nil
	}

	var builder strings.Builder

	// Reserving a modest buffer reduces reallocations during a scrape
	// while avoiding a large permanent allocation.
	builder.Grow(
		32 << 10,
	)

	r.writeHTTPMetrics(
		&builder,
	)

	r.writeRuntimeMetrics(
		&builder,
	)

	r.writePostgresMetrics(
		&builder,
	)

	r.writeRedisPoolMetrics(
		&builder,
	)

	r.writeQueueMetrics(
		ctx,
		&builder,
	)

	return []byte(
		builder.String(),
	)
}

func (r *Registry) httpSeries(
	method string,
	route string,
	statusClass string,
) *httpSeries {
	key :=
		method +
			"\x00" +
			route +
			"\x00" +
			statusClass

	if existing, ok :=
		r.series.Load(
			key,
		); ok {

		return existing.(*httpSeries)
	}

	if r.seriesCount.Load() >=
		maxHTTPSeries {

		return r.overflow
	}

	candidate :=
		&httpSeries{
			method: method,

			route: route,

			statusClass: statusClass,
		}

	actual,
		loaded :=
		r.series.LoadOrStore(
			key,
			candidate,
		)

	if loaded {
		return actual.(*httpSeries)
	}

	r.seriesCount.Add(
		1,
	)

	return candidate
}

func (r *Registry) writeHTTPMetrics(
	builder *strings.Builder,
) {
	snapshots :=
		r.httpSnapshots()

	writeMetricHeader(
		builder,
		"commerce_http_requests_total",
		"Total HTTP requests by method, route, and status class.",
		"counter",
	)

	for _, snapshot := range snapshots {

		fmt.Fprintf(
			builder,
			"commerce_http_requests_total%s %d\n",
			httpLabels(
				snapshot,
			),
			snapshot.requests,
		)
	}

	writeMetricHeader(
		builder,
		"commerce_http_errors_total",
		"Total HTTP responses with status code 400 or greater.",
		"counter",
	)

	for _, snapshot := range snapshots {

		fmt.Fprintf(
			builder,
			"commerce_http_errors_total%s %d\n",
			httpLabels(
				snapshot,
			),
			snapshot.errors,
		)
	}

	writeMetricHeader(
		builder,
		"commerce_http_requests_in_flight",
		"Current number of HTTP requests being processed.",
		"gauge",
	)

	fmt.Fprintf(
		builder,
		"commerce_http_requests_in_flight %d\n",
		r.inflight.Load(),
	)

	writeMetricHeader(
		builder,
		"commerce_http_request_duration_seconds",
		"HTTP request duration in seconds.",
		"histogram",
	)

	for _, snapshot := range snapshots {

		labels :=
			httpLabelsWithoutClosingBrace(
				snapshot,
			)

		for index, upperBound := range durationBuckets {

			fmt.Fprintf(
				builder,
				"commerce_http_request_duration_seconds_bucket%s,le=\"%s\"} %d\n",
				labels,
				formatSeconds(
					upperBound,
				),
				snapshot.
					durationBuckets[index],
			)
		}

		fmt.Fprintf(
			builder,
			"commerce_http_request_duration_seconds_bucket%s,le=\"+Inf\"} %d\n",
			labels,
			snapshot.requests,
		)

		fmt.Fprintf(
			builder,
			"commerce_http_request_duration_seconds_sum%s %s\n",
			httpLabels(
				snapshot,
			),
			strconv.FormatFloat(
				float64(
					snapshot.
						durationSumNanoseconds,
				)/float64(
					time.Second,
				),
				'f',
				6,
				64,
			),
		)

		fmt.Fprintf(
			builder,
			"commerce_http_request_duration_seconds_count%s %d\n",
			httpLabels(
				snapshot,
			),
			snapshot.requests,
		)
	}

	writeMetricHeader(
		builder,
		"commerce_http_response_size_bytes",
		"HTTP response size in bytes.",
		"histogram",
	)

	for _, snapshot := range snapshots {

		labels :=
			httpLabelsWithoutClosingBrace(
				snapshot,
			)

		for index, upperBound := range responseSizeBuckets {

			fmt.Fprintf(
				builder,
				"commerce_http_response_size_bytes_bucket%s,le=\"%d\"} %d\n",
				labels,
				upperBound,
				snapshot.
					responseBuckets[index],
			)
		}

		fmt.Fprintf(
			builder,
			"commerce_http_response_size_bytes_bucket%s,le=\"+Inf\"} %d\n",
			labels,
			snapshot.requests,
		)

		fmt.Fprintf(
			builder,
			"commerce_http_response_size_bytes_sum%s %d\n",
			httpLabels(
				snapshot,
			),
			snapshot.
				responseBytesSum,
		)

		fmt.Fprintf(
			builder,
			"commerce_http_response_size_bytes_count%s %d\n",
			httpLabels(
				snapshot,
			),
			snapshot.requests,
		)
	}
}

func (r *Registry) writeRuntimeMetrics(
	builder *strings.Builder,
) {
	var memory runtime.MemStats

	runtime.ReadMemStats(
		&memory,
	)

	writeMetricHeader(
		builder,
		"commerce_process_uptime_seconds",
		"Process uptime in seconds.",
		"gauge",
	)

	fmt.Fprintf(
		builder,
		"commerce_process_uptime_seconds %.3f\n",
		time.Since(
			r.startedAt,
		).Seconds(),
	)

	writeMetricHeader(
		builder,
		"commerce_go_goroutines",
		"Current number of goroutines.",
		"gauge",
	)

	fmt.Fprintf(
		builder,
		"commerce_go_goroutines %d\n",
		runtime.NumGoroutine(),
	)

	writeMetricHeader(
		builder,
		"commerce_go_heap_alloc_bytes",
		"Bytes of allocated heap objects.",
		"gauge",
	)

	fmt.Fprintf(
		builder,
		"commerce_go_heap_alloc_bytes %d\n",
		memory.HeapAlloc,
	)

	writeMetricHeader(
		builder,
		"commerce_go_heap_inuse_bytes",
		"Bytes in in-use heap spans.",
		"gauge",
	)

	fmt.Fprintf(
		builder,
		"commerce_go_heap_inuse_bytes %d\n",
		memory.HeapInuse,
	)

	writeMetricHeader(
		builder,
		"commerce_go_heap_objects",
		"Number of allocated heap objects.",
		"gauge",
	)

	fmt.Fprintf(
		builder,
		"commerce_go_heap_objects %d\n",
		memory.HeapObjects,
	)

	writeMetricHeader(
		builder,
		"commerce_go_gc_cycles_total",
		"Completed garbage collection cycles.",
		"counter",
	)

	fmt.Fprintf(
		builder,
		"commerce_go_gc_cycles_total %d\n",
		memory.NumGC,
	)

	writeMetricHeader(
		builder,
		"commerce_go_gc_pause_seconds_total",
		"Total stop-the-world GC pause duration.",
		"counter",
	)

	fmt.Fprintf(
		builder,
		"commerce_go_gc_pause_seconds_total %.6f\n",
		float64(
			memory.PauseTotalNs,
		)/
			float64(
				time.Second,
			),
	)

	writeMetricHeader(
		builder,
		"commerce_go_next_gc_bytes",
		"Target heap size for the next GC cycle.",
		"gauge",
	)

	fmt.Fprintf(
		builder,
		"commerce_go_next_gc_bytes %d\n",
		memory.NextGC,
	)
}

func (r *Registry) writePostgresMetrics(
	builder *strings.Builder,
) {
	if r.db == nil {
		return
	}

	stats :=
		r.db.Stat()

	maxConnections :=
		stats.MaxConns()

	acquiredConnections :=
		stats.AcquiredConns()

	saturation :=
		0.0

	if maxConnections > 0 {
		saturation =
			float64(
				acquiredConnections,
			) /
				float64(
					maxConnections,
				)
	}

	writeMetricHeader(
		builder,
		"commerce_postgres_pool_saturation_ratio",
		"Fraction of PostgreSQL pool connections currently acquired.",
		"gauge",
	)

	fmt.Fprintf(
		builder,
		"commerce_postgres_pool_saturation_ratio %.6f\n",
		saturation,
	)

	writeGauge(
		builder,
		"commerce_postgres_pool_max_connections",
		"Configured PostgreSQL pool connection limit.",
		int64(
			maxConnections,
		),
	)

	writeGauge(
		builder,
		"commerce_postgres_pool_total_connections",
		"Current total PostgreSQL pool connections.",
		int64(
			stats.TotalConns(),
		),
	)

	writeGauge(
		builder,
		"commerce_postgres_pool_acquired_connections",
		"Current acquired PostgreSQL pool connections.",
		int64(
			acquiredConnections,
		),
	)

	writeGauge(
		builder,
		"commerce_postgres_pool_idle_connections",
		"Current idle PostgreSQL pool connections.",
		int64(
			stats.IdleConns(),
		),
	)

	writeGauge(
		builder,
		"commerce_postgres_pool_constructing_connections",
		"PostgreSQL connections currently being constructed.",
		int64(
			stats.ConstructingConns(),
		),
	)

	writeCounter(
		builder,
		"commerce_postgres_pool_acquires_total",
		"Total successful PostgreSQL connection acquires.",
		stats.AcquireCount(),
	)

	writeCounter(
		builder,
		"commerce_postgres_pool_empty_acquires_total",
		"Successful PostgreSQL acquires that had to wait because the pool was empty.",
		stats.EmptyAcquireCount(),
	)

	writeCounter(
		builder,
		"commerce_postgres_pool_canceled_acquires_total",
		"PostgreSQL pool acquires canceled by context.",
		stats.CanceledAcquireCount(),
	)

	writeMetricHeader(
		builder,
		"commerce_postgres_pool_acquire_seconds_total",
		"Total time spent successfully acquiring PostgreSQL connections.",
		"counter",
	)

	fmt.Fprintf(
		builder,
		"commerce_postgres_pool_acquire_seconds_total %.6f\n",
		stats.AcquireDuration().
			Seconds(),
	)

	writeMetricHeader(
		builder,
		"commerce_postgres_pool_empty_acquire_wait_seconds_total",
		"Total wait time for PostgreSQL acquires while the pool was empty.",
		"counter",
	)

	fmt.Fprintf(
		builder,
		"commerce_postgres_pool_empty_acquire_wait_seconds_total %.6f\n",
		stats.EmptyAcquireWaitTime().
			Seconds(),
	)
}

func (r *Registry) writeRedisPoolMetrics(
	builder *strings.Builder,
) {
	if r.redis == nil {
		return
	}

	stats :=
		r.redis.PoolStats()

	writeGauge(
		builder,
		"commerce_redis_pool_total_connections",
		"Current total Redis pool connections.",
		int64(
			stats.TotalConns,
		),
	)

	writeGauge(
		builder,
		"commerce_redis_pool_idle_connections",
		"Current idle Redis pool connections.",
		int64(
			stats.IdleConns,
		),
	)

	writeGauge(
		builder,
		"commerce_redis_pool_pending_requests",
		"Requests currently waiting for a Redis connection.",
		int64(
			stats.PendingRequests,
		),
	)

	writeCounter(
		builder,
		"commerce_redis_pool_hits_total",
		"Redis pool requests served by an idle connection.",
		int64(
			stats.Hits,
		),
	)

	writeCounter(
		builder,
		"commerce_redis_pool_misses_total",
		"Redis pool requests that required another connection.",
		int64(
			stats.Misses,
		),
	)

	writeCounter(
		builder,
		"commerce_redis_pool_timeouts_total",
		"Redis pool wait timeouts.",
		int64(
			stats.Timeouts,
		),
	)

	writeCounter(
		builder,
		"commerce_redis_pool_waits_total",
		"Redis pool requests that waited for a connection.",
		int64(
			stats.WaitCount,
		),
	)

	writeMetricHeader(
		builder,
		"commerce_redis_pool_wait_seconds_total",
		"Total time spent waiting for Redis pool connections.",
		"counter",
	)

	fmt.Fprintf(
		builder,
		"commerce_redis_pool_wait_seconds_total %.6f\n",
		float64(
			stats.WaitDurationNs,
		)/
			float64(
				time.Second,
			),
	)

	writeCounter(
		builder,
		"commerce_redis_pool_unusable_connections_total",
		"Redis connections found unusable.",
		int64(
			stats.Unusable,
		),
	)

	writeCounter(
		builder,
		"commerce_redis_pool_stale_connections_total",
		"Redis stale connections removed from the pool.",
		int64(
			stats.StaleConns,
		),
	)
}

func (r *Registry) writeQueueMetrics(
	ctx context.Context,
	builder *strings.Builder,
) {
	if r.redis == nil {
		return
	}

	scrapeCtx, cancel :=
		context.WithTimeout(
			ctx,
			750*time.Millisecond,
		)

	defer cancel()

	pipeline :=
		r.redis.Pipeline()

	streamLength :=
		pipeline.XLen(
			scrapeCtx,
			queue.DefaultStream,
		)

	retryCount :=
		pipeline.ZCard(
			scrapeCtx,
			queue.DefaultRetrySet,
		)

	deadLetterLength :=
		pipeline.XLen(
			scrapeCtx,
			queue.DefaultDeadLetterStream,
		)

	pending :=
		pipeline.XPending(
			scrapeCtx,
			queue.DefaultStream,
			queue.DefaultGroup,
		)

	_, _ =
		pipeline.Exec(
			scrapeCtx,
		)

	success :=
		int64(
			1,
		)

	if err :=
		streamLength.Err(); err == nil {

		writeGauge(
			builder,
			"commerce_queue_stream_messages",
			"Current number of messages retained in the live job stream.",
			streamLength.Val(),
		)
	} else {
		success = 0
	}

	if err :=
		retryCount.Err(); err == nil {

		writeGauge(
			builder,
			"commerce_queue_retry_scheduled",
			"Current number of delayed retry jobs.",
			retryCount.Val(),
		)
	} else {
		success = 0
	}

	if err :=
		deadLetterLength.Err(); err == nil {

		writeGauge(
			builder,
			"commerce_queue_dead_letter_messages",
			"Current number of retained dead-letter jobs.",
			deadLetterLength.Val(),
		)
	} else {
		success = 0
	}

	if err :=
		pending.Err(); err == nil {

		result :=
			pending.Val()

		writeGauge(
			builder,
			"commerce_queue_pending_messages",
			"Current number of unacknowledged jobs in the worker consumer group.",
			result.Count,
		)
	} else {
		// Before the worker has created the consumer group this can
		// legitimately be unavailable.
		success = 0
	}

	writeGauge(
		builder,
		"commerce_queue_metrics_scrape_success",
		"Whether all queue metrics were retrieved successfully.",
		success,
	)
}

func (r *Registry) httpSnapshots() []httpSeriesSnapshot {
	result :=
		make(
			[]httpSeriesSnapshot,
			0,
			int(
				r.seriesCount.Load(),
			)+1,
		)

	r.series.Range(
		func(
			_,
			value any,
		) bool {
			series :=
				value.(*httpSeries)

			result =
				append(
					result,
					snapshotHTTPSeries(
						series,
					),
				)

			return true
		},
	)

	if r.overflow.requests.Load() >
		0 {

		result =
			append(
				result,
				snapshotHTTPSeries(
					r.overflow,
				),
			)
	}

	sort.Slice(
		result,
		func(
			left int,
			right int,
		) bool {
			if result[left].route !=
				result[right].route {

				return result[left].route <
					result[right].route
			}

			if result[left].method !=
				result[right].method {

				return result[left].method <
					result[right].method
			}

			return result[left].statusClass <
				result[right].statusClass
		},
	)

	return result
}

func snapshotHTTPSeries(
	series *httpSeries,
) httpSeriesSnapshot {
	snapshot :=
		httpSeriesSnapshot{
			method: series.method,

			route: series.route,

			statusClass: series.statusClass,

			requests: series.requests.Load(),

			errors: series.errors.Load(),

			durationSumNanoseconds: series.
				durationSumNanoseconds.
				Load(),

			responseBytesSum: series.
				responseBytesSum.
				Load(),
		}

	for index := range durationBucketCount {

		snapshot.durationBuckets[index] =
			series.
				durationBuckets[index].
				Load()
	}

	for index := range responseSizeBucketCount {

		snapshot.responseBuckets[index] =
			series.
				responseBuckets[index].
				Load()
	}

	return snapshot
}

func normalizeMethod(
	method string,
) string {
	method =
		strings.ToUpper(
			strings.TrimSpace(
				method,
			),
		)

	switch method {
	case "GET",
		"HEAD",
		"POST",
		"PUT",
		"PATCH",
		"DELETE",
		"OPTIONS":

		return method

	default:
		return "OTHER"
	}
}

func normalizeRoute(
	route string,
) string {
	route =
		strings.TrimSpace(
			route,
		)

	if route == "" {
		return "unmatched"
	}

	if len(route) > 256 {
		return "unmatched"
	}

	return route
}

func normalizeStatusClass(
	status int,
) string {
	switch {
	case status >= 100 &&
		status <= 199:
		return "1xx"

	case status >= 200 &&
		status <= 299:
		return "2xx"

	case status >= 300 &&
		status <= 399:
		return "3xx"

	case status >= 400 &&
		status <= 499:
		return "4xx"

	case status >= 500 &&
		status <= 599:
		return "5xx"

	default:
		return "other"
	}
}

func httpLabels(
	snapshot httpSeriesSnapshot,
) string {
	return fmt.Sprintf(
		"{method=\"%s\",route=\"%s\",status=\"%s\"}",
		escapeLabel(
			snapshot.method,
		),
		escapeLabel(
			snapshot.route,
		),
		escapeLabel(
			snapshot.statusClass,
		),
	)
}

func httpLabelsWithoutClosingBrace(
	snapshot httpSeriesSnapshot,
) string {
	return fmt.Sprintf(
		"{method=\"%s\",route=\"%s\",status=\"%s\"",
		escapeLabel(
			snapshot.method,
		),
		escapeLabel(
			snapshot.route,
		),
		escapeLabel(
			snapshot.statusClass,
		),
	)
}

func escapeLabel(
	value string,
) string {
	value =
		strings.ReplaceAll(
			value,
			`\`,
			`\\`,
		)

	value =
		strings.ReplaceAll(
			value,
			"\n",
			`\n`,
		)

	value =
		strings.ReplaceAll(
			value,
			`"`,
			`\"`,
		)

	return value
}

func formatSeconds(
	duration time.Duration,
) string {
	return strconv.FormatFloat(
		duration.Seconds(),
		'f',
		3,
		64,
	)
}

func writeMetricHeader(
	builder *strings.Builder,
	name string,
	help string,
	metricType string,
) {
	fmt.Fprintf(
		builder,
		"# HELP %s %s\n",
		name,
		help,
	)

	fmt.Fprintf(
		builder,
		"# TYPE %s %s\n",
		name,
		metricType,
	)
}

func writeGauge(
	builder *strings.Builder,
	name string,
	help string,
	value int64,
) {
	writeMetricHeader(
		builder,
		name,
		help,
		"gauge",
	)

	fmt.Fprintf(
		builder,
		"%s %d\n",
		name,
		value,
	)
}

func writeCounter(
	builder *strings.Builder,
	name string,
	help string,
	value int64,
) {
	writeMetricHeader(
		builder,
		name,
		help,
		"counter",
	)

	fmt.Fprintf(
		builder,
		"%s %d\n",
		name,
		value,
	)
}
