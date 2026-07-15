'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import AuthGuard from '@/components/AuthGuard';
import { apiFetch } from '@/lib/api';

interface Edition {
  id: string;
  isbn?: string;
  publisher?: string;
  publicationYear?: number;
  format?: string;
  language?: string;
  pageCount?: number;
}

interface BookDetail {
  id: string;
  title: string;
  description?: string;
  originalLanguage?: string;
  originalPublicationYear?: number;
  editions?: Edition[];
}

export default function BookDetailPage() {
  const params = useParams<{ id: string }>();
  const [book, setBook] = useState<BookDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!params.id) return;

    (async () => {
      try {
        const res = await apiFetch(`/api/v1/books/${params.id}`);
        if (res.ok) {
          setBook(await res.json());
        } else if (res.status === 404) {
          setError('Book not found.');
        } else {
          setError('Failed to load book.');
        }
      } catch {
        // 401 handled by apiFetch
      } finally {
        setLoading(false);
      }
    })();
  }, [params.id]);

  return (
    <AuthGuard>
      <div className="mx-auto max-w-3xl px-4 py-8 sm:px-6">
        {/* Back link */}
        <Link
          href="/books"
          className="mb-6 inline-flex items-center gap-1 text-sm text-stone-500 transition-colors hover:text-stone-800
            dark:text-stone-400 dark:hover:text-stone-200"
        >
          <svg
            aria-hidden="true"
            className="h-3.5 w-3.5"
            viewBox="0 0 24 24"
            fill="none"
            strokeWidth={2}
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M10.5 19.5L3 12m0 0l7.5-7.5M3 12h18"
            />
          </svg>
          Back to library
        </Link>

        {loading ? (
          <div className="flex items-center justify-center py-20">
            <div className="text-sm text-stone-400 dark:text-stone-500">Loading\u2026</div>
          </div>
        ) : error ? (
          <div className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700
            dark:border-red-800 dark:bg-red-950 dark:text-red-400">
            {error}
          </div>
        ) : book ? (
          <article>
            {/* Title */}
            <h1 className="font-serif text-3xl font-bold leading-tight tracking-tight text-stone-900 sm:text-4xl
              dark:text-stone-100">
              {book.title}
            </h1>

            {/* Meta row */}
            {(book.originalPublicationYear || book.originalLanguage) && (
              <div className="mt-3 flex items-center gap-3 text-sm text-stone-500 dark:text-stone-400">
                {book.originalPublicationYear && (
                  <span>Published {book.originalPublicationYear}</span>
                )}
                {book.originalPublicationYear && book.originalLanguage && (
                  <span className="text-stone-300 dark:text-stone-600">&middot;</span>
                )}
                {book.originalLanguage && (
                  <span className="rounded bg-stone-100 px-2 py-0.5 text-xs font-medium uppercase tracking-wide text-stone-600
                    dark:bg-stone-800 dark:text-stone-400">
                    {book.originalLanguage}
                  </span>
                )}
              </div>
            )}

            {/* Description */}
            {book.description && (
              <div className="mt-6 rounded-lg border border-stone-200 bg-white p-5
                dark:border-stone-700 dark:bg-stone-900">
                <h2 className="mb-2 text-xs font-semibold uppercase tracking-wider text-stone-400
                  dark:text-stone-500">
                  Description
                </h2>
                <p className="text-sm leading-relaxed text-stone-700 dark:text-stone-300">
                  {book.description}
                </p>
              </div>
            )}

            {/* Editions */}
            {book.editions && book.editions.length > 0 && (
              <div className="mt-6">
                <h2 className="mb-3 text-xs font-semibold uppercase tracking-wider text-stone-400
                  dark:text-stone-500">
                  Editions
                </h2>
                <div className="space-y-3">
                  {book.editions.map((edition) => (
                    <div
                      key={edition.id}
                      className="rounded-lg border border-stone-200 bg-white p-4
                        dark:border-stone-700 dark:bg-stone-900"
                    >
                      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm">
                        {edition.publisher && (
                          <span className="font-medium text-stone-800 dark:text-stone-200">
                            {edition.publisher}
                          </span>
                        )}
                        {edition.publicationYear && (
                          <span className="text-stone-500 dark:text-stone-400">
                            {edition.publicationYear}
                          </span>
                        )}
                        {edition.format && (
                          <span className="rounded bg-stone-100 px-2 py-0.5 text-xs text-stone-600
                            dark:bg-stone-800 dark:text-stone-400">
                            {edition.format}
                          </span>
                        )}
                        {edition.language && (
                          <span className="text-xs uppercase tracking-wide text-stone-400
                            dark:text-stone-500">
                            {edition.language}
                          </span>
                        )}
                      </div>
                      <div className="mt-1.5 flex flex-wrap gap-x-4 text-xs text-stone-400 dark:text-stone-500">
                        {edition.isbn && <span>ISBN {edition.isbn}</span>}
                        {edition.pageCount && (
                          <span>{edition.pageCount} pages</span>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </article>
        ) : null}
      </div>
    </AuthGuard>
  );
}
