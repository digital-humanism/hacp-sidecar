package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const (
	requestCount = 1000
	concurrency  = 5

	upstreamURL = "http://127.0.0.1:8000/api/test"
	sidecarURL  = "http://127.0.0.1:8080/api/test"

	tokensFile = "benchmarks/tokens.jsonl"
)

type TokenPair struct {
	Env string `json:"env"`
	Tok string `json:"tok"`
}

type Result struct {
	Latency time.Duration
	Success bool
	Status  int
	Err     error
}

type Stats struct {
	Total    time.Duration
	Success  int
	Failed   int
	Average  time.Duration
	P50      time.Duration
	P95      time.Duration
	P99      time.Duration
	Max      time.Duration
	Requests float64
}

func main() {
	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println(" High-rate Rotating Token Benchmark")
	fmt.Printf(" Requests: %d | Concurrency: %d\n", requestCount, concurrency)
	fmt.Println(" Unique DecisionToken per request")
	fmt.Println("============================================================")
	fmt.Println()

	tokens, err := loadTokens(tokensFile, requestCount)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load tokens: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d unique token pairs\n", len(tokens))
	fmt.Println()

	// ============================================================
	// HTTP clients
	// ============================================================

	//
	// Baseline client:
	// deliberately uses its own transport/pool.
	//
	baselineClient := newHTTPClient()

	//
	// Sidecar client:
	// uses a separate transport/pool so baseline connection state
	// cannot influence sidecar measurements.
	//
	sidecarClient := newHTTPClient()

	// ============================================================
	// Warm-up
	// ============================================================

	fmt.Println("Warm-up...")

	if err := warmupBaseline(baselineClient); err != nil {
		fmt.Printf("WARNING: baseline warm-up failed: %v\n", err)
	}

	if err := warmupSidecar(sidecarClient, tokens[0]); err != nil {
		fmt.Printf("WARNING: sidecar warm-up failed: %v\n", err)
	}

	fmt.Println()

	// ============================================================
	// Baseline
	// ============================================================

	fmt.Println("Baseline: upstream directly")

	baselineResults, baselineTotal :=
		runBaseline(
			baselineClient,
			requestCount,
			concurrency,
		)

	baselineStats :=
		calculateStats(
			baselineResults,
			baselineTotal,
		)

	printStats(
		"Baseline",
		baselineStats,
	)

	// Give the system a short pause between independent runs.
	time.Sleep(500 * time.Millisecond)

	// ============================================================
	// Sidecar
	// ============================================================

	fmt.Println()
	fmt.Println("Sidecar: unique rotating token per request")

	sidecarResults, sidecarTotal :=
		runSidecar(
			sidecarClient,
			tokens,
			concurrency,
		)

	sidecarStats :=
		calculateStats(
			sidecarResults,
			sidecarTotal,
		)

	printStats(
		"Sidecar rotating tokens",
		sidecarStats,
	)

	// ============================================================
	// Overhead
	// ============================================================

	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println(" Overhead")
	fmt.Println("============================================================")
	fmt.Println()

	avgOverhead :=
		sidecarStats.Average -
			baselineStats.Average

	p50Overhead :=
		sidecarStats.P50 -
			baselineStats.P50

	p95Overhead :=
		sidecarStats.P95 -
			baselineStats.P95

	p99Overhead :=
		sidecarStats.P99 -
			baselineStats.P99

	fmt.Printf(
		"Avg overhead: %s/request\n",
		formatDuration(avgOverhead),
	)

	fmt.Printf(
		"p50 overhead: %s\n",
		formatDuration(p50Overhead),
	)

	fmt.Printf(
		"p95 overhead: %s\n",
		formatDuration(p95Overhead),
	)

	fmt.Printf(
		"p99 overhead: %s\n",
		formatDuration(p99Overhead),
	)

	fmt.Println()

	// ============================================================
	// Verdict
	// ============================================================

	if baselineStats.Failed > 0 {
		fmt.Printf(
			"TEST INVALID: baseline failures detected (%d)\n",
			baselineStats.Failed,
		)
		os.Exit(2)
	}

	if sidecarStats.Failed > 0 {
		fmt.Printf(
			"TEST INVALID: sidecar failures detected (%d)\n",
			sidecarStats.Failed,
		)
		os.Exit(3)
	}

	if p99Overhead < 5*time.Millisecond {
		fmt.Printf(
			"PASS: high-rate rotating-token p99 overhead %s < 5 ms target\n",
			formatDuration(p99Overhead),
		)
		return
	}

	if p99Overhead < 10*time.Millisecond {
		fmt.Printf(
			"ACCEPTABLE: high-rate rotating-token p99 overhead %s < 10 ms\n",
			formatDuration(p99Overhead),
		)
		return
	}

	fmt.Printf(
		"TAIL LATENCY HIGH: high-rate rotating-token p99 overhead %s\n",
		formatDuration(p99Overhead),
	)
}

func newHTTPClient() *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,

		MaxIdleConns: 100,

		MaxIdleConnsPerHost: concurrency,

		MaxConnsPerHost: concurrency,

		IdleConnTimeout: 90 * time.Second,

		DisableKeepAlives: false,

		ForceAttemptHTTP2: false,
	}

	return &http.Client{
		Transport: transport,

		Timeout: 5 * time.Second,
	}
}

func loadTokens(
	path string,
	limit int,
) ([]TokenPair, error) {

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	tokens :=
		make(
			[]TokenPair,
			0,
			limit,
		)

	scanner :=
		bufio.NewScanner(file)

	//
	// Tokens can be relatively large.
	// Increase scanner buffer so one JSONL record is never truncated.
	//
	scanner.Buffer(
		make([]byte, 64*1024),
		1024*1024,
	)

	for scanner.Scan() {

		if len(tokens) >= limit {
			break
		}

		line := scanner.Bytes()

		var token TokenPair

		if err :=
			json.Unmarshal(
				line,
				&token,
			); err != nil {

			return nil,
				fmt.Errorf(
					"invalid token line %d: %w",
					len(tokens)+1,
					err,
				)
		}

		if token.Env == "" ||
			token.Tok == "" {

			return nil,
				fmt.Errorf(
					"token line %d has empty env or tok",
					len(tokens)+1,
				)
		}

		tokens =
			append(
				tokens,
				token,
			)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(tokens) < limit {
		return nil,
			fmt.Errorf(
				"not enough tokens: required=%d available=%d",
				limit,
				len(tokens),
			)
	}

	return tokens, nil
}

func warmupBaseline(
	client *http.Client,
) error {

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			2*time.Second,
		)

	defer cancel()

	req, err :=
		http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			upstreamURL,
			nil,
		)

	if err != nil {
		return err
	}

	resp, err :=
		client.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	_, _ =
		io.Copy(
			io.Discard,
			resp.Body,
		)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"unexpected baseline status: %d",
			resp.StatusCode,
		)
	}

	return nil
}

func warmupSidecar(
	client *http.Client,
	token TokenPair,
) error {

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			2*time.Second,
		)

	defer cancel()

	req, err :=
		http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			sidecarURL,
			nil,
		)

	if err != nil {
		return err
	}

	req.Header.Set(
		"X-HACP-Intent-Envelope",
		token.Env,
	)

	req.Header.Set(
		"X-HACP-Decision-Token",
		token.Tok,
	)

	resp, err :=
		client.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	_, _ =
		io.Copy(
			io.Discard,
			resp.Body,
		)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"unexpected sidecar status: %d",
			resp.StatusCode,
		)
	}

	return nil
}

func runBaseline(
	client *http.Client,
	count int,
	workers int,
) ([]Result, time.Duration) {

	results :=
		make(
			[]Result,
			count,
		)

	jobs :=
		make(
			chan int,
			count,
		)

	var wg sync.WaitGroup

	start :=
		time.Now()

	for workerID := 0; workerID < workers; workerID++ {

		wg.Add(1)

		go func() {
			defer wg.Done()

			for index := range jobs {
				results[index] =
					doBaselineRequest(
						client,
					)
			}
		}()
	}

	for i := 0; i < count; i++ {
		jobs <- i
	}

	close(jobs)

	wg.Wait()

	return results,
		time.Since(start)
}

func runSidecar(
	client *http.Client,
	tokens []TokenPair,
	workers int,
) ([]Result, time.Duration) {

	results :=
		make(
			[]Result,
			len(tokens),
		)

	jobs :=
		make(
			chan int,
			len(tokens),
		)

	var wg sync.WaitGroup

	var nextToken atomic.Uint64

	start :=
		time.Now()

	for workerID := 0; workerID < workers; workerID++ {

		wg.Add(1)

		go func() {
			defer wg.Done()

			for jobIndex := range jobs {

				//
				// Atomic token selection guarantees that every request
				// gets a different token even if worker scheduling changes.
				//
				tokenIndex :=
					int(
						nextToken.Add(1) - 1,
					)

				if tokenIndex >= len(tokens) {
					results[jobIndex] =
						Result{
							Success: false,
							Err: fmt.Errorf(
								"token index out of range: %d",
								tokenIndex,
							),
						}
					continue
				}

				results[jobIndex] =
					doSidecarRequest(
						client,
						tokens[tokenIndex],
					)
			}
		}()
	}

	for i := 0; i < len(tokens); i++ {
		jobs <- i
	}

	close(jobs)

	wg.Wait()

	return results,
		time.Since(start)
}

func doBaselineRequest(
	client *http.Client,
) Result {

	req, err :=
		http.NewRequest(
			http.MethodGet,
			upstreamURL,
			nil,
		)

	if err != nil {
		return Result{
			Success: false,
			Err:     err,
		}
	}

	start :=
		time.Now()

	resp, err :=
		client.Do(req)

	if err != nil {
		return Result{
			Latency: time.Since(start),
			Success: false,
			Err:     err,
		}
	}

	defer resp.Body.Close()

	_, readErr :=
		io.Copy(
			io.Discard,
			resp.Body,
		)

	latency :=
		time.Since(start)

	if readErr != nil {
		return Result{
			Latency: latency,
			Success: false,
			Status:  resp.StatusCode,
			Err:     readErr,
		}
	}

	return Result{
		Latency: latency,
		Success: resp.StatusCode == http.StatusOK,
		Status:  resp.StatusCode,
	}
}

func doSidecarRequest(
	client *http.Client,
	token TokenPair,
) Result {

	req, err :=
		http.NewRequest(
			http.MethodGet,
			sidecarURL,
			nil,
		)

	if err != nil {
		return Result{
			Success: false,
			Err:     err,
		}
	}

	req.Header.Set(
		"X-HACP-Intent-Envelope",
		token.Env,
	)

	req.Header.Set(
		"X-HACP-Decision-Token",
		token.Tok,
	)

	start :=
		time.Now()

	resp, err :=
		client.Do(req)

	if err != nil {
		return Result{
			Latency: time.Since(start),
			Success: false,
			Err:     err,
		}
	}

	defer resp.Body.Close()

	_, readErr :=
		io.Copy(
			io.Discard,
			resp.Body,
		)

	latency :=
		time.Since(start)

	if readErr != nil {
		return Result{
			Latency: latency,
			Success: false,
			Status:  resp.StatusCode,
			Err:     readErr,
		}
	}

	return Result{
		Latency: latency,
		Success: resp.StatusCode == http.StatusOK,
		Status:  resp.StatusCode,
	}
}

func calculateStats(
	results []Result,
	total time.Duration,
) Stats {

	latencies :=
		make(
			[]time.Duration,
			0,
			len(results),
		)

	success := 0
	failed := 0

	for _, result := range results {

		if result.Success {
			success++
		} else {
			failed++
		}

		//
		// Include latency even for failed requests if a request was
		// actually attempted and produced a measurable duration.
		//
		if result.Latency > 0 {
			latencies =
				append(
					latencies,
					result.Latency,
				)
		}
	}

	sort.Slice(
		latencies,
		func(i, j int) bool {
			return latencies[i] <
				latencies[j]
		},
	)

	var totalLatency time.Duration
	var maxLatency time.Duration

	for _, latency := range latencies {

		totalLatency += latency

		if latency > maxLatency {
			maxLatency = latency
		}
	}

	var average time.Duration

	if len(latencies) > 0 {
		average =
			time.Duration(
				int64(totalLatency) /
					int64(len(latencies)),
			)
	}

	requestsPerSecond := 0.0

	if total > 0 {
		requestsPerSecond =
			float64(len(results)) /
				total.Seconds()
	}

	return Stats{
		Total:    total,
		Success:  success,
		Failed:   failed,
		Average:  average,
		P50:      percentile(latencies, 50),
		P95:      percentile(latencies, 95),
		P99:      percentile(latencies, 99),
		Max:      maxLatency,
		Requests: requestsPerSecond,
	}
}

func percentile(
	sorted []time.Duration,
	p float64,
) time.Duration {

	if len(sorted) == 0 {
		return 0
	}

	//
	// Nearest-rank percentile.
	//
	rank :=
		int(
			math.Ceil(
				(p / 100.0) *
					float64(len(sorted)),
			),
		)

	index := rank - 1

	if index < 0 {
		index = 0
	}

	if index >= len(sorted) {
		index =
			len(sorted) - 1
	}

	return sorted[index]
}

func printStats(
	name string,
	stats Stats,
) {

	fmt.Println()
	fmt.Printf("--- %s ---\n", name)

	fmt.Printf(
		"Total:        %s\n",
		formatDuration(stats.Total),
	)

	fmt.Printf(
		"Success:      %d / %d\n",
		stats.Success,
		stats.Success+stats.Failed,
	)

	fmt.Printf(
		"Failed:       %d\n",
		stats.Failed,
	)

	fmt.Printf(
		"Average:      %s\n",
		formatDuration(stats.Average),
	)

	fmt.Printf(
		"p50:          %s\n",
		formatDuration(stats.P50),
	)

	fmt.Printf(
		"p95:          %s\n",
		formatDuration(stats.P95),
	)

	fmt.Printf(
		"p99:          %s\n",
		formatDuration(stats.P99),
	)

	fmt.Printf(
		"max:          %s\n",
		formatDuration(stats.Max),
	)

	fmt.Printf(
		"Requests/sec: %.2f\n",
		stats.Requests,
	)
}

func formatDuration(
	d time.Duration,
) string {

	ms :=
		float64(d) /
			float64(time.Millisecond)

	return fmt.Sprintf(
		"%.2f ms",
		ms,
	)
}
