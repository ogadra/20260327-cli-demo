package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// registerRetryInterval is the duration between registration retry attempts.
const registerRetryInterval = 3 * time.Second

// registerDeps holds injectable dependencies for broker registration.
type registerDeps struct {
	brokerURL string
	identity  Identity
	httpPost  func(ctx context.Context, url string, contentType string, body io.Reader) (*http.Response, error)
	afterFunc func(time.Duration) <-chan time.Time
	logf      func(string, ...any)
}

// registerRequest is the JSON body sent to POST /internal/runners/register.
type registerRequest struct {
	// RunnerID is the unique identifier for this runner.
	RunnerID string `json:"runnerId"`
	// PrivateURL is the URL that the broker uses to reach this runner.
	PrivateURL string `json:"privateUrl"`
}

// register sends POST /internal/runners/register to broker with retry.
// It retries every registerRetryInterval until success with HTTP 201 or context cancellation.
// This function blocks until registration succeeds.
func register(ctx context.Context, deps registerDeps) error {
	reqBody := registerRequest{
		RunnerID:   deps.identity.RunnerID,
		PrivateURL: deps.identity.PrivateURL,
	}
	payload, _ := json.Marshal(reqBody)

	url := deps.brokerURL + "/internal/runners/register"

	for {
		resp, err := deps.httpPost(ctx, url, "application/json", bytes.NewReader(payload))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusCreated {
				deps.logf("registered with broker: id=%s url=%s", deps.identity.RunnerID, deps.identity.PrivateURL)
				return nil
			}
			deps.logf("broker registration returned %d, retrying in %s", resp.StatusCode, registerRetryInterval)
		} else {
			deps.logf("broker registration failed: %v, retrying in %s", err, registerRetryInterval)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deps.afterFunc(registerRetryInterval):
		}
	}
}

// defaultHTTPPost performs an HTTP POST request with the given context.
func defaultHTTPPost(ctx context.Context, url string, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return http.DefaultClient.Do(req)
}
