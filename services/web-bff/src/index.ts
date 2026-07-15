import Fastify from "fastify";
import cors from "@fastify/cors";
import { loadConfig } from "./config.js";
import { authRoutes } from "./routes/auth.js";
import { booksRoutes } from "./routes/books.js";
import { searchRoutes } from "./routes/search.js";

const config = loadConfig();

const app = Fastify({
  logger: {
    name: "web-bff",
    timestamp: () => `,"time":"${new Date().toISOString()}"`,
  },
});

// CORS — allow Next.js dev server
await app.register(cors, {
  origin: "http://localhost:3000",
  methods: ["GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"],
  allowedHeaders: ["Authorization", "Content-Type"],
});

// Health
app.get("/health", async () => ({ status: "ok" }));

// Routes
await app.register(authRoutes, { config });
await app.register(booksRoutes, { config });
await app.register(searchRoutes, { config });

// Start
try {
  await app.listen({ port: config.port, host: "0.0.0.0" });
} catch (err) {
  app.log.error(err);
  process.exit(1);
}

export { app };
