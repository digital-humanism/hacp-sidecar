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

	controlTLSModeRequired = "required"
	controlTLSModeInsecure = "insecure"

	defaultControlMaxStaleness = 5 * time.Second
)

var (
	errUnsupportedControlMode = errors.New(
		"unsupported HACP_CONTROL_MODE",
	)

	errMissingControlPlaneAddress = errors.New(
		"HACP_CONTROL_PLANE_ADDR is required in distributed mode",
	)

	errMissingSidecarID = errors.New(
		"HACP_SIDECAR_ID is required in distributed mode",
	)

	errInvalidControlMaxStaleness = errors.New(
		"invalid HACP_CONTROL_MAX_STALENESS",
	)

	errAmbiguousStandaloneControl = errors.New(
		"standalone mode must not include distributed control-plane configuration",
	)

	errUnsupportedControlTLSMode = errors.New(
		"unsupported HACP_CONTROL_TLS_MODE",
	)

	errInsecureControlTLSOptions = errors.New(
		"insecure control-plane transport must not include TLS configuration",
	)
)

type controlRuntimeConfig struct {
	Mode         string
	Address      string
	SidecarID    string
	MaxStaleness time.Duration

	TLSMode    string
	CAFile     string
	ServerName string
}

// loadControlRuntimeConfig parses and validates the sidecar's distributed
// control-plane runtime configuration.
//
// Standalone mode is the default and must not include distributed
// control-plane settings.
//
// Distributed mode requires an explicit control-plane address and stable
// sidecar identifier. Max staleness defaults to five seconds when omitted.
//
// Distributed control-plane transport requires TLS by default. Plaintext
// transport is available only through explicit insecure opt-in.
func loadControlRuntimeConfig() (controlRuntimeConfig, error) {
	mode := strings.TrimSpace(
		os.Getenv("HACP_CONTROL_MODE"),
	)

	if mode == "" {
		mode = controlModeStandalone
	}

	address := strings.TrimSpace(
		os.Getenv("HACP_CONTROL_PLANE_ADDR"),
	)

	sidecarID := strings.TrimSpace(
		os.Getenv("HACP_SIDECAR_ID"),
	)

	maxStalenessRaw := strings.TrimSpace(
		os.Getenv("HACP_CONTROL_MAX_STALENESS"),
	)

	tlsModeRaw := strings.TrimSpace(
		os.Getenv("HACP_CONTROL_TLS_MODE"),
	)

	caFile := strings.TrimSpace(
		os.Getenv("HACP_CONTROL_CA_FILE"),
	)

	serverName := strings.TrimSpace(
		os.Getenv("HACP_CONTROL_SERVER_NAME"),
	)

	maxStaleness := defaultControlMaxStaleness
	if maxStalenessRaw != "" {
		parsed, err :=
			time.ParseDuration(
				maxStalenessRaw,
			)

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
		if address != "" ||
			sidecarID != "" ||
			tlsModeRaw != "" ||
			caFile != "" ||
			serverName != "" {

			return controlRuntimeConfig{},
				errAmbiguousStandaloneControl
		}

		return controlRuntimeConfig{
			Mode:         controlModeStandalone,
			MaxStaleness: maxStaleness,
		}, nil

	case controlModeDistributed:
		if address == "" {
			return controlRuntimeConfig{},
				errMissingControlPlaneAddress
		}

		if sidecarID == "" {
			return controlRuntimeConfig{},
				errMissingSidecarID
		}

	default:
		return controlRuntimeConfig{}, fmt.Errorf(
			"%w: %q",
			errUnsupportedControlMode,
			mode,
		)
	}

	tlsMode := tlsModeRaw
	if tlsMode == "" {
		tlsMode = controlTLSModeRequired
	}

	switch tlsMode {
	case controlTLSModeRequired:
		// System roots are used when CAFile is empty.

	case controlTLSModeInsecure:
		if caFile != "" || serverName != "" {
			return controlRuntimeConfig{},
				errInsecureControlTLSOptions
		}

	default:
		return controlRuntimeConfig{}, fmt.Errorf(
			"%w: %q",
			errUnsupportedControlTLSMode,
			tlsMode,
		)
	}

	return controlRuntimeConfig{
		Mode:         mode,
		Address:      address,
		SidecarID:    sidecarID,
		MaxStaleness: maxStaleness,

		TLSMode:    tlsMode,
		CAFile:     caFile,
		ServerName: serverName,
	}, nil
}
