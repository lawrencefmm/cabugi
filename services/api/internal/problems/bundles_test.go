package problems

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestValidateBundleContentsRejectsEmptyCaseList(t *testing.T) {
	err := ValidateBundleContents([]byte(`{"cases":[]}`))
	if err != ErrInvalidBundleContents {
		t.Fatalf("ValidateBundleContents() error = %v, want %v", err, ErrInvalidBundleContents)
	}
}

func TestS3BundleValidatorUploadsValidatedBundle(t *testing.T) {
	contents := []byte(`{"cases":[{"input":"1\n","expectedOutput":"2\n"}]}`)
	var uploadedPath string
	var uploadedBody string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodPut:
			uploadedPath = request.URL.Path
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatalf("io.ReadAll() error = %v", err)
			}
			uploadedBody = string(body)
			writer.WriteHeader(http.StatusOK)
		default:
			writer.WriteHeader(http.StatusMethodNotAllowed)
		}
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

	uploaded, err := validator.UploadBundle(context.Background(), "user-id", "bundle.json", contents)
	if err != nil {
		t.Fatalf("UploadBundle() error = %v", err)
	}
	if uploaded.SHA256 != sha256Hex(contents) {
		t.Fatalf("UploadBundle() checksum = %q, want %q", uploaded.SHA256, sha256Hex(contents))
	}
	if !strings.HasPrefix(uploaded.Key, "problem-drafts/user-id/") {
		t.Fatalf("UploadBundle() key = %q, want problem-drafts/user-id prefix", uploaded.Key)
	}
	if !strings.HasSuffix(uploaded.Key, "bundle.json") {
		t.Fatalf("UploadBundle() key = %q, want bundle.json suffix", uploaded.Key)
	}
	if uploadedBody != string(contents) {
		t.Fatalf("uploaded body = %q, want %q", uploadedBody, string(contents))
	}
	if uploadedPath != "/cabugi-hidden-tests/"+uploaded.Key {
		t.Fatalf("uploaded path = %q, want %q", uploadedPath, "/cabugi-hidden-tests/"+uploaded.Key)
	}
}

func sha256Hex(contents []byte) string {
	sum := sha256.Sum256(contents)
	return hex.EncodeToString(sum[:])
}
