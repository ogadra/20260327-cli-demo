FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./

# test stage
FROM build AS test
RUN CGO_ENABLED=0 go test -v ./...

# test with race detector (requires cgo + gcc)
FROM build AS test-race
RUN apk add --no-cache gcc musl-dev
RUN CGO_ENABLED=1 go test -race -v ./...

# binary build
FROM build AS binary
RUN CGO_ENABLED=0 go build -o /server .

# production image
FROM debian:bookworm-20260316-slim
RUN apt-get update && apt-get install -y bash coreutils && rm -rf /var/lib/apt/lists/*
COPY --from=binary /server /server
EXPOSE 8080
CMD ["/server"]
