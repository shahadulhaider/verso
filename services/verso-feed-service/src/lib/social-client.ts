import { request as undiciRequest } from "undici";
import CircuitBreaker from "opossum";

export interface FollowersResponse {
  count: number;
  userIds: string[];
}

function createFetchFollowers(socialServiceUrl: string) {
  return async (actorId: string): Promise<FollowersResponse> => {
    const url = `${socialServiceUrl}/v1/social/followers/${actorId}`;
    const res = await undiciRequest(url, {
      method: "GET",
      signal: AbortSignal.timeout(5000),
    });

    if (res.statusCode !== 200) {
      throw new Error(`Social service returned ${res.statusCode}`);
    }

    const body = await res.body.json() as FollowersResponse;
    return body;
  };
}

export function createSocialBreaker(
  socialServiceUrl: string,
): CircuitBreaker<[string], FollowersResponse> {
  const fetchFn = createFetchFollowers(socialServiceUrl);
  return new CircuitBreaker(fetchFn, {
    timeout: 5000,
    errorThresholdPercentage: 50,
    resetTimeout: 30000,
    volumeThreshold: 5,
    name: "social-followers",
  });
}
