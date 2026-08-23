package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	controlplanev1 "hacp-sidecar/gen/controlplane/v1"
	"hacp-sidecar/internal/budget"
	"hacp-sidecar/internal/evaluate"
	"hacp-sidecar/internal/provenance"
	"hacp-sidecar/internal/proxy"

	"google.golang.org/grpc"
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
	// Startup configuration
	// ============================================================

	controlConfig, err :=
		loadControlRuntimeConfig()

	if err != nil {
		log.Fatalf(
			"failed to load control-plane configuration: %v",
			err,
		)
	}

	// ============================================================
	// Trust
	// ============================================================

	keyResolver, err :=
		loadStartupTrustStore()

	if err != nil {
		log.Fatalf(
			"failed to load trust configuration: %v",
			err,
		)
	}

	// ============================================================
	// Control-plane runtime
	// ============================================================

	var controlConn *grpc.ClientConn

	var controlClient controlplanev1.ControlPlaneClient

	if controlConfig.Mode == controlModeDistributed {

		transportCredentials, err :=
			buildControlTransportCredentials(
				controlConfig,
			)

		if err != nil {
			log.Fatalf(
				"failed to build control-plane transport credentials: %v",
				err,
			)
		}

		controlConn, err =
			grpc.NewClient(
				controlConfig.Address,
				grpc.WithTransportCredentials(
					transportCredentials,
				),
			)

		if err != nil {
			log.Fatalf(
				"failed to create control-plane client: %v",
				err,
			)
		}

		defer func() {
			if err :=
				controlConn.Close(); err != nil {

				log.Printf(
					"control-plane connection close error: %v",
					err,
				)
			}
		}()

		controlClient =
			controlplanev1.NewControlPlaneClient(
				controlConn,
			)

		if controlConfig.TLSMode ==
			controlTLSModeInsecure {

			log.Printf(
				"WARNING: control-plane transport is explicitly configured as insecure",
			)
		}
	}

	controlRuntime, err :=
		newControlRuntime(
			controlConfig,
			controlClient,
			time.Now,
		)

	if err != nil {
		log.Fatalf(
			"failed to construct control-plane runtime: %v",
			err,
		)
	}

	// ============================================================
	// Evaluation dependencies
	// ============================================================

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
			controlRuntime.Revocations,
			budgetLedger,
			scopeGuard,
			provLog,
		)

	// Standalone mode intentionally leaves ControlState nil.
	// Distributed mode shares this exact ControlState with the
	// subscriber and readiness predicate.
	pipeline.ControlState =
		controlRuntime.ControlState

	handler :=
		proxy.NewHandler(
			pipeline,
			provLog,
			upstream,
		)

	trustAdminServer, err :=
		startTrustAdminServer(
			keyResolver,
			os.Getenv("HACP_TRUST_KEYS_FILE"),
		)

	if err != nil {
		log.Fatalf(
			"failed to start trust admin server: %v",
			err,
		)
	}

	if trustAdminServer != nil {
		log.Printf(
			"trust admin listening on %s",
			trustAdminServer.Addr,
		)
	}

	// ============================================================
	// Distributed subscriber
	// ============================================================

	if controlRuntime.Subscriber != nil {

		go func() {

			err :=
				controlRuntime.Subscriber.Run(
					ctx,
				)

			if err != nil &&
				!errors.Is(
					err,
					context.Canceled,
				) {

				log.Printf(
					"control-plane subscriber stopped: %v",
					err,
				)
			}
		}()
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
		"/readyz",
		makeReadinessHandler(
			controlRuntime.Ready,
		),
	)

	registerRevocationRoutes(
		mux,
		controlRuntime.LocalRevocations,
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
			"hacp-sidecar listening on :%s (upstream: %s provenance: %s control-mode: %s)",
			port,
			upstream,
			provenancePath,
			controlConfig.Mode,
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

	if err :=
		server.Shutdown(
			shutdownCtx,
		); err != nil {

		log.Printf(
			"server shutdown error: %v",
			err,
		)
	}

	provLog.Stop()
}

func registerRevocationRoutes(
	mux *http.ServeMux,
	store *evaluate.InMemoryRevocationStore,
) {

	if store != nil {

		mux.HandleFunc(
			"/revoke/token",
			makeRevokeHandler(
				store,
				"token",
			),
		)

		mux.HandleFunc(
			"/revoke/envelope",
			makeRevokeHandler(
				store,
				"envelope",
			),
		)

		mux.HandleFunc(
			"/revoke/key",
			makeRevokeHandler(
				store,
				"key",
			),
		)

		return
	}

	// Distributed mode reserves the legacy local revocation paths
	// and fails them locally. They must never fall through to the
	// protected upstream.
	unavailable :=
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {

			http.NotFound(
				w,
				r,
			)
		}

	mux.HandleFunc(
		"/revoke/token",
		unavailable,
	)

	mux.HandleFunc(
		"/revoke/envelope",
		unavailable,
	)

	mux.HandleFunc(
		"/revoke/key",
		unavailable,
	)
}

// makeRevokeHandler creates the temporary standalone HTTP revocation API.
//
// Distributed mode does not expose this mutation surface. Distributed
// revocation authority belongs exclusively to the control plane.
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
