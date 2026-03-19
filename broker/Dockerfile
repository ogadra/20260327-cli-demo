# checkov:skip=CKV_DOCKER_3:broker does not require non-root user at this stage
# checkov:skip=CKV_DOCKER_2:health check is managed by compose/orchestrator
# trivy:ignore:DS-0002 -- broker does not require non-root user at this stage
# trivy:ignore:DS-0026 -- health check is managed by compose/orchestrator
FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# test stage
FROM build AS test
RUN CGO_ENABLED=0 go test -v ./...

# integration test stage
FROM build AS test-integration
CMD ["go", "test", "-v", "-count=1", "-tags=integration", "./..."]

# test with race detector (requires cgo + gcc)
FROM build AS test-race
RUN apk add --no-cache gcc musl-dev \
    && CGO_ENABLED=1 go test -race -v ./...

# binary build
FROM build AS binary
RUN CGO_ENABLED=0 go build -o /server .

# production image
FROM debian:bookworm-20260316-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    bash=5.2* \
    coreutils=9.1* \
    && rm -rf /var/lib/apt/lists/*
COPY --from=binary /server /server
EXPOSE 8080
CMD ["/server"]
