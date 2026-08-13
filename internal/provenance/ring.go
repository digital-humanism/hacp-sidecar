package provenance

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"hacp-sidecar/internal/evaluate"
	"hacp-sidecar/internal/wire"
)

// Event is a provenance record
type Event struct {
	Timestamp  int64  `json:"timestamp"`
	RequestID  string `json:"request_id"`
	EnvelopeID string `json:"envelope_id"`
	TokenID    string `json:"token_id"`
	ActionHash string `json:"action_hash"`
	Decision   string `json:"decision"`
	ReasonCode string `json:"reason_code,omitempty"`
	LatencyNs  int64  `json:"latency_ns"`
}

// RingBuffer is a bounded, asynchronous provenance buffer
type RingBuffer struct {
	mu        sync.Mutex
	events    []Event
	capacity  int
	head      int
	count     int
	flushPath string
	flushCh   chan struct{}
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

// NewRingBuffer creates a ring buffer with async flush worker
func NewRingBuffer(capacity int, flushPath string) *RingBuffer {
	rb := &RingBuffer{
		events:    make([]Event, capacity),
		capacity:  capacity,
		flushPath: flushPath,
		flushCh:   make(chan struct{}, 1),
		stopCh:    make(chan struct{}),
	}
	rb.wg.Add(1)
	go rb.flushWorker()
	return rb
}

// Accept records a provenance event. Returns error if buffer cannot accept.
// Per enforcement.md §11: "If the ring buffer cannot accept a record,
// the request MUST be denied with TRACEABILITY_FAILURE."
// Note: In this implementation, overwrite is NOT allowed (fail-closed).
// Accept implements evaluate.ProvenanceWriter interface
func (rb *RingBuffer) Accept(env *wire.IntentEnvelope, tok *wire.DecisionToken, req *evaluate.RequestContext) error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count >= rb.capacity {
		return fmt.Errorf("ring buffer full: %d/%d", rb.count, rb.capacity)
	}

	idx := (rb.head + rb.count) % rb.capacity
	rb.events[idx] = Event{
		Timestamp:  time.Now().Unix(),
		RequestID:  req.RequestID,
		EnvelopeID: env.EnvelopeID,
		TokenID:    tok.TokenID,
		ActionHash: tok.ActionHash,
		Decision:   "ALLOW",
		ReasonCode: "",
		LatencyNs:  req.LatencyNs,
	}
	rb.count++

	// Signal flush worker (non-blocking)
	select {
	case rb.flushCh <- struct{}{}:
	default:
	}

	return nil
}

// AcceptDenied records a denied request
func (rb *RingBuffer) AcceptDenied(env *wire.IntentEnvelope, tok *wire.DecisionToken, req *evaluate.RequestContext, reason string) error {
	if env == nil || tok == nil {
		return fmt.Errorf("cannot record denied request without envelope/token")
	}

	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count >= rb.capacity {
		return fmt.Errorf("ring buffer full: %d/%d", rb.count, rb.capacity)
	}

	idx := (rb.head + rb.count) % rb.capacity
	rb.events[idx] = Event{
		Timestamp:  time.Now().Unix(),
		RequestID:  req.RequestID,
		EnvelopeID: env.EnvelopeID,
		TokenID:    tok.TokenID,
		ActionHash: tok.ActionHash,
		Decision:   "DENY",
		ReasonCode: reason,
		LatencyNs:  req.LatencyNs,
	}
	rb.count++

	select {
	case rb.flushCh <- struct{}{}:
	default:
	}

	return nil
}

// flushWorker periodically drains the buffer to disk
func (rb *RingBuffer) flushWorker() {
	defer rb.wg.Done()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rb.stopCh:
			rb.flush()
			return
		case <-ticker.C:
			rb.flush()
		case <-rb.flushCh:
			rb.flush()
		}
	}
}

// flush writes buffered events to disk (one file per flush for simplicity)
func (rb *RingBuffer) flush() {
	rb.mu.Lock()
	if rb.count == 0 {
		rb.mu.Unlock()
		return
	}

	// Copy events to flush outside the lock
	toFlush := make([]Event, rb.count)
	for i := 0; i < rb.count; i++ {
		idx := (rb.head + i) % rb.capacity
		toFlush[i] = rb.events[idx]
	}
	rb.head = (rb.head + rb.count) % rb.capacity
	rb.count = 0
	rb.mu.Unlock()

	// Append to file
	f, err := os.OpenFile(rb.flushPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Printf("provenance: flush open error: %v", err)
		return
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, e := range toFlush {
		if err := enc.Encode(e); err != nil {
			log.Printf("provenance: flush encode error: %v", err)
		}
	}
}

// Stop gracefully stops the flush worker
func (rb *RingBuffer) Stop() {
	close(rb.stopCh)
	rb.wg.Wait()
}

// PendingCount returns the number of buffered events (for testing)
func (rb *RingBuffer) PendingCount() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.count
}
