package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/lawrencefmm/cabugi/services/judge/internal/spike"
)

type BundleLoader interface {
	LoadCases(context.Context, string) ([]spike.TestCase, error)
}

type FileBundleLoader struct{}

type fileBundle struct {
	Cases []spike.TestCase `json:"cases"`
}

func (FileBundleLoader) LoadCases(_ context.Context, key string) ([]spike.TestCase, error) {
	contents, err := os.ReadFile(key)
	if err != nil {
		return nil, fmt.Errorf("read test bundle %s: %w", key, err)
	}

	var bundle fileBundle
	if err := json.Unmarshal(contents, &bundle); err != nil {
		return nil, fmt.Errorf("decode test bundle %s: %w", key, err)
	}

	return bundle.Cases, nil
}
