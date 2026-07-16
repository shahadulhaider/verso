'use client';

import FollowButton from './FollowButton';

interface ProfileHeaderProps {
  userId: string;
  displayName: string;
  bio?: string;
  location?: string;
  website?: string;
  followerCount: number;
  followingCount: number;
  shelfCount: number;
  isOwnProfile: boolean;
  isFollowing?: boolean;
  isAuthenticated?: boolean;
}

export default function ProfileHeader({
  userId,
  displayName,
  bio,
  location,
  website,
  followerCount,
  followingCount,
  shelfCount,
  isOwnProfile,
  isFollowing = false,
  isAuthenticated = false,
}: ProfileHeaderProps) {
  return (
    <div className="rounded-lg border border-stone-200 bg-white p-6
      dark:border-stone-700 dark:bg-stone-900">
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-center gap-4">
          <div className="flex h-16 w-16 items-center justify-center rounded-full
            bg-gradient-to-br from-amber-100 to-amber-200 text-2xl font-bold
            text-amber-800 dark:from-amber-900/40 dark:to-amber-800/40
            dark:text-amber-400">
            {displayName.charAt(0).toUpperCase()}
          </div>
          <div>
            <h1 className="font-serif text-2xl font-bold tracking-tight text-stone-900
              dark:text-stone-100">
              {displayName}
            </h1>
            {location && (
              <p className="mt-0.5 flex items-center gap-1 text-sm text-stone-500
                dark:text-stone-400">
                <svg aria-hidden="true" className="h-3.5 w-3.5" viewBox="0 0 24 24"
                  fill="none" strokeWidth={1.5} stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round"
                    d="M15 10.5a3 3 0 11-6 0 3 3 0 016 0z" />
                  <path strokeLinecap="round" strokeLinejoin="round"
                    d="M19.5 10.5c0 7.142-7.5 11.25-7.5 11.25S4.5 17.642 4.5 10.5a7.5 7.5 0 1115 0z" />
                </svg>
                {location}
              </p>
            )}
          </div>
        </div>

        {!isOwnProfile && isAuthenticated && (
          <FollowButton userId={userId} initialFollowing={isFollowing} />
        )}
      </div>

      {bio && (
        <p className="mt-4 text-sm leading-relaxed text-stone-600 dark:text-stone-400">
          {bio}
        </p>
      )}

      {website && (
        <a
          href={website}
          target="_blank"
          rel="noopener noreferrer"
          className="mt-2 inline-flex items-center gap-1 text-sm text-amber-700 hover:text-amber-800
            dark:text-amber-400 dark:hover:text-amber-300"
        >
          <svg aria-hidden="true" className="h-3.5 w-3.5" viewBox="0 0 24 24"
            fill="none" strokeWidth={1.5} stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round"
              d="M13.19 8.688a4.5 4.5 0 011.242 7.244l-4.5 4.5a4.5 4.5 0 01-6.364-6.364l1.757-1.757m13.35-.622l1.757-1.757a4.5 4.5 0 00-6.364-6.364l-4.5 4.5a4.5 4.5 0 001.242 7.244" />
          </svg>
          {website.replace(/^https?:\/\//, '')}
        </a>
      )}

      <div className="mt-5 flex items-center gap-6 border-t border-stone-100 pt-4
        dark:border-stone-800">
        {[
          { label: 'Followers', value: followerCount },
          { label: 'Following', value: followingCount },
          { label: 'Shelves', value: shelfCount },
        ].map(({ label, value }) => (
          <div key={label} className="text-center">
            <p className="text-lg font-semibold text-stone-900 dark:text-stone-100">{value}</p>
            <p className="text-xs text-stone-500 dark:text-stone-400">{label}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
