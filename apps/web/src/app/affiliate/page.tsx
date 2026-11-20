'use client';

import React, { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/api';
import { Copy, Check, Wallet, Users, DollarSign, ArrowUpRight, RefreshCw, ShieldCheck } from 'lucide-react';

export default function AffiliateAndWalletPage() {
  const [affData, setAffData] = useState<any>(null);
  const [walletData, setWalletData] = useState<any>(null);
  const [copied, setCopied] = useState(false);
  const [withdrawAmt, setWithdrawAmt] = useState('');
  const [payoutMethod, setPayoutMethod] = useState('credit_wallet');
  const [payoutDetails, setPayoutDetails] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [loading, setLoading] = useState(true);

  const loadAll = useCallback(async () => {
    setLoading(true);
    try {
      const [aRes, wRes] = await Promise.all([
        apiFetch<any>('/api/v1/affiliate/dashboard'),
        apiFetch<any>('/api/v1/wallet'),
      ]);
      if (aRes.data) setAffData(aRes.data);
      if (wRes.data) setWalletData(wRes.data);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadAll();
  }, [loadAll]);

  const copyRefLink = () => {
    if (!affData?.referral_link) return;
    navigator.clipboard.writeText(affData.referral_link);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleToggleAutoPay = async () => {
    const next = !(walletData?.wallet?.auto_pay_enabled);
    const res = await apiFetch<any>('/api/v1/wallet/autopay', {
      method: 'PUT',
      body: JSON.stringify({ auto_pay_enabled: next }),
    });
    if (res.data) await loadAll();
  };

  const handleWithdraw = async (e: React.FormEvent) => {
    e.preventDefault();
    const amt = parseFloat(withdrawAmt);
    if (!amt || amt <= 0) return;
    setSubmitting(true);
    try {
      const res = await apiFetch<any>('/api/v1/affiliate/withdraw', {
        method: 'POST',
        body: JSON.stringify({
          amount_usd: amt,
          payout_method: payoutMethod,
          payout_details: payoutDetails,
        }),
      });
      if (res.error) alert(res.error.message || 'Withdrawal failed');
      else {
        setWithdrawAmt('');
        setPayoutDetails('');
        await loadAll();
        alert('Withdrawal request submitted for admin approval!');
      }
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <div className="p-10 flex items-center justify-center gap-2 text-slate-500 text-sm">
        <RefreshCw className="w-4 h-4 animate-spin" />
        <span>Loading Credit Wallet &amp; Affiliate Hub...</span>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-extrabold text-slate-900 dark:text-white">Credit Wallet &amp; Affiliate Partner Hub</h1>
        <p className="text-xs text-slate-500 mt-1">Manage your double-entry hosting credit balance, Auto-Pay renewals, and 15% recurring referral commissions.</p>
      </div>

      {/* Credit Wallet Card */}
      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-3xl p-6 shadow-sm flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-4">
          <div className="w-12 h-12 rounded-2xl bg-emerald-500/10 text-emerald-600 flex items-center justify-center">
            <Wallet className="w-6 h-6" />
          </div>
          <div>
            <div className="text-xs uppercase font-bold text-slate-500">Account Credit Balance</div>
            <div className="text-3xl font-extrabold font-mono text-slate-900 dark:text-white">
              ${Number(walletData?.wallet?.balance_usd || 0).toFixed(2)} <span className="text-sm font-sans text-slate-400">USD</span>
            </div>
          </div>
        </div>
        <button
          onClick={handleToggleAutoPay}
          className={`px-4 py-2.5 rounded-xl text-xs font-bold transition cursor-pointer flex items-center gap-2 ${
            walletData?.wallet?.auto_pay_enabled
              ? 'bg-emerald-600 text-white hover:bg-emerald-500'
              : 'bg-slate-200 dark:bg-slate-800 text-slate-700 dark:text-slate-200'
          }`}
        >
          <ShieldCheck className="w-4 h-4" />
          <span>Auto-Pay Renewals: {walletData?.wallet?.auto_pay_enabled ? 'ENABLED' : 'DISABLED'}</span>
        </button>
      </div>

      {/* Affiliate KPIs */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div className="p-5 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl">
          <div className="text-xs text-slate-500 font-bold uppercase flex items-center gap-1.5"><Users className="w-3.5 h-3.5" /> Total Referrals</div>
          <div className="text-2xl font-extrabold mt-1">{affData?.account?.total_referrals || 0}</div>
        </div>
        <div className="p-5 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl">
          <div className="text-xs text-slate-500 font-bold uppercase flex items-center gap-1.5"><DollarSign className="w-3.5 h-3.5 text-emerald-500" /> Approved Commission</div>
          <div className="text-2xl font-extrabold text-emerald-600 mt-1">${Number(affData?.account?.approved_balance_usd || 0).toFixed(2)}</div>
        </div>
        <div className="p-5 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl">
          <div className="text-xs text-slate-500 font-bold uppercase flex items-center gap-1.5"><ArrowUpRight className="w-3.5 h-3.5 text-indigo-500" /> Total Paid Out</div>
          <div className="text-2xl font-extrabold mt-1">${Number(affData?.account?.total_paid_usd || 0).toFixed(2)}</div>
        </div>
      </div>

      {/* Referral Link + Withdrawal Form */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-3xl p-6 space-y-3">
          <h2 className="text-sm font-bold">Your Unique Referral Link ({affData?.account?.commission_rate_pct || 15}% Commission)</h2>
          <div className="flex gap-2">
            <input
              readOnly
              value={affData?.referral_link || ''}
              className="flex-1 p-2.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 font-mono text-xs"
            />
            <button onClick={copyRefLink} className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold cursor-pointer flex items-center gap-1.5">
              {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
              <span>{copied ? 'Copied' : 'Copy'}</span>
            </button>
          </div>
          <p className="text-xs text-slate-500">Share this link with friends or clients. Verified paid orders earn automatic commissions. Self-referrals are blocked.</p>
        </div>

        <form onSubmit={handleWithdraw} className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-3xl p-6 space-y-3">
          <h2 className="text-sm font-bold">Request Commission Payout</h2>
          <div className="grid grid-cols-2 gap-2">
            <input
              type="number"
              step="0.01"
              placeholder="Amount USD"
              value={withdrawAmt}
              onChange={(e) => setWithdrawAmt(e.target.value)}
              className="p-2.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-xs"
            />
            <select
              value={payoutMethod}
              onChange={(e) => setPayoutMethod(e.target.value)}
              className="p-2.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-xs"
            >
              <option value="credit_wallet">Hosting Credit Wallet (Instant)</option>
              <option value="usdt_bep20">USDT (BEP20)</option>
              <option value="usdt_trc20">USDT (TRC20)</option>
            </select>
          </div>
          {payoutMethod !== 'credit_wallet' && (
            <input
              type="text"
              placeholder="Receiving Wallet Address"
              value={payoutDetails}
              onChange={(e) => setPayoutDetails(e.target.value)}
              className="w-full p-2.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 font-mono text-xs"
            />
          )}
          <button
            type="submit"
            disabled={submitting || !withdrawAmt}
            className="w-full py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white text-xs font-bold cursor-pointer"
          >
            {submitting ? 'Submitting...' : 'Submit Payout Request'}
          </button>
        </form>
      </div>
    </div>
  );
}
