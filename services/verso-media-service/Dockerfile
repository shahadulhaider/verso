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
COPY services/verso-media-service/package.json services/verso-media-service/pnpm-lock.yaml* ./services/verso-media-service/
WORKDIR /build/services/verso-media-service
RUN pnpm install --frozen-lockfile --ignore-scripts 2>/dev/null || pnpm install --ignore-scripts

# ── Stage 2: Build TypeScript ──────────────────────────────────────────
FROM deps AS build
COPY services/verso-media-service/tsconfig.json /build/services/verso-media-service/
COPY services/verso-media-service/src/ /build/services/verso-media-service/src/
WORKDIR /build/services/verso-media-service
RUN pnpm run build

# ── Stage 3: Production image ─────────────────────────────────────────
FROM node:22-alpine AS runtime
RUN apk add --no-cache curl
RUN corepack enable && corepack prepare pnpm@latest --activate
WORKDIR /app

COPY --from=deps /build/services/verso-media-service/node_modules ./node_modules
COPY --from=build /build/services/verso-media-service/dist ./dist
COPY --from=deps /build/services/verso-media-service/package.json ./

ENV NODE_ENV=production
ENV SERVICE_PORT=8005

EXPOSE 8005
CMD ["node", "dist/index.js"]
