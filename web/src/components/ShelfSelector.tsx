'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { addToShelf, fetchShelves } from '@/lib/api';

interface Shelf {
  id: string;
  name: string;
  shelfType: string;
}

interface ShelfSelectorProps {
  workId: string;
  currentShelfId?: string;
  onAdded?: (shelfId: string) => void;
}

export default function ShelfSelector({ workId, currentShelfId, onAdded }: ShelfSelectorProps) {
  const [open, setOpen] = useState(false);
  const [shelves, setShelves] = useState<Shelf[]>([]);
  const [loading, setLoading] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    setLoading(true);
    fetchShelves()
      .then(async (res) => {
        if (res.ok) {
          const data = await res.json();
          setShelves(data.items ?? []);
        }
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [open]);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleSelect = useCallback(
    async (shelfId: string) => {
      try {
        const res = await addToShelf(shelfId, workId);
        if (res.ok) {
          onAdded?.(shelfId);
          setOpen(false);
        }
      } catch {
        /* empty — apiFetch redirects on 401 */
      }
    },
    [workId, onAdded],
  );

  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="inline-flex items-center gap-2 rounded-md border border-stone-200 bg-white
          px-3.5 py-2 text-sm font-medium text-stone-700 transition-colors
          hover:border-stone-300 hover:bg-stone-50
          dark:border-stone-700 dark:bg-stone-800 dark:text-stone-300
          dark:hover:border-stone-600 dark:hover:bg-stone-700"
      >
        <svg aria-hidden="true" className="h-4 w-4" viewBox="0 0 24 24" fill="none"
          strokeWidth={1.5} stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round"
            d="M12 4.5v15m7.5-7.5h-15" />
        </svg>
        {currentShelfId ? 'Move to\u2026' : 'Add to shelf'}
      </button>

      {open && (
        <div className="absolute right-0 z-20 mt-1 w-56 rounded-lg border border-stone-200
          bg-white py-1 shadow-lg dark:border-stone-700 dark:bg-stone-800">
          {loading ? (
            <p className="px-4 py-2 text-sm text-stone-400 dark:text-stone-500">Loading\u2026</p>
          ) : shelves.length === 0 ? (
            <p className="px-4 py-2 text-sm text-stone-400 dark:text-stone-500">No shelves found</p>
          ) : (
            shelves.map((shelf) => (
              <button
                key={shelf.id}
                type="button"
                disabled={shelf.id === currentShelfId}
                onClick={() => handleSelect(shelf.id)}
                className="flex w-full items-center gap-2 px-4 py-2 text-left text-sm
                  text-stone-700 transition-colors hover:bg-stone-50
                  disabled:cursor-not-allowed disabled:opacity-50
                  dark:text-stone-300 dark:hover:bg-stone-700/50"
              >
                {shelf.id === currentShelfId && (
                  <svg aria-hidden="true" className="h-3.5 w-3.5 text-amber-600 dark:text-amber-400"
                    viewBox="0 0 24 24" fill="none" strokeWidth={2} stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12.75l6 6 9-13.5" />
                  </svg>
                )}
                <span className={shelf.id === currentShelfId ? '' : 'pl-5.5'}>{shelf.name}</span>
              </button>
            ))
          )}
        </div>
      )}
    </div>
  );
}
