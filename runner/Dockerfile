FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod *.go ./

# test stage
FROM build AS test
RUN apk add --no-cache bash
RUN CGO_ENABLED=0 go test -v -cover ./...

# binary build
FROM build AS binary
RUN CGO_ENABLED=0 go build -o /server .

# production image
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y bash coreutils && rm -rf /var/lib/apt/lists/*
COPY --from=binary /server /server
EXPOSE 3000
CMD ["/server"]
