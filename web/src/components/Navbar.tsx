'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { clearToken, getUser, isAuthenticated, type User } from '@/lib/auth';
import { useTheme } from '@/lib/theme';

function ThemeToggle() {
  const { resolvedTheme, setTheme } = useTheme();

  return (
    <button
      type="button"
      onClick={() => setTheme(resolvedTheme === 'dark' ? 'light' : 'dark')}
      className="rounded-md p-1.5 text-stone-500 transition-colors hover:bg-stone-100
        hover:text-stone-800 dark:text-stone-400 dark:hover:bg-stone-800
        dark:hover:text-stone-200"
      aria-label={`Switch to ${resolvedTheme === 'dark' ? 'light' : 'dark'} mode`}
    >
      {resolvedTheme === 'dark' ? (
        /* Sun icon */
        <svg
          aria-hidden="true"
          className="h-4 w-4"
          viewBox="0 0 24 24"
          fill="none"
          strokeWidth={1.5}
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            d="M12 3v2.25m6.364.386l-1.591 1.591M21 12h-2.25m-.386 6.364l-1.591-1.591M12 18.75V21m-4.773-4.227l-1.591 1.591M5.25 12H3m4.227-4.773L5.636 5.636M15.75 12a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0z"
          />
        </svg>
      ) : (
        /* Moon icon */
        <svg
          aria-hidden="true"
          className="h-4 w-4"
          viewBox="0 0 24 24"
          fill="none"
          strokeWidth={1.5}
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            d="M21.752 15.002A9.718 9.718 0 0118 15.75c-5.385 0-9.75-4.365-9.75-9.75 0-1.33.266-2.597.748-3.752A9.753 9.753 0 003 11.25C3 16.635 7.365 21 12.75 21a9.753 9.753 0 009.002-5.998z"
          />
        </svg>
      )}
    </button>
  );
}

export default function Navbar() {
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [authed, setAuthed] = useState(false);

  useEffect(() => {
    setAuthed(isAuthenticated());
    setUser(getUser());
  }, []);

  const handleLogout = () => {
    clearToken();
    setAuthed(false);
    setUser(null);
    router.push('/login');
  };

  return (
    <nav className="fixed inset-x-0 top-0 z-50 border-b border-stone-200/80 bg-white/90
      backdrop-blur-sm dark:border-stone-800 dark:bg-stone-950/90">
      <div className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4 sm:px-6">
        {/* Logo */}
        <Link
          href="/books"
          className="flex items-center gap-2 text-stone-900 transition-colors
            hover:text-amber-800 dark:text-stone-100 dark:hover:text-amber-400"
        >
          <svg
            aria-hidden="true"
            className="h-5 w-5"
            viewBox="0 0 24 24"
            fill="none"
            strokeWidth={1.5}
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M12 6.042A8.967 8.967 0 006 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 016 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 016-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0018 18a8.967 8.967 0 00-6 2.292m0-14.25v14.25"
            />
          </svg>
          <span className="font-serif text-lg font-bold tracking-tight">Verso</span>
        </Link>

        {/* Nav links + theme toggle + auth */}
        <div className="flex items-center gap-4">
          {authed && (
            <>
              {[
                { href: '/feed', label: 'Feed' },
                { href: '/books', label: 'Books' },
                { href: '/library', label: 'Library' },
                { href: '/discover', label: 'Discover' },
                { href: '/profile', label: 'Profile' },
              ].map(({ href, label }) => (
                <Link key={href} href={href}
                  className="text-sm font-medium text-stone-600 transition-colors
                    hover:text-stone-900 dark:text-stone-400 dark:hover:text-stone-100">
                  {label}
                </Link>
              ))}
            </>
          )}

          <ThemeToggle />

          {authed ? (
            <div className="flex items-center gap-3">
              {user?.displayName && (
                <span className="hidden text-sm text-stone-500 dark:text-stone-400 sm:inline">
                  {user.displayName}
                </span>
              )}
              <button
                type="button"
                onClick={handleLogout}
                className="rounded-md border border-stone-200 px-3 py-1.5 text-xs font-medium
                  text-stone-600 transition-colors hover:border-stone-300 hover:text-stone-900
                  dark:border-stone-700 dark:text-stone-400 dark:hover:border-stone-600
                  dark:hover:text-stone-100"
              >
                Log out
              </button>
            </div>
          ) : (
            <Link
              href="/login"
              className="rounded-md bg-stone-900 px-3.5 py-1.5 text-xs font-medium
                text-white transition-colors hover:bg-stone-800
                dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200"
            >
              Sign in
            </Link>
          )}
        </div>
      </div>
    </nav>
  );
}
