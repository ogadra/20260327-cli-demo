# checkov:skip=CKV_DOCKER_2:health check is managed by compose/orchestrator
FROM node:24-slim AS deps
RUN corepack enable && corepack prepare pnpm@10.32.1 --activate
WORKDIR /app
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

FROM deps AS test
COPY . .
RUN pnpm exec vp check && pnpm exec vp test --run

FROM deps AS test-e2e
COPY . .
RUN pnpm exec playwright install --with-deps chromium && pnpm exec playwright test

FROM deps AS build
COPY . .
RUN pnpm exec vp build

FROM deps AS dev
RUN useradd -r -s /usr/sbin/nologin appuser
COPY --chown=appuser:appuser . .
USER appuser
EXPOSE 5173
CMD ["pnpm", "exec", "vp", "dev", "--host"]
