'use client';

import SemanticSearch from '@/components/SemanticSearch';

export default function DiscoverPage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-8 sm:px-6">
      <div className="mb-8">
        <h1 className="font-serif text-2xl font-bold tracking-tight text-stone-900
          dark:text-stone-100">
          Discover
        </h1>
        <p className="mt-1 text-sm text-stone-500 dark:text-stone-400">
          Find your next read with AI-powered recommendations
        </p>
      </div>
      <SemanticSearch />
    </div>
  );
}
