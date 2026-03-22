package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Identity holds the runner registration parameters resolved at startup.
type Identity struct {
	// RunnerID is the unique identifier for this runner.
	RunnerID string
	// PrivateURL is the URL that the broker uses to reach this runner.
	PrivateURL string
}

// identityDeps holds injectable dependencies for identity resolution.
type identityDeps struct {
	getenv   func(string) string
	hostname func() (string, error)
	httpGet  func(ctx context.Context, url string) ([]byte, error)
	port     string
}

// ecsTaskMeta is the subset of fields from GET $ECS_CONTAINER_METADATA_URI_V4/task.
type ecsTaskMeta struct {
	// TaskARN is the full ARN of the ECS task.
	TaskARN string `json:"TaskARN"`
}

// ecsNetwork represents a single network attachment in ECS container metadata.
type ecsNetwork struct {
	// IPv4Addresses holds the IPv4 addresses assigned to the container.
	IPv4Addresses []string `json:"IPv4Addresses"`
}

// ecsContainerMeta is the subset of fields from GET $ECS_CONTAINER_METADATA_URI_V4.
type ecsContainerMeta struct {
	// Networks holds the network attachments for the container.
	Networks []ecsNetwork `json:"Networks"`
}

// resolveIdentity determines the runner identity using a fallback chain:
// 1. ECS Task Metadata V4 when ECS_CONTAINER_METADATA_URI_V4 is set
// 2. Docker Compose hostname-based defaults
func resolveIdentity(ctx context.Context, deps identityDeps) (Identity, error) {
	ecsURI := deps.getenv("ECS_CONTAINER_METADATA_URI_V4")
	if ecsURI != "" {
		return resolveFromECS(ctx, deps, ecsURI)
	}
	return resolveFromHostname(deps)
}

// resolveFromECS resolves identity from ECS Task Metadata V4 endpoint.
func resolveFromECS(ctx context.Context, deps identityDeps, ecsURI string) (Identity, error) {
	taskBody, err := deps.httpGet(ctx, ecsURI+"/task")
	if err != nil {
		return Identity{}, fmt.Errorf("fetch ECS task metadata: %w", err)
	}
	var task ecsTaskMeta
	if err := json.Unmarshal(taskBody, &task); err != nil {
		return Identity{}, fmt.Errorf("parse ECS task metadata: %w", err)
	}
	idx := strings.LastIndex(task.TaskARN, "/")
	if idx < 0 || idx == len(task.TaskARN)-1 {
		return Identity{}, fmt.Errorf("invalid TaskARN: %s", task.TaskARN)
	}
	runnerID := task.TaskARN[idx+1:]

	containerBody, err := deps.httpGet(ctx, ecsURI)
	if err != nil {
		return Identity{}, fmt.Errorf("fetch ECS container metadata: %w", err)
	}
	var container ecsContainerMeta
	if err := json.Unmarshal(containerBody, &container); err != nil {
		return Identity{}, fmt.Errorf("parse ECS container metadata: %w", err)
	}
	if len(container.Networks) == 0 || len(container.Networks[0].IPv4Addresses) == 0 {
		return Identity{}, fmt.Errorf("no IPv4 address in ECS container metadata")
	}
	ip := container.Networks[0].IPv4Addresses[0]

	return Identity{
		RunnerID:   runnerID,
		PrivateURL: "http://" + ip + ":" + deps.port,
	}, nil
}

// resolveFromHostname resolves identity from the container hostname.
func resolveFromHostname(deps identityDeps) (Identity, error) {
	host, err := deps.hostname()
	if err != nil {
		return Identity{}, fmt.Errorf("get hostname: %w", err)
	}
	return Identity{
		RunnerID:   host,
		PrivateURL: "http://" + host + ":" + deps.port,
	}, nil
}

// defaultHTTPGet performs an HTTP GET request and returns the response body.
// It is used for fetching ECS metadata from the local endpoint.
func defaultHTTPGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata endpoint returned %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
