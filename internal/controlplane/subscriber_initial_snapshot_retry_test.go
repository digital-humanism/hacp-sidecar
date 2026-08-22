package controlplane

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	controlplanev1 "hacp-sidecar/gen/controlplane/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type initialSnapshotRetryClient struct {
	snapshotCalls atomic.Int32
}

func (c *initialSnapshotRetryClient) GetRevocationSnapshot(
	context.Context,
	*controlplanev1.GetRevocationSnapshotRequest,
	...grpc.CallOption,
) (*controlplanev1.RevocationSnapshot, error) {
	call := c.snapshotCalls.Add(1)

	if call == 1 {
		return nil, status.Error(
			codes.Unavailable,
			"control plane temporarily unavailable",
		)
	}

	return &controlplanev1.RevocationSnapshot{
		Revision: 7,
	}, nil
}

func (*initialSnapshotRetryClient) WatchRevocations(
	context.Context,
	*controlplanev1.WatchRevocationsRequest,
	...grpc.CallOption,
) (
	grpc.ServerStreamingClient[controlplanev1.WatchRevocationsResponse],
	error,
) {
	return nil, status.Error(
		codes.Unavailable,
		"watch unavailable",
	)
}

func TestSubscriberRetriesInitialSnapshotUntilAvailable(
	t *testing.T,
) {
	client :=
		&initialSnapshotRetryClient{}

	store :=
		newTestRevocationStore()

	state :=
		NewControlState(
			5 * time.Second,
		)

	subscriber :=
		NewSubscriber(
			client,
			store,
			"sidecar-initial-snapshot-retry-test",
		)

	subscriber.SetControlState(
		state,
	)

	subscriber.reconnectInitialBackoff =
		time.Millisecond

	subscriber.reconnectMaxBackoff =
		time.Millisecond

	ctx, cancel :=
		context.WithCancel(
			context.Background(),
		)

	errCh :=
		make(chan error, 1)

	go func() {
		errCh <- subscriber.Run(ctx)
	}()

	requireEventually(
		t,
		func() bool {
			return client.snapshotCalls.Load() >= 2 &&
				subscriber.LastSeenRevision() == 7 &&
				state.IsFresh(time.Now())
		},
	)

	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(
			err,
			context.Canceled,
		) {
			t.Fatalf(
				"Run() error = %v, want context.Canceled",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal(
			"subscriber did not stop after context cancellation",
		)
	}
}
