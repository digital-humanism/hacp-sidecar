package main

import (
	"net/http"
)

// makeReadinessHandler exposes authorization readiness independently
// from process liveness.
//
// The handler is fail-closed by default:
// a nil readiness predicate is treated as not ready.
func makeReadinessHandler(
	ready func() bool,
) http.HandlerFunc {

	return func(
		w http.ResponseWriter,
		_ *http.Request,
	) {

		if ready != nil && ready() {
			w.WriteHeader(
				http.StatusOK,
			)

			_, _ = w.Write(
				[]byte("ready\n"),
			)

			return
		}

		http.Error(
			w,
			"not ready",
			http.StatusServiceUnavailable,
		)
	}
}
