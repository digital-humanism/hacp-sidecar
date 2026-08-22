package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadinessHandlerReady(
	t *testing.T,
) {

	req :=
		httptest.NewRequest(
			http.MethodGet,
			"/readyz",
			nil,
		)

	rec :=
		httptest.NewRecorder()

	handler :=
		makeReadinessHandler(
			func() bool {
				return true
			},
		)

	handler.ServeHTTP(
		rec,
		req,
	)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	if rec.Body.String() != "ready\n" {
		t.Fatalf(
			"expected body %q, got %q",
			"ready\n",
			rec.Body.String(),
		)
	}
}

func TestReadinessHandlerNotReady(
	t *testing.T,
) {

	req :=
		httptest.NewRequest(
			http.MethodGet,
			"/readyz",
			nil,
		)

	rec :=
		httptest.NewRecorder()

	handler :=
		makeReadinessHandler(
			func() bool {
				return false
			},
		)

	handler.ServeHTTP(
		rec,
		req,
	)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusServiceUnavailable,
			rec.Code,
		)
	}

	if rec.Body.String() != "not ready\n" {
		t.Fatalf(
			"expected body %q, got %q",
			"not ready\n",
			rec.Body.String(),
		)
	}
}

func TestReadinessHandlerNilPredicateFailsClosed(
	t *testing.T,
) {

	req :=
		httptest.NewRequest(
			http.MethodGet,
			"/readyz",
			nil,
		)

	rec :=
		httptest.NewRecorder()

	handler :=
		makeReadinessHandler(
			nil,
		)

	handler.ServeHTTP(
		rec,
		req,
	)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusServiceUnavailable,
			rec.Code,
		)
	}
}
