'use client';

import { useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import AuthGuard from '@/components/AuthGuard';
import StarRating from '@/components/StarRating';
import { submitReview } from '@/lib/api';

export default function WriteReviewPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();

  const [overallRating, setOverallRating] = useState(0);
  const [plotRating, setPlotRating] = useState(0);
  const [charactersRating, setCharactersRating] = useState(0);
  const [pacingRating, setPacingRating] = useState(0);
  const [proseRating, setProseRating] = useState(0);
  const [body, setBody] = useState('');
  const [hasSpoilers, setHasSpoilers] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (overallRating === 0) {
      setError('Please select an overall rating.');
      return;
    }
    setSubmitting(true);
    setError('');
    try {
      const payload: Record<string, unknown> = {
        workId: params.id,
        rating: overallRating,
        body,
        hasSpoilers,
      };
      if (plotRating > 0) payload.plotRating = plotRating;
      if (charactersRating > 0) payload.charactersRating = charactersRating;
      if (pacingRating > 0) payload.pacingRating = pacingRating;
      if (proseRating > 0) payload.proseRating = proseRating;

      const res = await submitReview(payload);
      if (res.ok) {
        router.push(`/books/${params.id}`);
      } else {
        const data = await res.json().catch(() => ({}));
        setError((data as Record<string, string>).detail ?? 'Failed to submit review.');
      }
    } catch {
      /* empty — apiFetch redirects on 401 */
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <AuthGuard>
      <div className="mx-auto max-w-2xl px-4 py-8 sm:px-6">
        <Link href={`/books/${params.id}`}
          className="mb-6 inline-flex items-center gap-1 text-sm text-stone-500
            transition-colors hover:text-stone-800
            dark:text-stone-400 dark:hover:text-stone-200">
          <svg aria-hidden="true" className="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none"
            strokeWidth={2} stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round"
              d="M10.5 19.5L3 12m0 0l7.5-7.5M3 12h18" />
          </svg>
          Back to book
        </Link>

        <h1 className="mb-6 font-serif text-2xl font-bold tracking-tight text-stone-900
          dark:text-stone-100">
          Write a Review
        </h1>

        {error && (
          <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm
            text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-6">
          <fieldset className="rounded-lg border border-stone-200 bg-white p-5
            dark:border-stone-700 dark:bg-stone-900">
            <legend className="mb-2 text-sm font-medium text-stone-700 dark:text-stone-300">
              Overall rating *
            </legend>
            <StarRating value={overallRating} onChange={setOverallRating} size="lg" />
          </fieldset>

          <div className="rounded-lg border border-stone-200 bg-white p-5
            dark:border-stone-700 dark:bg-stone-900">
            <p className="mb-3 text-sm font-medium text-stone-700 dark:text-stone-300">
              Detailed ratings (optional)
            </p>
            <div className="grid gap-4 sm:grid-cols-2">
              {[
                { label: 'Plot', value: plotRating, set: setPlotRating },
                { label: 'Characters', value: charactersRating, set: setCharactersRating },
                { label: 'Pacing', value: pacingRating, set: setPacingRating },
                { label: 'Prose', value: proseRating, set: setProseRating },
              ].map(({ label, value, set }) => (
                <div key={label} className="flex items-center justify-between">
                  <span className="text-sm text-stone-600 dark:text-stone-400">{label}</span>
                  <StarRating value={value} onChange={set} size="sm" />
                </div>
              ))}
            </div>
          </div>

          <div>
            <label htmlFor="review-body"
              className="mb-1.5 block text-sm font-medium text-stone-700 dark:text-stone-300">
              Your review
            </label>
            <textarea id="review-body" rows={6} value={body}
              onChange={(e) => setBody(e.target.value)}
              placeholder="What did you think of this book?"
              className="w-full rounded-md border border-stone-200 bg-white px-3 py-2
                text-sm text-stone-900 placeholder:text-stone-400
                focus:border-amber-500 focus:outline-none focus:ring-1 focus:ring-amber-500
                dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100
                dark:placeholder:text-stone-500" />
            {body.length > 0 && body.length < 50 && (
              <p className="mt-1 text-xs text-stone-400 dark:text-stone-500">
                {50 - body.length} more characters encouraged
              </p>
            )}
          </div>

          <label className="flex items-center gap-2">
            <input type="checkbox" checked={hasSpoilers}
              onChange={(e) => setHasSpoilers(e.target.checked)}
              className="h-4 w-4 rounded border-stone-300 text-amber-600
                focus:ring-amber-500 dark:border-stone-600 dark:bg-stone-800" />
            <span className="text-sm text-stone-700 dark:text-stone-300">
              Contains spoilers
            </span>
          </label>

          <button type="submit" disabled={submitting || overallRating === 0}
            className="w-full rounded-md bg-stone-900 px-4 py-2.5 text-sm font-medium text-white
              transition-colors hover:bg-stone-800 disabled:opacity-60
              dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200">
            {submitting ? 'Submitting…' : 'Submit review'}
          </button>
        </form>
      </div>
    </AuthGuard>
  );
}
