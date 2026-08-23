package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"hacp-sidecar/internal/evaluate"
)

func TestRegisterRevocationRoutesDistributedFailsLocally(
	t *testing.T,
) {

	mux :=
		http.NewServeMux()

	upstreamCalled := false

	registerRevocationRoutes(
		mux,
		nil,
	)

	mux.HandleFunc(
		"/",
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {

			upstreamCalled = true

			w.WriteHeader(
				http.StatusOK,
			)
		},
	)

	for _, path := range []string{
		"/revoke/token",
		"/revoke/envelope",
		"/revoke/key",
	} {

		t.Run(
			path,
			func(
				t *testing.T,
			) {

				upstreamCalled = false

				req :=
					httptest.NewRequest(
						http.MethodPost,
						path+"?id=test",
						nil,
					)

				rec :=
					httptest.NewRecorder()

				mux.ServeHTTP(
					rec,
					req,
				)

				if rec.Code !=
					http.StatusNotFound {

					t.Fatalf(
						"status = %d, want %d",
						rec.Code,
						http.StatusNotFound,
					)
				}

				if upstreamCalled {
					t.Fatal(
						"distributed revocation route fell through to upstream",
					)
				}
			},
		)
	}
}

func TestRegisterRevocationRoutesStandaloneMutatesSharedStore(
	t *testing.T,
) {

	store :=
		evaluate.NewInMemoryRevocationStore()

	mux :=
		http.NewServeMux()

	registerRevocationRoutes(
		mux,
		store,
	)

	req :=
		httptest.NewRequest(
			http.MethodPost,
			"/revoke/token?id=token-001",
			nil,
		)

	rec :=
		httptest.NewRecorder()

	mux.ServeHTTP(
		rec,
		req,
	)

	if rec.Code !=
		http.StatusOK {

		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}

	if !store.IsTokenRevoked(
		"token-001",
	) {

		t.Fatal(
			"standalone revocation route did not mutate shared store",
		)
	}
}
