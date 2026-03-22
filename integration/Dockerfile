# checkov:skip=CKV_DOCKER_2:test-only container, no health check needed
FROM golang:1.26-alpine
RUN adduser -D -s /sbin/nologin appuser
WORKDIR /app
COPY go.mod ./
COPY . .
USER appuser
CMD ["go", "test", "-v", "-count=1", "-tags=integration", "-timeout=120s", "./..."]
