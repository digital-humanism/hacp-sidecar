// Rust HTTP local-response informational workload. Verification tooling only.
// This is a new client, not a reproduction of historical 70-http-performance.py.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"time"
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

func loadTokens(path string, expected int) ([]tokenPair, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	result := make([]tokenPair, 0, expected)
	for scanner.Scan() {
		if len(result) >= expected {
			return nil, fmt.Errorf("%s: more than %d fixture records", path, expected)
		}
		var tok tokenPair
		if err := json.Unmarshal(scanner.Bytes(), &tok); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, len(result)+1, err)
		}
		if tok.Env == "" || tok.Tok == "" {
			return nil, fmt.Errorf("%s line %d: empty env/tok", path, len(result)+1)
		}
		result = append(result, tok)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(result) != expected {
		return nil, fmt.Errorf("%s: expected %d records, got %d", path, expected, len(result))
	}
	return result, nil
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

func execute(phase string, client *http.Client, target string, tokens []tokenPair) (summary, error) {
	out := summary{Surface: "AUTHORIZATION_PLUS_LOCAL_RESPONSE", Phase: phase, Requests: len(tokens), Concurrency: 1}
	durations := make([]float64, 0, len(tokens))
	startAll := time.Now()
	for i, tok := range tokens {
		req, err := http.NewRequest(http.MethodGet, target, nil)
		if err != nil {
			return out, err
		}
		req.Header.Set("X-HACP-Intent-Envelope", tok.Env)
		req.Header.Set("X-HACP-Decision-Token", tok.Tok)
		started := time.Now()
		resp, err := client.Do(req)
		elapsed := float64(time.Since(started).Nanoseconds()) / 1e6
		durations = append(durations, elapsed)
		if err != nil {
			out.Failed++
			return out, fmt.Errorf("%s request %d transport: %w", phase, i+1, err)
		}
		_, readErr := io.Copy(io.Discard, resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil || closeErr != nil {
			out.Failed++
			return out, fmt.Errorf("%s request %d response read: %v; close: %v", phase, i+1, readErr, closeErr)
		}
		if resp.StatusCode != http.StatusOK || resp.Header.Get("X-HACP-Decision") != "ALLOW" || resp.Header.Get("X-HACP-Request-Id") == "" {
			out.Failed++
			return out, fmt.Errorf("%s request %d unexpected response: status=%d decision=%q request_id=%q reason=%q", phase, i+1, resp.StatusCode, resp.Header.Get("X-HACP-Decision"), resp.Header.Get("X-HACP-Request-Id"), resp.Header.Get("X-HACP-Reason"))
		}
		out.Success++
	}
	out.ElapsedMS = float64(time.Since(startAll).Nanoseconds()) / 1e6
	sort.Float64s(durations)
	var sum float64
	for _, d := range durations {
		sum += d
	}
	out.MeanMS = sum / float64(len(durations))
	out.P50MS = rank(durations, 0.50)
	out.P95MS = rank(durations, 0.95)
	out.P99MS = rank(durations, 0.99)
	out.MaxMS = durations[len(durations)-1]
	return out, nil
}

func run() error {
	url := flag.String("url", "http://127.0.0.1:18081/api/test", "Rust HTTP proxy local endpoint")
	warmup := flag.String("warmup-file", "", "fresh warmup JSONL fixture")
	measured := flag.String("measured-file", "", "fresh measured JSONL fixture")
	timeout := flag.Duration("timeout", 5*time.Second, "per-request HTTP timeout")
	flag.Parse()
	if *warmup == "" || *measured == "" {
		return errors.New("both -warmup-file and -measured-file are required")
	}
	if *timeout <= 0 {
		return errors.New("timeout must be positive")
	}
	if *url != "http://127.0.0.1:18081/api/test" {
		return errors.New("only the pinned loopback R8.2 target is permitted")
	}
	w, err := loadTokens(*warmup, 100)
	if err != nil {
		return err
	}
	m, err := loadTokens(*measured, 5000)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: *timeout, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	for _, phase := range []struct {
		name   string
		tokens []tokenPair
	}{{"warmup", w}, {"measured", m}} {
		s, err := execute(phase.name, client, *url, phase.tokens)
		enc := json.NewEncoder(os.Stdout)
		if jsonErr := enc.Encode(s); jsonErr != nil {
			return jsonErr
		}
		if err != nil {
			return err
		}
	}
	fmt.Println("R8_2_WORKLOAD=PASS")
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "R8_2_WORKLOAD=FAIL:", err)
		os.Exit(1)
	}
}
