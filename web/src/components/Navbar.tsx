'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { clearToken, getUser, isAuthenticated, type User } from '@/lib/auth';

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
    <nav className="fixed inset-x-0 top-0 z-50 border-b border-stone-200/80 bg-white/90 backdrop-blur-sm">
      <div className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4 sm:px-6">
        {/* Logo */}
        <Link
          href="/books"
          className="flex items-center gap-2 text-stone-900 transition-colors hover:text-amber-800"
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

        {/* Nav links + auth */}
        <div className="flex items-center gap-4">
          {authed && (
            <Link
              href="/books"
              className="text-sm font-medium text-stone-600 transition-colors hover:text-stone-900"
            >
              Books
            </Link>
          )}

          {authed ? (
            <div className="flex items-center gap-3">
              {user?.displayName && (
                <span className="hidden text-sm text-stone-500 sm:inline">
                  {user.displayName}
                </span>
              )}
              <button
                type="button"
                onClick={handleLogout}
                className="rounded-md border border-stone-200 px-3 py-1.5 text-xs font-medium
                  text-stone-600 transition-colors hover:border-stone-300 hover:text-stone-900"
              >
                Log out
              </button>
            </div>
          ) : (
            <Link
              href="/login"
              className="rounded-md bg-stone-900 px-3.5 py-1.5 text-xs font-medium
                text-white transition-colors hover:bg-stone-800"
            >
              Sign in
            </Link>
          )}
        </div>
      </div>
    </nav>
  );
}
