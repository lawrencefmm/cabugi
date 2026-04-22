package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/lawrencefmm/cabugi/services/judge/internal/config"
	"github.com/lawrencefmm/cabugi/services/judge/internal/spike"
	"github.com/lawrencefmm/cabugi/services/judge/internal/worker"
)

type processor interface {
	ProcessOne(context.Context) (bool, error)
}

func main() {
	runOnce := flag.Bool("once", false, "claim and process at most one queued submission")
	flag.Parse()

	cfg := config.Load()
	store, err := worker.NewPostgresStore(cfg.DatabaseURL, cfg.MaxJobAttempts, cfg.WorkerRetryDelay)
	if err != nil {
		log.Fatalf("create judge store: %v", err)
	}
	defer store.Close()

	bundleLoader, err := worker.NewS3BundleLoader(cfg.ObjectStorage)
	if err != nil {
		log.Fatalf("create bundle loader: %v", err)
	}

	jobProcessor := worker.NewProcessor(store, bundleLoader, spike.NewRunner())
	if *runOnce {
		if err := runOnceCommand(context.Background(), jobProcessor, os.Stdout); err != nil {
			log.Fatalf("process submission job: %v", err)
		}
		return
	}

	if err := runLoop(context.Background(), jobProcessor, cfg.WorkerPollInterval, os.Stdout); err != nil {
		log.Fatalf("run judge worker loop: %v", err)
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
