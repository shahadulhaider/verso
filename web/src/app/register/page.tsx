'use client';

import { useState, type FormEvent } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { apiFetch } from '@/lib/api';
import { setToken, setUser } from '@/lib/auth';

export default function RegisterPage() {
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const res = await apiFetch('/api/v1/auth/register', {
        method: 'POST',
        body: JSON.stringify({ email, password, displayName }),
      });

      if (!res.ok) {
        const body = await res.json().catch(() => null);
        setError(body?.detail ?? body?.message ?? 'Registration failed');
        return;
      }

      const data = await res.json();
      setToken(data.accessToken);
      setUser(data.user);
      router.push('/books');
    } catch (err) {
      if (err instanceof Error && err.message !== 'Unauthorized') {
        setError('Something went wrong. Please try again.');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-[calc(100vh-3.5rem)] items-center justify-center px-4">
      <div className="w-full max-w-sm">
        {/* Header */}
        <div className="mb-8 text-center">
          <h1 className="font-serif text-3xl font-bold tracking-tight text-stone-900
            dark:text-stone-100">
            Join Verso
          </h1>
          <p className="mt-2 text-sm text-stone-500 dark:text-stone-400">
            Start tracking and discovering your next great read.
          </p>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700
              dark:border-red-800 dark:bg-red-950 dark:text-red-400">
              {error}
            </div>
          )}

          <div>
            <label htmlFor="displayName" className="block text-sm font-medium text-stone-700
              dark:text-stone-300">
              Display name
            </label>
            <input
              id="displayName"
              type="text"
              required
              autoComplete="name"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              className="mt-1 block w-full rounded-md border border-stone-300 bg-white px-3 py-2
                text-sm text-stone-900 shadow-sm transition-colors placeholder:text-stone-400
                focus:border-amber-600/40 focus:outline-none focus:ring-2 focus:ring-amber-600/10
                dark:border-stone-600 dark:bg-stone-900 dark:text-stone-100
                dark:placeholder:text-stone-500"
              placeholder="Ada Lovelace"
            />
          </div>

          <div>
            <label htmlFor="email" className="block text-sm font-medium text-stone-700
              dark:text-stone-300">
              Email
            </label>
            <input
              id="email"
              type="email"
              required
              autoComplete="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="mt-1 block w-full rounded-md border border-stone-300 bg-white px-3 py-2
                text-sm text-stone-900 shadow-sm transition-colors placeholder:text-stone-400
                focus:border-amber-600/40 focus:outline-none focus:ring-2 focus:ring-amber-600/10
                dark:border-stone-600 dark:bg-stone-900 dark:text-stone-100
                dark:placeholder:text-stone-500"
              placeholder="you@example.com"
            />
          </div>

          <div>
            <label htmlFor="password" className="block text-sm font-medium text-stone-700
              dark:text-stone-300">
              Password
            </label>
            <input
              id="password"
              type="password"
              required
              autoComplete="new-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="mt-1 block w-full rounded-md border border-stone-300 bg-white px-3 py-2
                text-sm text-stone-900 shadow-sm transition-colors placeholder:text-stone-400
                focus:border-amber-600/40 focus:outline-none focus:ring-2 focus:ring-amber-600/10
                dark:border-stone-600 dark:bg-stone-900 dark:text-stone-100
                dark:placeholder:text-stone-500"
              placeholder="\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022"
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-md bg-stone-900 px-4 py-2.5 text-sm font-medium text-white
              shadow-sm transition-colors hover:bg-stone-800 disabled:cursor-not-allowed
              disabled:opacity-50 dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200"
          >
            {loading ? 'Creating account\u2026' : 'Create account'}
          </button>
        </form>

        {/* Link to login */}
        <p className="mt-6 text-center text-sm text-stone-500 dark:text-stone-400">
          Already have an account?{' '}
          <Link
            href="/login"
            className="font-medium text-amber-700 transition-colors hover:text-amber-800
              dark:text-amber-400 dark:hover:text-amber-300"
          >
            Sign in
          </Link>
        </p>
      </div>
    </div>
  );
}
