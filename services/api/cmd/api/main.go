package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
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
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg := config.Load()
	authReady := false
	var verifier auth.Verifier = auth.DisabledVerifier{}
	if clerkVerifier, err := auth.NewClerkVerifier(auth.LoadClerkConfig()); err == nil {
		verifier = clerkVerifier
		authReady = true
	} else {
		logger.Warn("api auth verifier disabled", slog.String("error", err.Error()))
	}

	databaseReady := true
	var problemStore problems.Store = problems.DisabledStore{}
	if store, err := problems.NewPostgresStore(cfg.DatabaseURL); err == nil {
		problemStore = store
		defer store.Close()
	} else {
		databaseReady = false
		logger.Warn("api problem store disabled", slog.String("error", err.Error()))
	}

	bundleValidationReady := false
	bundleValidator := problems.BundleValidator(problems.DisabledBundleValidator{})
	if validator, err := problems.NewS3BundleValidator(cfg.ObjectStorage); err == nil {
		if err := validator.CheckReady(ctx); err == nil {
			bundleValidator = validator
			bundleValidationReady = true
		} else {
			logger.Warn("api hidden test bundle validation disabled", slog.String("error", err.Error()))
		}
	} else {
		logger.Warn("api hidden test bundle validation disabled", slog.String("error", err.Error()))
	}

	var userStore users.Store = users.DisabledStore{}
	if store, err := users.NewPostgresStore(cfg.DatabaseURL); err == nil {
		userStore = store
		defer store.Close()
	} else {
		databaseReady = false
		logger.Warn("api user store disabled", slog.String("error", err.Error()))
	}

	var submissionStore submissions.Store = submissions.DisabledStore{}
	if store, err := submissions.NewPostgresStore(cfg.DatabaseURL); err == nil {
		submissionStore = store
		defer store.Close()
	} else {
		databaseReady = false
		logger.Warn("api submission store disabled", slog.String("error", err.Error()))
	}

	if err := failStartupIfRequired(cfg.RequireAuth, authReady, "auth verifier"); err != nil {
		logger.Error("api startup failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if err := failStartupIfRequired(cfg.RequireDatabase, databaseReady, "database"); err != nil {
		logger.Error("api startup failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if err := failStartupIfRequired(cfg.RequireHiddenBundleValidation, bundleValidationReady, "hidden test bundle validation"); err != nil {
		logger.Error("api startup failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	server := httpapi.NewServer(cfg.Address, logger, verifier, problemStore, userStore, submissionStore, cfg.AllowedOrigins, bundleValidator)

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("api shutdown error", slog.String("error", err.Error()))
		}
	}()

	logger.Info("api listening", slog.String("address", cfg.Address))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("api server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
