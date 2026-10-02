'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { Flame, Mail, ArrowRight, AlertCircle, CheckCircle2, ArrowLeft } from 'lucide-react';
import { apiFetch } from '@/lib/api';

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const res = await apiFetch<{ success: boolean; message: string }>('/api/v1/auth/forgot-password', {
        method: 'POST',
        body: JSON.stringify({ email: email.trim() }),
      });

      if (res.success) {
        setSubmitted(true);
      } else {
        setError(res.error?.message || 'Failed to request password reset. Please try again.');
      }
    } catch (err: any) {
      setError(err?.message || 'Network request failed. Please check your connection.');
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
            Reset Password
          </h2>
          <p className="mt-2 text-sm text-slate-600 dark:text-slate-400 font-medium">
            Enter your account email to receive a password reset link
          </p>
        </div>

        {/* Card */}
        <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-8 shadow-sm dark:shadow-md">
          {submitted ? (
            <div className="text-center space-y-4">
              <div className="mx-auto w-12 h-12 rounded-full bg-emerald-50 dark:bg-emerald-500/10 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                <CheckCircle2 className="w-7 h-7" />
              </div>
              <h3 className="text-lg font-bold text-slate-900 dark:text-white">Check your email</h3>
              <p className="text-sm text-slate-600 dark:text-slate-400 leading-relaxed">
                If an account with <strong className="text-slate-900 dark:text-slate-200">{email}</strong> exists, we’ve sent instructions and a secure link to reset your password.
              </p>
              <div className="pt-4">
                <Link
                  href="/login"
                  className="inline-flex items-center justify-center gap-2 w-full py-2.5 px-4 rounded-xl bg-slate-100 hover:bg-slate-200 dark:bg-surface-800 dark:hover:bg-surface-700 text-slate-900 dark:text-white font-semibold text-sm transition-colors"
                >
                  <ArrowLeft className="w-4 h-4" /> Return to Sign In
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
                    Account Email
                  </label>
                  <div className="relative">
                    <Mail className="w-5 h-5 text-slate-400 dark:text-slate-500 absolute left-3.5 top-1/2 -translate-y-1/2" />
                    <input
                      type="email"
                      required
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      placeholder="name@company.com"
                      className="w-full pl-11 pr-4 py-2.5 rounded-xl bg-white dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 placeholder-slate-400 focus:outline-none focus:border-[#16A34A] focus:ring-1 focus:ring-[#16A34A] text-sm font-medium transition-colors"
                    />
                  </div>
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
                      <span>Send Reset Link</span>
                      <ArrowRight className="w-4 h-4 group-hover:translate-x-0.5 transition-transform" />
                    </>
                  )}
                </button>
              </form>

              <div className="mt-6 pt-5 border-t border-slate-100 dark:border-surface-800 text-center">
                <Link
                  href="/login"
                  className="inline-flex items-center gap-1.5 text-xs font-semibold text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white transition-colors"
                >
                  <ArrowLeft className="w-3.5 h-3.5" /> Back to Sign In
                </Link>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
