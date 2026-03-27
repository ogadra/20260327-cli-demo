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

ENV PATH="/usr/local/bin:${PATH}"

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

RUN mkdir -p /etc/nix && \
    printf '%s\n' 'extra-experimental-features = nix-command flakes' > /etc/nix/nix.conf

RUN nix-channel --add https://nixos.org/channels/nixpkgs-unstable nixpkgs && \
    nix-channel --add https://github.com/nix-community/home-manager/archive/release-25.11.tar.gz home-manager && \
    nix-channel --update

RUN nix-env -e man man-db || true

RUN nix-shell '<home-manager>' -A install

RUN command -v home-manager && \
    HM_REAL="$(readlink -f "$(command -v home-manager)")" && \
    mkdir -p /usr/local/bin && \
    printf '%s\n' '#!/bin/sh' "exec \"$HM_REAL\" \"\$@\"" > /usr/local/bin/home-manager && \
    chmod +x /usr/local/bin/home-manager

RUN nix-env -e bash-interactive bash || true

RUN mkdir -p /root/.config/home-manager
COPY ./nix/home.nix /root/.config/home-manager/home.nix

RUN home-manager switch

RUN nix shell nixpkgs#gnused --command sh -c '\
    sed -i '\''s|TZ = "Asia/Tokyo";|TZ = "Pacific/Honolulu";|g'\'' /root/.config/home-manager/home.nix && \
    sed -i '\''s|LANG = "ja_JP.UTF-8";|LANG = "en_US.UTF-8";|g'\'' /root/.config/home-manager/home.nix && \
    sed -i '\''s|LC_ALL = "ja_JP.UTF-8";|LC_ALL = "en_US.UTF-8";|g'\'' /root/.config/home-manager/home.nix && \
    home-manager switch'

RUN nix build --impure --no-link --print-out-paths -f '<nixpkgs>' fastfetch cowsay pokemonsay lolcat figlet > /tmp/prebuilt-tools-paths.txt

RUN mkdir -p /app
COPY ./app/ /app/
WORKDIR /app

RUN nix flake archive

COPY --from=binary /server /server
EXPOSE 3000
CMD ["/server"]
