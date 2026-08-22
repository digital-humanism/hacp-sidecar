package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	controlModeStandalone  = "standalone"
	controlModeDistributed = "distributed"

	defaultControlMaxStaleness = 5 * time.Second
)

var (
	errUnsupportedControlMode     = errors.New("unsupported HACP_CONTROL_MODE")
	errMissingControlPlaneAddress = errors.New("HACP_CONTROL_PLANE_ADDR is required in distributed mode")
	errMissingSidecarID           = errors.New("HACP_SIDECAR_ID is required in distributed mode")
	errInvalidControlMaxStaleness = errors.New("invalid HACP_CONTROL_MAX_STALENESS")
	errAmbiguousStandaloneControl = errors.New("standalone mode must not include distributed control-plane configuration")
)

type controlRuntimeConfig struct {
	Mode         string
	Address      string
	SidecarID    string
	MaxStaleness time.Duration
}

// loadControlRuntimeConfig parses and validates the sidecar's distributed
// control-plane runtime configuration.
//
// Standalone mode is the default and does not require distributed
// control-plane settings.
//
// Distributed mode requires an explicit control-plane address and stable
// sidecar identifier. Max staleness defaults to five seconds when omitted.
func loadControlRuntimeConfig() (controlRuntimeConfig, error) {
	mode := strings.TrimSpace(os.Getenv("HACP_CONTROL_MODE"))
	if mode == "" {
		mode = controlModeStandalone
	}

	address := strings.TrimSpace(os.Getenv("HACP_CONTROL_PLANE_ADDR"))
	sidecarID := strings.TrimSpace(os.Getenv("HACP_SIDECAR_ID"))
	maxStalenessRaw := strings.TrimSpace(
		os.Getenv("HACP_CONTROL_MAX_STALENESS"),
	)

	maxStaleness := defaultControlMaxStaleness
	if maxStalenessRaw != "" {
		parsed, err := time.ParseDuration(maxStalenessRaw)
		if err != nil {
			return controlRuntimeConfig{}, fmt.Errorf(
				"%w: %q",
				errInvalidControlMaxStaleness,
				maxStalenessRaw,
			)
		}

		if parsed <= 0 {
			return controlRuntimeConfig{}, fmt.Errorf(
				"%w: must be greater than zero",
				errInvalidControlMaxStaleness,
			)
		}

		maxStaleness = parsed
	}

	switch mode {
	case controlModeStandalone:
		if address != "" || sidecarID != "" {
			return controlRuntimeConfig{}, errAmbiguousStandaloneControl
		}

	case controlModeDistributed:
		if address == "" {
			return controlRuntimeConfig{}, errMissingControlPlaneAddress
		}

		if sidecarID == "" {
			return controlRuntimeConfig{}, errMissingSidecarID
		}

	default:
		return controlRuntimeConfig{}, fmt.Errorf(
			"%w: %q",
			errUnsupportedControlMode,
			mode,
		)
	}

	return controlRuntimeConfig{
		Mode:         mode,
		Address:      address,
		SidecarID:    sidecarID,
		MaxStaleness: maxStaleness,
	}, nil
}
