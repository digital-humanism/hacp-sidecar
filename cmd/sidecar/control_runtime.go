package main

import (
	"errors"
	"time"

	controlplanev1 "hacp-sidecar/gen/controlplane/v1"
	"hacp-sidecar/internal/controlplane"
	"hacp-sidecar/internal/evaluate"
)

var (
	errMissingControlPlaneClient = errors.New(
		"distributed control runtime requires a control-plane client",
	)
)

// runtimeRevocationStore is the shared revocation contract required by the
// sidecar runtime.
//
// Both the standalone in-memory store and the distributed control-plane
// adapter satisfy this interface.
type runtimeRevocationStore interface {
	evaluate.RevocationStore

	RevokeKey(string)
	RevokeToken(string)
	RevokeEnvelope(string)
}

// controlRuntime contains the control-plane-related dependencies consumed by
// the sidecar runtime.
//
// Standalone mode intentionally leaves ControlState and Subscriber nil.
//
// Distributed mode constructs one shared revocation store and one shared
// ControlState. The same ControlState is consumed by the evaluator,
// subscriber, and readiness predicate.
type controlRuntime struct {
	Revocations runtimeRevocationStore

	ControlState *controlplane.ControlState
	Subscriber   *controlplane.Subscriber

	Ready func() bool
}

// newControlRuntime constructs the control-plane dependency graph without
// starting network activity or background goroutines.
func newControlRuntime(
	cfg controlRuntimeConfig,
	client controlplanev1.ControlPlaneClient,
	now func() time.Time,
) (*controlRuntime, error) {

	switch cfg.Mode {
	case controlModeStandalone:
		return &controlRuntime{
			Revocations: evaluate.NewInMemoryRevocationStore(),
			Ready: func() bool {
				return true
			},
		}, nil

	case controlModeDistributed:
		if client == nil {
			return nil, errMissingControlPlaneClient
		}

		store :=
			controlplane.NewRevocationStoreAdapter()

		state :=
			controlplane.NewControlState(
				cfg.MaxStaleness,
			)

		subscriber :=
			controlplane.NewSubscriber(
				client,
				store,
				cfg.SidecarID,
			)

		subscriber.SetControlState(
			state,
		)

		return &controlRuntime{
			Revocations:  store,
			ControlState: state,
			Subscriber:   subscriber,
			Ready: makeControlStateReadiness(
				state,
				now,
			),
		}, nil

	default:
		return nil, errUnsupportedControlMode
	}
}
