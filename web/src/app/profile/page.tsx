'use client';

import { useCallback, useEffect, useState } from 'react';
import AuthGuard from '@/components/AuthGuard';
import { fetchMyProfile, updateMyProfile } from '@/lib/api';

interface Profile {
  userId: string;
  displayName: string;
  bio: string;
  location: string;
  website: string;
  preferredLanguage: string;
  readingGoal: number;
  followerCount: number;
  followingCount: number;
  shelfCount: number;
}

export default function MyProfilePage() {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);

  const [form, setForm] = useState({
    displayName: '',
    bio: '',
    location: '',
    website: '',
    preferredLanguage: '',
    readingGoal: 0,
  });

  const loadProfile = useCallback(async () => {
    try {
      const res = await fetchMyProfile();
      if (res.ok) {
        const data = await res.json();
        setProfile(data);
        setForm({
          displayName: data.displayName ?? '',
          bio: data.bio ?? '',
          location: data.location ?? '',
          website: data.website ?? '',
          preferredLanguage: data.preferredLanguage ?? '',
          readingGoal: data.readingGoal ?? 0,
        });
      }
    } catch {
      /* empty — apiFetch redirects on 401 */
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadProfile();
  }, [loadProfile]);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setSaved(false);
    try {
      const res = await updateMyProfile(form);
      if (res.ok) {
        setSaved(true);
        setTimeout(() => setSaved(false), 2000);
      }
    } catch {
      /* empty — apiFetch redirects on 401 */
    } finally {
      setSaving(false);
    }
  };

  const updateField = (field: string, value: string | number) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  };

  return (
    <AuthGuard>
      <div className="mx-auto max-w-2xl px-4 py-8 sm:px-6">
        <h1 className="mb-6 font-serif text-2xl font-bold tracking-tight text-stone-900
          dark:text-stone-100">
          My Profile
        </h1>

        {loading ? (
          <div className="flex items-center justify-center py-20">
            <div className="text-sm text-stone-400 dark:text-stone-500">Loading…</div>
          </div>
        ) : (
          <>
            {profile && (
              <div className="mb-6 flex items-center gap-6 rounded-lg border border-stone-200
                bg-white p-5 dark:border-stone-700 dark:bg-stone-900">
                <div className="flex h-14 w-14 items-center justify-center rounded-full
                  bg-gradient-to-br from-amber-100 to-amber-200 text-xl font-bold text-amber-800
                  dark:from-amber-900/40 dark:to-amber-800/40 dark:text-amber-400">
                  {profile.displayName?.charAt(0).toUpperCase() ?? '?'}
                </div>
                <div className="flex gap-6">
                  {[
                    { label: 'Followers', value: profile.followerCount },
                    { label: 'Following', value: profile.followingCount },
                    { label: 'Shelves', value: profile.shelfCount },
                  ].map(({ label, value }) => (
                    <div key={label} className="text-center">
                      <p className="text-lg font-semibold text-stone-900 dark:text-stone-100">
                        {value ?? 0}
                      </p>
                      <p className="text-xs text-stone-500 dark:text-stone-400">{label}</p>
                    </div>
                  ))}
                </div>
              </div>
            )}

            <form onSubmit={handleSave} className="space-y-5">
              {[
                { id: 'displayName', label: 'Display name', type: 'text' },
                { id: 'location', label: 'Location', type: 'text' },
                { id: 'website', label: 'Website', type: 'url' },
                { id: 'preferredLanguage', label: 'Preferred language', type: 'text' },
              ].map(({ id, label, type }) => (
                <div key={id}>
                  <label htmlFor={id}
                    className="mb-1.5 block text-sm font-medium text-stone-700 dark:text-stone-300">
                    {label}
                  </label>
                  <input id={id} type={type}
                    value={(form as Record<string, string | number>)[id] as string}
                    onChange={(e) => updateField(id, e.target.value)}
                    className="w-full rounded-md border border-stone-200 bg-white px-3 py-2
                      text-sm text-stone-900 placeholder:text-stone-400
                      focus:border-amber-500 focus:outline-none focus:ring-1 focus:ring-amber-500
                      dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100
                      dark:placeholder:text-stone-500" />
                </div>
              ))}

              <div>
                <label htmlFor="bio"
                  className="mb-1.5 block text-sm font-medium text-stone-700 dark:text-stone-300">
                  Bio
                </label>
                <textarea id="bio" rows={3} value={form.bio}
                  onChange={(e) => updateField('bio', e.target.value)}
                  className="w-full rounded-md border border-stone-200 bg-white px-3 py-2
                    text-sm text-stone-900 placeholder:text-stone-400
                    focus:border-amber-500 focus:outline-none focus:ring-1 focus:ring-amber-500
                    dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100
                    dark:placeholder:text-stone-500" />
              </div>

              <div>
                <label htmlFor="readingGoal"
                  className="mb-1.5 block text-sm font-medium text-stone-700 dark:text-stone-300">
                  Reading goal (books/year)
                </label>
                <input id="readingGoal" type="number" min={0}
                  value={form.readingGoal}
                  onChange={(e) => updateField('readingGoal', parseInt(e.target.value) || 0)}
                  className="w-32 rounded-md border border-stone-200 bg-white px-3 py-2
                    text-sm text-stone-900 focus:border-amber-500 focus:outline-none
                    focus:ring-1 focus:ring-amber-500 dark:border-stone-700 dark:bg-stone-800
                    dark:text-stone-100" />
              </div>

              <div className="flex items-center gap-3 pt-2">
                <button type="submit" disabled={saving}
                  className="rounded-md bg-stone-900 px-4 py-2 text-sm font-medium text-white
                    transition-colors hover:bg-stone-800 disabled:opacity-60
                    dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200">
                  {saving ? 'Saving…' : 'Save changes'}
                </button>
                {saved && (
                  <span className="text-sm font-medium text-green-600 dark:text-green-400">
                    Saved!
                  </span>
                )}
              </div>
            </form>
          </>
        )}
      </div>
    </AuthGuard>
  );
}
