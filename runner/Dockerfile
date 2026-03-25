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
SHELL ["/bin/ash", "-o", "pipefail", "-c"]
RUN CGO_ENABLED=0 go test -v -coverprofile=cover.out -timeout=30s ./... \
    && total=$(go tool cover -func=cover.out | grep ^total: | awk '{print $NF}') \
    && echo "Total coverage: ${total}" \
    && test "${total}" = "100.0%" || { echo "FAIL: coverage ${total} is below 100.0%"; exit 1; }

# race detection stage
FROM build AS test-race
RUN apk add --no-cache bash~=5.3 gcc musl-dev
RUN CGO_ENABLED=1 go test -race -v -count=1 ./...

# binary build
FROM build AS binary
RUN CGO_ENABLED=0 go build -o /server .

# production image
FROM nixos/nix:2.34.3

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

RUN mkdir -p /etc/nix && \
    printf '%s\n' 'extra-experimental-features = nix-command flakes' > /etc/nix/nix.conf

RUN nix-channel --update && nix-env -iA nixpkgs.bash nixpkgs.coreutils

RUN nix build nixpkgs#cowsay && \
    nix build nixpkgs#pokemonsay

COPY --from=binary /server /server
EXPOSE 3000
CMD ["/server"]
