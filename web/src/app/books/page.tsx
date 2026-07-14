'use client';

import { useCallback, useEffect, useState } from 'react';
import AuthGuard from '@/components/AuthGuard';
import SearchBar from '@/components/SearchBar';
import BookCard from '@/components/BookCard';
import { apiFetch } from '@/lib/api';

interface Book {
  id: string;
  title: string;
  description?: string;
  originalLanguage?: string;
  originalPublicationYear?: number;
}

interface SearchResult {
  workId: string;
  title: string;
  description?: string;
  score: number;
}

export default function BooksPage() {
  const [books, setBooks] = useState<Book[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchMode, setSearchMode] = useState(false);

  // Fetch all books on mount
  useEffect(() => {
    (async () => {
      try {
        const res = await apiFetch('/api/v1/books');
        if (res.ok) {
          const data = await res.json();
          setBooks(data.items ?? []);
        }
      } catch {
        // apiFetch handles 401 redirect
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const handleSearch = useCallback(async (query: string) => {
    if (!query) {
      setSearchMode(false);
      setLoading(true);
      try {
        const res = await apiFetch('/api/v1/books');
        if (res.ok) {
          const data = await res.json();
          setBooks(data.items ?? []);
        }
      } catch {
        // handled
      } finally {
        setLoading(false);
      }
      return;
    }

    setSearchMode(true);
    setLoading(true);
    try {
      const res = await apiFetch(
        `/api/v1/search?q=${encodeURIComponent(query)}&type=work`,
      );
      if (res.ok) {
        const data = await res.json();
        const results: SearchResult[] = data.results ?? [];
        setBooks(
          results.map((r) => ({
            id: r.workId,
            title: r.title,
            description: r.description,
          })),
        );
      }
    } catch {
      // handled
    } finally {
      setLoading(false);
    }
  }, []);

  return (
    <AuthGuard>
      <div className="mx-auto max-w-5xl px-4 py-8 sm:px-6">
        {/* Page header */}
        <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h1 className="font-serif text-2xl font-bold tracking-tight text-stone-900">
              {searchMode ? 'Search results' : 'Library'}
            </h1>
            <p className="mt-1 text-sm text-stone-500">
              {searchMode
                ? `${books.length} result${books.length !== 1 ? 's' : ''} found`
                : 'Browse and discover your next great read.'}
            </p>
          </div>
          <SearchBar onSearch={handleSearch} />
        </div>

        {/* Book grid */}
        {loading ? (
          <div className="flex items-center justify-center py-20">
            <div className="text-sm text-stone-400">Loading\u2026</div>
          </div>
        ) : books.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-stone-300 py-20">
            <svg
              aria-hidden="true"
              className="mb-3 h-8 w-8 text-stone-300"
              viewBox="0 0 24 24"
              fill="none"
              strokeWidth={1.5}
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M12 6.042A8.967 8.967 0 006 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 016 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 016-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0018 18a8.967 8.967 0 00-6 2.292m0-14.25v14.25"
              />
            </svg>
            <p className="text-sm font-medium text-stone-500">No books found</p>
            <p className="mt-1 text-xs text-stone-400">
              {searchMode ? 'Try a different search term.' : 'The library is empty.'}
            </p>
          </div>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {books.map((book) => (
              <BookCard
                key={book.id}
                id={book.id}
                title={book.title}
                description={book.description}
                originalLanguage={book.originalLanguage}
                originalPublicationYear={book.originalPublicationYear}
              />
            ))}
          </div>
        )}
      </div>
    </AuthGuard>
  );
}
