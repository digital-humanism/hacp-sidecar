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
	"hacp-sidecar/internal/wire"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	port := os.Getenv("HACP_SIDECAR_PORT")
	if port == "" {
		port = "8080"
	}

	provenancePath := os.Getenv("HACP_PROVENANCE_FLUSH_PATH")
	if provenancePath == "" {
		provenancePath = "./provenance.jsonl"
	}

	// --- Wire up components ---
	keyResolver := wire.NewStaticKeyResolver()
	// Load test key from hacp-spec/harness/keys/test-ed25519-001.pub
	// pub hex: 9d17f1bbcc0845865e670f526413fb7a510380798fe300b6c98e28f3a3b0fdb3
	if err := keyResolver.AddKeyFromHex("key-ed25519-test-001",
		"9d17f1bbcc0845865e670f526413fb7a510380798fe300b6c98e28f3a3b0fdb3"); err != nil {
		log.Fatalf("failed to load test key: %v", err)
	}

	revocation := evaluate.NewInMemoryRevocationStore()
	budgetLedger := budget.NewLedger()
	scopeGuard := evaluate.NewDefaultScopeGuard()
	provLog := provenance.NewRingBuffer(10000, provenancePath)

	pipeline := evaluate.NewPipeline(
		keyResolver,
		revocation,
		budgetLedger,
		scopeGuard,
		provLog,
	)

	handler := proxy.NewHandler(pipeline, provLog, "http://upstream:8000")

	// --- HTTP mux ---
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/revoke/token", makeRevokeHandler(revocation, "token"))
	mux.HandleFunc("/revoke/envelope", makeRevokeHandler(revocation, "envelope"))
	mux.HandleFunc("/revoke/key", makeRevokeHandler(revocation, "key"))
	mux.Handle("/", handler)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		log.Printf("hacp-sidecar listening on :%s (provenance: %s)", port, provenancePath)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")

	provLog.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

// makeRevokeHandler creates a simple POST endpoint to revoke by ID
// For MVP / testing; in production this is replaced by gRPC control channel.
func makeRevokeHandler(store *evaluate.InMemoryRevocationStore, kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		switch kind {
		case "token":
			store.RevokeToken(id)
		case "envelope":
			store.RevokeEnvelope(id)
		case "key":
			store.RevokeKey(id)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("revoked\n"))
	}
}
