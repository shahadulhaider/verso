export interface Config {
  port: number;
  identityServiceUrl: string;
  catalogServiceUrl: string;
  searchServiceUrl: string;
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
  };
}
