'use client';

import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import AuthGuard from '@/components/AuthGuard';
import { createShelf, fetchShelves } from '@/lib/api';

interface Shelf {
  id: string;
  name: string;
  shelfType: string;
  itemCount: number;
}

const SHELF_ICONS: Record<string, string> = {
  'want-to-read': 'M17.593 3.322c1.1.128 1.907 1.077 1.907 2.185V21L12 17.25 4.5 21V5.507c0-1.108.806-2.057 1.907-2.185a48.507 48.507 0 0111.186 0z',
  'currently-reading': 'M12 6.042A8.967 8.967 0 006 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 016 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 016-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0018 18a8.967 8.967 0 00-6 2.292m0-14.25v14.25',
  read: 'M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z',
  dnf: 'M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636',
};

export default function LibraryPage() {
  const [shelves, setShelves] = useState<Shelf[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState('');
  const [creating, setCreating] = useState(false);

  const loadShelves = useCallback(async () => {
    try {
      const res = await fetchShelves();
      if (res.ok) {
        const data = await res.json();
        setShelves(data.items ?? []);
      }
    } catch {
      /* empty — apiFetch redirects on 401 */
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadShelves();
  }, [loadShelves]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newName.trim() || creating) return;
    setCreating(true);
    try {
      const res = await createShelf(newName.trim());
      if (res.ok) {
        setNewName('');
        setShowCreate(false);
        loadShelves();
      }
    } catch {
      /* empty — apiFetch redirects on 401 */
    } finally {
      setCreating(false);
    }
  };

  return (
    <AuthGuard>
      <div className="mx-auto max-w-5xl px-4 py-8 sm:px-6">
        <div className="mb-8 flex items-end justify-between">
          <div>
            <h1 className="font-serif text-2xl font-bold tracking-tight text-stone-900
              dark:text-stone-100">
              My Library
            </h1>
            <p className="mt-1 text-sm text-stone-500 dark:text-stone-400">
              Organize your reading with shelves
            </p>
          </div>
          <button type="button"
            onClick={() => setShowCreate(!showCreate)}
            className="inline-flex items-center gap-1.5 rounded-md bg-stone-900 px-3.5 py-2
              text-sm font-medium text-white transition-colors hover:bg-stone-800
              dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200">
            <svg aria-hidden="true" className="h-4 w-4" viewBox="0 0 24 24" fill="none"
              strokeWidth={1.5} stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
            </svg>
            New shelf
          </button>
        </div>

        {showCreate && (
          <form onSubmit={handleCreate}
            className="mb-6 flex items-center gap-3 rounded-lg border border-stone-200
              bg-white p-4 dark:border-stone-700 dark:bg-stone-900">
            <input type="text" placeholder="Shelf name" value={newName}
              onChange={(e) => setNewName(e.target.value)}
              className="flex-1 rounded-md border border-stone-200 bg-white px-3 py-2 text-sm
                text-stone-900 placeholder:text-stone-400 focus:border-amber-500
                focus:outline-none focus:ring-1 focus:ring-amber-500
                dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100
                dark:placeholder:text-stone-500" />
            <button type="submit" disabled={creating || !newName.trim()}
              className="rounded-md bg-amber-700 px-4 py-2 text-sm font-medium text-white
                transition-colors hover:bg-amber-800 disabled:opacity-60
                dark:bg-amber-600 dark:hover:bg-amber-700">
              {creating ? 'Creating…' : 'Create'}
            </button>
            <button type="button" onClick={() => { setShowCreate(false); setNewName(''); }}
              className="rounded-md px-3 py-2 text-sm text-stone-500 hover:text-stone-700
                dark:text-stone-400 dark:hover:text-stone-200">
              Cancel
            </button>
          </form>
        )}

        {loading ? (
          <div className="flex items-center justify-center py-20">
            <div className="text-sm text-stone-400 dark:text-stone-500">Loading…</div>
          </div>
        ) : shelves.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-lg border border-dashed
            border-stone-300 py-20 dark:border-stone-600">
            <p className="text-sm font-medium text-stone-500 dark:text-stone-400">
              No shelves yet
            </p>
          </div>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {shelves.map((shelf) => {
              const iconPath = SHELF_ICONS[shelf.shelfType] ?? SHELF_ICONS['want-to-read'];
              return (
                <Link key={shelf.id} href={`/library/${shelf.id}`}
                  className="group flex items-center gap-4 rounded-lg border border-stone-200
                    bg-white p-5 shadow-sm transition-all duration-200
                    hover:border-amber-600/30 hover:shadow-md hover:-translate-y-0.5
                    dark:border-stone-700 dark:bg-stone-900
                    dark:hover:border-amber-500/30 dark:hover:shadow-stone-950/50">
                  <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-lg
                    bg-stone-100 transition-colors group-hover:bg-amber-50
                    dark:bg-stone-800 dark:group-hover:bg-amber-900/30">
                    <svg aria-hidden="true" className="h-6 w-6 text-stone-500
                      group-hover:text-amber-700 dark:text-stone-400
                      dark:group-hover:text-amber-400"
                      viewBox="0 0 24 24" fill="none" strokeWidth={1.5} stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" d={iconPath} />
                    </svg>
                  </div>
                  <div className="min-w-0 flex-1">
                    <h3 className="font-serif text-base font-semibold text-stone-900
                      group-hover:text-amber-800 transition-colors
                      dark:text-stone-100 dark:group-hover:text-amber-400">
                      {shelf.name}
                    </h3>
                    <p className="mt-0.5 text-sm text-stone-500 dark:text-stone-400">
                      {shelf.itemCount} {shelf.itemCount === 1 ? 'book' : 'books'}
                    </p>
                  </div>
                  <svg aria-hidden="true" className="h-4 w-4 text-stone-300 transition-colors
                    group-hover:text-amber-600 dark:text-stone-600 dark:group-hover:text-amber-400"
                    viewBox="0 0 24 24" fill="none" strokeWidth={2} stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round"
                      d="M8.25 4.5l7.5 7.5-7.5 7.5" />
                  </svg>
                </Link>
              );
            })}
          </div>
        )}
      </div>
    </AuthGuard>
  );
}
