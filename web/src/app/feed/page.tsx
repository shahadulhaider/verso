'use client';

import { useCallback, useEffect, useState } from 'react';
import AuthGuard from '@/components/AuthGuard';
import ActivityCard from '@/components/ActivityCard';
import { fetchTimeline } from '@/lib/api';

interface FeedItem {
  id: string;
  actorId: string;
  actorName: string;
  verb: string;
  objectType: string;
  objectId: string;
  objectTitle: string;
  extra?: Record<string, unknown>;
  occurredAt: string;
}

export default function FeedPage() {
  const [items, setItems] = useState<FeedItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [mode, setMode] = useState<'chronological' | 'algorithmic'>('chronological');
  const [cursor, setCursor] = useState<string | undefined>();
  const [hasMore, setHasMore] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);

  const loadFeed = useCallback(async (feedMode: 'chronological' | 'algorithmic', nextCursor?: string) => {
    const isLoadMore = !!nextCursor;
    if (isLoadMore) setLoadingMore(true);
    else setLoading(true);

    try {
      const res = await fetchTimeline(feedMode, nextCursor);
      if (res.ok) {
        const data = await res.json();
        const newItems: FeedItem[] = data.items ?? [];
        setItems((prev) => (isLoadMore ? [...prev, ...newItems] : newItems));
        setCursor(data.nextCursor ?? undefined);
        setHasMore(!!data.nextCursor);
      }
    } catch {
      /* empty — apiFetch redirects on 401 */
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  }, []);

  useEffect(() => {
    loadFeed(mode);
  }, [mode, loadFeed]);

  const handleModeChange = (newMode: 'chronological' | 'algorithmic') => {
    if (newMode === mode) return;
    setMode(newMode);
    setItems([]);
    setCursor(undefined);
  };

  return (
    <AuthGuard>
      <div className="mx-auto max-w-2xl px-4 py-8 sm:px-6">
        <div className="mb-6 flex items-end justify-between">
          <div>
            <h1 className="font-serif text-2xl font-bold tracking-tight text-stone-900
              dark:text-stone-100">
              Feed
            </h1>
            <p className="mt-1 text-sm text-stone-500 dark:text-stone-400">
              See what people you follow are reading
            </p>
          </div>
          <div className="flex rounded-md border border-stone-200 dark:border-stone-700">
            {(['chronological', 'algorithmic'] as const).map((m) => (
              <button key={m} type="button"
                onClick={() => handleModeChange(m)}
                className={`px-3 py-1.5 text-xs font-medium capitalize transition-colors
                  ${mode === m
                    ? 'bg-stone-900 text-white dark:bg-stone-100 dark:text-stone-900'
                    : 'text-stone-600 hover:bg-stone-50 dark:text-stone-400 dark:hover:bg-stone-800'
                  } ${m === 'chronological' ? 'rounded-l-md' : 'rounded-r-md'}`}>
                {m === 'chronological' ? 'Latest' : 'For you'}
              </button>
            ))}
          </div>
        </div>

        {loading ? (
          <div className="flex items-center justify-center py-20">
            <div className="text-sm text-stone-400 dark:text-stone-500">Loading…</div>
          </div>
        ) : items.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-lg border border-dashed
            border-stone-300 py-20 dark:border-stone-600">
            <svg aria-hidden="true" className="mb-3 h-8 w-8 text-stone-300 dark:text-stone-600"
              viewBox="0 0 24 24" fill="none" strokeWidth={1.5} stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round"
                d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z" />
            </svg>
            <p className="text-sm font-medium text-stone-500 dark:text-stone-400">
              Follow users to see their activity
            </p>
            <p className="mt-1 text-xs text-stone-400 dark:text-stone-500">
              Discover readers on the Books or Discover pages
            </p>
          </div>
        ) : (
          <div className="space-y-3">
            {items.map((item) => (
              <ActivityCard key={item.id} {...item} />
            ))}
            {hasMore && (
              <div className="pt-2 text-center">
                <button type="button"
                  disabled={loadingMore}
                  onClick={() => loadFeed(mode, cursor)}
                  className="rounded-md border border-stone-200 bg-white px-4 py-2 text-sm
                    font-medium text-stone-700 transition-colors hover:bg-stone-50
                    disabled:opacity-60 dark:border-stone-700 dark:bg-stone-800
                    dark:text-stone-300 dark:hover:bg-stone-700">
                  {loadingMore ? 'Loading…' : 'Load more'}
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </AuthGuard>
  );
}
