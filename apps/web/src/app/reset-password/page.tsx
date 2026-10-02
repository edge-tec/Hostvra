'use client';

import React, { useState, Suspense } from 'react';
import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { Flame, Lock, ArrowRight, AlertCircle, CheckCircle2, ShieldCheck } from 'lucide-react';
import { apiFetch } from '@/lib/api';

function ResetPasswordForm() {
  const searchParams = useSearchParams();
  const token = searchParams.get('token') || '';

  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!token) {
      setError('Invalid or missing password reset token. Please request a new link.');
      return;
    }

    if (newPassword.length < 8) {
      setError('Password must be at least 8 characters long.');
      return;
    }

    if (newPassword !== confirmPassword) {
      setError('Passwords do not match. Please re-enter.');
      return;
    }

    setLoading(true);

    try {
      const res = await apiFetch<{ success: boolean; message: string }>('/api/v1/auth/reset-password', {
        method: 'POST',
        body: JSON.stringify({
          token,
          new_password: newPassword,
        }),
      });

      if (res.success) {
        setSuccess(true);
      } else {
        setError(res.error?.message || 'Failed to reset password. The link may have expired.');
      }
    } catch (err: any) {
      setError(err?.message || 'Network error occurred. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-8 shadow-sm dark:shadow-md">
      {!token ? (
        <div className="text-center space-y-4">
          <div className="mx-auto w-12 h-12 rounded-full bg-rose-50 dark:bg-rose-500/10 flex items-center justify-center text-rose-600 dark:text-rose-400">
            <AlertCircle className="w-7 h-7" />
          </div>
          <h3 className="text-lg font-bold text-slate-900 dark:text-white">Missing Reset Token</h3>
          <p className="text-sm text-slate-600 dark:text-slate-400">
            No password reset token was provided in the URL. Please request a new link.
          </p>
          <div className="pt-2">
            <Link
              href="/forgot-password"
              className="inline-flex items-center justify-center gap-2 py-2.5 px-4 rounded-xl bg-[#16A34A] hover:bg-[#15803D] text-white font-bold text-sm shadow-sm transition-all"
            >
              Request Password Reset
            </Link>
          </div>
        </div>
      ) : success ? (
        <div className="text-center space-y-4">
          <div className="mx-auto w-12 h-12 rounded-full bg-emerald-50 dark:bg-emerald-500/10 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
            <CheckCircle2 className="w-7 h-7" />
          </div>
          <h3 className="text-lg font-bold text-slate-900 dark:text-white">Password Reset Complete</h3>
          <p className="text-sm text-slate-600 dark:text-slate-400 leading-relaxed">
            Your account password has been updated securely with Argon2id encryption. You can now sign in with your new credentials.
          </p>
          <div className="pt-4">
            <Link
              href="/login"
              className="inline-flex items-center justify-center gap-2 w-full py-3 px-4 rounded-xl bg-[#16A34A] hover:bg-[#15803D] text-white font-bold text-sm shadow-sm transition-all"
            >
              <span>Sign In with New Password</span>
              <ArrowRight className="w-4 h-4" />
            </Link>
          </div>
        </div>
      ) : (
        <>
          {error && (
            <div className="mb-6 p-4 rounded-xl bg-rose-50 dark:bg-rose-500/10 border border-rose-200 dark:border-rose-500/20 text-rose-700 dark:text-rose-400 text-sm flex items-start gap-3">
              <AlertCircle className="w-5 h-5 flex-shrink-0 mt-0.5" />
              <span className="font-medium">{error}</span>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-5">
            <div>
              <label className="block text-xs font-bold uppercase tracking-wider text-slate-900 dark:text-slate-200 mb-2">
                New Password
              </label>
              <div className="relative">
                <Lock className="w-5 h-5 text-slate-400 dark:text-slate-500 absolute left-3.5 top-1/2 -translate-y-1/2" />
                <input
                  type="password"
                  required
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  placeholder="Min 8 characters"
                  className="w-full pl-11 pr-4 py-2.5 rounded-xl bg-white dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 placeholder-slate-400 focus:outline-none focus:border-[#16A34A] focus:ring-1 focus:ring-[#16A34A] text-sm font-medium transition-colors"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-bold uppercase tracking-wider text-slate-900 dark:text-slate-200 mb-2">
                Confirm New Password
              </label>
              <div className="relative">
                <Lock className="w-5 h-5 text-slate-400 dark:text-slate-500 absolute left-3.5 top-1/2 -translate-y-1/2" />
                <input
                  type="password"
                  required
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  placeholder="Re-type new password"
                  className="w-full pl-11 pr-4 py-2.5 rounded-xl bg-white dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 placeholder-slate-400 focus:outline-none focus:border-[#16A34A] focus:ring-1 focus:ring-[#16A34A] text-sm font-medium transition-colors"
                />
              </div>
            </div>

            <div className="flex items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
              <ShieldCheck className="w-4 h-4 text-emerald-500" />
              <span>Passwords are secured using Argon2id with memory hardness.</span>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 px-4 rounded-xl bg-[#16A34A] hover:bg-[#15803D] active:bg-[#166534] disabled:opacity-50 text-white font-bold text-sm shadow-sm transition-all flex items-center justify-center gap-2 cursor-pointer group"
            >
              {loading ? (
                <div className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin" />
              ) : (
                <>
                  <span>Set New Password</span>
                  <ArrowRight className="w-4 h-4 group-hover:translate-x-0.5 transition-transform" />
                </>
              )}
            </button>
          </form>
        </>
      )}
    </div>
  );
}

export default function ResetPasswordPage() {
  return (
    <div className="min-h-screen bg-[#F8FAFC] dark:bg-[#090d16] flex flex-col justify-center items-center px-4 sm:px-6 lg:px-8 relative overflow-hidden transition-colors">
      <div className="w-full max-w-md space-y-8 z-10">
        <div className="text-center">
          <Link href="/" className="inline-block group mb-4">
            <div className="mx-auto w-12 h-12 rounded-xl bg-[#16A34A] flex items-center justify-center shadow-lg shadow-emerald-600/20 group-hover:scale-105 transition-transform">
              <Flame className="w-6 h-6 text-white fill-white" />
            </div>
          </Link>
          <h2 className="text-3xl font-black tracking-tight text-slate-950 dark:text-white">
            Set New Password
          </h2>
          <p className="mt-2 text-sm text-slate-600 dark:text-slate-400 font-medium">
            Choose a strong, secure password for your account
          </p>
        </div>

        <Suspense fallback={
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-8 text-center text-slate-400">
            Loading...
          </div>
        }>
          <ResetPasswordForm />
        </Suspense>
      </div>
    </div>
  );
}
