'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import ProfileHeader from '@/components/ProfileHeader';
import BookCard from '@/components/BookCard';
import { fetchProfile } from '@/lib/api';
import { isAuthenticated, getUser } from '@/lib/auth';

interface PublicProfile {
  userId: string;
  displayName: string;
  bio?: string;
  location?: string;
  website?: string;
  followerCount: number;
  followingCount: number;
  shelfCount: number;
  isFollowing: boolean;
  publicShelves?: {
    id: string;
    name: string;
    items?: {
      workId: string;
      title: string;
      description?: string;
    }[];
  }[];
}

export default function PublicProfilePage() {
  const params = useParams<{ userId: string }>();
  const [profile, setProfile] = useState<PublicProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const authed = typeof window !== 'undefined' && isAuthenticated();
  const currentUser = typeof window !== 'undefined' ? getUser() : null;
  const isOwnProfile = currentUser?.id === params.userId;

  useEffect(() => {
    if (!params.userId) return;
    (async () => {
      try {
        const res = await fetchProfile(params.userId);
        if (res.ok) {
          setProfile(await res.json());
        } else if (res.status === 404) {
          setError('User not found.');
        } else {
          setError('Failed to load profile.');
        }
      } catch {
        /* empty — apiFetch redirects on 401 */
      } finally {
        setLoading(false);
      }
    })();
  }, [params.userId]);

  return (
    <div className="mx-auto max-w-3xl px-4 py-8 sm:px-6">
      <Link href="/books"
        className="mb-6 inline-flex items-center gap-1 text-sm text-stone-500
          transition-colors hover:text-stone-800
          dark:text-stone-400 dark:hover:text-stone-200">
        <svg aria-hidden="true" className="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none"
          strokeWidth={2} stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round"
            d="M10.5 19.5L3 12m0 0l7.5-7.5M3 12h18" />
        </svg>
        Back
      </Link>

      {loading ? (
        <div className="flex items-center justify-center py-20">
          <div className="text-sm text-stone-400 dark:text-stone-500">Loading…</div>
        </div>
      ) : error ? (
        <div className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700
          dark:border-red-800 dark:bg-red-950 dark:text-red-400">
          {error}
        </div>
      ) : profile ? (
        <>
          <ProfileHeader
            userId={profile.userId}
            displayName={profile.displayName}
            bio={profile.bio}
            location={profile.location}
            website={profile.website}
            followerCount={profile.followerCount}
            followingCount={profile.followingCount}
            shelfCount={profile.shelfCount}
            isOwnProfile={isOwnProfile}
            isFollowing={profile.isFollowing}
            isAuthenticated={authed}
          />

          {profile.publicShelves && profile.publicShelves.length > 0 && (
            <div className="mt-8 space-y-6">
              {profile.publicShelves.map((shelf) => (
                <div key={shelf.id}>
                  <h2 className="mb-3 text-xs font-semibold uppercase tracking-wider text-stone-400
                    dark:text-stone-500">
                    {shelf.name}
                  </h2>
                  {shelf.items && shelf.items.length > 0 ? (
                    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                      {shelf.items.map((item) => (
                        <BookCard key={item.workId} id={item.workId} title={item.title}
                          description={item.description} />
                      ))}
                    </div>
                  ) : (
                    <p className="text-sm text-stone-400 dark:text-stone-500">
                      No books on this shelf yet
                    </p>
                  )}
                </div>
              ))}
            </div>
          )}
        </>
      ) : null}
    </div>
  );
}
