'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Flame, Lock, Mail, ArrowRight, AlertCircle, ShieldCheck } from 'lucide-react';
import { apiFetch, setStoredToken, getStoredToken } from '@/lib/api';

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const token = getStoredToken();
    if (token) {
      if (typeof document !== 'undefined' && !document.cookie.includes('hostvra_token=')) {
        document.cookie = `hostvra_token=${encodeURIComponent(token)}; path=/; max-age=604800; SameSite=Lax`;
      }
      const urlParams = typeof window !== 'undefined' ? new URLSearchParams(window.location.search) : null;
      const redirectParam = urlParams?.get('redirect');
      const destination = redirectParam && redirectParam.startsWith('/') && !redirectParam.startsWith('//') ? redirectParam : '/dashboard';
      router.replace(destination);
    }
  }, [router]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const res = await apiFetch<{
        tokens: { access_token: string };
        user: { id: string; email: string; full_name: string };
        role: string;
        is_superadmin: boolean;
      }>('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email: email.trim(), password }),
      });

      if (res.success && res.data) {
        setStoredToken(res.data.tokens.access_token);
        if (typeof window !== 'undefined') {
          localStorage.setItem(
            'hostvra_user',
            JSON.stringify({
              id: res.data.user?.id,
              email: res.data.user?.email,
              role: res.data.role || 'customer',
              is_superadmin: Boolean(res.data.is_superadmin),
            })
          );
        }
        const urlParams = typeof window !== 'undefined' ? new URLSearchParams(window.location.search) : null;
        const redirectParam = urlParams?.get('redirect');
        const destination = redirectParam && redirectParam.startsWith('/') && !redirectParam.startsWith('//') ? redirectParam : '/dashboard';
        window.location.href = destination;
        return;
      } else {
        setError(res.error?.message || 'Invalid email or password');
      }
    } catch (err: any) {
      setError(err?.message || 'Authentication request failed. Please check network connection.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-[#F8FAFC] dark:bg-[#090d16] flex flex-col justify-center items-center px-4 sm:px-6 lg:px-8 relative overflow-hidden transition-colors">
      <div className="w-full max-w-md space-y-8 z-10">
        {/* Brand Header */}
        <div className="text-center">
          <Link href="/" className="inline-block group mb-4">
            <div className="mx-auto w-12 h-12 rounded-xl bg-[#16A34A] flex items-center justify-center shadow-lg shadow-emerald-600/20 group-hover:scale-105 transition-transform">
              <Flame className="w-6 h-6 text-white fill-white" />
            </div>
          </Link>
          <h2 className="text-3xl font-black tracking-tight text-slate-950 dark:text-white">
            Sign in to Hostvra
          </h2>
          <p className="mt-2 text-sm text-slate-600 dark:text-slate-400 font-medium">
            Self-hosted server and hosting control plane
          </p>
        </div>

        {/* Card */}
        <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-8 shadow-sm dark:shadow-md">
          {error && (
            <div className="mb-6 p-4 rounded-xl bg-rose-50 dark:bg-rose-500/10 border border-rose-200 dark:border-rose-500/20 text-rose-700 dark:text-rose-400 text-sm flex items-start gap-3">
              <AlertCircle className="w-5 h-5 flex-shrink-0 mt-0.5" />
              <span className="font-medium">{error}</span>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-5">
            <div>
              <label className="block text-xs font-bold uppercase tracking-wider text-slate-900 dark:text-slate-200 mb-2">
                Work Email
              </label>
              <div className="relative">
                <Mail className="w-5 h-5 text-slate-400 dark:text-slate-500 absolute left-3.5 top-1/2 -translate-y-1/2" />
                <input
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="admin@yourdomain.com"
                  className="w-full pl-11 pr-4 py-2.5 rounded-xl bg-white dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 placeholder-slate-400 focus:outline-none focus:border-[#16A34A] focus:ring-1 focus:ring-[#16A34A] text-sm font-medium transition-colors"
                />
              </div>
            </div>

            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="block text-xs font-bold uppercase tracking-wider text-slate-900 dark:text-slate-200">
                  Password
                </label>
              </div>
              <div className="relative">
                <Lock className="w-5 h-5 text-slate-400 dark:text-slate-500 absolute left-3.5 top-1/2 -translate-y-1/2" />
                <input
                  type="password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••••••"
                  className="w-full pl-11 pr-4 py-2.5 rounded-xl bg-white dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 placeholder-slate-400 focus:outline-none focus:border-[#16A34A] focus:ring-1 focus:ring-[#16A34A] text-sm font-medium transition-colors"
                />
              </div>
            </div>

            <div className="flex items-center justify-end text-xs pt-1">
              <Link href="/register" className="text-[#16A34A] dark:text-emerald-400 font-semibold hover:underline">
                Create Account
              </Link>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full mt-3 py-3 px-4 rounded-xl bg-[#16A34A] hover:bg-[#15803D] active:bg-[#166534] disabled:opacity-50 text-white font-bold text-sm shadow-sm transition-all flex items-center justify-center gap-2 cursor-pointer group"
            >
              {loading ? (
                <div className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin" />
              ) : (
                <>
                  <span>Sign In</span>
                  <ArrowRight className="w-4 h-4 group-hover:translate-x-0.5 transition-transform" />
                </>
              )}
            </button>
          </form>
        </div>

        {/* Footer */}
        <p className="text-center text-sm text-slate-600 dark:text-slate-400 font-medium">
          Need a new control plane account?{' '}
          <Link href="/register" className="text-[#16A34A] dark:text-emerald-400 font-bold hover:underline">
            Register Organization
          </Link>
        </p>
      </div>
    </div>
  );
}
