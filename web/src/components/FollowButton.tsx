'use client';

import { useCallback, useState } from 'react';
import { followUser, unfollowUser } from '@/lib/api';

interface FollowButtonProps {
  userId: string;
  initialFollowing?: boolean;
  onToggle?: (following: boolean) => void;
}

export default function FollowButton({
  userId,
  initialFollowing = false,
  onToggle,
}: FollowButtonProps) {
  const [following, setFollowing] = useState(initialFollowing);
  const [busy, setBusy] = useState(false);

  const handleToggle = useCallback(async () => {
    if (busy) return;
    setBusy(true);
    try {
      const res = following
        ? await unfollowUser(userId)
        : await followUser(userId);
      if (res.ok || res.status === 204) {
        const next = !following;
        setFollowing(next);
        onToggle?.(next);
      }
    } catch {
      /* empty — apiFetch redirects on 401 */
    } finally {
      setBusy(false);
    }
  }, [busy, following, userId, onToggle]);

  return (
    <button
      type="button"
      onClick={handleToggle}
      disabled={busy}
      className={`rounded-md px-4 py-1.5 text-sm font-medium transition-colors disabled:opacity-60 ${
        following
          ? 'border border-stone-200 bg-white text-stone-700 hover:border-red-300 hover:text-red-600 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-300 dark:hover:border-red-800 dark:hover:text-red-400'
          : 'bg-stone-900 text-white hover:bg-stone-800 dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200'
      }`}
    >
      {busy ? '\u2026' : following ? 'Following' : 'Follow'}
    </button>
  );
}
