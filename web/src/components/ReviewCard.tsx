'use client';

import { useState } from 'react';
import Link from 'next/link';
import StarRating from './StarRating';

interface ReviewCardProps {
  id: string;
  authorId: string;
  authorName: string;
  rating: number;
  body: string;
  hasSpoilers?: boolean;
  helpfulCount?: number;
  commentCount?: number;
  createdAt: string;
}

export default function ReviewCard({
  authorId,
  authorName,
  rating,
  body,
  hasSpoilers = false,
  helpfulCount = 0,
  commentCount = 0,
  createdAt,
}: ReviewCardProps) {
  const [spoilerRevealed, setSpoilerRevealed] = useState(false);

  const formattedDate = new Date(createdAt).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });

  return (
    <div className="rounded-lg border border-stone-200 bg-white p-5
      dark:border-stone-700 dark:bg-stone-900">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-stone-100
            text-sm font-semibold text-stone-600 dark:bg-stone-800 dark:text-stone-300">
            {authorName.charAt(0).toUpperCase()}
          </div>
          <div>
            <Link
              href={`/profile/${authorId}`}
              className="text-sm font-medium text-stone-800 hover:text-amber-700
                dark:text-stone-200 dark:hover:text-amber-400"
            >
              {authorName}
            </Link>
            <p className="text-xs text-stone-400 dark:text-stone-500">{formattedDate}</p>
          </div>
        </div>
        <StarRating value={rating} readOnly size="sm" />
      </div>

      <div className="mt-3">
        {hasSpoilers && !spoilerRevealed ? (
          <div className="rounded-md bg-stone-50 px-4 py-3 dark:bg-stone-800/50">
            <p className="text-sm text-stone-500 dark:text-stone-400">
              This review contains spoilers.
            </p>
            <button
              type="button"
              onClick={() => setSpoilerRevealed(true)}
              className="mt-1 text-sm font-medium text-amber-700 hover:text-amber-800
                dark:text-amber-400 dark:hover:text-amber-300"
            >
              Reveal spoiler
            </button>
          </div>
        ) : (
          <p className="text-sm leading-relaxed text-stone-700 dark:text-stone-300">
            {body}
          </p>
        )}
      </div>

      <div className="mt-4 flex items-center gap-4 text-xs text-stone-400 dark:text-stone-500">
        <span className="inline-flex items-center gap-1">
          <svg aria-hidden="true" className="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none"
            strokeWidth={1.5} stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round"
              d="M6.633 10.5c.806 0 1.533-.446 2.031-1.08a9.041 9.041 0 012.861-2.4c.723-.384 1.35-.956 1.653-1.715a4.498 4.498 0 00.322-1.672V3.75a.75.75 0 01.75-.75A2.25 2.25 0 0116.5 5.25c0 .372-.052.733-.148 1.075a11.697 11.697 0 01-2.006 4.174M4.5 19.5h15M4.5 19.5a2.25 2.25 0 01-2.25-2.25v-1.5a2.25 2.25 0 012.25-2.25h1.009c.543 0 1.07-.19 1.487-.545l3.004-2.569a.75.75 0 011 0l3.004 2.569c.418.355.944.545 1.487.545H19.5a2.25 2.25 0 012.25 2.25v1.5a2.25 2.25 0 01-2.25 2.25" />
          </svg>
          {helpfulCount} helpful
        </span>
        <span className="inline-flex items-center gap-1">
          <svg aria-hidden="true" className="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none"
            strokeWidth={1.5} stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round"
              d="M12 20.25c4.97 0 9-3.694 9-8.25s-4.03-8.25-9-8.25S3 7.444 3 12c0 2.104.859 4.023 2.273 5.48.432.447.74 1.04.586 1.641a4.483 4.483 0 01-.923 1.785A5.969 5.969 0 006 21c1.282 0 2.47-.402 3.445-1.087.81.22 1.668.337 2.555.337z" />
          </svg>
          {commentCount} comments
        </span>
      </div>
    </div>
  );
}
