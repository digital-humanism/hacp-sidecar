package main

import (
	"context"
	"errors"
	"testing"
	"time"

	controlplanev1 "hacp-sidecar/gen/controlplane/v1"

	"google.golang.org/grpc"
)

type runtimeTestControlPlaneClient struct{}

func (runtimeTestControlPlaneClient) GetRevocationSnapshot(
	context.Context,
	*controlplanev1.GetRevocationSnapshotRequest,
	...grpc.CallOption,
) (*controlplanev1.RevocationSnapshot, error) {

	return nil, errors.New("not used")
}

func (runtimeTestControlPlaneClient) WatchRevocations(
	context.Context,
	*controlplanev1.WatchRevocationsRequest,
	...grpc.CallOption,
) (
	grpc.ServerStreamingClient[controlplanev1.WatchRevocationsResponse],
	error,
) {

	return nil, errors.New("not used")
}

func TestNewControlRuntimeStandalone(
	t *testing.T,
) {

	cfg :=
		controlRuntimeConfig{
			Mode:         controlModeStandalone,
			MaxStaleness: defaultControlMaxStaleness,
		}

	runtime, err :=
		newControlRuntime(
			cfg,
			nil,
			time.Now,
		)

	if err != nil {
		t.Fatalf(
			"newControlRuntime() error = %v",
			err,
		)
	}

	if runtime.Revocations == nil {
		t.Fatal(
			"expected standalone revocation store",
		)
	}

	if runtime.LocalRevocations == nil {
		t.Fatal(
			"standalone runtime must expose local revocation mutations",
		)
	}

	if runtime.Revocations != runtime.LocalRevocations {
		t.Fatal(
			"standalone evaluator and local mutation API must share one revocation store",
		)
	}

	if runtime.ControlState != nil {
		t.Fatal(
			"expected standalone ControlState to be nil",
		)
	}

	if runtime.Subscriber != nil {
		t.Fatal(
			"expected standalone Subscriber to be nil",
		)
	}

	if runtime.Ready == nil {
		t.Fatal(
			"expected standalone readiness predicate",
		)
	}

	if !runtime.Ready() {
		t.Fatal(
			"expected standalone runtime to be ready",
		)
	}
}

func TestNewControlRuntimeDistributedRequiresClient(
	t *testing.T,
) {

	cfg :=
		controlRuntimeConfig{
			Mode:         controlModeDistributed,
			Address:      "control-plane:5000",
			SidecarID:    "sidecar-1",
			MaxStaleness: defaultControlMaxStaleness,
		}

	_, err :=
		newControlRuntime(
			cfg,
			nil,
			time.Now,
		)

	if !errors.Is(
		err,
		errMissingControlPlaneClient,
	) {
		t.Fatalf(
			"error = %v, want errMissingControlPlaneClient",
			err,
		)
	}
}

func TestNewControlRuntimeDistributedStartsNotReady(
	t *testing.T,
) {

	cfg :=
		controlRuntimeConfig{
			Mode:         controlModeDistributed,
			Address:      "control-plane:5000",
			SidecarID:    "sidecar-1",
			MaxStaleness: defaultControlMaxStaleness,
		}

	runtime, err :=
		newControlRuntime(
			cfg,
			runtimeTestControlPlaneClient{},
			time.Now,
		)

	if err != nil {
		t.Fatalf(
			"newControlRuntime() error = %v",
			err,
		)
	}

	if runtime.ControlState == nil {
		t.Fatal(
			"expected distributed ControlState",
		)
	}

	if runtime.Subscriber == nil {
		t.Fatal(
			"expected distributed Subscriber",
		)
	}

	if runtime.Ready == nil {
		t.Fatal(
			"expected distributed readiness predicate",
		)
	}

	if runtime.Ready() {
		t.Fatal(
			"expected distributed runtime to start not ready",
		)
	}
}

func TestNewControlRuntimeDistributedSharesControlState(
	t *testing.T,
) {

	cfg :=
		controlRuntimeConfig{
			Mode:         controlModeDistributed,
			Address:      "control-plane:5000",
			SidecarID:    "sidecar-1",
			MaxStaleness: defaultControlMaxStaleness,
		}

	runtime, err :=
		newControlRuntime(
			cfg,
			runtimeTestControlPlaneClient{},
			time.Now,
		)

	if err != nil {
		t.Fatalf(
			"newControlRuntime() error = %v",
			err,
		)
	}

	if runtime.Subscriber.ControlState() !=
		runtime.ControlState {

		t.Fatal(
			"subscriber and runtime do not share the same ControlState",
		)
	}
}

func TestNewControlRuntimeDistributedReadinessTracksSharedState(
	t *testing.T,
) {

	now :=
		time.Date(
			2026,
			time.August,
			22,
			12,
			0,
			0,
			0,
			time.UTC,
		)

	cfg :=
		controlRuntimeConfig{
			Mode:         controlModeDistributed,
			Address:      "control-plane:5000",
			SidecarID:    "sidecar-1",
			MaxStaleness: defaultControlMaxStaleness,
		}

	runtime, err :=
		newControlRuntime(
			cfg,
			runtimeTestControlPlaneClient{},
			func() time.Time {
				return now
			},
		)

	if err != nil {
		t.Fatalf(
			"newControlRuntime() error = %v",
			err,
		)
	}

	if runtime.Ready() {
		t.Fatal(
			"expected initial distributed state to be not ready",
		)
	}

	runtime.ControlState.MarkSnapshot(
		1,
		now,
	)

	if !runtime.Ready() {
		t.Fatal(
			"expected shared fresh state to make runtime ready",
		)
	}

	runtime.ControlState.MarkUnsafe()

	if runtime.Ready() {
		t.Fatal(
			"expected shared unsafe state to make runtime not ready",
		)
	}

	runtime.ControlState.MarkSnapshot(
		2,
		now,
	)

	if !runtime.Ready() {
		t.Fatal(
			"expected recovered shared state to restore readiness",
		)
	}
}

func TestNewControlRuntimeDistributedDisablesLocalRevocationMutation(
	t *testing.T,
) {
	cfg :=
		controlRuntimeConfig{
			Mode:         controlModeDistributed,
			Address:      "control-plane:5000",
			SidecarID:    "sidecar-1",
			MaxStaleness: defaultControlMaxStaleness,
		}

	runtime, err :=
		newControlRuntime(
			cfg,
			runtimeTestControlPlaneClient{},
			time.Now,
		)

	if err != nil {
		t.Fatalf(
			"newControlRuntime() error = %v",
			err,
		)
	}

	if runtime.LocalRevocations != nil {
		t.Fatal(
			"distributed runtime must not expose local revocation mutations",
		)
	}

	if runtime.Revocations == nil {
		t.Fatal(
			"distributed runtime must provide evaluator revocation state",
		)
	}
}
