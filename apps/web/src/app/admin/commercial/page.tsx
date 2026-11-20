'use client';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch } from '@/lib/api';
import {
  Tag,
  Users,
  Plus,
  Edit2,
  Trash2,
  CheckCircle2,
  XCircle,
  RefreshCw,
  Search,
  DollarSign,
  Play,
  Wallet,
  Percent,
  Clock,
  ShieldCheck,
  AlertCircle,
  X,
} from 'lucide-react';

interface Coupon {
  id: string;
  code: string;
  description: string;
  discount_type: 'percentage' | 'fixed';
  discount_value: number;
  max_uses: number;
  current_uses: number;
  max_uses_per_user: number;
  min_order_usd: number;
  first_order_only: boolean;
  recurring_discount: boolean;
  applicable_plan_ids?: string;
  is_active: boolean;
  expires_at?: string;
  created_at: string;
}

interface AffiliateAccount {
  id: string;
  user_id: string;
  referral_code: string;
  commission_rate_pct: number;
  pending_balance_usd: number;
  approved_balance_usd: number;
  total_paid_usd: number;
  total_referrals: number;
  status: string;
  payout_method: string;
  created_at: string;
}

interface AffiliateWithdrawal {
  id: string;
  affiliate_id: string;
  user_id: string;
  amount_usd: number;
  payout_method: string;
  payout_details?: string;
  status: 'pending' | 'approved' | 'rejected' | 'paid';
  admin_notes?: string;
  created_at: string;
  processed_at?: string;
}

const emptyForm = {
  code: '',
  description: '',
  discount_type: 'percentage' as 'percentage' | 'fixed',
  discount_value: '20',
  max_uses: '0',
  max_uses_per_user: '1',
  min_order_usd: '0',
  first_order_only: false,
  recurring_discount: false,
  is_active: true,
  expires_at: '',
};

export default function AdminCommercialPage() {
  const [activeTab, setActiveTab] = useState<'coupons' | 'affiliates' | 'ops'>('coupons');
  const [coupons, setCoupons] = useState<Coupon[]>([]);
  const [affiliates, setAffiliates] = useState<AffiliateAccount[]>([]);
  const [withdrawals, setWithdrawals] = useState<AffiliateWithdrawal[]>([]);
  const [loading, setLoading] = useState(true);
  const [banner, setBanner] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  // Coupon filters & modal
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'expired'>('all');
  const [showModal, setShowModal] = useState(false);
  const [editingCoupon, setEditingCoupon] = useState<Coupon | null>(null);
  const [form, setForm] = useState(emptyForm);
  const [saving, setSaving] = useState(false);

  // Withdrawal processing state
  const [processingId, setProcessingId] = useState<string | null>(null);
  const [adminNotesMap, setAdminNotesMap] = useState<Record<string, string>>({});

  // Dunning & Wallet Ops
  const [runningDunning, setRunningDunning] = useState(false);
  const [dunningResult, setDunningResult] = useState<any>(null);
  const [walletUserId, setWalletUserId] = useState('');
  const [walletDir, setWalletDir] = useState<'credit' | 'debit'>('credit');
  const [walletAmount, setWalletAmount] = useState('');
  const [walletNotes, setWalletNotes] = useState('');
  const [adjustingWallet, setAdjustingWallet] = useState(false);

  const loadAll = useCallback(async () => {
    setLoading(true);
    try {
      const [cRes, aRes] = await Promise.all([
        apiFetch<Coupon[]>('/api/v1/admin/commercial/coupons'),
        apiFetch<{ accounts: AffiliateAccount[]; withdrawals: AffiliateWithdrawal[] }>('/api/v1/admin/commercial/affiliates'),
      ]);
      if (cRes.data) setCoupons(Array.isArray(cRes.data) ? cRes.data : []);
      if (aRes.data) {
        setAffiliates(Array.isArray(aRes.data.accounts) ? aRes.data.accounts : []);
        setWithdrawals(Array.isArray(aRes.data.withdrawals) ? aRes.data.withdrawals : []);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadAll();
  }, [loadAll]);

  const isExpired = (c: Coupon) => {
    if (!c.is_active) return true;
    if (c.expires_at && new Date(c.expires_at).getTime() < Date.now()) return true;
    if (c.max_uses > 0 && c.current_uses >= c.max_uses) return true;
    return false;
  };

  const couponKPIs = useMemo(() => {
    const total = coupons.length;
    const active = coupons.filter((c) => !isExpired(c)).length;
    const expired = total - active;
    const redemptions = coupons.reduce((sum, c) => sum + (c.current_uses || 0), 0);
    return { total, active, expired, redemptions };
  }, [coupons]);

  const affiliateKPIs = useMemo(() => {
    const totalAffiliates = affiliates.length;
    const totalReferrals = affiliates.reduce((s, a) => s + (a.total_referrals || 0), 0);
    const approvedBalance = affiliates.reduce((s, a) => s + (a.approved_balance_usd || 0), 0);
    const totalPaid = affiliates.reduce((s, a) => s + (a.total_paid_usd || 0), 0);
    const pendingWithdrawals = withdrawals.filter((w) => w.status === 'pending').length;
    return { totalAffiliates, totalReferrals, approvedBalance, totalPaid, pendingWithdrawals };
  }, [affiliates, withdrawals]);

  const filteredCoupons = useMemo(() => {
    return coupons.filter((c) => {
      const q = search.trim().toLowerCase();
      if (q && !c.code.toLowerCase().includes(q) && !(c.description || '').toLowerCase().includes(q)) {
        return false;
      }
      if (statusFilter === 'active' && isExpired(c)) return false;
      if (statusFilter === 'expired' && !isExpired(c)) return false;
      return true;
    });
  }, [coupons, search, statusFilter]);

  const openCreateModal = () => {
    setEditingCoupon(null);
    setForm(emptyForm);
    setShowModal(true);
  };

  const openEditModal = (c: Coupon) => {
    setEditingCoupon(c);
    setForm({
      code: c.code,
      description: c.description || '',
      discount_type: c.discount_type || 'percentage',
      discount_value: String(c.discount_value),
      max_uses: String(c.max_uses || 0),
      max_uses_per_user: String(c.max_uses_per_user || 1),
      min_order_usd: String(c.min_order_usd || 0),
      first_order_only: Boolean(c.first_order_only),
      recurring_discount: Boolean(c.recurring_discount),
      is_active: Boolean(c.is_active),
      expires_at: c.expires_at ? c.expires_at.slice(0, 16) : '',
    });
    setShowModal(true);
  };

  const handleSaveCoupon = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setBanner(null);
    try {
      const payload: any = {
        code: form.code.trim().toUpperCase(),
        description: form.description.trim(),
        discount_type: form.discount_type,
        discount_value: parseFloat(form.discount_value) || 0,
        max_uses: parseInt(form.max_uses, 10) || 0,
        max_uses_per_user: parseInt(form.max_uses_per_user, 10) || 1,
        min_order_usd: parseFloat(form.min_order_usd) || 0,
        first_order_only: form.first_order_only,
        recurring_discount: form.recurring_discount,
        is_active: form.is_active,
        expires_at: form.expires_at ? new Date(form.expires_at).toISOString() : null,
      };
      const url = editingCoupon
        ? `/api/v1/admin/commercial/coupons/${editingCoupon.id}`
        : '/api/v1/admin/commercial/coupons';
      const method = editingCoupon ? 'PUT' : 'POST';
      const res = await apiFetch<Coupon>(url, { method, body: JSON.stringify(payload) });
      if (res.error) {
        setBanner({ type: 'error', text: res.error.message || 'Failed to save coupon' });
      } else {
        setShowModal(false);
        setBanner({ type: 'success', text: `Coupon ${payload.code} ${editingCoupon ? 'updated' : 'created'} successfully.` });
        await loadAll();
      }
    } finally {
      setSaving(false);
    }
  };

  const handleToggleActive = async (c: Coupon) => {
    const res = await apiFetch<Coupon>(`/api/v1/admin/commercial/coupons/${c.id}`, {
      method: 'PUT',
      body: JSON.stringify({ ...c, is_active: !c.is_active }),
    });
    if (res.error) setBanner({ type: 'error', text: res.error.message || 'Toggle failed' });
    else {
      setBanner({ type: 'success', text: `Coupon ${c.code} ${!c.is_active ? 'activated' : 'deactivated'}.` });
      await loadAll();
    }
  };

  const handleDeleteCoupon = async (c: Coupon) => {
    if (!confirm(`Delete coupon ${c.code}? This cannot be undone.`)) return;
    const res = await apiFetch<any>(`/api/v1/admin/commercial/coupons/${c.id}`, { method: 'DELETE' });
    if (res.error) setBanner({ type: 'error', text: res.error.message || 'Delete failed' });
    else {
      setBanner({ type: 'success', text: `Coupon ${c.code} deleted.` });
      await loadAll();
    }
  };

  const handleProcessWithdrawal = async (w: AffiliateWithdrawal, approve: boolean) => {
    const actionLabel = approve ? 'APPROVE & PAY' : 'REJECT & REFUND TO AFFILIATE BALANCE';
    if (!confirm(`${actionLabel} withdrawal of $${w.amount_usd.toFixed(2)} via ${w.payout_method}?`)) return;
    setProcessingId(w.id);
    setBanner(null);
    try {
      const res = await apiFetch<AffiliateWithdrawal>(`/api/v1/admin/commercial/withdrawals/${w.id}/process`, {
        method: 'POST',
        body: JSON.stringify({
          approve,
          admin_notes: adminNotesMap[w.id] || (approve ? 'Approved by admin' : 'Rejected by admin'),
        }),
      });
      if (res.error) setBanner({ type: 'error', text: res.error.message || 'Failed to process withdrawal' });
      else {
        setBanner({
          type: 'success',
          text: `Withdrawal $${w.amount_usd.toFixed(2)} marked ${approve ? 'PAID' : 'REJECTED'}.`,
        });
        await loadAll();
      }
    } finally {
      setProcessingId(null);
    }
  };

  const handleRunDunning = async () => {
    setRunningDunning(true);
    setBanner(null);
    try {
      const res = await apiFetch<any>('/api/v1/admin/commercial/dunning/run', { method: 'POST' });
      if (res.error) setBanner({ type: 'error', text: res.error.message || 'Dunning pass failed' });
      else {
        setDunningResult(res.data);
        setBanner({ type: 'success', text: 'Billing dunning pass completed.' });
      }
    } finally {
      setRunningDunning(false);
    }
  };

  const handleAdjustWallet = async (e: React.FormEvent) => {
    e.preventDefault();
    setAdjustingWallet(true);
    setBanner(null);
    try {
      const res = await apiFetch<any>('/api/v1/admin/commercial/wallet/adjust', {
        method: 'POST',
        body: JSON.stringify({
          user_id: walletUserId.trim(),
          direction: walletDir,
          amount_usd: parseFloat(walletAmount) || 0,
          notes: walletNotes.trim() || 'Admin manual adjustment',
        }),
      });
      if (res.error) setBanner({ type: 'error', text: res.error.message || 'Wallet adjustment failed' });
      else {
        setBanner({ type: 'success', text: `Wallet ${walletDir} of $${Number(walletAmount).toFixed(2)} recorded.` });
        setWalletAmount('');
        setWalletNotes('');
      }
    } finally {
      setAdjustingWallet(false);
    }
  };

  return (
    <DashboardShell>
      <div className="p-6 max-w-7xl mx-auto space-y-6">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-extrabold text-slate-900 dark:text-white flex items-center gap-2">
              <Tag className="w-6 h-6 text-emerald-600" />
              <span>Commercial Management — Coupons, Affiliates &amp; Billing Ops</span>
            </h1>
            <p className="text-xs text-slate-500 mt-1">
              Manage promotional coupons, partner referral payouts, automated dunning, and customer wallet adjustments.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={loadAll}
              className="px-3 py-2 rounded-xl border border-slate-200 dark:border-slate-700 text-xs font-bold flex items-center gap-1.5 hover:bg-slate-100 dark:hover:bg-slate-800 cursor-pointer"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
              <span>Refresh</span>
            </button>
            {activeTab === 'coupons' && (
              <button
                onClick={openCreateModal}
                className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold flex items-center gap-1.5 cursor-pointer shadow-sm"
              >
                <Plus className="w-4 h-4" />
                <span>Create Coupon</span>
              </button>
            )}
          </div>
        </div>

        {banner && (
          <div
            className={`p-3.5 rounded-2xl text-xs font-semibold flex items-center justify-between ${
              banner.type === 'success'
                ? 'bg-emerald-50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-500/30'
                : 'bg-rose-50 dark:bg-rose-500/10 text-rose-700 dark:text-rose-300 border border-rose-200 dark:border-rose-500/30'
            }`}
          >
            <span className="flex items-center gap-2">
              {banner.type === 'success' ? <CheckCircle2 className="w-4 h-4" /> : <AlertCircle className="w-4 h-4" />}
              {banner.text}
            </span>
            <button onClick={() => setBanner(null)} className="cursor-pointer">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {/* Tabs */}
        <div className="flex gap-2 border-b border-slate-200 dark:border-slate-800 pb-2">
          <button
            onClick={() => setActiveTab('coupons')}
            className={`px-4 py-2 rounded-xl text-xs font-bold cursor-pointer flex items-center gap-2 ${
              activeTab === 'coupons'
                ? 'bg-emerald-600 text-white'
                : 'text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800'
            }`}
          >
            <Percent className="w-4 h-4" />
            <span>Coupons ({coupons.length})</span>
          </button>
          <button
            onClick={() => setActiveTab('affiliates')}
            className={`px-4 py-2 rounded-xl text-xs font-bold cursor-pointer flex items-center gap-2 ${
              activeTab === 'affiliates'
                ? 'bg-emerald-600 text-white'
                : 'text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800'
            }`}
          >
            <Users className="w-4 h-4" />
            <span>Affiliates &amp; Payouts</span>
            {affiliateKPIs.pendingWithdrawals > 0 && (
              <span className="px-1.5 py-0.5 rounded-full bg-amber-500 text-white text-[10px]">
                {affiliateKPIs.pendingWithdrawals}
              </span>
            )}
          </button>
          <button
            onClick={() => setActiveTab('ops')}
            className={`px-4 py-2 rounded-xl text-xs font-bold cursor-pointer flex items-center gap-2 ${
              activeTab === 'ops'
                ? 'bg-emerald-600 text-white'
                : 'text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800'
            }`}
          >
            <Wallet className="w-4 h-4" />
            <span>Dunning &amp; Wallet Ops</span>
          </button>
        </div>

        {/* TAB 1: COUPONS */}
        {activeTab === 'coupons' && (
          <div className="space-y-5">
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
              <div className="p-4 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800">
                <div className="text-[11px] font-bold uppercase text-slate-500">Total Coupons</div>
                <div className="text-2xl font-extrabold mt-1">{couponKPIs.total}</div>
              </div>
              <div className="p-4 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800">
                <div className="text-[11px] font-bold uppercase text-slate-500">Active Coupons</div>
                <div className="text-2xl font-extrabold text-emerald-600 mt-1">{couponKPIs.active}</div>
              </div>
              <div className="p-4 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800">
                <div className="text-[11px] font-bold uppercase text-slate-500">Expired / Inactive</div>
                <div className="text-2xl font-extrabold text-amber-600 mt-1">{couponKPIs.expired}</div>
              </div>
              <div className="p-4 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800">
                <div className="text-[11px] font-bold uppercase text-slate-500">Total Redemptions</div>
                <div className="text-2xl font-extrabold text-indigo-600 mt-1">{couponKPIs.redemptions}</div>
              </div>
            </div>

            <div className="flex flex-wrap items-center gap-3">
              <div className="relative flex-1 min-w-[220px]">
                <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
                <input
                  type="text"
                  placeholder="Search coupon code or description..."
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="w-full pl-9 pr-3 py-2 rounded-xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 text-xs"
                />
              </div>
              <div className="flex gap-1.5">
                {(['all', 'active', 'expired'] as const).map((f) => (
                  <button
                    key={f}
                    onClick={() => setStatusFilter(f)}
                    className={`px-3 py-2 rounded-xl text-xs font-bold capitalize cursor-pointer ${
                      statusFilter === f
                        ? 'bg-slate-900 dark:bg-slate-100 text-white dark:text-slate-900'
                        : 'bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 text-slate-600 dark:text-slate-300'
                    }`}
                  >
                    {f}
                  </button>
                ))}
              </div>
            </div>

            <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl overflow-x-auto">
              <table className="w-full text-xs text-left">
                <thead className="bg-slate-50 dark:bg-slate-800/60 text-slate-500 uppercase text-[10px]">
                  <tr>
                    <th className="p-3.5">Code</th>
                    <th className="p-3.5">Discount</th>
                    <th className="p-3.5">Usage</th>
                    <th className="p-3.5">Rules</th>
                    <th className="p-3.5">Status</th>
                    <th className="p-3.5">Expires</th>
                    <th className="p-3.5 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                  {filteredCoupons.length === 0 ? (
                    <tr>
                      <td colSpan={7} className="p-8 text-center text-slate-400">
                        No coupons match your filter. Click &quot;Create Coupon&quot; to add one.
                      </td>
                    </tr>
                  ) : (
                    filteredCoupons.map((c) => {
                      const expired = isExpired(c);
                      return (
                        <tr key={c.id} className="hover:bg-slate-50/70 dark:hover:bg-slate-800/40">
                          <td className="p-3.5">
                            <div className="font-mono font-extrabold text-slate-900 dark:text-white">{c.code}</div>
                            <div className="text-[11px] text-slate-500">{c.description || '—'}</div>
                          </td>
                          <td className="p-3.5 font-mono font-bold text-emerald-600">
                            {c.discount_type === 'fixed' ? `$${c.discount_value.toFixed(2)} OFF` : `${c.discount_value}% OFF`}
                          </td>
                          <td className="p-3.5 font-mono">
                            {c.current_uses} / {c.max_uses > 0 ? c.max_uses : '∞'}
                            <div className="text-[10px] text-slate-400">Max {c.max_uses_per_user}/user</div>
                          </td>
                          <td className="p-3.5 space-x-1">
                            {c.min_order_usd > 0 && (
                              <span className="px-1.5 py-0.5 rounded bg-slate-100 dark:bg-slate-800 text-[10px]">
                                Min ${c.min_order_usd}
                              </span>
                            )}
                            {c.first_order_only && (
                              <span className="px-1.5 py-0.5 rounded bg-blue-100 dark:bg-blue-500/20 text-blue-700 dark:text-blue-300 text-[10px]">
                                First Order
                              </span>
                            )}
                            {c.recurring_discount && (
                              <span className="px-1.5 py-0.5 rounded bg-purple-100 dark:bg-purple-500/20 text-purple-700 dark:text-purple-300 text-[10px]">
                                Recurring
                              </span>
                            )}
                            {!c.min_order_usd && !c.first_order_only && !c.recurring_discount && (
                              <span className="text-slate-400">Standard</span>
                            )}
                          </td>
                          <td className="p-3.5">
                            <button
                              onClick={() => handleToggleActive(c)}
                              className={`px-2 py-0.5 rounded-full text-[10px] font-bold uppercase cursor-pointer ${
                                !expired
                                  ? 'bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-300'
                                  : 'bg-slate-200 dark:bg-slate-700 text-slate-600 dark:text-slate-300'
                              }`}
                            >
                              {!expired ? 'Active' : c.is_active ? 'Expired/Used' : 'Inactive'}
                            </button>
                          </td>
                          <td className="p-3.5 text-slate-500">
                            {c.expires_at ? new Date(c.expires_at).toLocaleDateString() : 'Never'}
                          </td>
                          <td className="p-3.5 text-right space-x-1.5">
                            <button
                              onClick={() => openEditModal(c)}
                              className="p-1.5 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-600 dark:text-slate-300 cursor-pointer"
                              title="Edit coupon"
                            >
                              <Edit2 className="w-3.5 h-3.5" />
                            </button>
                            <button
                              onClick={() => handleDeleteCoupon(c)}
                              className="p-1.5 rounded-lg hover:bg-rose-50 dark:hover:bg-rose-500/10 text-rose-600 cursor-pointer"
                              title="Delete coupon"
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
                          </td>
                        </tr>
                      );
                    })
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* TAB 2: AFFILIATES & WITHDRAWALS */}
        {activeTab === 'affiliates' && (
          <div className="space-y-6">
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
              <div className="p-4 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800">
                <div className="text-[11px] font-bold uppercase text-slate-500">Total Affiliates</div>
                <div className="text-2xl font-extrabold mt-1">{affiliateKPIs.totalAffiliates}</div>
                <div className="text-[11px] text-slate-400">{affiliateKPIs.totalReferrals} total referrals</div>
              </div>
              <div className="p-4 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800">
                <div className="text-[11px] font-bold uppercase text-slate-500">Approved Unpaid Balance</div>
                <div className="text-2xl font-extrabold text-emerald-600 mt-1">
                  ${affiliateKPIs.approvedBalance.toFixed(2)}
                </div>
              </div>
              <div className="p-4 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800">
                <div className="text-[11px] font-bold uppercase text-slate-500">Total Paid Out</div>
                <div className="text-2xl font-extrabold text-indigo-600 mt-1">${affiliateKPIs.totalPaid.toFixed(2)}</div>
              </div>
              <div className="p-4 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800">
                <div className="text-[11px] font-bold uppercase text-slate-500">Pending Withdrawals</div>
                <div className="text-2xl font-extrabold text-amber-600 mt-1">{affiliateKPIs.pendingWithdrawals}</div>
              </div>
            </div>

            {/* Withdrawal Requests */}
            <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl overflow-x-auto">
              <div className="p-4 border-b border-slate-100 dark:border-slate-800 font-bold text-sm flex items-center gap-2">
                <DollarSign className="w-4 h-4 text-emerald-600" />
                <span>Affiliate Withdrawal Requests</span>
              </div>
              <table className="w-full text-xs text-left">
                <thead className="bg-slate-50 dark:bg-slate-800/60 text-slate-500 uppercase text-[10px]">
                  <tr>
                    <th className="p-3.5">Date</th>
                    <th className="p-3.5">Amount</th>
                    <th className="p-3.5">Method &amp; Destination</th>
                    <th className="p-3.5">Status</th>
                    <th className="p-3.5">Admin Notes</th>
                    <th className="p-3.5 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                  {withdrawals.length === 0 ? (
                    <tr>
                      <td colSpan={6} className="p-6 text-center text-slate-400">
                        No withdrawal requests yet.
                      </td>
                    </tr>
                  ) : (
                    withdrawals.map((w) => (
                      <tr key={w.id}>
                        <td className="p-3.5 text-slate-500">{new Date(w.created_at).toLocaleString()}</td>
                        <td className="p-3.5 font-mono font-extrabold text-slate-900 dark:text-white">
                          ${w.amount_usd.toFixed(2)}
                        </td>
                        <td className="p-3.5">
                          <div className="font-bold uppercase text-[11px]">{w.payout_method}</div>
                          <div className="font-mono text-[10px] text-slate-500 break-all">{w.payout_details || 'Credit Wallet'}</div>
                        </td>
                        <td className="p-3.5">
                          <span
                            className={`px-2 py-0.5 rounded-full text-[10px] font-bold uppercase ${
                              w.status === 'paid' || w.status === 'approved'
                                ? 'bg-emerald-100 text-emerald-700'
                                : w.status === 'rejected'
                                ? 'bg-rose-100 text-rose-700'
                                : 'bg-amber-100 text-amber-700'
                            }`}
                          >
                            {w.status}
                          </span>
                        </td>
                        <td className="p-3.5">
                          {w.status === 'pending' ? (
                            <input
                              type="text"
                              placeholder="Optional note / TxHash..."
                              value={adminNotesMap[w.id] || ''}
                              onChange={(e) => setAdminNotesMap((p) => ({ ...p, [w.id]: e.target.value }))}
                              className="w-full p-1.5 rounded-lg bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-700 text-xs"
                            />
                          ) : (
                            <span className="text-slate-500">{w.admin_notes || '—'}</span>
                          )}
                        </td>
                        <td className="p-3.5 text-right">
                          {w.status === 'pending' ? (
                            <div className="flex justify-end gap-1.5">
                              <button
                                disabled={processingId === w.id}
                                onClick={() => handleProcessWithdrawal(w, true)}
                                className="px-2.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-[11px] cursor-pointer flex items-center gap-1 disabled:opacity-50"
                              >
                                <CheckCircle2 className="w-3.5 h-3.5" />
                                <span>Approve</span>
                              </button>
                              <button
                                disabled={processingId === w.id}
                                onClick={() => handleProcessWithdrawal(w, false)}
                                className="px-2.5 py-1.5 rounded-lg bg-rose-600 hover:bg-rose-500 text-white font-bold text-[11px] cursor-pointer flex items-center gap-1 disabled:opacity-50"
                              >
                                <XCircle className="w-3.5 h-3.5" />
                                <span>Reject</span>
                              </button>
                            </div>
                          ) : (
                            <span className="text-[11px] text-slate-400">Processed</span>
                          )}
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>

            {/* Affiliate Accounts Directory */}
            <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl overflow-x-auto">
              <div className="p-4 border-b border-slate-100 dark:border-slate-800 font-bold text-sm flex items-center gap-2">
                <Users className="w-4 h-4 text-indigo-500" />
                <span>Registered Affiliate Partners ({affiliates.length})</span>
              </div>
              <table className="w-full text-xs text-left">
                <thead className="bg-slate-50 dark:bg-slate-800/60 text-slate-500 uppercase text-[10px]">
                  <tr>
                    <th className="p-3.5">Referral Code</th>
                    <th className="p-3.5">User ID</th>
                    <th className="p-3.5">Rate</th>
                    <th className="p-3.5">Referrals</th>
                    <th className="p-3.5">Approved Balance</th>
                    <th className="p-3.5">Total Paid</th>
                    <th className="p-3.5">Status</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                  {affiliates.length === 0 ? (
                    <tr>
                      <td colSpan={7} className="p-6 text-center text-slate-400">
                        No affiliate accounts registered yet.
                      </td>
                    </tr>
                  ) : (
                    affiliates.map((a) => (
                      <tr key={a.id}>
                        <td className="p-3.5 font-mono font-bold text-emerald-600">{a.referral_code}</td>
                        <td className="p-3.5 font-mono text-[11px] text-slate-500">{a.user_id}</td>
                        <td className="p-3.5 font-bold">{a.commission_rate_pct}%</td>
                        <td className="p-3.5 font-bold">{a.total_referrals}</td>
                        <td className="p-3.5 font-mono font-bold text-emerald-600">${a.approved_balance_usd.toFixed(2)}</td>
                        <td className="p-3.5 font-mono">${a.total_paid_usd.toFixed(2)}</td>
                        <td className="p-3.5">
                          <span className="px-2 py-0.5 rounded-full bg-emerald-100 text-emerald-700 text-[10px] font-bold uppercase">
                            {a.status}
                          </span>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* TAB 3: DUNNING & WALLET OPS */}
        {activeTab === 'ops' && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl p-6 space-y-4">
              <div className="flex items-center gap-2 font-bold text-sm">
                <Clock className="w-4 h-4 text-emerald-600" />
                <span>Automated Subscription Dunning Engine</span>
              </div>
              <p className="text-xs text-slate-500">
                Manually trigger a dunning evaluation pass to generate upcoming renewal invoices, execute Auto-Pay from customer credit wallets, and suspend overdue subscriptions past grace period.
              </p>
              <button
                onClick={handleRunDunning}
                disabled={runningDunning}
                className="px-4 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white text-xs font-bold flex items-center gap-2 cursor-pointer"
              >
                <Play className="w-3.5 h-3.5" />
                <span>{runningDunning ? 'Running Dunning Pass...' : 'Run Dunning Pass Now'}</span>
              </button>
              {dunningResult && (
                <div className="p-4 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-xs font-mono space-y-1">
                  <div>Renewals Generated: {dunningResult.renewals_generated ?? 0}</div>
                  <div>Auto-Paid From Wallet: {dunningResult.auto_paid_from_wallet ?? 0}</div>
                  <div>Overdue Suspended: {dunningResult.suspended_overdue ?? 0}</div>
                  <div>Notices Dispatched: {dunningResult.notices_sent ?? 0}</div>
                </div>
              )}
            </div>

            <form
              onSubmit={handleAdjustWallet}
              className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl p-6 space-y-3"
            >
              <div className="flex items-center gap-2 font-bold text-sm">
                <ShieldCheck className="w-4 h-4 text-indigo-500" />
                <span>Admin Customer Credit Wallet Adjustment</span>
              </div>
              <input
                type="text"
                required
                placeholder="Customer User UUID"
                value={walletUserId}
                onChange={(e) => setWalletUserId(e.target.value)}
                className="w-full p-2.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-xs font-mono"
              />
              <div className="grid grid-cols-2 gap-2">
                <select
                  value={walletDir}
                  onChange={(e) => setWalletDir(e.target.value as 'credit' | 'debit')}
                  className="p-2.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-xs"
                >
                  <option value="credit">Credit (+USD)</option>
                  <option value="debit">Debit (-USD)</option>
                </select>
                <input
                  type="number"
                  step="0.01"
                  required
                  placeholder="Amount USD"
                  value={walletAmount}
                  onChange={(e) => setWalletAmount(e.target.value)}
                  className="p-2.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-xs"
                />
              </div>
              <input
                type="text"
                placeholder="Audit reason / notes"
                value={walletNotes}
                onChange={(e) => setWalletNotes(e.target.value)}
                className="w-full p-2.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-xs"
              />
              <button
                type="submit"
                disabled={adjustingWallet}
                className="w-full py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white text-xs font-bold cursor-pointer"
              >
                {adjustingWallet ? 'Applying Adjustment...' : 'Apply Double-Entry Adjustment'}
              </button>
            </form>
          </div>
        )}

        {/* CREATE / EDIT COUPON MODAL */}
        {showModal && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4">
            <form
              onSubmit={handleSaveCoupon}
              className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-3xl p-6 w-full max-w-lg space-y-4 shadow-2xl"
            >
              <div className="flex items-center justify-between">
                <h2 className="text-base font-extrabold">{editingCoupon ? `Edit Coupon ${editingCoupon.code}` : 'Create Promotional Coupon'}</h2>
                <button type="button" onClick={() => setShowModal(false)} className="cursor-pointer text-slate-400">
                  <X className="w-5 h-5" />
                </button>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-[11px] font-bold text-slate-500 uppercase">Coupon Code</label>
                  <input
                    type="text"
                    required
                    value={form.code}
                    onChange={(e) => setForm({ ...form, code: e.target.value.toUpperCase() })}
                    placeholder="LAUNCH25"
                    className="w-full mt-1 p-2.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-xs font-mono uppercase"
                  />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-slate-500 uppercase">Discount Type</label>
                  <select
                    value={form.discount_type}
                    onChange={(e) => setForm({ ...form, discount_type: e.target.value as 'percentage' | 'fixed' })}
                    className="w-full mt-1 p-2.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-xs"
                  >
                    <option value="percentage">Percentage (%)</option>
                    <option value="fixed">Fixed USD ($)</option>
                  </select>
                </div>
              </div>
              <div>
                <label className="text-[11px] font-bold text-slate-500 uppercase">Description</label>
                <input
                  type="text"
                  value={form.description}
                  onChange={(e) => setForm({ ...form, description: e.target.value })}
                  placeholder="25% off first cloud hosting order"
                  className="w-full mt-1 p-2.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-xs"
                />
              </div>
              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label className="text-[11px] font-bold text-slate-500 uppercase">Value</label>
                  <input
                    type="number"
                    step="0.01"
                    required
                    value={form.discount_value}
                    onChange={(e) => setForm({ ...form, discount_value: e.target.value })}
                    className="w-full mt-1 p-2.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-xs"
                  />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-slate-500 uppercase">Max Uses (0=∞)</label>
                  <input
                    type="number"
                    value={form.max_uses}
                    onChange={(e) => setForm({ ...form, max_uses: e.target.value })}
                    className="w-full mt-1 p-2.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-xs"
                  />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-slate-500 uppercase">Per Customer</label>
                  <input
                    type="number"
                    value={form.max_uses_per_user}
                    onChange={(e) => setForm({ ...form, max_uses_per_user: e.target.value })}
                    className="w-full mt-1 p-2.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-xs"
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-[11px] font-bold text-slate-500 uppercase">Min Order USD</label>
                  <input
                    type="number"
                    step="0.01"
                    value={form.min_order_usd}
                    onChange={(e) => setForm({ ...form, min_order_usd: e.target.value })}
                    className="w-full mt-1 p-2.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-xs"
                  />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-slate-500 uppercase">Expires At (Optional)</label>
                  <input
                    type="datetime-local"
                    value={form.expires_at}
                    onChange={(e) => setForm({ ...form, expires_at: e.target.value })}
                    className="w-full mt-1 p-2.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-xs"
                  />
                </div>
              </div>
              <div className="flex flex-wrap gap-4 text-xs pt-1">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={form.first_order_only}
                    onChange={(e) => setForm({ ...form, first_order_only: e.target.checked })}
                  />
                  <span>First order only</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={form.recurring_discount}
                    onChange={(e) => setForm({ ...form, recurring_discount: e.target.checked })}
                  />
                  <span>Recurring discount</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={form.is_active}
                    onChange={(e) => setForm({ ...form, is_active: e.target.checked })}
                  />
                  <span>Active</span>
                </label>
              </div>
              <div className="flex justify-end gap-2 pt-3 border-t border-slate-100 dark:border-slate-800">
                <button
                  type="button"
                  onClick={() => setShowModal(false)}
                  className="px-4 py-2 rounded-xl border border-slate-200 dark:border-slate-700 text-xs font-bold cursor-pointer"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={saving}
                  className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white text-xs font-bold cursor-pointer"
                >
                  {saving ? 'Saving...' : editingCoupon ? 'Update Coupon' : 'Create Coupon'}
                </button>
              </div>
            </form>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
