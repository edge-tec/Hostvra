'use client';

import React, { useEffect, useState, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { apiFetch, getStoredToken } from '@/lib/api';
import { Check, Tag, ShoppingCart, ArrowRight, ShieldCheck, Globe, Coins, CreditCard, Wallet, AlertCircle } from 'lucide-react';

interface HostingPlan {
  id: string;
  name: string;
  slug?: string;
  description: string;
  price_monthly: number;
  price_yearly: number;
  disk_space_mb?: number;
  bandwidth_mb?: number;
  disk_gb?: number;
  bandwidth_gb?: number;
  max_websites: number;
}

function OrderCartContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [plans, setPlans] = useState<HostingPlan[]>([]);
  const [selectedPlanId, setSelectedPlanId] = useState<string>('');
  const [cycle, setCycle] = useState<'monthly' | 'yearly'>((searchParams.get('cycle') as 'monthly' | 'yearly') || 'monthly');
  const [domainName, setDomainName] = useState<string>(searchParams.get('domain') || '');
  const [couponCode, setCouponCode] = useState<string>((searchParams.get('coupon') || '').toUpperCase());
  const [couponResult, setCouponResult] = useState<any>(null);
  const [couponError, setCouponError] = useState<string>('');
  const [applyingCoupon, setApplyingCoupon] = useState(false);
  const [paymentMethod, setPaymentMethod] = useState<'crypto' | 'card' | 'credit_wallet'>('crypto');
  const [submitting, setSubmitting] = useState(false);
  const [checkoutError, setCheckoutError] = useState<string>('');
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  useEffect(() => {
    setIsAuthenticated(Boolean(getStoredToken()));
    apiFetch<HostingPlan[]>('/api/v1/billing/plans')
      .then((res) => {
        if (res.data && res.data.length > 0) {
          setPlans(res.data);
          const prePlan = searchParams.get('plan');
          const match = prePlan ? res.data.find((p) => p.id === prePlan || p.slug === prePlan) : null;
          setSelectedPlanId(match ? match.id : res.data[0].id);
        }
      })
      .catch(() => {});
  }, [searchParams]);

  const selectedPlan = plans.find((p) => p.id === selectedPlanId) || plans[0];
  const basePrice = selectedPlan
    ? cycle === 'yearly'
      ? selectedPlan.price_yearly || selectedPlan.price_monthly * 10
      : selectedPlan.price_monthly
    : 0;
  const discountUSD = couponResult?.valid ? Number(couponResult.discount_usd || 0) : 0;
  const finalTotal = Math.max(0, basePrice - discountUSD);

  const handleApplyCoupon = async () => {
    if (!couponCode.trim()) return;
    setApplyingCoupon(true);
    setCouponError('');
    setCouponResult(null);
    try {
      const res = await apiFetch<any>('/api/v1/billing/coupons/validate', {
        method: 'POST',
        body: JSON.stringify({ code: couponCode.trim(), order_amount_usd: basePrice }),
      });
      if (res.data?.valid) setCouponResult(res.data);
      else setCouponError(res.error?.message || 'Invalid coupon code');
    } catch (e: any) {
      setCouponError(e.message || 'Invalid coupon code');
    } finally {
      setApplyingCoupon(false);
    }
  };

  const handleProceedCheckout = async () => {
    if (!selectedPlan) return;
    setCheckoutError('');

    const token = getStoredToken();
    if (!token) {
      const nextParams = new URLSearchParams();
      nextParams.set('plan', selectedPlan.id);
      nextParams.set('cycle', cycle);
      if (domainName.trim()) nextParams.set('domain', domainName.trim());
      if (couponResult?.code) nextParams.set('coupon', couponResult.code);
      const nextUrl = `/order?${nextParams.toString()}`;
      router.push(`/register?next=${encodeURIComponent(nextUrl)}`);
      return;
    }

    setSubmitting(true);
    try {
      const res = await apiFetch<{
        invoice: { id: string; invoice_number: string };
        paid: boolean;
        next_url: string;
      }>('/api/v1/billing/orders/checkout', {
        method: 'POST',
        body: JSON.stringify({
          plan_id: selectedPlan.id,
          billing_cycle: cycle,
          domain_name: domainName.trim() || undefined,
          coupon_code: couponResult?.code || (couponCode.trim() ? couponCode.trim().toUpperCase() : undefined),
          payment_method: paymentMethod,
        }),
      });

      if (res.error || !res.data) {
        setCheckoutError(res.error?.message || 'Checkout failed. Please verify your selection and try again.');
        return;
      }

      router.push(res.data.next_url || `/billing?tab=invoices&pay_invoice=${res.data.invoice.id}`);
    } catch (e: any) {
      setCheckoutError(e.message || 'Checkout request failed');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-950 text-slate-900 dark:text-slate-100 py-10 px-4">
      <div className="max-w-5xl mx-auto space-y-8">
        <div className="flex items-center justify-between flex-wrap gap-3">
          <div>
            <h1 className="text-2xl font-extrabold tracking-tight flex items-center gap-2">
              <ShoppingCart className="w-6 h-6 text-emerald-600" />
              <span>Configure Your Cloud Hosting Order</span>
            </h1>
            <p className="text-xs text-slate-500 mt-1">
              Select plan, attach a domain, apply promo codes (try WELCOME20), and checkout via Zero-Fee Crypto, Card, or Credit Wallet.
            </p>
          </div>
          <div className="flex items-center gap-4 text-xs font-bold">
            {isAuthenticated && (
              <Link href="/dashboard" className="text-slate-600 dark:text-slate-300 hover:text-emerald-600">
                Dashboard
              </Link>
            )}
            <Link href="/" className="text-emerald-600 hover:underline">
              &larr; Back to Home
            </Link>
          </div>
        </div>

        {checkoutError && (
          <div className="p-4 rounded-2xl bg-rose-50 dark:bg-rose-500/10 border border-rose-200 dark:border-rose-500/30 text-rose-700 dark:text-rose-300 text-xs font-semibold flex items-center gap-2">
            <AlertCircle className="w-4 h-4 flex-shrink-0" />
            <span>{checkoutError}</span>
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 space-y-6">
            {/* Step 1: Select Hosting Plan */}
            <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-3xl p-6 space-y-4">
              <div className="flex items-center justify-between">
                <h2 className="text-sm font-bold uppercase tracking-wider text-slate-500">1. Choose Hosting Plan</h2>
                <div className="inline-flex rounded-xl bg-slate-100 dark:bg-slate-800 p-1 text-xs font-bold">
                  <button
                    onClick={() => {
                      setCycle('monthly');
                      setCouponResult(null);
                    }}
                    className={`px-3 py-1 rounded-lg cursor-pointer ${cycle === 'monthly' ? 'bg-white dark:bg-slate-900 shadow-sm' : 'text-slate-500'}`}
                  >
                    Monthly
                  </button>
                  <button
                    onClick={() => {
                      setCycle('yearly');
                      setCouponResult(null);
                    }}
                    className={`px-3 py-1 rounded-lg cursor-pointer ${cycle === 'yearly' ? 'bg-white dark:bg-slate-900 shadow-sm text-emerald-600' : 'text-slate-500'}`}
                  >
                    Yearly (Save)
                  </button>
                </div>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                {plans.map((p) => {
                  const active = p.id === selectedPlan?.id;
                  const price = cycle === 'yearly' ? p.price_yearly || p.price_monthly * 10 : p.price_monthly;
                  const diskGB = p.disk_gb ?? (p.disk_space_mb ? Math.round(p.disk_space_mb / 1024) : 0);
                  const bwGB = p.bandwidth_gb ?? (p.bandwidth_mb ? Math.round(p.bandwidth_mb / 1024) : 0);
                  return (
                    <button
                      key={p.id}
                      onClick={() => {
                        setSelectedPlanId(p.id);
                        setCouponResult(null);
                      }}
                      className={`p-4 rounded-2xl border text-left transition cursor-pointer ${
                        active
                          ? 'border-emerald-500 ring-2 ring-emerald-500/30 bg-emerald-50/40 dark:bg-emerald-500/10'
                          : 'border-slate-200 dark:border-slate-800 hover:border-slate-300'
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <span className="font-extrabold text-sm">{p.name}</span>
                        {active && <Check className="w-4 h-4 text-emerald-600" />}
                      </div>
                      <div className="text-xl font-extrabold font-mono mt-1">
                        ${Number(price || 0).toFixed(2)}
                        <span className="text-xs font-sans font-normal text-slate-500">/{cycle === 'yearly' ? 'yr' : 'mo'}</span>
                      </div>
                      <div className="text-[11px] text-slate-500 mt-2">
                        {p.max_websites} Sites • {diskGB} GB NVMe • {bwGB} GB BW
                      </div>
                    </button>
                  );
                })}
              </div>
            </div>

            {/* Step 2: Optional Domain */}
            <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-3xl p-6 space-y-3">
              <h2 className="text-sm font-bold uppercase tracking-wider text-slate-500 flex items-center gap-1.5">
                <Globe className="w-4 h-4 text-emerald-500" />
                <span>2. Attach Domain Name (Optional)</span>
              </h2>
              <input
                type="text"
                placeholder="example.com (leave blank to configure later)"
                value={domainName}
                onChange={(e) => setDomainName(e.target.value)}
                className="w-full p-3 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-xs font-mono"
              />
            </div>

            {/* Step 3: Payment Method */}
            <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-3xl p-6 space-y-3">
              <h2 className="text-sm font-bold uppercase tracking-wider text-slate-500">3. Payment Method</h2>
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                <button
                  type="button"
                  onClick={() => setPaymentMethod('crypto')}
                  className={`p-3.5 rounded-2xl border text-left cursor-pointer transition ${
                    paymentMethod === 'crypto'
                      ? 'border-emerald-500 ring-2 ring-emerald-500/30 bg-emerald-50/40 dark:bg-emerald-500/10'
                      : 'border-slate-200 dark:border-slate-800'
                  }`}
                >
                  <Coins className="w-4 h-4 text-emerald-600 mb-1.5" />
                  <div className="text-xs font-extrabold">Zero-Fee Crypto</div>
                  <div className="text-[10px] text-slate-500">USDT, BTC, ETH, BNB, SOL</div>
                </button>
                <button
                  type="button"
                  onClick={() => setPaymentMethod('card')}
                  className={`p-3.5 rounded-2xl border text-left cursor-pointer transition ${
                    paymentMethod === 'card'
                      ? 'border-emerald-500 ring-2 ring-emerald-500/30 bg-emerald-50/40 dark:bg-emerald-500/10'
                      : 'border-slate-200 dark:border-slate-800'
                  }`}
                >
                  <CreditCard className="w-4 h-4 text-blue-600 mb-1.5" />
                  <div className="text-xs font-extrabold">Card / Gateway</div>
                  <div className="text-[10px] text-slate-500">Stripe / Local Gateways</div>
                </button>
                <button
                  type="button"
                  onClick={() => setPaymentMethod('credit_wallet')}
                  className={`p-3.5 rounded-2xl border text-left cursor-pointer transition ${
                    paymentMethod === 'credit_wallet'
                      ? 'border-emerald-500 ring-2 ring-emerald-500/30 bg-emerald-50/40 dark:bg-emerald-500/10'
                      : 'border-slate-200 dark:border-slate-800'
                  }`}
                >
                  <Wallet className="w-4 h-4 text-indigo-600 mb-1.5" />
                  <div className="text-xs font-extrabold">Credit Wallet</div>
                  <div className="text-[10px] text-slate-500">Instant balance settlement</div>
                </button>
              </div>
            </div>
          </div>

          {/* Order Summary Sidebar */}
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-3xl p-6 h-fit space-y-5 shadow-sm">
            <h2 className="text-sm font-bold uppercase tracking-wider text-slate-500">Order Summary</h2>
            <div className="space-y-2 text-xs">
              <div className="flex justify-between">
                <span className="text-slate-500">Selected Plan</span>
                <span className="font-bold">{selectedPlan?.name || 'Loading...'}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">Billing Cycle</span>
                <span className="font-bold capitalize">{cycle}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">Subtotal</span>
                <span className="font-mono">${basePrice.toFixed(2)}</span>
              </div>
              {discountUSD > 0 && (
                <div className="flex justify-between text-emerald-600 font-bold">
                  <span>Promo ({couponResult.code})</span>
                  <span className="font-mono">-${discountUSD.toFixed(2)}</span>
                </div>
              )}
              <div className="pt-3 border-t border-slate-100 dark:border-slate-800 flex justify-between text-base font-extrabold">
                <span>Total Due</span>
                <span className="font-mono text-emerald-600">${finalTotal.toFixed(2)}</span>
              </div>
            </div>

            {/* Coupon Box */}
            <div className="space-y-2">
              <div className="flex gap-2">
                <input
                  type="text"
                  placeholder="Promo code (e.g. WELCOME20)"
                  value={couponCode}
                  onChange={(e) => setCouponCode(e.target.value.toUpperCase())}
                  className="flex-1 p-2.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-xs font-mono uppercase"
                />
                <button
                  onClick={handleApplyCoupon}
                  disabled={applyingCoupon || !couponCode.trim()}
                  className="px-3.5 py-2 rounded-xl bg-slate-900 dark:bg-slate-700 text-white text-xs font-bold cursor-pointer flex items-center gap-1 disabled:opacity-50"
                >
                  <Tag className="w-3.5 h-3.5" />
                  <span>Apply</span>
                </button>
              </div>
              {couponError && <p className="text-[11px] text-rose-500">{couponError}</p>}
              {couponResult?.valid && <p className="text-[11px] text-emerald-600 font-semibold">Applied: {couponResult.description}</p>}
            </div>

            <button
              onClick={handleProceedCheckout}
              disabled={submitting || !selectedPlan}
              className="w-full py-3.5 rounded-2xl bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white font-bold text-sm flex items-center justify-center gap-2 cursor-pointer shadow-lg shadow-emerald-600/20"
            >
              <span>
                {submitting
                  ? 'Creating Authoritative Invoice...'
                  : isAuthenticated
                  ? 'Complete Secure Checkout'
                  : 'Create Account & Checkout'}
              </span>
              <ArrowRight className="w-4 h-4" />
            </button>
            <div className="text-[11px] text-slate-400 flex items-center justify-center gap-1.5">
              <ShieldCheck className="w-3.5 h-3.5 text-emerald-500" />
              <span>Server-verified pricing &amp; instant automated provisioning</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default function DirectOrderCartPage() {
  return (
    <Suspense fallback={<div className="p-10 text-center text-sm text-slate-500">Loading Order Cart...</div>}>
      <OrderCartContent />
    </Suspense>
  );
}
