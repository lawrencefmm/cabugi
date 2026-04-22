package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/lawrencefmm/cabugi/services/judge/internal/config"
	"github.com/lawrencefmm/cabugi/services/judge/internal/spike"
	"github.com/lawrencefmm/cabugi/services/judge/internal/worker"
)

func main() {
	runOnce := flag.Bool("once", false, "claim and process at most one queued submission")
	flag.Parse()

	cfg := config.Load()
	store, err := worker.NewPostgresStore(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("create judge store: %v", err)
	}
	defer store.Close()

	bundleLoader, err := worker.NewS3BundleLoader(cfg.ObjectStorage)
	if err != nil {
		log.Fatalf("create bundle loader: %v", err)
	}

	processor := worker.NewProcessor(store, bundleLoader, spike.NewRunner())
	processed, err := processor.ProcessOne(context.Background())
	if err != nil {
		log.Fatalf("process submission job: %v", err)
	}

	if !processed {
		fmt.Fprintln(os.Stdout, "no queued submissions")
		return
	}

	fmt.Fprintln(os.Stdout, "processed one submission")

	if !*runOnce {
		return
	}
}
