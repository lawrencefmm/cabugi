package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/lawrencefmm/cabugi/services/api/internal/auth"
	"github.com/lawrencefmm/cabugi/services/api/internal/config"
	"github.com/lawrencefmm/cabugi/services/api/internal/httpapi"
	"github.com/lawrencefmm/cabugi/services/api/internal/problems"
	"github.com/lawrencefmm/cabugi/services/api/internal/submissions"
	"github.com/lawrencefmm/cabugi/services/api/internal/users"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	var verifier auth.Verifier = auth.DisabledVerifier{}
	if clerkVerifier, err := auth.NewClerkVerifier(auth.LoadClerkConfig()); err == nil {
		verifier = clerkVerifier
	} else {
		log.Printf("api auth verifier disabled: %v", err)
	}

	var problemStore problems.Store = problems.DisabledStore{}
	if store, err := problems.NewPostgresStore(cfg.DatabaseURL); err == nil {
		problemStore = store
		defer store.Close()
	} else {
		log.Printf("api problem store disabled: %v", err)
	}

	var userStore users.Store = users.DisabledStore{}
	if store, err := users.NewPostgresStore(cfg.DatabaseURL); err == nil {
		userStore = store
		defer store.Close()
	} else {
		log.Printf("api user store disabled: %v", err)
	}

	var submissionStore submissions.Store = submissions.DisabledStore{}
	if store, err := submissions.NewPostgresStore(cfg.DatabaseURL); err == nil {
		submissionStore = store
		defer store.Close()
	} else {
		log.Printf("api submission store disabled: %v", err)
	}

	server := httpapi.NewServer(cfg.Address, verifier, problemStore, userStore, submissionStore)

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("api shutdown error: %v", err)
		}
	}()

	log.Printf("api listening on %s", cfg.Address)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("api server failed: %v", err)
	}
}
