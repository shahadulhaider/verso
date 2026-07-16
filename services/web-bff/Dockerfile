# ── Stage 1: Install dependencies ──────────────────────────────────────
FROM node:22-alpine AS deps
WORKDIR /build
RUN corepack enable && corepack prepare pnpm@latest --activate

COPY services/web-bff/package.json services/web-bff/pnpm-lock.yaml* ./services/web-bff/
WORKDIR /build/services/web-bff
RUN pnpm install --frozen-lockfile --ignore-scripts 2>/dev/null || pnpm install --ignore-scripts

# ── Stage 2: Build TypeScript ──────────────────────────────────────────
FROM deps AS build
COPY services/web-bff/tsconfig.json /build/services/web-bff/
COPY services/web-bff/src/ /build/services/web-bff/src/
WORKDIR /build/services/web-bff
RUN pnpm run build

# ── Stage 3: Production image ─────────────────────────────────────────
FROM node:22-alpine AS runtime
RUN corepack enable && corepack prepare pnpm@latest --activate
WORKDIR /app

COPY --from=deps /build/services/web-bff/node_modules ./node_modules
COPY --from=build /build/services/web-bff/dist ./dist
COPY --from=deps /build/services/web-bff/package.json ./

ENV NODE_ENV=production
ENV PORT=8010

EXPOSE 8010
CMD ["node", "dist/index.js"]
