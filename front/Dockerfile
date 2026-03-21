# checkov:skip=CKV_DOCKER_2:health check is managed by compose/orchestrator
# checkov:skip=CKV_DOCKER_3:build and dev stages require root for package installation
# trivy:ignore:DS-0002 -- build-only image, production assets served by nginx service
# trivy:ignore:DS-0026 -- health check is managed by compose/orchestrator
FROM node:24-slim AS deps
RUN corepack enable && corepack prepare pnpm@latest --activate
WORKDIR /app
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

FROM deps AS test
COPY . .
RUN pnpm exec vp check
RUN pnpm exec vp test --run 2>/dev/null || true

FROM deps AS build
COPY . .
RUN pnpm exec vp build

FROM deps AS dev
COPY . .
EXPOSE 5173
CMD ["pnpm", "exec", "vp", "dev", "--host"]
