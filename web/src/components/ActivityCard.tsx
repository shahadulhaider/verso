'use client';

import Link from 'next/link';

interface ActivityCardProps {
  actorId: string;
  actorName: string;
  verb: string;
  objectType: string;
  objectId: string;
  objectTitle: string;
  extra?: Record<string, unknown>;
  occurredAt: string;
}

function objectLink(objectType: string, objectId: string): string {
  switch (objectType) {
    case 'work':
    case 'book':
      return `/books/${objectId}`;
    case 'review':
      return `/books/${objectId}`;
    case 'profile':
    case 'user':
      return `/profile/${objectId}`;
    default:
      return '#';
  }
}

function verbLabel(verb: string): string {
  switch (verb) {
    case 'reviewed':
      return 'wrote a review for';
    case 'rated':
      return 'rated';
    case 'shelved':
      return 'added to shelf';
    case 'followed':
      return 'started following';
    default:
      return verb;
  }
}

export default function ActivityCard({
  actorId,
  actorName,
  verb,
  objectType,
  objectId,
  objectTitle,
  extra,
  occurredAt,
}: ActivityCardProps) {
  const timeAgo = formatRelativeTime(occurredAt);

  return (
    <div className="rounded-lg border border-stone-200 bg-white p-4
      dark:border-stone-700 dark:bg-stone-900">
      <div className="flex items-start gap-3">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full
          bg-stone-100 text-sm font-semibold text-stone-600
          dark:bg-stone-800 dark:text-stone-300">
          {actorName.charAt(0).toUpperCase()}
        </div>
        <div className="min-w-0 flex-1">
          <p className="text-sm text-stone-700 dark:text-stone-300">
            <Link
              href={`/profile/${actorId}`}
              className="font-medium text-stone-900 hover:text-amber-700
                dark:text-stone-100 dark:hover:text-amber-400"
            >
              {actorName}
            </Link>{' '}
            {verbLabel(verb)}{' '}
            <Link
              href={objectLink(objectType, objectId)}
              className="font-medium text-stone-900 hover:text-amber-700
                dark:text-stone-100 dark:hover:text-amber-400"
            >
              {objectTitle}
            </Link>
          </p>
          {extra?.rating != null && (
            <div className="mt-1 flex items-center gap-1 text-xs text-amber-600 dark:text-amber-400">
              {'★'.repeat(Math.round(extra.rating as number))}
            </div>
          )}
          <p className="mt-1 text-xs text-stone-400 dark:text-stone-500">{timeAgo}</p>
        </div>
      </div>
    </div>
  );
}

function formatRelativeTime(dateStr: string): string {
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diffMs = now - then;
  const diffMins = Math.floor(diffMs / 60000);

  if (diffMins < 1) return 'just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  const diffHours = Math.floor(diffMins / 60);
  if (diffHours < 24) return `${diffHours}h ago`;
  const diffDays = Math.floor(diffHours / 24);
  if (diffDays < 30) return `${diffDays}d ago`;
  return new Date(dateStr).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
  });
}
