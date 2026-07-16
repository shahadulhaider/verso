import { getToken, clearToken } from './auth';

export async function apiFetch(
  path: string,
  options: RequestInit = {},
): Promise<Response> {
  const token = getToken();
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(options.headers as Record<string, string> | undefined),
  };

  const res = await fetch(path, { ...options, headers });

  if (res.status === 401) {
    clearToken();
    window.location.href = '/login';
    throw new Error('Unauthorized');
  }

  return res;
}

// ---------------------------------------------------------------------------
// Profiles
// ---------------------------------------------------------------------------

export function fetchMyProfile() {
  return apiFetch('/api/v1/profiles/me');
}

export function updateMyProfile(body: Record<string, unknown>) {
  return apiFetch('/api/v1/profiles/me', {
    method: 'PATCH',
    body: JSON.stringify(body),
  });
}

export function fetchProfile(userId: string) {
  return apiFetch(`/api/v1/profiles/${userId}`);
}

// ---------------------------------------------------------------------------
// Library / Shelves
// ---------------------------------------------------------------------------

export function fetchShelves() {
  return apiFetch('/api/v1/library/shelves');
}

export function createShelf(name: string, description?: string) {
  return apiFetch('/api/v1/library/shelves', {
    method: 'POST',
    body: JSON.stringify({ name, description }),
  });
}

export function fetchShelfItems(shelfId: string, cursor?: string) {
  const qs = cursor ? `?cursor=${encodeURIComponent(cursor)}` : '';
  return apiFetch(`/api/v1/library/shelves/${shelfId}/items${qs}`);
}

export function addToShelf(shelfId: string, workId: string) {
  return apiFetch(`/api/v1/library/shelves/${shelfId}/items`, {
    method: 'POST',
    body: JSON.stringify({ workId }),
  });
}

export function removeFromShelf(shelfId: string, itemId: string) {
  return apiFetch(`/api/v1/library/shelves/${shelfId}/items/${itemId}`, {
    method: 'DELETE',
  });
}

// ---------------------------------------------------------------------------
// Reviews
// ---------------------------------------------------------------------------

export function fetchWorkReviews(workId: string, cursor?: string) {
  const qs = cursor ? `?cursor=${encodeURIComponent(cursor)}` : '';
  return apiFetch(`/api/v1/works/${workId}/reviews${qs}`);
}

export function fetchAggregateRating(workId: string) {
  return apiFetch(`/api/v1/works/${workId}/aggregate-rating`);
}

export function submitReview(body: Record<string, unknown>) {
  return apiFetch('/api/v1/reviews', {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

// ---------------------------------------------------------------------------
// Social (follow / unfollow)
// ---------------------------------------------------------------------------

export function followUser(targetUserId: string) {
  return apiFetch('/api/v1/social/follow', {
    method: 'POST',
    body: JSON.stringify({ targetUserId }),
  });
}

export function unfollowUser(targetUserId: string) {
  return apiFetch('/api/v1/social/follow', {
    method: 'DELETE',
    body: JSON.stringify({ targetUserId }),
  });
}

export function fetchFollowers(userId: string, cursor?: string) {
  const qs = cursor ? `?cursor=${encodeURIComponent(cursor)}` : '';
  return apiFetch(`/api/v1/social/${userId}/followers${qs}`);
}

export function fetchFollowing(userId: string, cursor?: string) {
  const qs = cursor ? `?cursor=${encodeURIComponent(cursor)}` : '';
  return apiFetch(`/api/v1/social/${userId}/following${qs}`);
}

// ---------------------------------------------------------------------------
// Feed
// ---------------------------------------------------------------------------

export function fetchTimeline(mode: 'chronological' | 'algorithmic' = 'chronological', cursor?: string) {
  const params = new URLSearchParams({ mode });
  if (cursor) params.set('cursor', cursor);
  return apiFetch(`/api/v1/feed/timeline?${params.toString()}`);
}

// ---------------------------------------------------------------------------
// Semantic search
// ---------------------------------------------------------------------------

export function semanticSearch(query: string) {
  return apiFetch(`/api/v1/search/semantic?q=${encodeURIComponent(query)}`);
}
