'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Flame, Lock, Mail, ArrowRight, AlertCircle, ShieldCheck, Key, ArrowLeft } from 'lucide-react';
import { apiFetch, setStoredToken, getStoredToken } from '@/lib/api';

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // 2FA Challenge State
  const [twoFactorRequired, setTwoFactorRequired] = useState(false);
  const [tempToken, setTempToken] = useState('');
  const [twoFactorCode, setTwoFactorCode] = useState('');
  const [useRecoveryCode, setUseRecoveryCode] = useState(false);

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

  const handlePasswordSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const res = await apiFetch<{
        two_factor_required?: boolean;
        temp_token?: string;
        tokens?: { access_token: string };
        user?: { id: string; email: string; full_name: string };
        role?: string;
        is_superadmin?: boolean;
      }>('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email: email.trim(), password }),
      });

      if (res.success && res.data) {
        // If 2FA is required, transition to the verification card
        if (res.data.two_factor_required && res.data.temp_token) {
          setTwoFactorRequired(true);
          setTempToken(res.data.temp_token);
          setTwoFactorCode('');
          return;
        }

        // Direct login success
        if (res.data.tokens?.access_token) {
          handleAuthSuccess(res.data.tokens.access_token, res.data.user, res.data.role, res.data.is_superadmin);
          return;
        }
      }

      setError(res.error?.message || 'Invalid email or password');
    } catch (err: any) {
      setError(err?.message || 'Authentication request failed. Please check network connection.');
    } finally {
      setLoading(false);
    }
  };

  const handle2FASubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!twoFactorCode.trim()) {
      setError(useRecoveryCode ? 'Please enter your emergency recovery code' : 'Please enter the 6-digit code');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const res = await apiFetch<{
        tokens: { access_token: string };
        user: { id: string; email: string; full_name: string };
        role: string;
        is_superadmin: boolean;
      }>('/api/v1/auth/2fa/verify', {
        method: 'POST',
        body: JSON.stringify({
          temp_token: tempToken,
          code: twoFactorCode.trim(),
        }),
      });

      if (res.success && res.data?.tokens?.access_token) {
        handleAuthSuccess(res.data.tokens.access_token, res.data.user, res.data.role, res.data.is_superadmin);
      } else {
        setError(res.error?.message || 'Invalid two-factor authentication code or recovery code');
      }
    } catch (err: any) {
      setError(err?.message || 'Verification failed. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleAuthSuccess = (accessToken: string, user: any, role?: string, isSuperAdmin?: boolean) => {
    setStoredToken(accessToken);
    if (typeof window !== 'undefined') {
      localStorage.setItem(
        'hostvra_user',
        JSON.stringify({
          id: user?.id,
          email: user?.email,
          role: role || 'customer',
          is_superadmin: Boolean(isSuperAdmin),
        })
      );
    }
    const urlParams = typeof window !== 'undefined' ? new URLSearchParams(window.location.search) : null;
    const redirectParam = urlParams?.get('redirect');
    const destination = redirectParam && redirectParam.startsWith('/') && !redirectParam.startsWith('//') ? redirectParam : '/dashboard';
    window.location.href = destination;
  };

  const resetToLoginForm = () => {
    setTwoFactorRequired(false);
    setTempToken('');
    setTwoFactorCode('');
    setError(null);
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
            {twoFactorRequired ? 'Two-Factor Challenge' : 'Sign in to Hostvra'}
          </h2>
          <p className="mt-2 text-sm text-slate-600 dark:text-slate-400 font-medium">
            {twoFactorRequired
              ? 'Multi-factor security verification required'
              : 'Self-hosted server and hosting control plane'}
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

          {!twoFactorRequired ? (
            /* STEP 1: EMAIL & PASSWORD */
            <form onSubmit={handlePasswordSubmit} className="space-y-5">
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

              <div className="flex items-center justify-between text-xs pt-1">
                <Link href="/forgot-password" className="text-slate-500 hover:text-slate-900 dark:hover:text-slate-200 transition-colors">
                  Forgot password?
                </Link>
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
          ) : (
            /* STEP 2: TOTP 2FA CHALLENGE */
            <form onSubmit={handle2FASubmit} className="space-y-6">
              <div className="text-center space-y-2">
                <div className="w-12 h-12 mx-auto rounded-2xl bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200 dark:border-emerald-800/40 flex items-center justify-center">
                  {useRecoveryCode ? (
                    <Key className="w-6 h-6 text-[#16A34A]" />
                  ) : (
                    <ShieldCheck className="w-6 h-6 text-[#16A34A]" />
                  )}
                </div>
                <p className="text-xs text-slate-600 dark:text-slate-400 max-w-xs mx-auto">
                  {useRecoveryCode
                    ? 'Enter one of your 12-character emergency recovery codes (e.g. 4H7K-9P2M-3X8W).'
                    : 'Open Google Authenticator or your compatible authenticator app and enter the 6-digit code.'}
                </p>
              </div>

              <div>
                <label className="block text-xs font-bold uppercase tracking-wider text-slate-900 dark:text-slate-200 mb-2 text-center">
                  {useRecoveryCode ? 'Emergency Recovery Code' : '6-Digit Verification Code'}
                </label>
                <input
                  type="text"
                  autoFocus
                  required
                  autoComplete="one-time-code"
                  value={twoFactorCode}
                  onChange={(e) => setTwoFactorCode(e.target.value)}
                  placeholder={useRecoveryCode ? 'XXXX-XXXX-XXXX' : '123456'}
                  maxLength={useRecoveryCode ? 16 : 6}
                  className="w-full text-center tracking-widest text-xl font-mono px-4 py-3 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-white placeholder-slate-400 focus:outline-none focus:border-[#16A34A] focus:ring-2 focus:ring-[#16A34A]/20 transition-all font-semibold"
                />
              </div>

              <button
                type="submit"
                disabled={loading}
                className="w-full py-3 px-4 rounded-xl bg-[#16A34A] hover:bg-[#15803D] active:bg-[#166534] disabled:opacity-50 text-white font-bold text-sm shadow-sm transition-all flex items-center justify-center gap-2 cursor-pointer"
              >
                {loading ? (
                  <div className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin" />
                ) : (
                  <>
                    <span>Verify & Continue</span>
                    <ArrowRight className="w-4 h-4" />
                  </>
                )}
              </button>

              <div className="flex flex-col items-center gap-2 pt-2 text-xs">
                <button
                  type="button"
                  onClick={() => {
                    setUseRecoveryCode(!useRecoveryCode);
                    setTwoFactorCode('');
                    setError(null);
                  }}
                  className="text-slate-600 dark:text-slate-400 hover:text-slate-950 dark:hover:text-white transition-colors underline cursor-pointer"
                >
                  {useRecoveryCode ? 'Use authenticator app instead' : 'Lost your phone? Use an emergency recovery code'}
                </button>

                <button
                  type="button"
                  onClick={resetToLoginForm}
                  className="text-slate-500 hover:text-slate-800 dark:hover:text-slate-300 flex items-center gap-1 mt-2 cursor-pointer transition-colors"
                >
                  <ArrowLeft className="w-3.5 h-3.5" />
                  <span>Back to Sign In</span>
                </button>
              </div>
            </form>
          )}
        </div>

        {/* Footer */}
        {!twoFactorRequired && (
          <p className="text-center text-sm text-slate-600 dark:text-slate-400 font-medium">
            Need a new control plane account?{' '}
            <Link href="/register" className="text-[#16A34A] dark:text-emerald-400 font-bold hover:underline">
              Register Organization
            </Link>
          </p>
        )}
      </div>
    </div>
  );
}
