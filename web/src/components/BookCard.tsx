'use client';

import Link from 'next/link';

interface BookCardProps {
  id: string;
  title: string;
  description?: string;
  originalLanguage?: string;
  originalPublicationYear?: number;
}

export default function BookCard({
  id,
  title,
  description,
  originalLanguage,
  originalPublicationYear,
}: BookCardProps) {
  return (
    <Link
      href={`/books/${id}`}
      className="group block rounded-lg border border-stone-200 bg-white p-5
        shadow-sm transition-all duration-200 hover:border-amber-600/30
        hover:shadow-md hover:-translate-y-0.5
        dark:border-stone-700 dark:bg-stone-900 dark:hover:border-amber-500/30
        dark:hover:shadow-stone-950/50"
    >
      <h3
        className="font-serif text-lg font-semibold leading-snug text-stone-900
          group-hover:text-amber-800 transition-colors line-clamp-2
          dark:text-stone-100 dark:group-hover:text-amber-400"
      >
        {title}
      </h3>

      {(originalPublicationYear || originalLanguage) && (
        <div className="mt-1.5 flex items-center gap-2 text-xs text-stone-400 dark:text-stone-500">
          {originalPublicationYear && <span>{originalPublicationYear}</span>}
          {originalPublicationYear && originalLanguage && (
            <span className="text-stone-300 dark:text-stone-600">&middot;</span>
          )}
          {originalLanguage && (
            <span className="uppercase tracking-wide">{originalLanguage}</span>
          )}
        </div>
      )}

      {description && (
        <p className="mt-3 text-sm leading-relaxed text-stone-500 line-clamp-3 dark:text-stone-400">
          {description}
        </p>
      )}

      <div
        className="mt-4 flex items-center gap-1 text-xs font-medium text-amber-700
          opacity-0 transition-opacity group-hover:opacity-100 dark:text-amber-400"
      >
        View details
        <svg
          aria-hidden="true"
          className="h-3 w-3"
          fill="none"
          viewBox="0 0 24 24"
          strokeWidth={2.5}
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3"
          />
        </svg>
      </div>
    </Link>
  );
}
