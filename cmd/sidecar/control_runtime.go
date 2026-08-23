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

// controlRuntime contains the control-plane-related dependencies consumed by
// the sidecar runtime.
//
// Standalone mode intentionally leaves ControlState and Subscriber nil.
//
// Distributed mode constructs one shared revocation store and one shared
// ControlState. The same ControlState is consumed by the evaluator,
// subscriber, and readiness predicate.
type controlRuntime struct {
	// Revocations is the read-only revocation view consumed by evaluation.
	//
	// Standalone mode uses an in-memory store.
	// Distributed mode uses the control-plane-backed adapter.
	Revocations evaluate.RevocationStore

	// LocalRevocations exposes the temporary local HTTP mutation surface only
	// in standalone mode.
	//
	// It is intentionally nil in distributed mode. Distributed revocation
	// authority belongs exclusively to the control plane.
	LocalRevocations *evaluate.InMemoryRevocationStore

	ControlState *controlplane.ControlState
	Subscriber   *controlplane.Subscriber
	Ready        func() bool
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
		store :=
			evaluate.NewInMemoryRevocationStore()

		return &controlRuntime{
			Revocations:      store,
			LocalRevocations: store,
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
