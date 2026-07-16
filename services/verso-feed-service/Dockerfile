# ── Stage 1: Install dependencies ──────────────────────────────────────
FROM node:22-alpine AS deps
RUN corepack enable && corepack prepare pnpm@latest --activate
WORKDIR /build

# Copy and build shared @verso/jwt lib (needed by file: dep in package.json)
COPY libs/node/packages/tsconfig.base.json libs/node/packages/tsconfig.base.json
COPY libs/node/packages/jwt/package.json libs/node/packages/jwt/package.json
COPY libs/node/packages/jwt/tsconfig.json libs/node/packages/jwt/tsconfig.json
COPY libs/node/packages/jwt/src/index.ts libs/node/packages/jwt/src/index.ts
RUN cd libs/node/packages/jwt && pnpm install --ignore-scripts && npx tsc

# Copy service manifests and install
COPY services/verso-feed-service/package.json services/verso-feed-service/pnpm-lock.yaml* ./services/verso-feed-service/
WORKDIR /build/services/verso-feed-service
RUN pnpm install --frozen-lockfile --ignore-scripts 2>/dev/null || pnpm install --ignore-scripts

# ── Stage 2: Build TypeScript ──────────────────────────────────────────
FROM deps AS build
COPY services/verso-feed-service/tsconfig.json /build/services/verso-feed-service/
COPY services/verso-feed-service/src/ /build/services/verso-feed-service/src/
WORKDIR /build/services/verso-feed-service
RUN pnpm run build

# ── Stage 3: Production image ─────────────────────────────────────────
FROM node:22-alpine AS runtime
RUN apk add --no-cache curl
RUN corepack enable && corepack prepare pnpm@latest --activate
WORKDIR /app

COPY --from=deps /build/services/verso-feed-service/node_modules ./node_modules
COPY --from=build /build/services/verso-feed-service/dist ./dist
COPY --from=deps /build/services/verso-feed-service/package.json ./

ENV NODE_ENV=production
ENV SERVICE_PORT=8009

EXPOSE 8009
CMD ["node", "dist/index.js"]
