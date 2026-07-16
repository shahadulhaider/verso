import Fastify from "fastify";
import cors from "@fastify/cors";
import { loadConfig } from "./config.js";
import { authRoutes } from "./routes/auth.js";
import { booksRoutes } from "./routes/books.js";
import { searchRoutes } from "./routes/search.js";
import { profilesRoutes } from "./routes/profiles.js";
import { libraryRoutes } from "./routes/library.js";
import { reviewsRoutes } from "./routes/reviews.js";
import { socialRoutes } from "./routes/social.js";
import { feedRoutes } from "./routes/feed.js";
import { mediaRoutes } from "./routes/media.js";

const config = loadConfig();

const app = Fastify({
  logger: {
    name: "web-bff",
    timestamp: () => `,"time":"${new Date().toISOString()}"`,
  },
});

await app.register(cors, {
  origin: ["http://localhost:3100", "http://localhost:3000"],
  methods: ["GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"],
  allowedHeaders: ["Authorization", "Content-Type"],
});

app.get("/health", async () => ({ status: "ok" }));

await app.register(authRoutes, { config });
await app.register(booksRoutes, { config });
await app.register(searchRoutes, { config });
await app.register(profilesRoutes, { config });
await app.register(libraryRoutes, { config });
await app.register(reviewsRoutes, { config });
await app.register(socialRoutes, { config });
await app.register(feedRoutes, { config });
await app.register(mediaRoutes, { config });

// Start
try {
  await app.listen({ port: config.port, host: "0.0.0.0" });
} catch (err) {
  app.log.error(err);
  process.exit(1);
}

export { app };
