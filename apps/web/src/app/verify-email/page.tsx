'use client';

import React, { useState, useEffect, Suspense } from 'react';
import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { Flame, CheckCircle2, AlertCircle, ArrowRight, RefreshCw, Mail } from 'lucide-react';
import { apiFetch } from '@/lib/api';

function VerifyEmailContent() {
  const searchParams = useSearchParams();
  const token = searchParams.get('token') || '';

  const [loading, setLoading] = useState(true);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [resendEmail, setResendEmail] = useState('');
  const [resending, setResending] = useState(false);
  const [resendSuccess, setResendSuccess] = useState(false);

  useEffect(() => {
    if (!token) {
      setLoading(false);
      setError('No verification token provided in the URL.');
      return;
    }

    const verify = async () => {
      try {
        const res = await apiFetch<{ success: boolean; message: string }>(`/api/v1/auth/verify-email?token=${encodeURIComponent(token)}`);
        if (res.success) {
          setSuccess(true);
        } else {
          setError(res.error?.message || 'Invalid or expired email verification link.');
        }
      } catch (err: any) {
        setError(err?.message || 'Verification failed. Please try again.');
      } finally {
        setLoading(false);
      }
    };

    verify();
  }, [token]);

  const handleResend = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!resendEmail.trim()) return;

    setResending(true);
    try {
      const res = await apiFetch<{ success: boolean; message: string }>('/api/v1/auth/resend-verification', {
        method: 'POST',
        body: JSON.stringify({ email: resendEmail.trim() }),
      });
      if (res.success) {
        setResendSuccess(true);
      }
    } catch {
      setResendSuccess(true); // Always show generic confirmation
    } finally {
      setResending(false);
    }
  };

  return (
    <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-8 shadow-sm dark:shadow-md">
      {loading ? (
        <div className="text-center py-8 space-y-4">
          <div className="mx-auto w-10 h-10 border-3 border-[#16A34A] border-t-transparent rounded-full animate-spin" />
          <h3 className="text-base font-bold text-slate-900 dark:text-white">Verifying your email...</h3>
          <p className="text-xs text-slate-500 dark:text-slate-400">Please hold on while we confirm your account.</p>
        </div>
      ) : success ? (
        <div className="text-center space-y-4">
          <div className="mx-auto w-12 h-12 rounded-full bg-emerald-50 dark:bg-emerald-500/10 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
            <CheckCircle2 className="w-7 h-7" />
          </div>
          <h3 className="text-lg font-bold text-slate-900 dark:text-white">Email Successfully Verified!</h3>
          <p className="text-sm text-slate-600 dark:text-slate-400 leading-relaxed">
            Your Hostvra account email address has been verified. You now have full access to your hosting dashboard.
          </p>
          <div className="pt-4">
            <Link
              href="/login"
              className="inline-flex items-center justify-center gap-2 w-full py-3 px-4 rounded-xl bg-[#16A34A] hover:bg-[#15803D] text-white font-bold text-sm shadow-sm transition-all"
            >
              <span>Sign In to Dashboard</span>
              <ArrowRight className="w-4 h-4" />
            </Link>
          </div>
        </div>
      ) : (
        <div className="space-y-6">
          <div className="text-center space-y-3">
            <div className="mx-auto w-12 h-12 rounded-full bg-rose-50 dark:bg-rose-500/10 flex items-center justify-center text-rose-600 dark:text-rose-400">
              <AlertCircle className="w-7 h-7" />
            </div>
            <h3 className="text-lg font-bold text-slate-900 dark:text-white">Verification Failed</h3>
            <p className="text-sm text-slate-600 dark:text-slate-400">
              {error || 'The verification link is invalid or has expired.'}
            </p>
          </div>

          <div className="pt-4 border-t border-slate-100 dark:border-surface-800">
            {resendSuccess ? (
              <div className="p-4 rounded-xl bg-emerald-50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 text-sm text-center">
                ✓ If an unverified account exists, a new verification link has been sent!
              </div>
            ) : (
              <form onSubmit={handleResend} className="space-y-3">
                <label className="block text-xs font-bold uppercase tracking-wider text-slate-900 dark:text-slate-200">
                  Resend Verification Email
                </label>
                <div className="relative">
                  <Mail className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
                  <input
                    type="email"
                    required
                    value={resendEmail}
                    onChange={(e) => setResendEmail(e.target.value)}
                    placeholder="Enter your registered email"
                    className="w-full pl-10 pr-4 py-2 rounded-xl bg-white dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 placeholder-slate-400 text-sm focus:outline-none focus:border-[#16A34A]"
                  />
                </div>
                <button
                  type="submit"
                  disabled={resending}
                  className="w-full py-2.5 px-4 rounded-xl bg-slate-900 hover:bg-slate-800 dark:bg-surface-800 dark:hover:bg-surface-700 text-white font-semibold text-xs transition-colors flex items-center justify-center gap-1.5"
                >
                  {resending ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : 'Send New Verification Link'}
                </button>
              </form>
            )}
          </div>

          <div className="text-center">
            <Link href="/login" className="text-xs font-semibold text-[#16A34A] dark:text-emerald-400 hover:underline">
              Return to Sign In
            </Link>
          </div>
        </div>
      )}
    </div>
  );
}

export default function VerifyEmailPage() {
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
            Email Verification
          </h2>
          <p className="mt-2 text-sm text-slate-600 dark:text-slate-400 font-medium">
            Confirming your Hostvra account credentials
          </p>
        </div>

        <Suspense fallback={
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-8 text-center text-slate-400">
            Loading...
          </div>
        }>
          <VerifyEmailContent />
        </Suspense>
      </div>
    </div>
  );
}
