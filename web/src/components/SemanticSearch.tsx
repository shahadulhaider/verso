'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { semanticSearch } from '@/lib/api';
import BookCard from './BookCard';

interface SemanticResult {
  workId: string;
  title: string;
  description?: string;
  score: number;
}

interface SemanticSearchProps {
  onResultsChange?: (count: number) => void;
}

export default function SemanticSearch({ onResultsChange }: SemanticSearchProps) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SemanticResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [searched, setSearched] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const doSearch = useCallback(async (q: string) => {
    if (!q.trim()) {
      setResults([]);
      setSearched(false);
      onResultsChange?.(0);
      return;
    }

    setLoading(true);
    setError('');
    setSearched(true);
    try {
      const res = await semanticSearch(q);
      if (res.ok) {
        const data = await res.json();
        const items: SemanticResult[] = data.results ?? [];
        setResults(items);
        onResultsChange?.(items.length);
      } else if (res.status === 503) {
        setError('AI search is temporarily unavailable. Try again later.');
        setResults([]);
      } else {
        setError('Search failed. Please try again.');
        setResults([]);
      }
    } catch {
      setError('Search failed. Please try again.');
      setResults([]);
    } finally {
      setLoading(false);
    }
  }, [onResultsChange]);

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => doSearch(query), 500);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [query, doSearch]);

  return (
    <div>
      <div className="relative">
        <svg aria-hidden="true" className="absolute left-3.5 top-1/2 h-5 w-5 -translate-y-1/2
          text-stone-400 dark:text-stone-500" viewBox="0 0 24 24" fill="none"
          strokeWidth={1.5} stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round"
            d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.455 2.456L21.75 6l-1.036.259a3.375 3.375 0 00-2.455 2.456z" />
        </svg>
        <input
          type="text"
          placeholder="Find books like\u2026"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          className="w-full rounded-lg border border-stone-200 bg-white py-3 pl-11 pr-4
            text-sm text-stone-900 placeholder:text-stone-400
            focus:border-amber-500 focus:outline-none focus:ring-1 focus:ring-amber-500
            dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100
            dark:placeholder:text-stone-500 dark:focus:border-amber-500
            dark:focus:ring-amber-500"
        />
        {loading && (
          <div className="absolute right-3.5 top-1/2 -translate-y-1/2">
            <div className="h-4 w-4 animate-spin rounded-full border-2 border-stone-300
              border-t-amber-600 dark:border-stone-600 dark:border-t-amber-400" />
          </div>
        )}
      </div>

      {error && (
        <div className="mt-4 rounded-md border border-amber-200 bg-amber-50 px-4 py-3
          text-sm text-amber-800 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-300">
          {error}
        </div>
      )}

      {searched && !loading && !error && results.length === 0 && (
        <div className="mt-8 flex flex-col items-center justify-center py-12">
          <p className="text-sm font-medium text-stone-500 dark:text-stone-400">
            No similar books found
          </p>
          <p className="mt-1 text-xs text-stone-400 dark:text-stone-500">
            Try describing the kind of book you&apos;re looking for
          </p>
        </div>
      )}

      {results.length > 0 && (
        <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {results.map((r) => (
            <div key={r.workId} className="relative">
              <BookCard
                id={r.workId}
                title={r.title}
                description={r.description}
              />
              <span className="absolute right-2 top-2 rounded-full bg-amber-100 px-2 py-0.5
                text-[10px] font-semibold text-amber-800 dark:bg-amber-900/50
                dark:text-amber-400">
                {Math.round(r.score * 100)}% match
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
