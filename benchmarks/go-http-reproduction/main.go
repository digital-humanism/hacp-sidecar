// Go HTTP local reproduction client (B15). Verification tooling only.
// Accepted boundary: authorization + real upstream forwarding + real upstream response.
// The mandatory negative fail-closed preflight is performed by the procedure
// document (docs/verification/go-http-local-reproduction.md), not by this client.
//
// Timing metrics are INFORMATIONAL. No latency threshold and no Gate D
// threshold determine PASS/FAIL.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"time"
)

const (
	pinnedURL     = "http://127.0.0.1:18082/api/test"
	warmupCount   = 100
	measuredCount = 5000
	surfaceName   = "AUTHORIZATION_PLUS_REAL_FORWARDING"

	exitOK          = 0
	exitBehavioral  = 1
	exitFixture     = 2
	exitEnvironment = 3
	// exitCleanup (4) is reserved for the orchestrator; this client owns
	// no processes.
)

type tokenPair struct {
	Env string `json:"env"`
	Tok string `json:"tok"`
}

type summary struct {
	Surface     string  `json:"surface"`
	Phase       string  `json:"phase"`
	Requests    int     `json:"requests"`
	Success     int     `json:"success"`
	Failed      int     `json:"failed"`
	Concurrency int     `json:"concurrency"`
	ElapsedMS   float64 `json:"elapsed_ms"`
	MeanMS      float64 `json:"mean_ms"`
	P50MS       float64 `json:"p50_ms"`
	P95MS       float64 `json:"p95_ms"`
	P99MS       float64 `json:"p99_ms"`
	MaxMS       float64 `json:"max_ms"`
}

type classError struct {
	code   int
	reason string
}

func (e *classError) Error() string {
	return e.reason
}

func abort(code int, reason string) {
	fmt.Println("GO_HTTP_REPRODUCTION=FAIL")
	fmt.Fprintf(os.Stderr, "GO_HTTP_FAIL[code=%d]: %s\n", code, reason)
	os.Exit(code)
}

func abortError(err error) {
	if ce, ok := err.(*classError); ok {
		abort(ce.code, ce.reason)
	}
	abort(exitFixture, err.Error())
}

func loadTokens(path string, expected int) ([]tokenPair, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open fixture %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	records := 0
	result := make([]tokenPair, 0, expected)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytesTrimSpace(line)) == 0 {
			return nil, fmt.Errorf("%s line %d: empty record", path, records+1)
		}
		records++
		if records > expected {
			return nil, fmt.Errorf("%s: more than %d records", path, expected)
		}

		var tok tokenPair
		if err := json.Unmarshal(line, &tok); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, records, err)
		}
		if tok.Env == "" || tok.Tok == "" {
			return nil, fmt.Errorf("%s line %d: empty env/tok", path, records)
		}

		result = append(result, tok)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	if records != expected {
		return nil, fmt.Errorf("%s: expected %d records, got %d", path, expected, records)
	}

	return result, nil
}

func bytesTrimSpace(b []byte) []byte {
	start := 0
	for start < len(b) {
		c := b[start]
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			break
		}
		start++
	}
	return b[start:]
}

func rank(sorted []float64, percentile float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	i := int(math.Ceil(percentile*float64(len(sorted)))) - 1
	if i < 0 {
		i = 0
	}
	if i >= len(sorted) {
		i = len(sorted) - 1
	}
	return sorted[i]
}

func execute(phase string, client *http.Client, tokens []tokenPair) (summary, error) {
	out := summary{
		Surface:     surfaceName,
		Phase:       phase,
		Requests:    len(tokens),
		Concurrency: 1,
	}

	durations := make([]float64, 0, len(tokens))
	startAll := time.Now()

	for i, tok := range tokens {
		req, err := http.NewRequest(http.MethodGet, pinnedURL, nil)
		if err != nil {
			out.Failed++
			return out, &classError{exitEnvironment, fmt.Sprintf("%s request %d: build request: %v", phase, i+1, err)}
		}

		req.Header.Set("X-HACP-Intent-Envelope", tok.Env)
		req.Header.Set("X-HACP-Decision-Token", tok.Tok)

		started := time.Now()
		resp, err := client.Do(req)
		elapsed := float64(time.Since(started).Nanoseconds()) / 1e6

		if err != nil {
			out.Failed++
			return out, &classError{exitEnvironment, fmt.Sprintf("%s request %d transport: %v", phase, i+1, err)}
		}

		body, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil || closeErr != nil {
			out.Failed++
			return out, &classError{exitEnvironment, fmt.Sprintf("%s request %d response read=%v close=%v", phase, i+1, readErr, closeErr)}
		}

		durations = append(durations, elapsed)

		decision := resp.Header.Get("X-HACP-Decision")
		requestID := resp.Header.Get("X-HACP-Request-Id")

		// A5: 502 + ALLOW means the upstream transport failed after a
		// successful authorization. This is an environment/topology
		// failure, not a success and not a DENY.
		if resp.StatusCode == http.StatusBadGateway && decision == "ALLOW" {
			out.Failed++
			return out, &classError{
				exitEnvironment,
				fmt.Sprintf("%s request %d (A5): 502 + X-HACP-Decision=ALLOW — upstream transport failure", phase, i+1),
			}
		}

		// A1
		if resp.StatusCode != http.StatusOK {
			out.Failed++
			return out, &classError{
				exitBehavioral,
				fmt.Sprintf("%s request %d (A1): status=%d, want 200; decision=%q request_id=%q", phase, i+1, resp.StatusCode, decision, requestID),
			}
		}

		// A2
		if decision != "ALLOW" {
			out.Failed++
			return out, &classError{
				exitBehavioral,
				fmt.Sprintf("%s request %d (A2): X-HACP-Decision=%q, want ALLOW", phase, i+1, decision),
			}
		}

		// A3
		if requestID == "" {
			out.Failed++
			return out, &classError{
				exitBehavioral,
				fmt.Sprintf("%s request %d (A3): empty X-HACP-Request-Id", phase, i+1),
			}
		}

		// A4: real upstream response body must carry the mock-upstream
		// path marker. Parsing the JSON field keeps the check robust to
		// JSON rendering (the mock upstream emits compact JSON).
		var upstreamBody struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(body, &upstreamBody); err != nil || upstreamBody.Path != "/api/test" {
			out.Failed++
			return out, &classError{
				exitBehavioral,
				fmt.Sprintf("%s request %d (A4): upstream origin marker path=/api/test not found in body", phase, i+1),
			}
		}

		out.Success++
	}

	out.ElapsedMS = float64(time.Since(startAll).Nanoseconds()) / 1e6

	sort.Float64s(durations)

	var sum float64
	for _, d := range durations {
		sum += d
	}
	if len(durations) > 0 {
		out.MeanMS = sum / float64(len(durations))
	}

	out.P50MS = rank(durations, 0.50)
	out.P95MS = rank(durations, 0.95)
	out.P99MS = rank(durations, 0.99)
	if len(durations) > 0 {
		out.MaxMS = durations[len(durations)-1]
	}

	return out, nil
}

func emitSummary(s summary) {
	if err := json.NewEncoder(os.Stdout).Encode(s); err != nil {
		abort(exitEnvironment, fmt.Sprintf("write summary: %v", err))
	}
}

func main() {
	url := flag.String("url", pinnedURL, "pinned reproduction target (only the pinned loopback URL is permitted)")
	warmupFile := flag.String("warmup-file", "", "fresh warmup JSONL fixture (exactly 100 records)")
	measuredFile := flag.String("measured-file", "", "fresh measured JSONL fixture (exactly 5000 records)")
	timeout := flag.Duration("timeout", 5*time.Second, "per-request HTTP timeout")
	flag.Parse()

	if *url != pinnedURL {
		abort(exitEnvironment, fmt.Sprintf("only the pinned target %s is permitted, got %s", pinnedURL, *url))
	}
	if *warmupFile == "" || *measuredFile == "" {
		abort(exitFixture, "both -warmup-file and -measured-file are required")
	}
	if *timeout <= 0 {
		abort(exitFixture, "timeout must be positive")
	}

	warmup, err := loadTokens(*warmupFile, warmupCount)
	if err != nil {
		abort(exitFixture, err.Error())
	}
	measured, err := loadTokens(*measuredFile, measuredCount)
	if err != nil {
		abort(exitFixture, err.Error())
	}

	fmt.Printf("WARMUP_FIXTURE_RECORDS=%d\n", len(warmup))
	fmt.Printf("MEASURED_FIXTURE_RECORDS=%d\n", len(measured))

	client := &http.Client{
		Timeout: *timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Warm-up phase. Any failure stops the run; the measured phase is
	// not started.
	warmupSummary, err := execute("warmup", client, warmup)
	emitSummary(warmupSummary)
	if err != nil {
		abortError(err)
	}

	// Measured phase, serial (concurrency = 1), one fresh token pair per
	// request.
	measuredSummary, err := execute("measured", client, measured)
	emitSummary(measuredSummary)
	if err != nil {
		abortError(err)
	}

	fmt.Println("GO_HTTP_REPRODUCTION=PASS")
	os.Exit(exitOK)
}
