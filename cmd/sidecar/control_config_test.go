package main

import (
	"errors"
	"testing"
	"time"
)

func clearControlRuntimeEnv(t *testing.T) {
	t.Helper()

	t.Setenv("HACP_CONTROL_MODE", "")
	t.Setenv("HACP_CONTROL_PLANE_ADDR", "")
	t.Setenv("HACP_SIDECAR_ID", "")
	t.Setenv("HACP_CONTROL_MAX_STALENESS", "")
	t.Setenv("HACP_CONTROL_TLS_MODE", "")
	t.Setenv("HACP_CONTROL_CA_FILE", "")
	t.Setenv("HACP_CONTROL_SERVER_NAME", "")
}

func TestLoadControlRuntimeConfigDefaultsToStandalone(
	t *testing.T,
) {
	clearControlRuntimeEnv(t)

	cfg, err := loadControlRuntimeConfig()
	if err != nil {
		t.Fatalf(
			"loadControlRuntimeConfig() error = %v",
			err,
		)
	}

	if cfg.Mode != controlModeStandalone {
		t.Fatalf(
			"mode = %q, want %q",
			cfg.Mode,
			controlModeStandalone,
		)
	}

	if cfg.Address != "" {
		t.Fatalf(
			"address = %q, want empty",
			cfg.Address,
		)
	}

	if cfg.SidecarID != "" {
		t.Fatalf(
			"sidecar ID = %q, want empty",
			cfg.SidecarID,
		)
	}

	if cfg.MaxStaleness != defaultControlMaxStaleness {
		t.Fatalf(
			"max staleness = %v, want %v",
			cfg.MaxStaleness,
			defaultControlMaxStaleness,
		)
	}
}

func TestLoadControlRuntimeConfigExplicitStandalone(
	t *testing.T,
) {
	clearControlRuntimeEnv(t)

	t.Setenv(
		"HACP_CONTROL_MODE",
		controlModeStandalone,
	)

	cfg, err := loadControlRuntimeConfig()
	if err != nil {
		t.Fatalf(
			"loadControlRuntimeConfig() error = %v",
			err,
		)
	}

	if cfg.Mode != controlModeStandalone {
		t.Fatalf(
			"mode = %q, want %q",
			cfg.Mode,
			controlModeStandalone,
		)
	}
}

func TestLoadControlRuntimeConfigDistributed(
	t *testing.T,
) {
	clearControlRuntimeEnv(t)

	t.Setenv(
		"HACP_CONTROL_MODE",
		controlModeDistributed,
	)
	t.Setenv(
		"HACP_CONTROL_PLANE_ADDR",
		"control-plane:5000",
	)
	t.Setenv(
		"HACP_SIDECAR_ID",
		"sidecar-1",
	)
	t.Setenv(
		"HACP_CONTROL_MAX_STALENESS",
		"12s",
	)

	cfg, err := loadControlRuntimeConfig()
	if err != nil {
		t.Fatalf(
			"loadControlRuntimeConfig() error = %v",
			err,
		)
	}

	if cfg.Mode != controlModeDistributed {
		t.Fatalf(
			"mode = %q, want %q",
			cfg.Mode,
			controlModeDistributed,
		)
	}

	if cfg.Address != "control-plane:5000" {
		t.Fatalf(
			"address = %q, want %q",
			cfg.Address,
			"control-plane:5000",
		)
	}

	if cfg.SidecarID != "sidecar-1" {
		t.Fatalf(
			"sidecar ID = %q, want %q",
			cfg.SidecarID,
			"sidecar-1",
		)
	}

	if cfg.MaxStaleness != 12*time.Second {
		t.Fatalf(
			"max staleness = %v, want %v",
			cfg.MaxStaleness,
			12*time.Second,
		)
	}
}

func TestLoadControlRuntimeConfigDistributedUsesDefaultStaleness(
	t *testing.T,
) {
	clearControlRuntimeEnv(t)

	t.Setenv(
		"HACP_CONTROL_MODE",
		controlModeDistributed,
	)
	t.Setenv(
		"HACP_CONTROL_PLANE_ADDR",
		"control-plane:5000",
	)
	t.Setenv(
		"HACP_SIDECAR_ID",
		"sidecar-1",
	)

	cfg, err := loadControlRuntimeConfig()
	if err != nil {
		t.Fatalf(
			"loadControlRuntimeConfig() error = %v",
			err,
		)
	}

	if cfg.MaxStaleness != defaultControlMaxStaleness {
		t.Fatalf(
			"max staleness = %v, want %v",
			cfg.MaxStaleness,
			defaultControlMaxStaleness,
		)
	}
}

func TestLoadControlRuntimeConfigDistributedRequiresAddress(
	t *testing.T,
) {
	clearControlRuntimeEnv(t)

	t.Setenv(
		"HACP_CONTROL_MODE",
		controlModeDistributed,
	)
	t.Setenv(
		"HACP_SIDECAR_ID",
		"sidecar-1",
	)

	_, err := loadControlRuntimeConfig()
	if !errors.Is(
		err,
		errMissingControlPlaneAddress,
	) {
		t.Fatalf(
			"error = %v, want errMissingControlPlaneAddress",
			err,
		)
	}
}

func TestLoadControlRuntimeConfigDistributedRequiresSidecarID(
	t *testing.T,
) {
	clearControlRuntimeEnv(t)

	t.Setenv(
		"HACP_CONTROL_MODE",
		controlModeDistributed,
	)
	t.Setenv(
		"HACP_CONTROL_PLANE_ADDR",
		"control-plane:5000",
	)

	_, err := loadControlRuntimeConfig()
	if !errors.Is(
		err,
		errMissingSidecarID,
	) {
		t.Fatalf(
			"error = %v, want errMissingSidecarID",
			err,
		)
	}
}

func TestLoadControlRuntimeConfigRejectsUnsupportedMode(
	t *testing.T,
) {
	clearControlRuntimeEnv(t)

	t.Setenv(
		"HACP_CONTROL_MODE",
		"legacy",
	)

	_, err := loadControlRuntimeConfig()
	if !errors.Is(
		err,
		errUnsupportedControlMode,
	) {
		t.Fatalf(
			"error = %v, want errUnsupportedControlMode",
			err,
		)
	}
}

func TestLoadControlRuntimeConfigRejectsInvalidMaxStaleness(
	t *testing.T,
) {
	clearControlRuntimeEnv(t)

	t.Setenv(
		"HACP_CONTROL_MAX_STALENESS",
		"not-a-duration",
	)

	_, err := loadControlRuntimeConfig()
	if !errors.Is(
		err,
		errInvalidControlMaxStaleness,
	) {
		t.Fatalf(
			"error = %v, want errInvalidControlMaxStaleness",
			err,
		)
	}
}

func TestLoadControlRuntimeConfigRejectsZeroMaxStaleness(
	t *testing.T,
) {
	clearControlRuntimeEnv(t)

	t.Setenv(
		"HACP_CONTROL_MAX_STALENESS",
		"0s",
	)

	_, err := loadControlRuntimeConfig()
	if !errors.Is(
		err,
		errInvalidControlMaxStaleness,
	) {
		t.Fatalf(
			"error = %v, want errInvalidControlMaxStaleness",
			err,
		)
	}
}

func TestLoadControlRuntimeConfigRejectsNegativeMaxStaleness(
	t *testing.T,
) {
	clearControlRuntimeEnv(t)

	t.Setenv(
		"HACP_CONTROL_MAX_STALENESS",
		"-1s",
	)

	_, err := loadControlRuntimeConfig()
	if !errors.Is(
		err,
		errInvalidControlMaxStaleness,
	) {
		t.Fatalf(
			"error = %v, want errInvalidControlMaxStaleness",
			err,
		)
	}
}

func TestLoadControlRuntimeConfigStandaloneRejectsDistributedAddress(
	t *testing.T,
) {
	clearControlRuntimeEnv(t)

	t.Setenv(
		"HACP_CONTROL_MODE",
		controlModeStandalone,
	)
	t.Setenv(
		"HACP_CONTROL_PLANE_ADDR",
		"control-plane:5000",
	)

	_, err := loadControlRuntimeConfig()
	if !errors.Is(
		err,
		errAmbiguousStandaloneControl,
	) {
		t.Fatalf(
			"error = %v, want errAmbiguousStandaloneControl",
			err,
		)
	}
}

func TestLoadControlRuntimeConfigStandaloneRejectsSidecarID(
	t *testing.T,
) {
	clearControlRuntimeEnv(t)

	t.Setenv(
		"HACP_CONTROL_MODE",
		controlModeStandalone,
	)
	t.Setenv(
		"HACP_SIDECAR_ID",
		"sidecar-1",
	)

	_, err := loadControlRuntimeConfig()
	if !errors.Is(
		err,
		errAmbiguousStandaloneControl,
	) {
		t.Fatalf(
			"error = %v, want errAmbiguousStandaloneControl",
			err,
		)
	}
}

func TestLoadControlRuntimeConfigDistributedRequiresTLSByDefault(
	t *testing.T,
) {
	clearControlRuntimeEnv(t)

	t.Setenv(
		"HACP_CONTROL_MODE",
		controlModeDistributed,
	)

	t.Setenv(
		"HACP_CONTROL_PLANE_ADDR",
		"control-plane:5000",
	)

	t.Setenv(
		"HACP_SIDECAR_ID",
		"sidecar-1",
	)

	cfg, err :=
		loadControlRuntimeConfig()

	if err != nil {
		t.Fatalf(
			"loadControlRuntimeConfig() error = %v",
			err,
		)
	}

	if cfg.TLSMode != controlTLSModeRequired {
		t.Fatalf(
			"TLS mode = %q, want %q",
			cfg.TLSMode,
			controlTLSModeRequired,
		)
	}
}

func TestLoadControlRuntimeConfigDistributedExplicitTLS(
	t *testing.T,
) {
	clearControlRuntimeEnv(t)

	t.Setenv(
		"HACP_CONTROL_MODE",
		controlModeDistributed,
	)

	t.Setenv(
		"HACP_CONTROL_PLANE_ADDR",
		"control-plane:5000",
	)

	t.Setenv(
		"HACP_SIDECAR_ID",
		"sidecar-1",
	)

	t.Setenv(
		"HACP_CONTROL_TLS_MODE",
		controlTLSModeRequired,
	)

	t.Setenv(
		"HACP_CONTROL_CA_FILE",
		"/etc/hacp/control-ca.pem",
	)

	t.Setenv(
		"HACP_CONTROL_SERVER_NAME",
		"control-plane.internal",
	)

	cfg, err :=
		loadControlRuntimeConfig()

	if err != nil {
		t.Fatalf(
			"loadControlRuntimeConfig() error = %v",
			err,
		)
	}

	if cfg.TLSMode != controlTLSModeRequired {
		t.Fatalf(
			"TLS mode = %q, want %q",
			cfg.TLSMode,
			controlTLSModeRequired,
		)
	}

	if cfg.CAFile != "/etc/hacp/control-ca.pem" {
		t.Fatalf(
			"CA file = %q, want %q",
			cfg.CAFile,
			"/etc/hacp/control-ca.pem",
		)
	}

	if cfg.ServerName != "control-plane.internal" {
		t.Fatalf(
			"server name = %q, want %q",
			cfg.ServerName,
			"control-plane.internal",
		)
	}
}

func TestLoadControlRuntimeConfigDistributedExplicitInsecure(
	t *testing.T,
) {
	clearControlRuntimeEnv(t)

	t.Setenv(
		"HACP_CONTROL_MODE",
		controlModeDistributed,
	)

	t.Setenv(
		"HACP_CONTROL_PLANE_ADDR",
		"control-plane:5000",
	)

	t.Setenv(
		"HACP_SIDECAR_ID",
		"sidecar-1",
	)

	t.Setenv(
		"HACP_CONTROL_TLS_MODE",
		controlTLSModeInsecure,
	)

	cfg, err :=
		loadControlRuntimeConfig()

	if err != nil {
		t.Fatalf(
			"loadControlRuntimeConfig() error = %v",
			err,
		)
	}

	if cfg.TLSMode != controlTLSModeInsecure {
		t.Fatalf(
			"TLS mode = %q, want %q",
			cfg.TLSMode,
			controlTLSModeInsecure,
		)
	}
}

func TestLoadControlRuntimeConfigRejectsUnsupportedTLSMode(
	t *testing.T,
) {
	clearControlRuntimeEnv(t)

	t.Setenv(
		"HACP_CONTROL_MODE",
		controlModeDistributed,
	)

	t.Setenv(
		"HACP_CONTROL_PLANE_ADDR",
		"control-plane:5000",
	)

	t.Setenv(
		"HACP_SIDECAR_ID",
		"sidecar-1",
	)

	t.Setenv(
		"HACP_CONTROL_TLS_MODE",
		"optional",
	)

	_, err :=
		loadControlRuntimeConfig()

	if !errors.Is(
		err,
		errUnsupportedControlTLSMode,
	) {
		t.Fatalf(
			"error = %v, want errUnsupportedControlTLSMode",
			err,
		)
	}
}

func TestLoadControlRuntimeConfigRejectsInsecureWithCAFile(
	t *testing.T,
) {
	clearControlRuntimeEnv(t)

	t.Setenv(
		"HACP_CONTROL_MODE",
		controlModeDistributed,
	)

	t.Setenv(
		"HACP_CONTROL_PLANE_ADDR",
		"control-plane:5000",
	)

	t.Setenv(
		"HACP_SIDECAR_ID",
		"sidecar-1",
	)

	t.Setenv(
		"HACP_CONTROL_TLS_MODE",
		controlTLSModeInsecure,
	)

	t.Setenv(
		"HACP_CONTROL_CA_FILE",
		"/etc/hacp/control-ca.pem",
	)

	_, err :=
		loadControlRuntimeConfig()

	if !errors.Is(
		err,
		errInsecureControlTLSOptions,
	) {
		t.Fatalf(
			"error = %v, want errInsecureControlTLSOptions",
			err,
		)
	}
}

func TestLoadControlRuntimeConfigRejectsInsecureWithServerName(
	t *testing.T,
) {
	clearControlRuntimeEnv(t)

	t.Setenv(
		"HACP_CONTROL_MODE",
		controlModeDistributed,
	)

	t.Setenv(
		"HACP_CONTROL_PLANE_ADDR",
		"control-plane:5000",
	)

	t.Setenv(
		"HACP_SIDECAR_ID",
		"sidecar-1",
	)

	t.Setenv(
		"HACP_CONTROL_TLS_MODE",
		controlTLSModeInsecure,
	)

	t.Setenv(
		"HACP_CONTROL_SERVER_NAME",
		"control-plane.internal",
	)

	_, err :=
		loadControlRuntimeConfig()

	if !errors.Is(
		err,
		errInsecureControlTLSOptions,
	) {
		t.Fatalf(
			"error = %v, want errInsecureControlTLSOptions",
			err,
		)
	}
}

func TestLoadControlRuntimeConfigStandaloneRejectsTLSConfiguration(
	t *testing.T,
) {
	clearControlRuntimeEnv(t)

	t.Setenv(
		"HACP_CONTROL_MODE",
		controlModeStandalone,
	)

	t.Setenv(
		"HACP_CONTROL_TLS_MODE",
		controlTLSModeRequired,
	)

	_, err :=
		loadControlRuntimeConfig()

	if !errors.Is(
		err,
		errAmbiguousStandaloneControl,
	) {
		t.Fatalf(
			"error = %v, want errAmbiguousStandaloneControl",
			err,
		)
	}
}
