package main

import (
	"context"
	"errors"
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

func failStartupIfRequired(required bool, ready bool, dependencyName string) error {
	if required && !ready {
		return errors.New(dependencyName + " is required but not ready")
	}

	return nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	authReady := false
	var verifier auth.Verifier = auth.DisabledVerifier{}
	if clerkVerifier, err := auth.NewClerkVerifier(auth.LoadClerkConfig()); err == nil {
		verifier = clerkVerifier
		authReady = true
	} else {
		log.Printf("api auth verifier disabled: %v", err)
	}

	databaseReady := true
	var problemStore problems.Store = problems.DisabledStore{}
	if store, err := problems.NewPostgresStore(cfg.DatabaseURL); err == nil {
		problemStore = store
		defer store.Close()
	} else {
		databaseReady = false
		log.Printf("api problem store disabled: %v", err)
	}

	bundleValidationReady := false
	bundleValidator := problems.BundleValidator(problems.DisabledBundleValidator{})
	if validator, err := problems.NewS3BundleValidator(cfg.ObjectStorage); err == nil {
		if err := validator.CheckReady(ctx); err == nil {
			bundleValidator = validator
			bundleValidationReady = true
		} else {
			log.Printf("api hidden test bundle validation disabled: %v", err)
		}
	} else {
		log.Printf("api hidden test bundle validation disabled: %v", err)
	}

	var userStore users.Store = users.DisabledStore{}
	if store, err := users.NewPostgresStore(cfg.DatabaseURL); err == nil {
		userStore = store
		defer store.Close()
	} else {
		databaseReady = false
		log.Printf("api user store disabled: %v", err)
	}

	var submissionStore submissions.Store = submissions.DisabledStore{}
	if store, err := submissions.NewPostgresStore(cfg.DatabaseURL); err == nil {
		submissionStore = store
		defer store.Close()
	} else {
		databaseReady = false
		log.Printf("api submission store disabled: %v", err)
	}

	if err := failStartupIfRequired(cfg.RequireAuth, authReady, "auth verifier"); err != nil {
		log.Fatalf("api startup failed: %v", err)
	}
	if err := failStartupIfRequired(cfg.RequireDatabase, databaseReady, "database"); err != nil {
		log.Fatalf("api startup failed: %v", err)
	}
	if err := failStartupIfRequired(cfg.RequireHiddenBundleValidation, bundleValidationReady, "hidden test bundle validation"); err != nil {
		log.Fatalf("api startup failed: %v", err)
	}

	server := httpapi.NewServer(cfg.Address, verifier, problemStore, userStore, submissionStore, cfg.AllowedOrigins, bundleValidator)

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
