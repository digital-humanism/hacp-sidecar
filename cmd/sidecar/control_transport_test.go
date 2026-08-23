package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildControlTransportCredentialsRequiredUsesTLS(
	t *testing.T,
) {

	cfg :=
		controlRuntimeConfig{
			TLSMode: controlTLSModeRequired,
		}

	creds, err :=
		buildControlTransportCredentials(
			cfg,
		)

	if err != nil {
		t.Fatalf(
			"buildControlTransportCredentials() error = %v",
			err,
		)
	}

	if creds == nil {
		t.Fatal(
			"expected TLS transport credentials",
		)
	}

	info :=
		creds.Info()

	if info.SecurityProtocol == "" {
		t.Fatal(
			"expected non-empty TLS security protocol",
		)
	}
}

func TestBuildControlTransportCredentialsInsecure(
	t *testing.T,
) {

	cfg :=
		controlRuntimeConfig{
			TLSMode: controlTLSModeInsecure,
		}

	creds, err :=
		buildControlTransportCredentials(
			cfg,
		)

	if err != nil {
		t.Fatalf(
			"buildControlTransportCredentials() error = %v",
			err,
		)
	}

	if creds == nil {
		t.Fatal(
			"expected insecure transport credentials",
		)
	}

	info :=
		creds.Info()

	if info.SecurityProtocol != "insecure" {
		t.Fatalf(
			"security protocol = %q, want %q",
			info.SecurityProtocol,
			"insecure",
		)
	}
}

func TestBuildControlTransportCredentialsRejectsUnsupportedMode(
	t *testing.T,
) {

	cfg :=
		controlRuntimeConfig{
			TLSMode: "optional",
		}

	_, err :=
		buildControlTransportCredentials(
			cfg,
		)

	if !errors.Is(
		err,
		errUnsupportedControlTransportMode,
	) {
		t.Fatalf(
			"error = %v, want errUnsupportedControlTransportMode",
			err,
		)
	}
}

func TestBuildControlTransportCredentialsMissingCAFileFails(
	t *testing.T,
) {

	cfg :=
		controlRuntimeConfig{
			TLSMode: controlTLSModeRequired,
			CAFile: filepath.Join(
				t.TempDir(),
				"missing-ca.pem",
			),
		}

	_, err :=
		buildControlTransportCredentials(
			cfg,
		)

	if !errors.Is(
		err,
		errLoadControlCAFile,
	) {
		t.Fatalf(
			"error = %v, want errLoadControlCAFile",
			err,
		)
	}
}

func TestBuildControlTransportCredentialsInvalidCAPEMFails(
	t *testing.T,
) {

	dir :=
		t.TempDir()

	caPath :=
		filepath.Join(
			dir,
			"invalid-ca.pem",
		)

	if err :=
		os.WriteFile(
			caPath,
			[]byte("not a certificate"),
			0o600,
		); err != nil {

		t.Fatalf(
			"WriteFile() error = %v",
			err,
		)
	}

	cfg :=
		controlRuntimeConfig{
			TLSMode: controlTLSModeRequired,
			CAFile:  caPath,
		}

	_, err :=
		buildControlTransportCredentials(
			cfg,
		)

	if !errors.Is(
		err,
		errInvalidControlCAPEM,
	) {
		t.Fatalf(
			"error = %v, want errInvalidControlCAPEM",
			err,
		)
	}
}

func TestBuildControlTransportCredentialsAcceptsCustomCA(
	t *testing.T,
) {
	dir := t.TempDir()

	caPath := filepath.Join(
		dir,
		"ca.pem",
	)

	_, privateKey, err :=
		ed25519.GenerateKey(
			rand.Reader,
		)
	if err != nil {
		t.Fatalf(
			"GenerateKey() error = %v",
			err,
		)
	}

	template :=
		&x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				CommonName: "HACP Test Control CA",
			},
			NotBefore: time.Now().Add(
				-time.Minute,
			),
			NotAfter: time.Now().Add(
				time.Hour,
			),
			IsCA:                  true,
			BasicConstraintsValid: true,
			KeyUsage: x509.KeyUsageCertSign |
				x509.KeyUsageDigitalSignature,
		}

	der, err :=
		x509.CreateCertificate(
			rand.Reader,
			template,
			template,
			privateKey.Public(),
			privateKey,
		)
	if err != nil {
		t.Fatalf(
			"CreateCertificate() error = %v",
			err,
		)
	}

	pemBytes :=
		pem.EncodeToMemory(
			&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: der,
			},
		)

	if pemBytes == nil {
		t.Fatal(
			"EncodeToMemory() returned nil",
		)
	}

	if err :=
		os.WriteFile(
			caPath,
			pemBytes,
			0o600,
		); err != nil {

		t.Fatalf(
			"WriteFile() error = %v",
			err,
		)
	}

	cfg :=
		controlRuntimeConfig{
			TLSMode: controlTLSModeRequired,
			CAFile:  caPath,
		}

	creds, err :=
		buildControlTransportCredentials(
			cfg,
		)

	if err != nil {
		t.Fatalf(
			"buildControlTransportCredentials() error = %v",
			err,
		)
	}

	if creds == nil {
		t.Fatal(
			"expected TLS transport credentials",
		)
	}
}

func TestBuildControlTransportCredentialsPreservesServerName(
	t *testing.T,
) {

	cfg :=
		controlRuntimeConfig{
			TLSMode:    controlTLSModeRequired,
			ServerName: "control-plane.internal",
		}

	creds, err :=
		buildControlTransportCredentials(
			cfg,
		)

	if err != nil {
		t.Fatalf(
			"buildControlTransportCredentials() error = %v",
			err,
		)
	}

	info :=
		creds.Info()

	if info.ServerName != "control-plane.internal" {
		t.Fatalf(
			"server name = %q, want %q",
			info.ServerName,
			"control-plane.internal",
		)
	}
}
