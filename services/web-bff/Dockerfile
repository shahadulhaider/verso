# ── Stage 1: Install dependencies ──────────────────────────────────────
FROM node:22-alpine AS deps
WORKDIR /app
RUN corepack enable && corepack prepare pnpm@latest --activate
COPY package.json pnpm-lock.yaml* ./
RUN pnpm install --frozen-lockfile --ignore-scripts 2>/dev/null || pnpm install --ignore-scripts

# ── Stage 2: Build TypeScript ──────────────────────────────────────────
FROM deps AS build
COPY tsconfig.json ./
COPY src/ ./src/
RUN pnpm run build

# ── Stage 3: Production image ─────────────────────────────────────────
FROM node:22-alpine AS runtime
RUN corepack enable && corepack prepare pnpm@latest --activate
WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY --from=build /app/dist ./dist
COPY --from=deps /app/package.json ./

ENV NODE_ENV=production
ENV PORT=8010

EXPOSE 8010
CMD ["node", "dist/index.js"]
