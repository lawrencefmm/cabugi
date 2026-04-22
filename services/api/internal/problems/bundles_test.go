package problems

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	appconfig "github.com/lawrencefmm/cabugi/services/api/internal/config"
)

func TestValidateBundleMetadataRejectsPartialMetadata(t *testing.T) {
	err := ValidateBundleMetadata("bundles/two-sum.json", "")
	if err != ErrInvalidBundleChecksum {
		t.Fatalf("ValidateBundleMetadata() error = %v, want %v", err, ErrInvalidBundleChecksum)
	}
}

func TestS3BundleValidatorValidatesObjectChecksum(t *testing.T) {
	contents := []byte(`{"cases":[{"input":"1\n"}]}`)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/cabugi-hidden-tests/bundles/two-sum.json" {
			t.Fatalf("request path = %q, want %q", request.URL.Path, "/cabugi-hidden-tests/bundles/two-sum.json")
		}

		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(contents)
	}))
	defer server.Close()

	validator, err := NewS3BundleValidator(appconfig.ObjectStorageConfig{
		Endpoint:        server.URL,
		Region:          "us-east-1",
		Bucket:          "cabugi-hidden-tests",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		UsePathStyle:    true,
	})
	if err != nil {
		t.Fatalf("NewS3BundleValidator() error = %v", err)
	}

	err = validator.ValidateBundle(context.Background(), "bundles/two-sum.json", sha256Hex(contents))
	if err != nil {
		t.Fatalf("ValidateBundle() error = %v", err)
	}
}

func TestS3BundleValidatorRejectsChecksumMismatch(t *testing.T) {
	contents := []byte(`{"cases":[{"input":"1\n"}]}`)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(contents)
	}))
	defer server.Close()

	validator, err := NewS3BundleValidator(appconfig.ObjectStorageConfig{
		Endpoint:        server.URL,
		Region:          "us-east-1",
		Bucket:          "cabugi-hidden-tests",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		UsePathStyle:    true,
	})
	if err != nil {
		t.Fatalf("NewS3BundleValidator() error = %v", err)
	}

	err = validator.ValidateBundle(context.Background(), "bundles/two-sum.json", sha256Hex([]byte("different")))
	if err == nil {
		t.Fatal("ValidateBundle() should fail on checksum mismatch")
	}
}

func sha256Hex(contents []byte) string {
	sum := sha256.Sum256(contents)
	return hex.EncodeToString(sum[:])
}
