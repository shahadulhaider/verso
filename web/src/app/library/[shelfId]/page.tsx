'use client';

import { useCallback, useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import AuthGuard from '@/components/AuthGuard';
import BookCard from '@/components/BookCard';
import { fetchShelfItems, removeFromShelf } from '@/lib/api';

interface ShelfItem {
  id: string;
  workId: string;
  title: string;
  description?: string;
  originalLanguage?: string;
  originalPublicationYear?: number;
}

export default function ShelfDetailPage() {
  const params = useParams<{ shelfId: string }>();
  const [items, setItems] = useState<ShelfItem[]>([]);
  const [loading, setLoading] = useState(true);

  const loadItems = useCallback(async () => {
    if (!params.shelfId) return;
    try {
      const res = await fetchShelfItems(params.shelfId);
      if (res.ok) {
        const data = await res.json();
        setItems(data.items ?? []);
      }
    } catch {
      /* empty — apiFetch redirects on 401 */
    } finally {
      setLoading(false);
    }
  }, [params.shelfId]);

  useEffect(() => {
    loadItems();
  }, [loadItems]);

  const handleRemove = async (itemId: string) => {
    if (!params.shelfId) return;
    try {
      const res = await removeFromShelf(params.shelfId, itemId);
      if (res.ok || res.status === 204) {
        setItems((prev) => prev.filter((i) => i.id !== itemId));
      }
    } catch {
      /* empty — apiFetch redirects on 401 */
    }
  };

  return (
    <AuthGuard>
      <div className="mx-auto max-w-5xl px-4 py-8 sm:px-6">
        <Link href="/library"
          className="mb-6 inline-flex items-center gap-1 text-sm text-stone-500
            transition-colors hover:text-stone-800
            dark:text-stone-400 dark:hover:text-stone-200">
          <svg aria-hidden="true" className="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none"
            strokeWidth={2} stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round"
              d="M10.5 19.5L3 12m0 0l7.5-7.5M3 12h18" />
          </svg>
          Back to library
        </Link>

        <h1 className="mb-6 font-serif text-2xl font-bold tracking-tight text-stone-900
          dark:text-stone-100">
          Shelf
        </h1>

        {loading ? (
          <div className="flex items-center justify-center py-20">
            <div className="text-sm text-stone-400 dark:text-stone-500">Loading…</div>
          </div>
        ) : items.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-lg border border-dashed
            border-stone-300 py-20 dark:border-stone-600">
            <p className="text-sm font-medium text-stone-500 dark:text-stone-400">
              This shelf is empty
            </p>
            <Link href="/books"
              className="mt-2 text-sm font-medium text-amber-700 hover:text-amber-800
                dark:text-amber-400 dark:hover:text-amber-300">
              Browse books →
            </Link>
          </div>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {items.map((item) => (
              <div key={item.id} className="relative">
                <BookCard
                  id={item.workId}
                  title={item.title}
                  description={item.description}
                  originalLanguage={item.originalLanguage}
                  originalPublicationYear={item.originalPublicationYear}
                />
                <button type="button"
                  onClick={() => handleRemove(item.id)}
                  className="absolute right-2 top-2 rounded-md bg-white/80 p-1 text-stone-400
                    opacity-0 transition-all hover:bg-red-50 hover:text-red-500
                    group-hover:opacity-100 [div:hover>&]:opacity-100
                    dark:bg-stone-800/80 dark:text-stone-500
                    dark:hover:bg-red-950 dark:hover:text-red-400"
                  aria-label="Remove from shelf">
                  <svg aria-hidden="true" className="h-4 w-4" viewBox="0 0 24 24" fill="none"
                    strokeWidth={1.5} stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </AuthGuard>
  );
}
