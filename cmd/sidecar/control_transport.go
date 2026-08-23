package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	errLoadControlCAFile = errors.New(
		"failed to load HACP_CONTROL_CA_FILE",
	)

	errInvalidControlCAPEM = errors.New(
		"HACP_CONTROL_CA_FILE does not contain valid PEM certificates",
	)

	errUnsupportedControlTransportMode = errors.New(
		"unsupported control-plane TLS mode",
	)
)

// buildControlTransportCredentials creates the gRPC transport credentials
// required by the configured control-plane transport policy.
//
// TLS is the production default. When no custom CA file is configured,
// the host system trust roots are used.
//
// When a custom CA file is configured, its certificates are appended to the
// host system roots.
//
// Plaintext transport is available only through explicit insecure opt-in.
func buildControlTransportCredentials(
	cfg controlRuntimeConfig,
) (
	credentials.TransportCredentials,
	error,
) {

	switch cfg.TLSMode {
	case controlTLSModeRequired:
		return buildControlTLSCredentials(
			cfg,
		)

	case controlTLSModeInsecure:
		return insecure.NewCredentials(), nil

	default:
		return nil, fmt.Errorf(
			"%w: %q",
			errUnsupportedControlTransportMode,
			cfg.TLSMode,
		)
	}
}

func buildControlTLSCredentials(
	cfg controlRuntimeConfig,
) (
	credentials.TransportCredentials,
	error,
) {

	roots, err :=
		x509.SystemCertPool()

	if err != nil || roots == nil {
		roots =
			x509.NewCertPool()
	}

	if cfg.CAFile != "" {
		pemBytes, err :=
			os.ReadFile(
				cfg.CAFile,
			)

		if err != nil {
			return nil, fmt.Errorf(
				"%w: %v",
				errLoadControlCAFile,
				err,
			)
		}

		if ok :=
			roots.AppendCertsFromPEM(
				pemBytes,
			); !ok {

			return nil,
				errInvalidControlCAPEM
		}
	}

	tlsConfig :=
		&tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:    roots,
			ServerName: cfg.ServerName,
		}

	return credentials.NewTLS(
		tlsConfig,
	), nil
}
