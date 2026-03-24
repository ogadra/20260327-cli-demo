# checkov:skip=CKV_DOCKER_2:Lambda functions do not need health checks
# checkov:skip=CKV_DOCKER_3:Lambda runtime manages the user context
FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# test stage
FROM build AS test
SHELL ["/bin/ash", "-o", "pipefail", "-c"]
RUN CGO_ENABLED=0 go test -v -coverpkg=./... -coverprofile=cover.out ./... \
    && total=$(go tool cover -func=cover.out | grep ^total: | awk '{print $NF}') \
    && echo "Total coverage: ${total}" \
    && test "${total}" = "100.0%" || { echo "FAIL: coverage ${total} is below 100.0%"; exit 1; }

# test with race detector
FROM build AS test-race
RUN apk add --no-cache gcc musl-dev \
    && CGO_ENABLED=1 go test -race -v ./...

# binary build for arm64 Lambda
FROM build AS binary
RUN mkdir -p /out/connect /out/disconnect /out/message /out/login \
    && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /out/connect/bootstrap ./cmd/connect \
    && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /out/disconnect/bootstrap ./cmd/disconnect \
    && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /out/message/bootstrap ./cmd/message \
    && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /out/login/bootstrap ./cmd/login
