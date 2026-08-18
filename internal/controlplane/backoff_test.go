package controlplane

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNextReconnectBackoff(t *testing.T) {
	const (
		initial = 100 * time.Millisecond
		maximum = 800 * time.Millisecond
	)

	tests := []struct {
		name    string
		current time.Duration
		want    time.Duration
	}{
		{
			name:    "first retry uses initial backoff",
			current: 0,
			want:    initial,
		},
		{
			name:    "backoff doubles",
			current: 100 * time.Millisecond,
			want:    200 * time.Millisecond,
		},
		{
			name:    "backoff continues doubling",
			current: 200 * time.Millisecond,
			want:    400 * time.Millisecond,
		},
		{
			name:    "backoff reaches maximum",
			current: 400 * time.Millisecond,
			want:    maximum,
		},
		{
			name:    "backoff is capped",
			current: maximum,
			want:    maximum,
		},
		{
			name:    "backoff above maximum remains capped",
			current: 2 * time.Second,
			want:    maximum,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextReconnectBackoff(
				tt.current,
				initial,
				maximum,
			)

			if got != tt.want {
				t.Fatalf(
					"nextReconnectBackoff(%s) = %s, want %s",
					tt.current,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestWaitForReconnectHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()

	err := waitForReconnect(
		ctx,
		10*time.Second,
	)

	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"waitForReconnect error = %v, want %v",
			err,
			context.Canceled,
		)
	}

	if elapsed >= time.Second {
		t.Fatalf(
			"waitForReconnect took %s after context cancellation; want prompt cancellation",
			elapsed,
		)
	}
}
