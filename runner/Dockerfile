# checkov:skip=CKV_DOCKER_3:runner executes user commands and requires root
# checkov:skip=CKV_DOCKER_2:health check is managed by compose/orchestrator
FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./

# test stage
FROM build AS test
RUN apk add --no-cache bash~=5.3
RUN CGO_ENABLED=0 go test -v -cover -timeout=30s ./...

# race detection stage
FROM build AS test-race
RUN apk add --no-cache bash~=5.3 gcc musl-dev
RUN CGO_ENABLED=1 go test -race -v -count=1 ./...

# binary build
FROM build AS binary
RUN CGO_ENABLED=0 go build -o /server .

# production image
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    bash=5.2* \
    coreutils=9.1* \
    && rm -rf /var/lib/apt/lists/*
COPY --from=binary /server /server
EXPOSE 3000
CMD ["/server"]
