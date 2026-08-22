package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hacp-sidecar/internal/budget"
	"hacp-sidecar/internal/evaluate"
	"hacp-sidecar/internal/provenance"
	"hacp-sidecar/internal/proxy"
)

func main() {

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	port := os.Getenv(
		"HACP_SIDECAR_PORT",
	)

	if port == "" {
		port = "8080"
	}

	upstream := os.Getenv(
		"HACP_UPSTREAM",
	)

	if upstream == "" {
		upstream = "http://127.0.0.1:8000"
	}

	provenancePath := os.Getenv(
		"HACP_PROVENANCE_FLUSH_PATH",
	)

	if provenancePath == "" {
		provenancePath = "./provenance.jsonl"
	}

	// ============================================================
	// Dependencies
	// ============================================================

	keyResolver, err := loadStartupTrustStore()
	if err != nil {
		log.Fatalf(
			"failed to load trust configuration: %v",
			err,
		)
	}

	revocation :=
		evaluate.NewInMemoryRevocationStore()

	budgetLedger :=
		budget.NewLedger()

	scopeGuard :=
		evaluate.NewDefaultScopeGuard()

	provLog :=
		provenance.NewRingBuffer(
			10000,
			provenancePath,
		)

	pipeline :=
		evaluate.NewPipeline(
			keyResolver,
			revocation,
			budgetLedger,
			scopeGuard,
			provLog,
		)

	handler :=
		proxy.NewHandler(
			pipeline,
			provLog,
			upstream,
		)

	trustAdminServer, err := startTrustAdminServer(
		keyResolver,
		os.Getenv("HACP_TRUST_KEYS_FILE"),
	)
	if err != nil {
		log.Fatalf("failed to start trust admin server: %v", err)
	}
	if trustAdminServer != nil {
		log.Printf("trust admin listening on %s", trustAdminServer.Addr)
	}

	// ============================================================
	// HTTP routes
	// ============================================================

	mux :=
		http.NewServeMux()

	mux.HandleFunc(
		"/healthz",
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {

			w.WriteHeader(
				http.StatusOK,
			)

			_, _ = w.Write(
				[]byte("ok\n"),
			)
		},
	)

	mux.HandleFunc(
		"/revoke/token",
		makeRevokeHandler(
			revocation,
			"token",
		),
	)

	mux.HandleFunc(
		"/revoke/envelope",
		makeRevokeHandler(
			revocation,
			"envelope",
		),
	)

	mux.HandleFunc(
		"/revoke/key",
		makeRevokeHandler(
			revocation,
			"key",
		),
	)

	mux.Handle(
		"/",
		handler,
	)

	server :=
		&http.Server{
			Addr:    ":" + port,
			Handler: mux,

			ReadHeaderTimeout: 5 * time.Second,
		}

	go func() {

		log.Printf(
			"hacp-sidecar listening on :%s (upstream: %s provenance: %s)",
			port,
			upstream,
			provenancePath,
		)

		err :=
			server.ListenAndServe()

		if err != nil &&
			err != http.ErrServerClosed {

			log.Fatalf(
				"server error: %v",
				err,
			)
		}
	}()

	// ============================================================
	// Graceful shutdown
	// ============================================================

	<-ctx.Done()

	log.Println(
		"hacp-sidecar shutting down",
	)

	shutdownCtx,
		shutdownCancel :=
		context.WithTimeout(
			context.Background(),
			5*time.Second,
		)

	defer shutdownCancel()

	if err := server.Shutdown(
		shutdownCtx,
	); err != nil {

		log.Printf(
			"server shutdown error: %v",
			err,
		)
	}

	provLog.Stop()
}

// makeRevokeHandler creates the temporary HTTP revocation API.
//
// Gate E is expected to replace this management surface with the
// distributed control channel.
func makeRevokeHandler(
	store *evaluate.InMemoryRevocationStore,
	kind string,
) http.HandlerFunc {

	return func(
		w http.ResponseWriter,
		r *http.Request,
	) {

		if r.Method != http.MethodPost {

			http.Error(
				w,
				"method not allowed",
				http.StatusMethodNotAllowed,
			)

			return
		}

		id :=
			r.URL.Query().Get(
				"id",
			)

		if id == "" {

			http.Error(
				w,
				"missing id",
				http.StatusBadRequest,
			)

			return
		}

		switch kind {

		case "token":
			store.RevokeToken(id)

		case "envelope":
			store.RevokeEnvelope(id)

		case "key":
			store.RevokeKey(id)

		default:

			http.Error(
				w,
				"unsupported revocation kind",
				http.StatusInternalServerError,
			)

			return
		}

		w.WriteHeader(
			http.StatusOK,
		)

		_, _ = w.Write(
			[]byte("revoked\n"),
		)
	}
}
