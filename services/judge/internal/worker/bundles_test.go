package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	judgeconfig "github.com/lawrencefmm/cabugi/services/judge/internal/config"
)

func TestS3BundleLoaderLoadsCasesFromObjectStorage(t *testing.T) {
	contents := []byte(`{"cases":[{"Input":"21\n","ExpectedOutput":"42\n"}]}`)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Fatalf("request method = %q, want %q", request.Method, http.MethodGet)
		}
		if request.URL.Path != "/hidden-tests/bundles/two-sum.json" {
			t.Fatalf("request path = %q, want %q", request.URL.Path, "/hidden-tests/bundles/two-sum.json")
		}

		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(contents)
	}))
	defer server.Close()

	loader, err := NewS3BundleLoader(judgeconfig.ObjectStorageConfig{
		Endpoint:        server.URL,
		Region:          "us-east-1",
		Bucket:          "hidden-tests",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		UsePathStyle:    true,
	})
	if err != nil {
		t.Fatalf("NewS3BundleLoader() error = %v", err)
	}

	cases, err := loader.LoadCases(context.Background(), "bundles/two-sum.json", sha256Hex(contents))
	if err != nil {
		t.Fatalf("LoadCases() error = %v", err)
	}
	if len(cases) != 1 {
		t.Fatalf("LoadCases() returned %d cases, want 1", len(cases))
	}
	if cases[0].Input != "21\n" || cases[0].ExpectedOutput != "42\n" {
		t.Fatalf("LoadCases() returned unexpected cases: %#v", cases)
	}
}

func TestS3BundleLoaderRejectsChecksumMismatch(t *testing.T) {
	contents := []byte(`{"cases":[{"Input":"1\n","ExpectedOutput":"1\n"}]}`)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(contents)
	}))
	defer server.Close()

	loader, err := NewS3BundleLoader(judgeconfig.ObjectStorageConfig{
		Endpoint:        server.URL,
		Region:          "us-east-1",
		Bucket:          "hidden-tests",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		UsePathStyle:    true,
	})
	if err != nil {
		t.Fatalf("NewS3BundleLoader() error = %v", err)
	}

	_, err = loader.LoadCases(context.Background(), "bundles/two-sum.json", sha256Hex([]byte("different")))
	if err == nil {
		t.Fatal("LoadCases() should fail when the bundle checksum does not match")
	}
}

func sha256Hex(contents []byte) string {
	sum := sha256.Sum256(contents)
	return hex.EncodeToString(sum[:])
}
