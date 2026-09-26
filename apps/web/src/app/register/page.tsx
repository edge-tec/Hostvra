'use client';

import React, { useState, useEffect, Suspense } from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { Flame, Lock, Mail, User, Building, ArrowRight, AlertCircle, Sparkles, CheckCircle2 } from 'lucide-react';
import { apiFetch, setStoredToken, HostingPlan } from '@/lib/api';

function RegisterForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const planSlug = searchParams.get('plan');
  const cycleParam = (searchParams.get('cycle') as 'monthly' | 'yearly') || 'monthly';
  const isTrialParam = searchParams.get('trial') === 'true';

  const [fullName, setFullName] = useState('');
  const [email, setEmail] = useState('');
  const [orgName, setOrgName] = useState('');
  const [password, setPassword] = useState('');
  const [selectedPlan, setSelectedPlan] = useState<HostingPlan | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!planSlug) return;
    async function fetchPlan() {
      try {
        const res = await apiFetch<HostingPlan>(`/api/v1/billing/plans/${planSlug}`);
        if (res.success && res.data) {
          setSelectedPlan(res.data);
        }
      } catch {
        // Ignored
      }
    }
    fetchPlan();
  }, [planSlug]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    const res = await apiFetch<{
      tokens: { access_token: string };
      user: { id: string; email: string; full_name: string };
      role: string;
      is_superadmin: boolean;
    }>('/api/v1/auth/register', {
      method: 'POST',
      body: JSON.stringify({
        full_name: fullName,
        email,
        organization_name: orgName || `${fullName.split(' ')[0]}'s Cloud`,
        password,
        plan_slug: planSlug || undefined,
      }),
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

      // If a hosting plan was selected during registration
      if (selectedPlan) {
        try {
          const subRes = await apiFetch<{ trial_activated?: boolean }>('/api/v1/billing/subscriptions', {
            method: 'POST',
            body: JSON.stringify({
              plan_id: selectedPlan.id,
              billing_cycle: cycleParam,
              auto_renew: true,
              start_trial: isTrialParam || selectedPlan.trial_allowed,
              payment_method: isTrialParam ? 'free_trial' : 'pending',
            }),
          });

          if (subRes.success && subRes.data?.trial_activated) {
            router.push('/billing?trial_activated=true');
            return;
          } else {
            router.push('/billing?tab=invoices&new_order=true');
            return;
          }
        } catch {
          // If subscription activation fails, fallback to dashboard
          router.push('/billing');
          return;
        }
      }

      router.push('/dashboard');
    } else {
      setLoading(false);
      setError(res.error?.message || 'Registration failed. Please verify your details.');
    }
  };

  return (
    <div className="w-full max-w-md space-y-6 z-10">
      <div className="text-center">
        <Link href="/" className="inline-flex items-center gap-2 group mb-4">
          <div className="w-12 h-12 rounded-2xl bg-gradient-to-tr from-indigo-600 to-purple-600 flex items-center justify-center shadow-xl shadow-indigo-600/25 group-hover:scale-105 transition-transform">
            <Flame className="w-6 h-6 text-white fill-white" />
          </div>
        </Link>
        <h2 className="text-3xl font-black tracking-tight text-slate-950 dark:text-white">
          Create Hostvra Account
        </h2>
        <p className="mt-1.5 text-sm text-slate-600 dark:text-slate-400 font-medium">
          Deploy websites, databases, and enterprise email in minutes
        </p>
      </div>

      {/* Selected Plan Summary Banner */}
      {selectedPlan && (
        <div className="p-4 rounded-2xl bg-indigo-50 dark:bg-indigo-950/40 border border-indigo-200 dark:border-indigo-800/60 flex items-center justify-between text-xs">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-indigo-600/20 text-indigo-600 dark:text-indigo-400 flex items-center justify-center flex-shrink-0">
              <Sparkles className="w-5 h-5" />
            </div>
            <div>
              <div className="font-bold text-slate-900 dark:text-white flex items-center gap-1.5">
                <span>{selectedPlan.name}</span>
                {isTrialParam && (
                  <span className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/20 text-emerald-700 dark:text-emerald-400 font-black uppercase">
                    14-Day Free Trial
                  </span>
                )}
              </div>
              <div className="text-slate-500 dark:text-slate-400 mt-0.5">
                {cycleParam === 'yearly'
                  ? `$${selectedPlan.price_yearly}/yr (~$${(selectedPlan.price_yearly / 12).toFixed(2)}/mo)`
                  : `$${selectedPlan.price_monthly}/mo`}
              </div>
            </div>
          </div>
          <Link
            href="/#pricing"
            className="text-[11px] font-bold text-indigo-600 dark:text-indigo-400 hover:underline"
          >
            Change
          </Link>
        </div>
      )}

      <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-8 shadow-xl dark:shadow-2xl">
        {error && (
          <div className="mb-6 p-4 rounded-xl bg-rose-50 dark:bg-rose-500/10 border border-rose-200 dark:border-rose-500/20 text-rose-700 dark:text-rose-400 text-sm flex items-start gap-3">
            <AlertCircle className="w-5 h-5 flex-shrink-0 mt-0.5" />
            <span className="font-medium">{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-xs font-bold uppercase tracking-wider text-slate-900 dark:text-slate-200 mb-2">
              Full Name
            </label>
            <div className="relative">
              <User className="w-5 h-5 text-slate-400 dark:text-slate-500 absolute left-3.5 top-1/2 -translate-y-1/2" />
              <input
                type="text"
                required
                value={fullName}
                onChange={(e) => setFullName(e.target.value)}
                placeholder="Alex Mercer"
                className="w-full pl-11 pr-4 py-2.5 rounded-xl bg-white dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 placeholder-slate-400 focus:outline-none focus:border-indigo-600 focus:ring-1 focus:ring-indigo-600 text-sm font-medium"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-bold uppercase tracking-wider text-slate-900 dark:text-slate-200 mb-2">
              Email Address
            </label>
            <div className="relative">
              <Mail className="w-5 h-5 text-slate-400 dark:text-slate-500 absolute left-3.5 top-1/2 -translate-y-1/2" />
              <input
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="alex@example.com"
                className="w-full pl-11 pr-4 py-2.5 rounded-xl bg-white dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 placeholder-slate-400 focus:outline-none focus:border-indigo-600 focus:ring-1 focus:ring-indigo-600 text-sm font-medium"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-bold uppercase tracking-wider text-slate-900 dark:text-slate-200 mb-2">
              Organization / Company (Optional)
            </label>
            <div className="relative">
              <Building className="w-5 h-5 text-slate-400 dark:text-slate-500 absolute left-3.5 top-1/2 -translate-y-1/2" />
              <input
                type="text"
                value={orgName}
                onChange={(e) => setOrgName(e.target.value)}
                placeholder="Acme Cloud Ltd"
                className="w-full pl-11 pr-4 py-2.5 rounded-xl bg-white dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 placeholder-slate-400 focus:outline-none focus:border-indigo-600 focus:ring-1 focus:ring-indigo-600 text-sm font-medium"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-bold uppercase tracking-wider text-slate-900 dark:text-slate-200 mb-2">
              Password (min. 8 characters)
            </label>
            <div className="relative">
              <Lock className="w-5 h-5 text-slate-400 dark:text-slate-500 absolute left-3.5 top-1/2 -translate-y-1/2" />
              <input
                type="password"
                required
                minLength={8}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••••••"
                className="w-full pl-11 pr-4 py-2.5 rounded-xl bg-white dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 placeholder-slate-400 focus:outline-none focus:border-indigo-600 focus:ring-1 focus:ring-indigo-600 text-sm font-medium"
              />
            </div>
          </div>

          <div className="pt-2">
            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 px-4 rounded-xl bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 active:scale-[0.99] disabled:opacity-50 text-white font-bold text-sm shadow-lg shadow-indigo-600/30 transition-all flex items-center justify-center gap-2 cursor-pointer group"
            >
              {loading ? (
                <div className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin" />
              ) : (
                <>
                  <span>
                    {isTrialParam ? 'Start 14-Day Free Trial' : selectedPlan ? 'Continue to Checkout' : 'Create Account'}
                  </span>
                  <ArrowRight className="w-4 h-4 group-hover:translate-x-0.5 transition-transform" />
                </>
              )}
            </button>
          </div>
        </form>
      </div>

      <p className="text-center text-sm text-slate-600 dark:text-slate-400 font-medium">
        Already have an account?{' '}
        <Link href="/login" className="text-indigo-600 dark:text-indigo-400 font-bold hover:underline">
          Sign In
        </Link>
      </p>
    </div>
  );
}

export default function RegisterPage() {
  return (
    <div className="min-h-screen bg-[#F8FAFC] dark:bg-[#07090e] flex flex-col justify-center items-center px-4 sm:px-6 lg:px-8 relative overflow-hidden py-12 transition-colors">
      <div className="absolute top-1/4 left-1/2 -translate-x-1/2 w-96 h-96 bg-indigo-500/10 blur-[120px] rounded-full pointer-events-none" />
      <Suspense
        fallback={
          <div className="flex items-center gap-3 text-slate-400">
            <div className="w-5 h-5 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
            <span>Loading...</span>
          </div>
        }
      >
        <RegisterForm />
      </Suspense>
    </div>
  );
}
