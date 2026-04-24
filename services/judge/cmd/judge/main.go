package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lawrencefmm/cabugi/services/judge/internal/config"
	judgeobs "github.com/lawrencefmm/cabugi/services/judge/internal/observability"
	"github.com/lawrencefmm/cabugi/services/judge/internal/sandbox"
	"github.com/lawrencefmm/cabugi/services/judge/internal/worker"
)

type processor interface {
	ProcessOne(context.Context) (bool, error)
}

func main() {
	runOnce := flag.Bool("once", false, "claim and process at most one queued submission")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg := config.Load()
	store, err := worker.NewPostgresStore(cfg.DatabaseURL, cfg.MaxJobAttempts, cfg.WorkerRetryDelay, cfg.JobLeaseDuration)
	if err != nil {
		logger.Error("judge startup failed", slog.String("error", "create judge store: "+err.Error()))
		os.Exit(1)
	}
	defer store.Close()

	bundleLoader, err := worker.NewS3BundleLoader(cfg.ObjectStorage)
	if err != nil {
		logger.Error("judge startup failed", slog.String("error", "create bundle loader: "+err.Error()))
		os.Exit(1)
	}

	observer := judgeobs.New(logger)
	observabilityServer := newObservabilityServer(cfg.ObservabilityAddress, observer)
	go runObservabilityServer(ctx, observabilityServer, logger, stop)

	jobProcessor := worker.NewProcessor(store, bundleLoader, sandbox.NewRunner(), observer, cfg.JobLeaseRenewAfter)
	if *runOnce {
		if err := runOnceCommand(ctx, jobProcessor, os.Stdout); err != nil {
			logger.Error("process submission job failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
		return
	}

	if err := runLoop(ctx, jobProcessor, cfg.WorkerPollInterval, os.Stdout); err != nil {
		logger.Error("judge worker loop failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func newObservabilityServer(address string, observer *judgeobs.Observer) *http.Server {
	mux := http.NewServeMux()
	observer.AttachRoutes(mux)
	return &http.Server{Addr: address, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
}

func runObservabilityServer(ctx context.Context, server *http.Server, logger *slog.Logger, stop context.CancelFunc) {
	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("judge observability shutdown error", slog.String("error", err.Error()))
		}
	}()

	logger.Info("judge observability listening", slog.String("address", server.Addr))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("judge observability server failed", slog.String("error", err.Error()))
		stop()
	}
}

func runOnceCommand(ctx context.Context, jobProcessor processor, writer io.Writer) error {
	processed, err := jobProcessor.ProcessOne(ctx)
	if err != nil {
		return err
	}

	if !processed {
		_, _ = fmt.Fprintln(writer, "no queued submissions")
		return nil
	}

	_, _ = fmt.Fprintln(writer, "handled one submission")
	return nil
}

func runLoop(ctx context.Context, jobProcessor processor, pollInterval time.Duration, writer io.Writer) error {
	for {
		processed, err := jobProcessor.ProcessOne(ctx)
		if err != nil {
			return err
		}
		if processed {
			_, _ = fmt.Fprintln(writer, "handled one submission")
			continue
		}

		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}
