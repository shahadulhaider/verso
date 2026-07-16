export interface Config {
  port: number;
  identityServiceUrl: string;
  catalogServiceUrl: string;
  searchServiceUrl: string;
  profileServiceUrl: string;
  mediaServiceUrl: string;
  libraryServiceUrl: string;
  reviewServiceUrl: string;
  socialServiceUrl: string;
  feedServiceUrl: string;
}

export function loadConfig(): Config {
  return {
    port: parseInt(process.env.PORT ?? "8010", 10),
    identityServiceUrl:
      process.env.IDENTITY_SERVICE_URL ?? "http://localhost:8001",
    catalogServiceUrl:
      process.env.CATALOG_SERVICE_URL ?? "http://localhost:8002",
    searchServiceUrl:
      process.env.SEARCH_SERVICE_URL ?? "http://localhost:8003",
    profileServiceUrl:
      process.env.PROFILE_SERVICE_URL ?? "http://localhost:8004",
    mediaServiceUrl:
      process.env.MEDIA_SERVICE_URL ?? "http://localhost:8005",
    libraryServiceUrl:
      process.env.LIBRARY_SERVICE_URL ?? "http://localhost:8006",
    reviewServiceUrl:
      process.env.REVIEW_SERVICE_URL ?? "http://localhost:8007",
    socialServiceUrl:
      process.env.SOCIAL_SERVICE_URL ?? "http://localhost:8008",
    feedServiceUrl:
      process.env.FEED_SERVICE_URL ?? "http://localhost:8009",
  };
}
