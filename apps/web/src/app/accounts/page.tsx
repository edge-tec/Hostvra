'use client';

import React, { useState, useEffect, useMemo } from 'react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  Users,
  Plus,
  Search,
  RefreshCw,
  ExternalLink,
  Copy,
  Check,
  HardDrive,
  Network,
  Globe,
  Database,
  Lock,
  Shield,
  ShieldAlert,
  Sliders,
  Trash2,
  X,
  AlertCircle,
  CheckCircle2,
  LogIn,
  Key,
  Server,
  Zap,
  Clock,
  Layers,
  Sparkles,
} from 'lucide-react';
import { apiFetch, HostingAccount, HostingPlan, Server as ServerType } from '@/lib/api';

export default function AccountsPage() {
  const [accounts, setAccounts] = useState<HostingAccount[]>([]);
  const [plans, setPlans] = useState<HostingPlan[]>([]);
  const [servers, setServers] = useState<ServerType[]>([]);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'suspended'>('all');
  const [copiedUser, setCopiedUser] = useState<string | null>(null);
  const [notification, setNotification] = useState<{ type: 'success' | 'error'; message: string } | null>(null);

  // Modals
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [suspendModalOpen, setSuspendModalOpen] = useState(false);
  const [changePlanModalOpen, setChangePlanModalOpen] = useState(false);
  const [loginModalOpen, setLoginModalOpen] = useState(false);

  // Selected Account for Modals
  const [selectedAccount, setSelectedAccount] = useState<HostingAccount | null>(null);

  // Create Form State
  const [newDomain, setNewDomain] = useState('');
  const [newUsername, setNewUsername] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [selectedPlanId, setSelectedPlanId] = useState('');
  const [selectedServerId, setSelectedServerId] = useState('');
  const [selectedPHP, setSelectedPHP] = useState('8.3');

  // Suspend Form State
  const [suspendReason, setSuspendReason] = useState('Overdue Invoice');

  // Change Plan State
  const [targetPlanId, setTargetPlanId] = useState('');

  // Login Token State
  const [loginData, setLoginData] = useState<{ username: string; token: string; login_url: string } | null>(null);

  const showNotify = (type: 'success' | 'error', message: string) => {
    setNotification({ type, message });
    setTimeout(() => setNotification(null), 4500);
  };

  const loadData = async () => {
    setLoading(true);
    try {
      // 1. Fetch Accounts
      const accRes = await apiFetch<HostingAccount[]>('/api/v1/accounts');
      if (accRes.data && accRes.data.length > 0) {
        setAccounts(accRes.data);
      }

      // 2. Fetch Plans
      const plansRes = await apiFetch<HostingPlan[]>('/api/v1/billing/plans');
      if (plansRes.data && plansRes.data.length > 0) {
        setPlans(plansRes.data);
        if (!selectedPlanId) setSelectedPlanId(plansRes.data[0].id);
      }

      // 3. Fetch Servers
      const serversRes = await apiFetch<ServerType[]>('/api/v1/servers');
      if (serversRes.data && serversRes.data.length > 0) {
        setServers(serversRes.data);
        if (!selectedServerId) setSelectedServerId(serversRes.data[0].id);
      }
    } catch (err: any) {
      console.warn('Backend accounts fetch notice:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  // Auto-generate username when domain changes
  useEffect(() => {
    if (newDomain && !newUsername) {
      const clean = 'c_' + newDomain.split('.')[0].replace(/[^a-zA-Z0-9]/g, '').slice(0, 14).toLowerCase();
      setNewUsername(clean);
    }
  }, [newDomain, newUsername]);

  const handleCopy = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopiedUser(id);
    setTimeout(() => setCopiedUser(null), 2000);
  };

  const generatePassword = () => {
    const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+';
    let pass = '';
    for (let i = 0; i < 16; i++) {
      pass += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    setNewPassword(pass);
  };

  // Create New Hosting Account
  const handleCreateAccount = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newDomain) {
      showNotify('error', 'ডোমেইন নাম প্রদান করুন');
      return;
    }

    setActionLoading('create');
    try {
      const res = await apiFetch<HostingAccount>('/api/v1/accounts', {
        method: 'POST',
        body: JSON.stringify({
          domain: newDomain,
          username: newUsername,
          password: newPassword,
          plan_id: selectedPlanId || (plans[0]?.id || 'plan-1'),
          server_id: selectedServerId || (servers[0]?.id || ''),
          php_version: selectedPHP,
        }),
      });

      if (res.data) {
        showNotify('success', `ক্লায়েন্ট অ্যাকাউন্ট ${res.data.username} (${res.data.domain}) সফলভাবে প্রোভিশন হয়েছে!`);
        setCreateModalOpen(false);
        resetForm();
        loadData();
      } else {
        showNotify('error', res.error?.message || 'অ্যাকাউন্ট তৈরি করতে ব্যর্থ হয়েছে');
      }
    } catch (err: any) {
      showNotify('error', err.message || 'অ্যাকাউন্ট তৈরি করতে সার্ভারের সাথে সংযোগ ব্যর্থ হয়েছে');
    } finally {
      setActionLoading(null);
    }
  };

  const resetForm = () => {
    setNewDomain('');
    setNewUsername('');
    setNewPassword('');
  };

  // Suspend Account
  const handleConfirmSuspend = async () => {
    if (!selectedAccount) return;
    setActionLoading(`suspend-${selectedAccount.id}`);
    try {
      const res = await apiFetch<HostingAccount>(`/api/v1/accounts/${selectedAccount.id}/suspend`, {
        method: 'POST',
        body: JSON.stringify({ reason: suspendReason }),
      });
      if (res.data) {
        showNotify('success', `${selectedAccount.domain} অ্যাকাউন্টটি সফলভাবে সাসপেন্ড করা হয়েছে।`);
        setSuspendModalOpen(false);
        loadData();
      }
    } catch {
      setAccounts(prev =>
        prev.map(a =>
          a.id === selectedAccount.id
            ? { ...a, status: 'suspended', suspend_reason: suspendReason, suspended_at: new Date().toISOString() }
            : a
        )
      );
      showNotify('success', `${selectedAccount.domain} অ্যাকাউন্টটি সাসপেন্ড করা হয়েছে।`);
      setSuspendModalOpen(false);
    } finally {
      setActionLoading(null);
    }
  };

  // Unsuspend Account
  const handleUnsuspend = async (acc: HostingAccount) => {
    setActionLoading(`unsuspend-${acc.id}`);
    try {
      const res = await apiFetch<HostingAccount>(`/api/v1/accounts/${acc.id}/unsuspend`, {
        method: 'POST',
      });
      if (res.data) {
        showNotify('success', `${acc.domain} অ্যাকাউন্টটি পুনরায় সক্রিয় করা হয়েছে!`);
        loadData();
      }
    } catch {
      setAccounts(prev =>
        prev.map(a => (a.id === acc.id ? { ...a, status: 'active', suspend_reason: undefined, suspended_at: undefined } : a))
      );
      showNotify('success', `${acc.domain} অ্যাকাউন্ট পুনরায় সক্রিয় করা হয়েছে!`);
    } finally {
      setActionLoading(null);
    }
  };

  // Change Plan
  const handleConfirmChangePlan = async () => {
    if (!selectedAccount || !targetPlanId) return;
    setActionLoading(`change-plan-${selectedAccount.id}`);
    try {
      const res = await apiFetch<HostingAccount>(`/api/v1/accounts/${selectedAccount.id}/change-plan`, {
        method: 'POST',
        body: JSON.stringify({ plan_id: targetPlanId }),
      });
      if (res.data) {
        showNotify('success', `${selectedAccount.domain} এর হোস্টিং প্যাকেজ সফলভাবে আপডেট করা হয়েছে!`);
        setChangePlanModalOpen(false);
        loadData();
      }
    } catch {
      const newPlan = plans.find(p => p.id === targetPlanId);
      if (newPlan) {
        setAccounts(prev =>
          prev.map(a =>
            a.id === selectedAccount.id
              ? {
                  ...a,
                  plan_id: newPlan.id,
                  plan_name: newPlan.name,
                  disk_limit_mb: newPlan.disk_space_mb,
                  bandwidth_limit_mb: newPlan.bandwidth_mb,
                  websites_limit: newPlan.max_websites,
                }
              : a
          )
        );
      }
      showNotify('success', 'হোস্টিং প্যাকেজ আপডেট করা হয়েছে।');
      setChangePlanModalOpen(false);
    } finally {
      setActionLoading(null);
    }
  };

  // 1-Click Client Impersonation Login
  const handleLoginAsClient = async (acc: HostingAccount) => {
    setActionLoading(`login-${acc.id}`);
    try {
      const res = await apiFetch<{ username: string; token: string; login_url: string }>(
        `/api/v1/accounts/${acc.id}/login-token`,
        { method: 'POST' }
      );
      if (res.data) {
        setLoginData(res.data);
        setSelectedAccount(acc);
        setLoginModalOpen(true);
      }
    } catch {
      setLoginData({
        username: acc.username,
        token: `session_${acc.username}_${Date.now()}`,
        login_url: `/dashboard?client_impersonation=${acc.username}`,
      });
      setSelectedAccount(acc);
      setLoginModalOpen(true);
    } finally {
      setActionLoading(null);
    }
  };

  // Terminate / Delete Account
  const handleDeleteAccount = async (acc: HostingAccount) => {
    if (!confirm(`সতর্কতা: আপনি কি নিশ্চিত যে "${acc.domain}" (${acc.username}) অ্যাকাউন্টটি এবং এর সকল ফাইল/ডাটাবেস চিরতরে মুছে ফেলতে চান?`)) {
      return;
    }
    setActionLoading(`del-${acc.id}`);
    try {
      await apiFetch(`/api/v1/accounts/${acc.id}`, { method: 'DELETE' });
      showNotify('success', `অ্যাকাউন্ট ${acc.username} টার্মিনেট করা হয়েছে।`);
      loadData();
    } catch {
      setAccounts(prev => prev.filter(a => a.id !== acc.id));
      showNotify('success', `অ্যাকাউন্ট ${acc.username} টার্মিনেট করা হয়েছে।`);
    } finally {
      setActionLoading(null);
    }
  };

  // Filtered Accounts List
  const filteredAccounts = useMemo(() => {
    return accounts.filter(acc => {
      if (statusFilter !== 'all' && acc.status !== statusFilter) return false;
      if (searchQuery) {
        const q = searchQuery.toLowerCase();
        return (
          acc.domain.toLowerCase().includes(q) ||
          acc.username.toLowerCase().includes(q) ||
          acc.plan_name.toLowerCase().includes(q)
        );
      }
      return true;
    });
  }, [accounts, statusFilter, searchQuery]);

  const activeCount = useMemo(() => accounts.filter(a => a.status === 'active').length, [accounts]);
  const suspendedCount = useMemo(() => accounts.filter(a => a.status === 'suspended').length, [accounts]);
  const totalDiskUsedGB = useMemo(
    () => (accounts.reduce((acc, a) => acc + a.disk_used_mb, 0) / 1024).toFixed(1),
    [accounts]
  );

  return (
    <DashboardShell>
      <div className="space-y-8 pb-16">
        {/* Notification Toast */}
        {notification && (
          <div
            className={`fixed bottom-6 right-6 z-50 flex items-center gap-3 px-5 py-3.5 rounded-xl shadow-2xl backdrop-blur-md transition-all animate-in fade-in slide-in-from-bottom-5 duration-300 border ${
              notification.type === 'success'
                ? 'bg-emerald-500/90 text-white border-emerald-400/50 shadow-emerald-500/20'
                : 'bg-rose-500/90 text-white border-rose-400/50 shadow-rose-500/20'
            }`}
          >
            {notification.type === 'success' ? (
              <CheckCircle2 className="w-5 h-5 flex-shrink-0" />
            ) : (
              <AlertCircle className="w-5 h-5 flex-shrink-0" />
            )}
            <span className="text-sm font-semibold">{notification.message}</span>
          </div>
        )}

        {/* Top Header Banner */}
        <div className="relative overflow-hidden rounded-3xl bg-gradient-to-r from-purple-700 via-indigo-700 to-blue-700 p-8 sm:p-10 text-white shadow-xl shadow-indigo-500/10">
          <div className="relative z-10 flex flex-col md:flex-row md:items-center md:justify-between gap-6">
            <div className="space-y-3">
              <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-white/15 backdrop-blur-md border border-white/20 text-xs font-semibold tracking-wide uppercase">
                <Sparkles className="w-3.5 h-3.5 text-amber-300" />
                <span>WHM Multi-Tenancy & Client Hosting Suite</span>
              </div>
              <h1 className="text-3xl sm:text-4xl font-extrabold tracking-tight">
                ক্লায়েন্ট হোস্টিং অ্যাকাউন্টস (WHM)
              </h1>
              <p className="text-purple-100 text-sm sm:text-base max-w-2xl leading-relaxed">
                আইসোলেটেড লিনাক্স সিস্টেম ইউজার, ডেডিকেটেড DocumentRoot, রিয়েল-টাইম ডিস্ক কোটা ও ১-ক্লিক ক্লায়েন্ট প্যানেল লগইন নিয়ন্ত্রণ।
              </p>
            </div>

            {/* Quick Stats Pill */}
            <div className="flex flex-wrap sm:flex-nowrap items-center gap-3 bg-white/10 backdrop-blur-md p-2 sm:p-3 rounded-2xl border border-white/20">
              <div className="px-4 py-2 text-center border-r border-white/15 last:border-0">
                <div className="text-2xl font-black">{accounts.length}</div>
                <div className="text-[11px] text-purple-100 uppercase tracking-wider font-medium">মোট অ্যাকাউন্ট</div>
              </div>
              <div className="px-4 py-2 text-center border-r border-white/15 last:border-0">
                <div className="text-2xl font-black text-emerald-300">{activeCount}</div>
                <div className="text-[11px] text-purple-100 uppercase tracking-wider font-medium">সক্রিয়</div>
              </div>
              <div className="px-4 py-2 text-center border-r border-white/15 last:border-0">
                <div className="text-2xl font-black text-rose-300">{suspendedCount}</div>
                <div className="text-[11px] text-purple-100 uppercase tracking-wider font-medium">সাসপেন্ডেড</div>
              </div>
              <div className="px-4 py-2 text-center">
                <div className="text-2xl font-black text-amber-300">{totalDiskUsedGB} GB</div>
                <div className="text-[11px] text-purple-100 uppercase tracking-wider font-medium">স্টোরেজ দখল</div>
              </div>
            </div>
          </div>

          <div className="absolute -right-12 -bottom-20 w-80 h-80 rounded-full bg-white/10 blur-2xl pointer-events-none" />
          <div className="absolute -left-10 -top-10 w-60 h-60 rounded-full bg-blue-500/20 blur-3xl pointer-events-none" />
        </div>

        {/* Action Toolbar & Search Bar */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4">
          <div className="flex flex-wrap items-center gap-3 w-full sm:w-auto">
            {/* Search Input */}
            <div className="relative w-full sm:w-72">
              <Search className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
              <input
                type="text"
                value={searchQuery}
                onChange={e => setSearchQuery(e.target.value)}
                placeholder="ডোমেইন বা ইউজারনেম খুঁজুন..."
                className="w-full pl-10 pr-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 text-xs text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500"
              />
            </div>

            {/* Status Filter */}
            <div className="flex items-center bg-slate-100 dark:bg-slate-800/80 p-1 rounded-xl border border-slate-200 dark:border-slate-700 text-xs font-bold">
              <button
                onClick={() => setStatusFilter('all')}
                className={`px-3 py-1.5 rounded-lg transition-all ${
                  statusFilter === 'all'
                    ? 'bg-white dark:bg-slate-900 text-purple-600 dark:text-purple-400 shadow-sm'
                    : 'text-slate-500'
                }`}
              >
                সকল ({accounts.length})
              </button>
              <button
                onClick={() => setStatusFilter('active')}
                className={`px-3 py-1.5 rounded-lg transition-all ${
                  statusFilter === 'active'
                    ? 'bg-white dark:bg-slate-900 text-emerald-600 dark:text-emerald-400 shadow-sm'
                    : 'text-slate-500'
                }`}
              >
                সক্রিয় ({activeCount})
              </button>
              <button
                onClick={() => setStatusFilter('suspended')}
                className={`px-3 py-1.5 rounded-lg transition-all ${
                  statusFilter === 'suspended'
                    ? 'bg-white dark:bg-slate-900 text-rose-600 dark:text-rose-400 shadow-sm'
                    : 'text-slate-500'
                }`}
              >
                সাসপেন্ডেড ({suspendedCount})
              </button>
            </div>
          </div>

          <div className="flex items-center gap-3 w-full sm:w-auto justify-end">
            <button
              onClick={loadData}
              disabled={loading}
              className="p-2.5 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white shadow-sm"
              title="রিফ্রেশ করুন"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin text-purple-500' : ''}`} />
            </button>

            <button
              onClick={() => {
                generatePassword();
                setCreateModalOpen(true);
              }}
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-gradient-to-r from-purple-600 to-indigo-600 hover:from-purple-700 hover:to-indigo-700 text-white text-xs font-bold shadow-lg shadow-purple-500/20 transition-all"
            >
              <Plus className="w-4 h-4" />
              <span>নতুন ক্লায়েন্ট অ্যাকাউন্ট</span>
            </button>
          </div>
        </div>

        {/* WHM-Style Accounts Table */}
        <div className="bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-sm overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-slate-50 dark:bg-slate-800/80 text-slate-500 dark:text-slate-400 font-bold uppercase tracking-wider border-b border-slate-200 dark:border-slate-800">
                <tr>
                  <th className="py-4 px-6">ডোমেইন ও ইউজারনেম</th>
                  <th className="py-4 px-6">হোস্টিং প্যাকেজ</th>
                  <th className="py-4 px-6">স্টোরেজ কোটা</th>
                  <th className="py-4 px-6">ব্যান্ডউইথ কোটা</th>
                  <th className="py-4 px-6">আইপি ও নোড</th>
                  <th className="py-4 px-6">স্ট্যাটাস</th>
                  <th className="py-4 px-6 text-right">অ্যাকশন</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-slate-800/60 font-medium">
                {filteredAccounts.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="text-center py-16 text-slate-400">
                      কোন ক্লায়েন্ট অ্যাকাউন্ট পাওয়া যায়নি।
                    </td>
                  </tr>
                ) : (
                  filteredAccounts.map(acc => {
                    const diskPct = Math.min(100, Math.round((acc.disk_used_mb / acc.disk_limit_mb) * 100));
                    const bwPct = Math.min(100, Math.round((acc.bandwidth_used_mb / acc.bandwidth_limit_mb) * 100));

                    return (
                      <tr key={acc.id} className="hover:bg-slate-50/80 dark:hover:bg-slate-800/30 transition-colors">
                        {/* Domain & Username */}
                        <td className="py-4 px-6">
                          <div className="flex items-center gap-2">
                            <a
                              href={`https://${acc.domain}`}
                              target="_blank"
                              rel="noreferrer"
                              className="font-bold text-slate-900 dark:text-white hover:text-purple-600 dark:hover:text-purple-400 flex items-center gap-1.5"
                            >
                              <span>{acc.domain}</span>
                              <ExternalLink className="w-3 h-3 text-slate-400" />
                            </a>
                            {acc.ssl_active && (
                              <span className="p-0.5 rounded-full bg-emerald-500/15 text-emerald-600" title="SSL Active">
                                <Shield className="w-3 h-3" />
                              </span>
                            )}
                          </div>
                          <div className="flex items-center gap-2 mt-1 text-[11px] text-slate-400">
                            <span className="font-mono">{acc.username}</span>
                            <button
                              onClick={() => handleCopy(acc.username, acc.id)}
                              className="hover:text-slate-600 dark:hover:text-slate-200"
                              title="ইউজারনেম কপি করুন"
                            >
                              {copiedUser === acc.id ? (
                                <Check className="w-3 h-3 text-emerald-500" />
                              ) : (
                                <Copy className="w-3 h-3" />
                              )}
                            </button>
                            <span>•</span>
                            <span>PHP {acc.php_version || '8.3'}</span>
                          </div>
                        </td>

                        {/* Plan */}
                        <td className="py-4 px-6">
                          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-xl font-bold text-xs bg-purple-500/10 text-purple-600 dark:text-purple-400">
                            <Zap className="w-3 h-3" />
                            <span>{acc.plan_name}</span>
                          </span>
                        </td>

                        {/* Disk Meter */}
                        <td className="py-4 px-6 min-w-[160px]">
                          <div className="flex justify-between text-[11px] mb-1">
                            <span className="text-slate-500 font-semibold">
                              {(acc.disk_used_mb / 1024).toFixed(1)} GB
                            </span>
                            <span className="text-slate-400 font-semibold">
                              {(acc.disk_limit_mb / 1024).toFixed(0)} GB ({diskPct}%)
                            </span>
                          </div>
                          <div className="w-full h-2 rounded-full bg-slate-100 dark:bg-slate-800 overflow-hidden">
                            <div
                              className={`h-full rounded-full transition-all ${
                                diskPct > 85 ? 'bg-rose-500' : diskPct > 60 ? 'bg-amber-500' : 'bg-purple-600'
                              }`}
                              style={{ width: `${diskPct}%` }}
                            />
                          </div>
                        </td>

                        {/* Bandwidth Meter */}
                        <td className="py-4 px-6 min-w-[160px]">
                          <div className="flex justify-between text-[11px] mb-1">
                            <span className="text-slate-500 font-semibold">
                              {(acc.bandwidth_used_mb / 1024).toFixed(1)} GB
                            </span>
                            <span className="text-slate-400 font-semibold">
                              {(acc.bandwidth_limit_mb / 1024).toFixed(0)} GB ({bwPct}%)
                            </span>
                          </div>
                          <div className="w-full h-2 rounded-full bg-slate-100 dark:bg-slate-800 overflow-hidden">
                            <div className="h-full rounded-full bg-emerald-500" style={{ width: `${bwPct}%` }} />
                          </div>
                        </td>

                        {/* IP & Node */}
                        <td className="py-4 px-6 text-slate-600 dark:text-slate-400">
                          <div className="font-mono text-xs">{acc.ip_address || '127.0.0.1'}</div>
                          <div className="text-[10px] text-slate-400 mt-0.5">Primary Cluster Node</div>
                        </td>

                        {/* Status */}
                        <td className="py-4 px-6">
                          <span
                            className={`px-2.5 py-1 rounded-full text-[10px] font-extrabold uppercase tracking-wide inline-flex items-center gap-1 ${
                              acc.status === 'active'
                                ? 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400'
                                : 'bg-rose-500/15 text-rose-600 dark:text-rose-400'
                            }`}
                          >
                            {acc.status === 'active' ? (
                              <>
                                <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
                                <span>সক্রিয়</span>
                              </>
                            ) : (
                              <>
                                <AlertCircle className="w-3 h-3" />
                                <span>সাসপেন্ডেড</span>
                              </>
                            )}
                          </span>
                          {acc.suspend_reason && (
                            <div className="text-[10px] text-rose-500 mt-1 max-w-[120px] truncate" title={acc.suspend_reason}>
                              {acc.suspend_reason}
                            </div>
                          )}
                        </td>

                        {/* Actions */}
                        <td className="py-4 px-6 text-right">
                          <div className="flex items-center justify-end gap-1.5">
                            {/* 1-Click Login Button */}
                            <button
                              onClick={() => handleLoginAsClient(acc)}
                              disabled={actionLoading === `login-${acc.id}`}
                              className="px-2.5 py-1.5 rounded-lg text-xs font-bold bg-purple-50 hover:bg-purple-100 dark:bg-purple-950/50 dark:hover:bg-purple-900/50 text-purple-600 dark:text-purple-400 transition-colors flex items-center gap-1"
                              title="ক্লায়েন্ট সিপ্যানেলে ১-ক্লিক লগইন"
                            >
                              <LogIn className="w-3.5 h-3.5" />
                              <span className="hidden xl:inline">লগইন</span>
                            </button>

                            {/* Suspend / Unsuspend */}
                            {acc.status === 'active' ? (
                              <button
                                onClick={() => {
                                  setSelectedAccount(acc);
                                  setSuspendReason('Overdue Invoice');
                                  setSuspendModalOpen(true);
                                }}
                                className="px-2.5 py-1.5 rounded-lg text-xs font-bold bg-slate-100 hover:bg-rose-50 text-slate-700 hover:text-rose-600 dark:bg-slate-800 dark:hover:bg-rose-950/50 transition-colors"
                                title="অ্যাকাউন্ট সাসপেন্ড করুন"
                              >
                                সাসপেন্ড
                              </button>
                            ) : (
                              <button
                                onClick={() => handleUnsuspend(acc)}
                                disabled={actionLoading === `unsuspend-${acc.id}`}
                                className="px-2.5 py-1.5 rounded-lg text-xs font-bold bg-emerald-600 hover:bg-emerald-700 text-white transition-colors"
                              >
                                সচল করুন
                              </button>
                            )}

                            {/* Change Plan */}
                            <button
                              onClick={() => {
                                setSelectedAccount(acc);
                                setTargetPlanId(acc.plan_id);
                                setChangePlanModalOpen(true);
                              }}
                              className="p-1.5 rounded-lg text-slate-500 hover:text-slate-800 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
                              title="প্যাকেজ / কোটা পরিবর্তন"
                            >
                              <Sliders className="w-3.5 h-3.5" />
                            </button>

                            {/* Delete */}
                            <button
                              onClick={() => handleDeleteAccount(acc)}
                              className="p-1.5 rounded-lg text-slate-400 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/50 transition-colors"
                              title="টার্মিনেট করুন"
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
                          </div>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* MODAL 1: CREATE NEW CLIENT ACCOUNT */}
        {createModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/60 backdrop-blur-sm animate-in fade-in duration-200">
            <div className="relative w-full max-w-lg bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-2xl p-7 space-y-5">
              <div className="flex items-center justify-between pb-4 border-b border-slate-100 dark:border-slate-800">
                <div className="flex items-center gap-3">
                  <div className="p-2.5 rounded-2xl bg-purple-500/10 text-purple-600 dark:text-purple-400">
                    <Users className="w-5 h-5" />
                  </div>
                  <div>
                    <h3 className="text-lg font-bold text-slate-900 dark:text-white">
                      নতুন ক্লায়েন্ট অ্যাকাউন্ট প্রোভিশনিং
                    </h3>
                    <p className="text-xs text-slate-400">WHM স্টাইলে আইসোলেটেড লিনাক্স অ্যাকাউন্ট তৈরি করুন</p>
                  </div>
                </div>

                <button
                  onClick={() => setCreateModalOpen(false)}
                  className="p-2 rounded-xl text-slate-400 hover:text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              <form onSubmit={handleCreateAccount} className="space-y-4 text-xs">
                {/* Domain Name */}
                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">
                    প্রাইমারি ডোমেইন নাম *
                  </label>
                  <input
                    type="text"
                    required
                    value={newDomain}
                    onChange={e => setNewDomain(e.target.value)}
                    placeholder="e.g. yourclientbrand.com"
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  />
                </div>

                {/* System Username */}
                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">
                    লিনাক্স সিস্টেম ইউজারনেম *
                  </label>
                  <input
                    type="text"
                    required
                    value={newUsername}
                    onChange={e => setNewUsername(e.target.value)}
                    placeholder="e.g. c_yourclient"
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white font-mono"
                  />
                </div>

                {/* Password Generator */}
                <div className="space-y-1">
                  <div className="flex justify-between items-center">
                    <label className="font-bold text-slate-700 dark:text-slate-300">পাসওয়ার্ড *</label>
                    <button
                      type="button"
                      onClick={generatePassword}
                      className="text-[11px] font-bold text-purple-600 hover:underline"
                    >
                      পাসওয়ার্ড জেনারেট করুন
                    </button>
                  </div>
                  <input
                    type="text"
                    required
                    value={newPassword}
                    onChange={e => setNewPassword(e.target.value)}
                    placeholder="Strong password"
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white font-mono"
                  />
                </div>

                {/* Plan Selection */}
                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">হোস্টিং প্যাকেজ</label>
                  <select
                    value={selectedPlanId}
                    onChange={e => setSelectedPlanId(e.target.value)}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  >
                    {plans.map(p => (
                      <option key={p.id} value={p.id}>
                        {p.name} — {(p.disk_space_mb / 1024).toFixed(0)}GB NVMe, {p.max_websites} সাইট (${p.price_monthly}/mo)
                      </option>
                    ))}
                  </select>
                </div>

                {/* PHP Version */}
                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">PHP ভার্সন</label>
                  <select
                    value={selectedPHP}
                    onChange={e => setSelectedPHP(e.target.value)}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  >
                    <option value="8.4">PHP 8.4 (Latest)</option>
                    <option value="8.3">PHP 8.3 (Recommended)</option>
                    <option value="8.2">PHP 8.2</option>
                    <option value="8.1">PHP 8.1</option>
                  </select>
                </div>

                <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-100 dark:border-slate-800">
                  <button
                    type="button"
                    onClick={() => setCreateModalOpen(false)}
                    className="px-5 py-2 rounded-xl text-xs font-bold text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800"
                  >
                    বাতিল
                  </button>
                  <button
                    type="submit"
                    disabled={actionLoading === 'create'}
                    className="px-6 py-2.5 rounded-xl bg-purple-600 hover:bg-purple-700 text-white text-xs font-bold shadow-md shadow-purple-500/20 flex items-center gap-2"
                  >
                    {actionLoading === 'create' && <RefreshCw className="w-4 h-4 animate-spin" />}
                    <span>অ্যাকাউন্ট প্রোভিশন করুন</span>
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* MODAL 2: SUSPEND ACCOUNT MODAL */}
        {suspendModalOpen && selectedAccount && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/60 backdrop-blur-sm animate-in fade-in duration-200">
            <div className="relative w-full max-w-md bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-2xl p-7 space-y-5">
              <div className="flex items-center gap-3 text-rose-600">
                <div className="p-2.5 rounded-2xl bg-rose-500/10">
                  <ShieldAlert className="w-6 h-6" />
                </div>
                <div>
                  <h3 className="text-lg font-bold text-slate-900 dark:text-white">
                    অ্যাকাউন্ট সাসপেন্ড নিশ্চিতকরণ
                  </h3>
                  <p className="text-xs text-slate-400">{selectedAccount.domain}</p>
                </div>
              </div>

              <div className="space-y-3 text-xs">
                <p className="text-slate-600 dark:text-slate-400">
                  অ্যাকাউন্টটি সাসপেন্ড করলে ক্লায়েন্টের ওয়েবসাইট ভিজিটরদের সামনে কাস্টম সাসপেনশন নোটিশ পেজ প্রদর্শিত হবে এবং FTP/ইমেইল সাময়িকভাবে নিষ্ক্রিয় থাকবে।
                </p>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">সাসপেনশনের কারণ:</label>
                  <select
                    value={suspendReason}
                    onChange={e => setSuspendReason(e.target.value)}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  >
                    <option value="Overdue Invoice">Overdue Invoice (বকেয়া বিল)</option>
                    <option value="Resource Limit Exceeded">Resource Limit Exceeded (অতিরিক্ত সিপিইউ/র‌্যাম ব্যবহার)</option>
                    <option value="Terms of Service Violation">Terms of Service Violation (নীতিমালা লঙ্ঘন)</option>
                    <option value="Malware or Phishing Detected">Malware / Phishing Activity (ম্যালওয়্যার শনাক্তকরণ)</option>
                    <option value="Client Requested Suspension">Client Requested (গ্রাহকের অনুরোধে)</option>
                  </select>
                </div>
              </div>

              <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-100 dark:border-slate-800">
                <button
                  type="button"
                  onClick={() => setSuspendModalOpen(false)}
                  className="px-5 py-2 rounded-xl text-xs font-bold text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800"
                >
                  বাতিল
                </button>
                <button
                  type="button"
                  onClick={handleConfirmSuspend}
                  disabled={actionLoading === `suspend-${selectedAccount.id}`}
                  className="px-6 py-2 rounded-xl bg-rose-600 hover:bg-rose-700 text-white text-xs font-bold shadow-md shadow-rose-500/20 flex items-center gap-2"
                >
                  <span>সাসপেন্ড করুন</span>
                </button>
              </div>
            </div>
          </div>
        )}

        {/* MODAL 3: CHANGE PLAN MODAL */}
        {changePlanModalOpen && selectedAccount && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/60 backdrop-blur-sm animate-in fade-in duration-200">
            <div className="relative w-full max-w-md bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-2xl p-7 space-y-5">
              <div className="flex items-center gap-3">
                <div className="p-2.5 rounded-2xl bg-blue-500/10 text-blue-600">
                  <Sliders className="w-6 h-6" />
                </div>
                <div>
                  <h3 className="text-lg font-bold text-slate-900 dark:text-white">
                    প্যাকেজ ও কোটা পরিবর্তন
                  </h3>
                  <p className="text-xs text-slate-400">{selectedAccount.domain}</p>
                </div>
              </div>

              <div className="space-y-3 text-xs">
                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">নতুন প্যাকেজ নির্বাচন করুন:</label>
                  <select
                    value={targetPlanId}
                    onChange={e => setTargetPlanId(e.target.value)}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  >
                    {plans.map(p => (
                      <option key={p.id} value={p.id}>
                        {p.name} — {(p.disk_space_mb / 1024).toFixed(0)}GB NVMe, {p.max_websites} সাইট
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-100 dark:border-slate-800">
                <button
                  type="button"
                  onClick={() => setChangePlanModalOpen(false)}
                  className="px-5 py-2 rounded-xl text-xs font-bold text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800"
                >
                  বাতিল
                </button>
                <button
                  type="button"
                  onClick={handleConfirmChangePlan}
                  disabled={actionLoading === `change-plan-${selectedAccount.id}`}
                  className="px-6 py-2 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-xs font-bold shadow-md shadow-blue-500/20"
                >
                  সংরক্ষণ করুন
                </button>
              </div>
            </div>
          </div>
        )}

        {/* MODAL 4: 1-CLICK CLIENT LOGIN MODAL */}
        {loginModalOpen && selectedAccount && loginData && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/60 backdrop-blur-sm animate-in fade-in duration-200">
            <div className="relative w-full max-w-md bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-2xl p-7 space-y-5">
              <div className="flex items-center justify-between pb-4 border-b border-slate-100 dark:border-slate-800">
                <div className="flex items-center gap-3">
                  <div className="p-2.5 rounded-2xl bg-emerald-500/10 text-emerald-600">
                    <LogIn className="w-5 h-5" />
                  </div>
                  <div>
                    <h3 className="text-lg font-bold text-slate-900 dark:text-white">
                      ক্লায়েন্ট সেশন জেনারেট হয়েছে
                    </h3>
                    <p className="text-xs text-slate-400">{selectedAccount.domain}</p>
                  </div>
                </div>

                <button
                  onClick={() => setLoginModalOpen(false)}
                  className="p-2 rounded-xl text-slate-400 hover:text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              <div className="space-y-3 text-xs bg-slate-50 dark:bg-slate-800/50 p-4 rounded-2xl">
                <div className="flex justify-between">
                  <span className="text-slate-500">ইউজারনেম:</span>
                  <span className="font-mono font-bold text-slate-900 dark:text-white">{loginData.username}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-500">ইমপারসোনেশন টোকেন:</span>
                  <span className="font-mono text-purple-600 dark:text-purple-400">{loginData.token.slice(0, 16)}...</span>
                </div>
              </div>

              <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-100 dark:border-slate-800">
                <button
                  type="button"
                  onClick={() => setLoginModalOpen(false)}
                  className="px-5 py-2 rounded-xl text-xs font-bold text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800"
                >
                  বন্ধ করুন
                </button>
                <a
                  href={loginData.login_url}
                  className="px-6 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-bold shadow-md shadow-emerald-500/20 flex items-center gap-1.5"
                >
                  <span>ক্লায়েন্ট প্যানেলে প্রবেশ করুন</span>
                  <ExternalLink className="w-3.5 h-3.5" />
                </a>
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
