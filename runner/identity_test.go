package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestResolveIdentityECS verifies that resolveIdentity uses ECS metadata
// when ECS_CONTAINER_METADATA_URI_V4 is set.
func TestResolveIdentityECS(t *testing.T) {
	taskJSON := `{"TaskARN":"arn:aws:ecs:ap-northeast-1:123456789:task/cluster/abc123"}`
	containerJSON := `{"Networks":[{"IPv4Addresses":["10.0.1.5"]}]}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/task" {
			w.Write([]byte(taskJSON))
		} else {
			w.Write([]byte(containerJSON))
		}
	}))
	defer ts.Close()

	deps := identityDeps{
		getenv:   func(k string) string { return ts.URL },
		hostname: func() (string, error) { return "", errors.New("should not be called") },
		httpGet:  defaultHTTPGet,
		port:     "3000",
	}

	id, err := resolveIdentity(context.Background(), deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.RunnerID != "abc123" {
		t.Errorf("RunnerID = %q, want %q", id.RunnerID, "abc123")
	}
	if id.PrivateURL != "http://10.0.1.5:3000" {
		t.Errorf("PrivateURL = %q, want %q", id.PrivateURL, "http://10.0.1.5:3000")
	}
}

// TestResolveIdentityECSTaskFetchError verifies that resolveIdentity returns
// an error when fetching the ECS task metadata fails.
func TestResolveIdentityECSTaskFetchError(t *testing.T) {
	deps := identityDeps{
		getenv: func(k string) string { return "http://169.254.170.2/v4" },
		httpGet: func(ctx context.Context, url string) ([]byte, error) {
			return nil, errors.New("connection refused")
		},
		port: "3000",
	}

	_, err := resolveIdentity(context.Background(), deps)
	if err == nil {
		t.Fatal("expected error when task metadata fetch fails")
	}
}

// TestResolveIdentityECSTaskInvalidJSON verifies that resolveIdentity returns
// an error when ECS task metadata contains invalid JSON.
func TestResolveIdentityECSTaskInvalidJSON(t *testing.T) {
	deps := identityDeps{
		getenv: func(k string) string { return "http://169.254.170.2/v4" },
		httpGet: func(ctx context.Context, url string) ([]byte, error) {
			if url == "http://169.254.170.2/v4/task" {
				return []byte("{invalid"), nil
			}
			return nil, errors.New("unexpected call")
		},
		port: "3000",
	}

	_, err := resolveIdentity(context.Background(), deps)
	if err == nil {
		t.Fatal("expected error when task metadata is invalid JSON")
	}
}

// TestResolveIdentityECSTaskARNNoSlash verifies that resolveIdentity returns
// an error when TaskARN does not contain a slash.
func TestResolveIdentityECSTaskARNNoSlash(t *testing.T) {
	deps := identityDeps{
		getenv: func(k string) string { return "http://169.254.170.2/v4" },
		httpGet: func(ctx context.Context, url string) ([]byte, error) {
			return []byte(`{"TaskARN":"noslash"}`), nil
		},
		port: "3000",
	}

	_, err := resolveIdentity(context.Background(), deps)
	if err == nil {
		t.Fatal("expected error when TaskARN has no slash")
	}
}

// TestResolveIdentityECSTaskARNTrailingSlash verifies that resolveIdentity returns
// an error when TaskARN ends with a slash.
func TestResolveIdentityECSTaskARNTrailingSlash(t *testing.T) {
	deps := identityDeps{
		getenv: func(k string) string { return "http://169.254.170.2/v4" },
		httpGet: func(ctx context.Context, url string) ([]byte, error) {
			return []byte(`{"TaskARN":"arn:aws:ecs:region:account:task/cluster/"}`), nil
		},
		port: "3000",
	}

	_, err := resolveIdentity(context.Background(), deps)
	if err == nil {
		t.Fatal("expected error when TaskARN ends with slash")
	}
}

// TestResolveIdentityECSContainerFetchError verifies that resolveIdentity returns
// an error when fetching the ECS container metadata fails.
func TestResolveIdentityECSContainerFetchError(t *testing.T) {
	callCount := 0
	deps := identityDeps{
		getenv: func(k string) string { return "http://169.254.170.2/v4" },
		httpGet: func(ctx context.Context, url string) ([]byte, error) {
			callCount++
			if callCount == 1 {
				return []byte(`{"TaskARN":"arn:aws:ecs:r:a:task/c/id123"}`), nil
			}
			return nil, errors.New("connection refused")
		},
		port: "3000",
	}

	_, err := resolveIdentity(context.Background(), deps)
	if err == nil {
		t.Fatal("expected error when container metadata fetch fails")
	}
}

// TestResolveIdentityECSContainerInvalidJSON verifies that resolveIdentity returns
// an error when ECS container metadata contains invalid JSON.
func TestResolveIdentityECSContainerInvalidJSON(t *testing.T) {
	callCount := 0
	deps := identityDeps{
		getenv: func(k string) string { return "http://169.254.170.2/v4" },
		httpGet: func(ctx context.Context, url string) ([]byte, error) {
			callCount++
			if callCount == 1 {
				return []byte(`{"TaskARN":"arn:aws:ecs:r:a:task/c/id123"}`), nil
			}
			return []byte("{bad"), nil
		},
		port: "3000",
	}

	_, err := resolveIdentity(context.Background(), deps)
	if err == nil {
		t.Fatal("expected error when container metadata is invalid JSON")
	}
}

// TestResolveIdentityECSEmptyNetworks verifies that resolveIdentity returns
// an error when ECS container metadata has no networks.
func TestResolveIdentityECSEmptyNetworks(t *testing.T) {
	callCount := 0
	deps := identityDeps{
		getenv: func(k string) string { return "http://169.254.170.2/v4" },
		httpGet: func(ctx context.Context, url string) ([]byte, error) {
			callCount++
			if callCount == 1 {
				return []byte(`{"TaskARN":"arn:aws:ecs:r:a:task/c/id123"}`), nil
			}
			return []byte(`{"Networks":[]}`), nil
		},
		port: "3000",
	}

	_, err := resolveIdentity(context.Background(), deps)
	if err == nil {
		t.Fatal("expected error when Networks is empty")
	}
}

// TestResolveIdentityECSEmptyIPv4 verifies that resolveIdentity returns
// an error when ECS container metadata has a network with no IPv4 addresses.
func TestResolveIdentityECSEmptyIPv4(t *testing.T) {
	callCount := 0
	deps := identityDeps{
		getenv: func(k string) string { return "http://169.254.170.2/v4" },
		httpGet: func(ctx context.Context, url string) ([]byte, error) {
			callCount++
			if callCount == 1 {
				return []byte(`{"TaskARN":"arn:aws:ecs:r:a:task/c/id123"}`), nil
			}
			return []byte(`{"Networks":[{"IPv4Addresses":[]}]}`), nil
		},
		port: "3000",
	}

	_, err := resolveIdentity(context.Background(), deps)
	if err == nil {
		t.Fatal("expected error when IPv4Addresses is empty")
	}
}

// TestResolveIdentityHostnameFallback verifies that resolveIdentity falls back
// to hostname-based resolution when ECS metadata is not available.
func TestResolveIdentityHostnameFallback(t *testing.T) {
	deps := identityDeps{
		getenv:   func(k string) string { return "" },
		hostname: func() (string, error) { return "abcdef123456", nil },
		httpGet: func(ctx context.Context, url string) ([]byte, error) {
			return nil, errors.New("should not be called")
		},
		port: "3000",
	}

	id, err := resolveIdentity(context.Background(), deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.RunnerID != "abcdef123456" {
		t.Errorf("RunnerID = %q, want %q", id.RunnerID, "abcdef123456")
	}
	if id.PrivateURL != "http://abcdef123456:3000" {
		t.Errorf("PrivateURL = %q, want %q", id.PrivateURL, "http://abcdef123456:3000")
	}
}

// TestResolveIdentityHostnameError verifies that resolveIdentity returns
// an error when hostname lookup fails.
func TestResolveIdentityHostnameError(t *testing.T) {
	deps := identityDeps{
		getenv:   func(k string) string { return "" },
		hostname: func() (string, error) { return "", errors.New("no hostname") },
		port:     "3000",
	}

	_, err := resolveIdentity(context.Background(), deps)
	if err == nil {
		t.Fatal("expected error when hostname fails")
	}
}

// TestDefaultHTTPGetSuccess verifies that defaultHTTPGet returns the body
// of a successful HTTP response.
func TestDefaultHTTPGetSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer ts.Close()

	body, err := defaultHTTPGet(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != "ok" {
		t.Errorf("body = %q, want %q", string(body), "ok")
	}
}

// TestDefaultHTTPGetNon200 verifies that defaultHTTPGet returns an error
// when the server responds with a non-200 status code.
func TestDefaultHTTPGetNon200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	_, err := defaultHTTPGet(context.Background(), ts.URL)
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

// TestDefaultHTTPGetCanceledContext verifies that defaultHTTPGet returns an error
// when the context is already canceled.
func TestDefaultHTTPGetCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := defaultHTTPGet(ctx, "http://127.0.0.1:1/unreachable")
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

// TestDefaultHTTPGetInvalidURL verifies that defaultHTTPGet returns an error
// when the URL is malformed and cannot be parsed into a request.
func TestDefaultHTTPGetInvalidURL(t *testing.T) {
	_, err := defaultHTTPGet(context.Background(), "://bad-url")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}
