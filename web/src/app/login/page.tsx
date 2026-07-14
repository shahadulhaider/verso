'use client';

import { useState, type FormEvent } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { apiFetch } from '@/lib/api';
import { setToken, setUser } from '@/lib/auth';

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const res = await apiFetch('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email, password }),
      });

      if (!res.ok) {
        const body = await res.json().catch(() => null);
        setError(body?.detail ?? body?.message ?? 'Invalid credentials');
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
          <h1 className="font-serif text-3xl font-bold tracking-tight text-stone-900">
            Welcome back
          </h1>
          <p className="mt-2 text-sm text-stone-500">
            Sign in to continue your reading journey.
          </p>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
              {error}
            </div>
          )}

          <div>
            <label htmlFor="email" className="block text-sm font-medium text-stone-700">
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
                focus:border-amber-600/40 focus:outline-none focus:ring-2 focus:ring-amber-600/10"
              placeholder="you@example.com"
            />
          </div>

          <div>
            <label htmlFor="password" className="block text-sm font-medium text-stone-700">
              Password
            </label>
            <input
              id="password"
              type="password"
              required
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="mt-1 block w-full rounded-md border border-stone-300 bg-white px-3 py-2
                text-sm text-stone-900 shadow-sm transition-colors placeholder:text-stone-400
                focus:border-amber-600/40 focus:outline-none focus:ring-2 focus:ring-amber-600/10"
              placeholder="\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022"
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-md bg-stone-900 px-4 py-2.5 text-sm font-medium text-white
              shadow-sm transition-colors hover:bg-stone-800 disabled:cursor-not-allowed
              disabled:opacity-50"
          >
            {loading ? 'Signing in\u2026' : 'Sign in'}
          </button>
        </form>

        {/* Link to register */}
        <p className="mt-6 text-center text-sm text-stone-500">
          Don&apos;t have an account?{' '}
          <Link
            href="/register"
            className="font-medium text-amber-700 transition-colors hover:text-amber-800"
          >
            Create one
          </Link>
        </p>
      </div>
    </div>
  );
}
