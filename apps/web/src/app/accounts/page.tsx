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
  Webhook,
  Palette,
  Cpu,
  Activity,
  Gauge,
  ChevronDown,
  ChevronUp,
} from 'lucide-react';
import {
  apiFetch,
  HostingAccount,
  HostingPlan,
  Server as ServerType,
  setStoredToken,
  updateAccountLimits,
} from '@/lib/api';
import { WebhookIntegrationsTab } from '@/components/accounts/WebhookIntegrationsTab';
import { ResellerMultiTenancyTab } from '@/components/accounts/ResellerMultiTenancyTab';

export default function AccountsPage() {
  const [activeTab, setActiveTab] = useState<'accounts' | 'webhooks' | 'reseller'>('accounts');
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
  const [limitsModalOpen, setLimitsModalOpen] = useState(false);
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
  const [showAdvancedCreate, setShowAdvancedCreate] = useState(false);
  const [createRamMB, setCreateRamMB] = useState(1024);
  const [createStorageGB, setCreateStorageGB] = useState(10);
  const [createCpuPct, setCreateCpuPct] = useState(100);
  const [createTasksMax, setCreateTasksMax] = useState(100);

  // Suspend Form State
  const [suspendReason, setSuspendReason] = useState('Overdue Invoice');

  // Resource Limits & System Isolation State
  const [targetPlanId, setTargetPlanId] = useState('');
  const [limitRamMB, setLimitRamMB] = useState(1024);
  const [limitStorageGB, setLimitStorageGB] = useState(10);
  const [limitCpuPct, setLimitCpuPct] = useState(100);
  const [limitBwGB, setLimitBwGB] = useState(100);
  const [limitInodes, setLimitInodes] = useState(250000);
  const [limitTasks, setLimitTasks] = useState(100);
  const [limitWebsites, setLimitWebsites] = useState(1);
  const [limitDatabases, setLimitDatabases] = useState(5);
  const [limitMailboxes, setLimitMailboxes] = useState(10);
  const [limitPHP, setLimitPHP] = useState('8.3');
  const [limitOpenBaseDir, setLimitOpenBaseDir] = useState(true);

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
      if (accRes.data) {
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
      showNotify('error', 'Please provide a domain name');
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
          memory_limit_mb: Number(createRamMB) || 1024,
          disk_limit_mb: (Number(createStorageGB) || 10) * 1024,
          cpu_limit_percent: Number(createCpuPct) || 100,
          tasks_max: Number(createTasksMax) || 100,
        }),
      });

      if (res.data) {
        showNotify('success', `Client account ${res.data.username} (${res.data.domain}) provisioned with isolated Linux user & cgroups quotas!`);
        setCreateModalOpen(false);
        resetForm();
        loadData();
      } else {
        showNotify('error', res.error?.message || 'Failed to create client account');
      }
    } catch (err: any) {
      showNotify('error', err.message || 'Server connection failed while creating account');
    } finally {
      setActionLoading(null);
    }
  };

  const resetForm = () => {
    setNewDomain('');
    setNewUsername('');
    setNewPassword('');
    setShowAdvancedCreate(false);
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
        showNotify('success', `Account ${selectedAccount.domain} suspended successfully.`);
        setSuspendModalOpen(false);
        loadData();
      } else {
        showNotify('error', res.error?.message || 'Failed to suspend account');
      }
    } catch (err: any) {
      showNotify('error', err?.message || 'Failed to connect to server');
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
        showNotify('success', `Account ${acc.domain} reactivated successfully!`);
        loadData();
      } else {
        showNotify('error', res.error?.message || 'Failed to reactivate account');
      }
    } catch (err: any) {
      showNotify('error', err?.message || 'Failed to connect to server');
    } finally {
      setActionLoading(null);
    }
  };

  // Open Manage Limits & Isolation Modal
  const handleOpenLimitsModal = (acc: HostingAccount) => {
    setSelectedAccount(acc);
    setTargetPlanId(acc.plan_id);
    setLimitRamMB(acc.memory_limit_mb || 1024);
    setLimitStorageGB(Math.round((acc.disk_limit_mb || 10240) / 1024));
    setLimitCpuPct(acc.cpu_limit_percent || 100);
    setLimitBwGB(Math.round((acc.bandwidth_limit_mb || 102400) / 1024));
    setLimitInodes(acc.inodes_limit || 250000);
    setLimitTasks(acc.tasks_max || 100);
    setLimitWebsites(acc.websites_limit || 1);
    setLimitDatabases(acc.databases_limit || 5);
    setLimitMailboxes(acc.mailboxes_limit || 10);
    setLimitPHP(acc.php_version || '8.3');
    setLimitOpenBaseDir(true);
    setLimitsModalOpen(true);
  };

  // Apply Package Template to Limit Fields
  const handleApplyPlanToLimits = (planId: string) => {
    setTargetPlanId(planId);
    const p = plans.find(plan => plan.id === planId);
    if (p) {
      setLimitStorageGB(Math.round(p.disk_space_mb / 1024));
      setLimitBwGB(Math.round(p.bandwidth_mb / 1024));
      setLimitRamMB(p.disk_space_mb >= 10240 ? 1024 : 512);
      setLimitWebsites(p.max_websites || 1);
      setLimitDatabases(p.max_databases || 5);
      setLimitMailboxes(p.max_mailboxes || 10);
    }
  };

  // Save Custom Resource Limits (RAM, Storage, CPU, Isolation)
  const handleSaveLimits = async () => {
    if (!selectedAccount) return;
    setActionLoading(`limits-${selectedAccount.id}`);
    try {
      const res = await updateAccountLimits(selectedAccount.id, {
        memory_limit_mb: Number(limitRamMB) || 1024,
        disk_limit_mb: (Number(limitStorageGB) || 10) * 1024,
        cpu_limit_percent: Number(limitCpuPct) || 100,
        bandwidth_limit_mb: (Number(limitBwGB) || 100) * 1024,
        inodes_limit: Number(limitInodes) || 250000,
        tasks_max: Number(limitTasks) || 100,
        websites_limit: Number(limitWebsites) || 1,
        databases_limit: Number(limitDatabases) || 5,
        mailboxes_limit: Number(limitMailboxes) || 10,
        php_version: limitPHP,
        open_basedir: limitOpenBaseDir,
      });

      if (res.data) {
        showNotify('success', `Resource limits & cgroups v2 isolation for ${selectedAccount.domain} updated successfully!`);
        setLimitsModalOpen(false);
        loadData();
      } else {
        showNotify('error', res.error?.message || 'Failed to update resource limits');
      }
    } catch (err: any) {
      showNotify('error', err?.message || 'Failed to connect to server');
    } finally {
      setActionLoading(null);
    }
  };

  // 1-Click Client WHM Impersonation Login
  const handleLoginAsClient = async (acc: HostingAccount) => {
    if (!confirm(`Log in directly to WHM client control panel for "${acc.username}" (${acc.domain})?`)) {
      return;
    }
    setActionLoading(`login-${acc.id}`);
    try {
      const res = await apiFetch<{
        username: string;
        domain: string;
        tokens?: { access_token: string; refresh_token: string };
        token?: string;
        login_url: string;
        user?: { id: string; email: string };
      }>(
        `/api/v1/accounts/${acc.id}/login-token`,
        { method: 'POST' }
      );
      if (res.data) {
        const token = res.data.tokens?.access_token || res.data.token;
        if (token) {
          const currentToken = localStorage.getItem('hostvra_access_token') || localStorage.getItem('hostvra_token');
          if (currentToken) {
            localStorage.setItem('hostvra_admin_backup_token', currentToken);
          }
          localStorage.setItem('hostvra_admin_return_url', '/accounts');
          setStoredToken(token);
          localStorage.setItem('hostvra_token', token);
          if (typeof document !== 'undefined') {
            document.cookie = `hostvra_token=${encodeURIComponent(token)}; path=/; max-age=3600; SameSite=Lax`;
          }
          localStorage.setItem(
            'hostvra_user',
            JSON.stringify({
              id: res.data.user?.id,
              email: res.data.user?.email,
              username: acc.username,
              domain: acc.domain,
              role: 'customer',
              is_superadmin: false,
              impersonated: true,
            })
          );
          showNotify('success', `Client session authenticated for ${acc.username}. Entering client panel...`);
          window.location.href = '/dashboard';
        } else {
          showNotify('error', 'Authentication session token not returned by server');
        }
      } else {
        showNotify('error', res.error?.message || 'Failed to generate client login session');
      }
    } catch (err: any) {
      showNotify('error', err?.message || 'Failed to connect to authentication server');
    } finally {
      setActionLoading(null);
    }
  };

  // Terminate / Delete Account
  const handleDeleteAccount = async (acc: HostingAccount) => {
    if (!confirm(`Warning: Are you sure you want to permanently delete account "${acc.domain}" (${acc.username}) and all its files and databases?`)) {
      return;
    }
    setActionLoading(`del-${acc.id}`);
    try {
      const res = await apiFetch(`/api/v1/accounts/${acc.id}`, { method: 'DELETE' });
      if (res.error) {
        showNotify('error', res.error.message || 'Failed to terminate account');
        return;
      }
      showNotify('success', `Account ${acc.username} (${acc.domain}) terminated successfully.`);
      loadData();
    } catch (err: any) {
      showNotify('error', err?.message || 'Failed to connect to server');
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
        <div className="relative overflow-hidden rounded-3xl bg-white dark:bg-slate-900 p-8 sm:p-10 text-slate-900 dark:text-white border border-slate-200 dark:border-slate-800 shadow-sm">
          <div className="relative z-10 flex flex-col md:flex-row md:items-center md:justify-between gap-6">
            <div className="space-y-3">
              <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-purple-50 dark:bg-purple-950/50 border border-purple-100 dark:border-purple-900 text-xs font-semibold tracking-wide uppercase text-purple-700 dark:text-purple-300">
                <Sparkles className="w-3.5 h-3.5 text-purple-600" />
                <span>WHM Multi-Tenancy & Client Hosting Suite</span>
              </div>
              <h1 className="text-3xl sm:text-4xl font-extrabold tracking-tight text-slate-900 dark:text-white">
                Client Hosting Accounts (WHM)
              </h1>
              <p className="text-slate-600 dark:text-slate-400 text-sm sm:text-base max-w-2xl leading-relaxed">
                Isolated Linux system users, dedicated DocumentRoot, real-time disk quotas, and 1-click client panel login control.
              </p>
            </div>

            {/* Quick Stats Pill */}
            <div className="flex flex-wrap sm:flex-nowrap items-center gap-3 bg-slate-50 dark:bg-slate-800/80 p-2 sm:p-3 rounded-2xl border border-slate-200 dark:border-slate-700">
              <div className="px-4 py-2 text-center border-r border-slate-200 dark:border-slate-700 last:border-0">
                <div className="text-2xl font-black text-slate-900 dark:text-white">{accounts.length}</div>
                <div className="text-[11px] text-slate-500 dark:text-slate-400 uppercase tracking-wider font-semibold">Total Accounts</div>
              </div>
              <div className="px-4 py-2 text-center border-r border-slate-200 dark:border-slate-700 last:border-0">
                <div className="text-2xl font-black text-emerald-600 dark:text-emerald-400">{activeCount}</div>
                <div className="text-[11px] text-slate-500 dark:text-slate-400 uppercase tracking-wider font-semibold">Active</div>
              </div>
              <div className="px-4 py-2 text-center border-r border-slate-200 dark:border-slate-700 last:border-0">
                <div className="text-2xl font-black text-rose-600 dark:text-rose-400">{suspendedCount}</div>
                <div className="text-[11px] text-slate-500 dark:text-slate-400 uppercase tracking-wider font-semibold">Suspended</div>
              </div>
              <div className="px-4 py-2 text-center">
                <div className="text-2xl font-black text-amber-600 dark:text-amber-400">{totalDiskUsedGB} GB</div>
                <div className="text-[11px] text-slate-500 dark:text-slate-400 uppercase tracking-wider font-semibold">Storage Used</div>
              </div>
            </div>
          </div>
        </div>

        {/* Phase Three Navigation Tabs */}
        <div className="flex flex-wrap items-center gap-2 border-b border-slate-200 dark:border-slate-800 pb-4">
          <button
            onClick={() => setActiveTab('accounts')}
            className={`px-4 py-2.5 rounded-2xl text-xs font-bold flex items-center gap-2 transition-all ${
              activeTab === 'accounts'
                ? 'bg-[#16A34A] text-white shadow-sm'
                : 'bg-white dark:bg-slate-900 text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white border border-slate-200 dark:border-slate-800 hover:bg-slate-50'
            }`}
          >
            <Users className="w-4 h-4" />
            <span>Accounts & WHM ({accounts.length})</span>
          </button>
          <button
            onClick={() => setActiveTab('webhooks')}
            className={`px-4 py-2.5 rounded-2xl text-xs font-bold flex items-center gap-2 transition-all ${
              activeTab === 'webhooks'
                ? 'bg-[#16A34A] text-white shadow-sm'
                : 'bg-white dark:bg-slate-900 text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white border border-slate-200 dark:border-slate-800 hover:bg-slate-50'
            }`}
          >
            <Webhook className="w-4 h-4" />
            <span>WHMCS / Blesta Webhooks</span>
          </button>
          <button
            onClick={() => setActiveTab('reseller')}
            className={`px-4 py-2.5 rounded-2xl text-xs font-bold flex items-center gap-2 transition-all ${
              activeTab === 'reseller'
                ? 'bg-[#16A34A] text-white shadow-sm'
                : 'bg-white dark:bg-slate-900 text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white border border-slate-200 dark:border-slate-800 hover:bg-slate-50'
            }`}
          >
            <Palette className="w-4 h-4" />
            <span>Reseller Quotas & White-Label</span>
          </button>
        </div>

        {activeTab === 'accounts' && (
          <>
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
                placeholder="Search domain or username..."
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
                All ({accounts.length})
              </button>
              <button
                onClick={() => setStatusFilter('active')}
                className={`px-3 py-1.5 rounded-lg transition-all ${
                  statusFilter === 'active'
                    ? 'bg-white dark:bg-slate-900 text-emerald-600 dark:text-emerald-400 shadow-sm'
                    : 'text-slate-500'
                }`}
              >
                Active ({activeCount})
              </button>
              <button
                onClick={() => setStatusFilter('suspended')}
                className={`px-3 py-1.5 rounded-lg transition-all ${
                  statusFilter === 'suspended'
                    ? 'bg-white dark:bg-slate-900 text-rose-600 dark:text-rose-400 shadow-sm'
                    : 'text-slate-500'
                }`}
              >
                Suspended ({suspendedCount})
              </button>
            </div>
          </div>

          <div className="flex items-center gap-3 w-full sm:w-auto justify-end">
            <button
              onClick={loadData}
              disabled={loading}
              className="p-2.5 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white shadow-sm"
              title="Refresh"
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
              <span>New Client Account</span>
            </button>
          </div>
        </div>

        {/* WHM-Style Accounts Table */}
        <div className="bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-sm overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-slate-50 dark:bg-slate-800/80 text-slate-500 dark:text-slate-400 font-bold uppercase tracking-wider border-b border-slate-200 dark:border-slate-800">
                <tr>
                  <th className="py-4 px-6">Domain & Username</th>
                  <th className="py-4 px-6">Hosting Package</th>
                  <th className="py-4 px-6">RAM & CPU Limit</th>
                  <th className="py-4 px-6">Storage Quota</th>
                  <th className="py-4 px-6">Bandwidth Quota</th>
                  <th className="py-4 px-6">IP & Node</th>
                  <th className="py-4 px-6">Status</th>
                  <th className="py-4 px-6 text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-slate-800/60 font-medium">
                {filteredAccounts.length === 0 ? (
                  <tr>
                    <td colSpan={8} className="text-center py-16 text-slate-400">
                      No client accounts found.
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
                              title="Copy username"
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

                        {/* RAM & CPU Quota (Linux cgroups v2 Isolated) */}
                        <td className="py-4 px-6 min-w-[150px]">
                          <div className="flex items-center gap-1.5 font-bold text-slate-800 dark:text-slate-200">
                            <Cpu className="w-3.5 h-3.5 text-purple-600 dark:text-purple-400" />
                            <span>
                              {acc.memory_limit_mb
                                ? acc.memory_limit_mb >= 1024
                                  ? `${(acc.memory_limit_mb / 1024).toFixed(0)} GB`
                                  : `${acc.memory_limit_mb} MB`
                                : '1 GB'}{' '}
                              RAM
                            </span>
                          </div>
                          <div className="text-[11px] text-slate-400 mt-0.5 flex items-center gap-1 font-medium">
                            <span className="text-emerald-600 dark:text-emerald-400 font-semibold">{acc.cpu_limit_percent || 100}% CPU</span>
                            <span>•</span>
                            <span>{acc.tasks_max || 100} procs</span>
                          </div>
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
                                <span>Active</span>
                              </>
                            ) : (
                              <>
                                <AlertCircle className="w-3 h-3" />
                                <span>Suspended</span>
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
                              title="1-Click Client WHM Panel Login"
                            >
                              <LogIn className="w-3.5 h-3.5" />
                              <span className="hidden xl:inline">Login</span>
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
                                title="Suspend account"
                              >
                                Suspend
                              </button>
                            ) : (
                              <button
                                onClick={() => handleUnsuspend(acc)}
                                disabled={actionLoading === `unsuspend-${acc.id}`}
                                className="px-2.5 py-1.5 rounded-lg text-xs font-bold bg-emerald-600 hover:bg-emerald-700 text-white transition-colors"
                              >
                                Unsuspend
                              </button>
                            )}

                            {/* Resource Limits & System Isolation */}
                            <button
                              onClick={() => handleOpenLimitsModal(acc)}
                              className="p-1.5 rounded-lg text-slate-500 hover:text-purple-600 hover:bg-purple-50 dark:hover:bg-purple-950/50 transition-colors"
                              title="Set RAM, Storage & Resource Limits (Isolation)"
                            >
                              <Sliders className="w-3.5 h-3.5" />
                            </button>

                            {/* Delete */}
                            <button
                              onClick={() => handleDeleteAccount(acc)}
                              className="p-1.5 rounded-lg text-slate-400 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/50 transition-colors"
                              title="Terminate account"
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
          </>
        )}

        {/* Phase Three: WHMCS / Blesta Webhooks Tab */}
        {activeTab === 'webhooks' && (
          <WebhookIntegrationsTab servers={servers} plans={plans} showNotify={showNotify} />
        )}

        {/* Phase Three: Reseller Quotas & White-Label Branding Tab */}
        {activeTab === 'reseller' && (
          <ResellerMultiTenancyTab showNotify={showNotify} />
        )}

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
                      Provision New Client Account
                    </h3>
                    <p className="text-xs text-slate-400">Create WHM-style isolated Linux hosting account</p>
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
                    Primary Domain Name *
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
                    Linux System Username *
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
                    <label className="font-bold text-slate-700 dark:text-slate-300">Password *</label>
                    <button
                      type="button"
                      onClick={generatePassword}
                      className="text-[11px] font-bold text-purple-600 hover:underline"
                    >
                      Generate Password
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
                  <label className="font-bold text-slate-700 dark:text-slate-300">Hosting Package</label>
                  <select
                    value={selectedPlanId}
                    onChange={e => {
                      setSelectedPlanId(e.target.value);
                      const p = plans.find(plan => plan.id === e.target.value);
                      if (p) {
                        setCreateStorageGB(Math.round(p.disk_space_mb / 1024));
                        setCreateRamMB(p.disk_space_mb >= 10240 ? 1024 : 512);
                      }
                    }}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  >
                    {plans.map(p => (
                      <option key={p.id} value={p.id}>
                        {p.name} — {(p.disk_space_mb / 1024).toFixed(0)}GB NVMe, {p.max_websites} Sites (${p.price_monthly}/mo)
                      </option>
                    ))}
                  </select>
                </div>

                {/* PHP Version */}
                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">PHP Version</label>
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

                {/* Collapsible Resource Limits & System Isolation */}
                <div className="pt-2 border-t border-slate-100 dark:border-slate-800">
                  <button
                    type="button"
                    onClick={() => setShowAdvancedCreate(!showAdvancedCreate)}
                    className="flex items-center justify-between w-full text-left py-1 text-xs font-bold text-purple-600 dark:text-purple-400 hover:text-purple-700 dark:hover:text-purple-300"
                  >
                    <span className="flex items-center gap-1.5">
                      <Sliders className="w-3.5 h-3.5" />
                      <span>Customize Resource Limits & System Isolation</span>
                    </span>
                    {showAdvancedCreate ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
                  </button>

                  {showAdvancedCreate && (
                    <div className="mt-3 p-3.5 rounded-2xl bg-slate-50 dark:bg-slate-800/60 border border-slate-200 dark:border-slate-700 space-y-3 animate-in fade-in duration-150">
                      <div className="text-[11px] text-slate-500 dark:text-slate-400">
                        Configures Linux POSIX user isolation, systemd cgroups v2 slice, memory ceiling, and CPU quotas.
                      </div>

                      <div className="grid grid-cols-2 gap-3">
                        <div>
                          <label className="block text-[11px] font-semibold text-slate-700 dark:text-slate-300 mb-1">
                            RAM Limit (MB)
                          </label>
                          <input
                            type="number"
                            min="256"
                            step="128"
                            value={createRamMB}
                            onChange={e => setCreateRamMB(Number(e.target.value))}
                            className="w-full px-3 py-1.5 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white text-xs font-medium"
                          />
                        </div>
                        <div>
                          <label className="block text-[11px] font-semibold text-slate-700 dark:text-slate-300 mb-1">
                            Storage Quota (GB)
                          </label>
                          <input
                            type="number"
                            min="1"
                            step="1"
                            value={createStorageGB}
                            onChange={e => setCreateStorageGB(Number(e.target.value))}
                            className="w-full px-3 py-1.5 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white text-xs font-medium"
                          />
                        </div>
                        <div>
                          <label className="block text-[11px] font-semibold text-slate-700 dark:text-slate-300 mb-1">
                            CPU Limit (%)
                          </label>
                          <input
                            type="number"
                            min="10"
                            max="800"
                            step="10"
                            value={createCpuPct}
                            onChange={e => setCreateCpuPct(Number(e.target.value))}
                            className="w-full px-3 py-1.5 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white text-xs font-medium"
                          />
                        </div>
                        <div>
                          <label className="block text-[11px] font-semibold text-slate-700 dark:text-slate-300 mb-1">
                            Max Tasks/Procs
                          </label>
                          <input
                            type="number"
                            min="10"
                            max="1000"
                            step="10"
                            value={createTasksMax}
                            onChange={e => setCreateTasksMax(Number(e.target.value))}
                            className="w-full px-3 py-1.5 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white text-xs font-medium"
                          />
                        </div>
                      </div>
                    </div>
                  )}
                </div>

                <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-100 dark:border-slate-800">
                  <button
                    type="button"
                    onClick={() => setCreateModalOpen(false)}
                    className="px-5 py-2 rounded-xl text-xs font-bold text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={actionLoading === 'create'}
                    className="px-6 py-2.5 rounded-xl bg-purple-600 hover:bg-purple-700 text-white text-xs font-bold shadow-md shadow-purple-500/20 flex items-center gap-2"
                  >
                    {actionLoading === 'create' && <RefreshCw className="w-4 h-4 animate-spin" />}
                    <span>Provision Account</span>
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
                    Confirm Account Suspension
                  </h3>
                  <p className="text-xs text-slate-400">{selectedAccount.domain}</p>
                </div>
              </div>

              <div className="space-y-3 text-xs">
                <p className="text-slate-600 dark:text-slate-400">
                  Suspending this account will display a custom suspension notice to website visitors and temporarily disable FTP and email access.
                </p>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">Suspension Reason:</label>
                  <select
                    value={suspendReason}
                    onChange={e => setSuspendReason(e.target.value)}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  >
                    <option value="Overdue Invoice">Overdue Invoice</option>
                    <option value="Resource Limit Exceeded">Resource Limit Exceeded (CPU/RAM)</option>
                    <option value="Terms of Service Violation">Terms of Service Violation</option>
                    <option value="Malware or Phishing Detected">Malware / Phishing Activity Detected</option>
                    <option value="Client Requested Suspension">Client Requested Suspension</option>
                  </select>
                </div>
              </div>

              <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-100 dark:border-slate-800">
                <button
                  type="button"
                  onClick={() => setSuspendModalOpen(false)}
                  className="px-5 py-2 rounded-xl text-xs font-bold text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={handleConfirmSuspend}
                  disabled={actionLoading === `suspend-${selectedAccount.id}`}
                  className="px-6 py-2 rounded-xl bg-rose-600 hover:bg-rose-700 text-white text-xs font-bold shadow-md shadow-rose-500/20 flex items-center gap-2"
                >
                  <span>Suspend Account</span>
                </button>
              </div>
            </div>
          </div>
        )}

        {/* MODAL 3: RESOURCE LIMITS & SYSTEM ISOLATION MODAL */}
        {limitsModalOpen && selectedAccount && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/60 backdrop-blur-sm animate-in fade-in duration-200">
            <div className="relative w-full max-w-2xl max-h-[92vh] flex flex-col bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-2xl overflow-hidden">
              {/* Header */}
              <div className="flex items-center justify-between p-6 pb-4 border-b border-slate-100 dark:border-slate-800 flex-shrink-0">
                <div className="flex items-center gap-3">
                  <div className="p-2.5 rounded-2xl bg-purple-500/10 text-purple-600 dark:text-purple-400">
                    <Sliders className="w-6 h-6" />
                  </div>
                  <div>
                    <h3 className="text-lg font-bold text-slate-900 dark:text-white">
                      Resource Limits & Security Isolation
                    </h3>
                    <p className="text-xs text-slate-400">
                      WHM Account: <span className="font-semibold text-slate-700 dark:text-slate-300">{selectedAccount.domain}</span> ({selectedAccount.username})
                    </p>
                  </div>
                </div>

                <button
                  onClick={() => setLimitsModalOpen(false)}
                  className="p-2 rounded-xl text-slate-400 hover:text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              {/* Scrollable Body */}
              <div className="p-6 space-y-6 overflow-y-auto text-xs">
                {/* Isolation Architecture Card */}
                <div className="p-4 rounded-2xl bg-gradient-to-r from-purple-950/30 via-slate-900/40 to-slate-900/20 border border-purple-500/20 dark:border-purple-500/30 space-y-2">
                  <div className="flex items-center gap-2 text-purple-600 dark:text-purple-400 font-bold">
                    <Shield className="w-4 h-4" />
                    <span>Active Linux POSIX & cgroups v2 Isolation</span>
                  </div>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-[11px] text-slate-600 dark:text-slate-300">
                    <div className="flex items-center gap-1.5 font-mono">
                      <span className="text-slate-400">System User:</span>
                      <span className="font-bold text-slate-900 dark:text-white">{selectedAccount.username}</span>
                      <span className="text-emerald-500 font-semibold">(0750)</span>
                    </div>
                    <div className="flex items-center gap-1.5 font-mono">
                      <span className="text-slate-400">Home/Root:</span>
                      <span className="truncate text-slate-900 dark:text-white">/home/{selectedAccount.username}</span>
                    </div>
                    <div className="flex items-center gap-1.5 font-mono">
                      <span className="text-slate-400">Systemd Slice:</span>
                      <span className="text-purple-600 dark:text-purple-400">hostvra-user-{selectedAccount.username}.slice</span>
                    </div>
                    <div className="flex items-center gap-1.5 font-mono">
                      <span className="text-slate-400">FPM Pool:</span>
                      <span className="truncate text-slate-900 dark:text-white">php{limitPHP}-fpm-{selectedAccount.username}.sock</span>
                    </div>
                  </div>
                </div>

                {/* Package Template Quick Preset */}
                <div className="space-y-1.5">
                  <label className="font-bold text-slate-700 dark:text-slate-300">
                    Apply Package Template Preset:
                  </label>
                  <div className="flex flex-wrap items-center gap-2">
                    <select
                      value={targetPlanId}
                      onChange={e => handleApplyPlanToLimits(e.target.value)}
                      className="flex-1 px-4 py-2 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white text-xs font-medium"
                    >
                      <option value="">-- Custom Limits Configuration --</option>
                      {plans.map(p => (
                        <option key={p.id} value={p.id}>
                          {p.name} ({(p.disk_space_mb / 1024).toFixed(0)} GB Disk, {(p.bandwidth_mb / 1024).toFixed(0)} GB Bandwidth, {p.max_websites} Sites)
                        </option>
                      ))}
                    </select>
                  </div>
                </div>

                {/* Core Hardware Resource Limits */}
                <div className="space-y-4">
                  <h4 className="font-bold text-xs uppercase tracking-wider text-slate-400 flex items-center gap-1.5">
                    <Cpu className="w-3.5 h-3.5 text-purple-500" />
                    <span>Hardware Resource Quotas (cgroups v2)</span>
                  </h4>

                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    {/* RAM Limit */}
                    <div className="p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 space-y-2.5">
                      <div className="flex justify-between items-center">
                        <label className="font-bold text-slate-700 dark:text-slate-300 flex items-center gap-1.5">
                          <Gauge className="w-3.5 h-3.5 text-purple-600 dark:text-purple-400" />
                          <span>RAM Limit (MB)</span>
                        </label>
                        <span className="font-bold text-purple-600 dark:text-purple-400 font-mono">
                          {limitRamMB >= 1024 ? `${(limitRamMB / 1024).toFixed(1)} GB` : `${limitRamMB} MB`}
                        </span>
                      </div>
                      <input
                        type="range"
                        min="256"
                        max="8192"
                        step="128"
                        value={limitRamMB}
                        onChange={e => setLimitRamMB(Number(e.target.value))}
                        className="w-full accent-purple-600 cursor-pointer"
                      />
                      <div className="flex items-center gap-1.5 pt-1">
                        {[512, 1024, 2048, 4096].map(mb => (
                          <button
                            key={mb}
                            type="button"
                            onClick={() => setLimitRamMB(mb)}
                            className={`px-2 py-1 rounded-lg text-[10px] font-bold border transition-colors ${
                              limitRamMB === mb
                                ? 'bg-purple-600 text-white border-purple-600'
                                : 'border-slate-200 dark:border-slate-700 text-slate-500 hover:text-slate-800 dark:hover:text-white'
                            }`}
                          >
                            {mb >= 1024 ? `${mb / 1024}GB` : `${mb}MB`}
                          </button>
                        ))}
                      </div>
                      <p className="text-[10px] text-slate-400 leading-tight">
                        Hard limit (`memory.max`). Exceeding throttles or triggers OOM protection solely for this tenant.
                      </p>
                    </div>

                    {/* Storage Quota */}
                    <div className="p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 space-y-2.5">
                      <div className="flex justify-between items-center">
                        <label className="font-bold text-slate-700 dark:text-slate-300 flex items-center gap-1.5">
                          <HardDrive className="w-3.5 h-3.5 text-purple-600 dark:text-purple-400" />
                          <span>Storage Quota (GB)</span>
                        </label>
                        <span className="font-bold text-purple-600 dark:text-purple-400 font-mono">
                          {limitStorageGB} GB
                        </span>
                      </div>
                      <input
                        type="range"
                        min="1"
                        max="200"
                        step="1"
                        value={limitStorageGB}
                        onChange={e => setLimitStorageGB(Number(e.target.value))}
                        className="w-full accent-purple-600 cursor-pointer"
                      />
                      <div className="flex items-center gap-1.5 pt-1">
                        {[5, 10, 25, 50, 100].map(gb => (
                          <button
                            key={gb}
                            type="button"
                            onClick={() => setLimitStorageGB(gb)}
                            className={`px-2 py-1 rounded-lg text-[10px] font-bold border transition-colors ${
                              limitStorageGB === gb
                                ? 'bg-purple-600 text-white border-purple-600'
                                : 'border-slate-200 dark:border-slate-700 text-slate-500 hover:text-slate-800 dark:hover:text-white'
                            }`}
                          >
                            {gb}GB
                          </button>
                        ))}
                      </div>
                      <p className="text-[10px] text-slate-400 leading-tight">
                        Filesystem quota on `/home/{selectedAccount.username}` preventing disk saturation.
                      </p>
                    </div>

                    {/* CPU Limit */}
                    <div className="p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 space-y-2.5">
                      <div className="flex justify-between items-center">
                        <label className="font-bold text-slate-700 dark:text-slate-300 flex items-center gap-1.5">
                          <Cpu className="w-3.5 h-3.5 text-purple-600 dark:text-purple-400" />
                          <span>CPU Limit (%)</span>
                        </label>
                        <span className="font-bold text-purple-600 dark:text-purple-400 font-mono">
                          {limitCpuPct}% ({limitCpuPct >= 100 ? `${(limitCpuPct / 100).toFixed(1)} Core` : `${limitCpuPct}% Core`})
                        </span>
                      </div>
                      <input
                        type="range"
                        min="10"
                        max="800"
                        step="10"
                        value={limitCpuPct}
                        onChange={e => setLimitCpuPct(Number(e.target.value))}
                        className="w-full accent-purple-600 cursor-pointer"
                      />
                      <div className="flex items-center gap-1.5 pt-1">
                        {[50, 100, 200, 400].map(pct => (
                          <button
                            key={pct}
                            type="button"
                            onClick={() => setLimitCpuPct(pct)}
                            className={`px-2 py-1 rounded-lg text-[10px] font-bold border transition-colors ${
                              limitCpuPct === pct
                                ? 'bg-purple-600 text-white border-purple-600'
                                : 'border-slate-200 dark:border-slate-700 text-slate-500 hover:text-slate-800 dark:hover:text-white'
                            }`}
                          >
                            {pct}%
                          </button>
                        ))}
                      </div>
                      <p className="text-[10px] text-slate-400 leading-tight">
                        cgroups v2 `cpu.max` CFS bandwidth slice. Enforces fair sharing across cores.
                      </p>
                    </div>

                    {/* Max Tasks / Procs */}
                    <div className="p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 space-y-2.5">
                      <div className="flex justify-between items-center">
                        <label className="font-bold text-slate-700 dark:text-slate-300 flex items-center gap-1.5">
                          <Activity className="w-3.5 h-3.5 text-purple-600 dark:text-purple-400" />
                          <span>Max Tasks & Processes</span>
                        </label>
                        <span className="font-bold text-purple-600 dark:text-purple-400 font-mono">
                          {limitTasks} Procs
                        </span>
                      </div>
                      <input
                        type="range"
                        min="10"
                        max="500"
                        step="10"
                        value={limitTasks}
                        onChange={e => setLimitTasks(Number(e.target.value))}
                        className="w-full accent-purple-600 cursor-pointer"
                      />
                      <div className="flex items-center gap-1.5 pt-1">
                        {[50, 100, 200, 300].map(procs => (
                          <button
                            key={procs}
                            type="button"
                            onClick={() => setLimitTasks(procs)}
                            className={`px-2 py-1 rounded-lg text-[10px] font-bold border transition-colors ${
                              limitTasks === procs
                                ? 'bg-purple-600 text-white border-purple-600'
                                : 'border-slate-200 dark:border-slate-700 text-slate-500 hover:text-slate-800 dark:hover:text-white'
                            }`}
                          >
                            {procs}
                          </button>
                        ))}
                      </div>
                      <p className="text-[10px] text-slate-400 leading-tight">
                        systemd `TasksMax` setting. Prevents fork bombs or thread exhaustion attacks.
                      </p>
                    </div>
                  </div>
                </div>

                {/* Network & Inodes Limits */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div className="space-y-1">
                    <label className="font-bold text-slate-700 dark:text-slate-300 flex items-center gap-1.5">
                      <Network className="w-3.5 h-3.5 text-purple-500" />
                      <span>Bandwidth Quota (GB / Month)</span>
                    </label>
                    <input
                      type="number"
                      min="1"
                      step="5"
                      value={limitBwGB}
                      onChange={e => setLimitBwGB(Number(e.target.value))}
                      className="w-full px-3.5 py-2 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white text-xs font-medium"
                    />
                  </div>

                  <div className="space-y-1">
                    <label className="font-bold text-slate-700 dark:text-slate-300 flex items-center gap-1.5">
                      <Layers className="w-3.5 h-3.5 text-purple-500" />
                      <span>Inodes Quota (Files & Dirs)</span>
                    </label>
                    <input
                      type="number"
                      min="10000"
                      step="10000"
                      value={limitInodes}
                      onChange={e => setLimitInodes(Number(e.target.value))}
                      className="w-full px-3.5 py-2 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white text-xs font-medium"
                    />
                  </div>
                </div>

                {/* PHP Runtime & Strict open_basedir Security Isolation */}
                <div className="p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 space-y-3">
                  <h4 className="font-bold text-xs uppercase tracking-wider text-slate-400 flex items-center gap-1.5">
                    <Shield className="w-3.5 h-3.5 text-emerald-500" />
                    <span>PHP-FPM Runtime & Directory Confinement</span>
                  </h4>

                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div className="space-y-1">
                      <label className="font-bold text-slate-700 dark:text-slate-300">Dedicated PHP Pool Version</label>
                      <select
                        value={limitPHP}
                        onChange={e => setLimitPHP(e.target.value)}
                        className="w-full px-3.5 py-2 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white text-xs font-medium"
                      >
                        <option value="8.4">PHP 8.4 (Latest)</option>
                        <option value="8.3">PHP 8.3 (Stable)</option>
                        <option value="8.2">PHP 8.2</option>
                        <option value="8.1">PHP 8.1</option>
                      </select>
                    </div>

                    <div className="space-y-1">
                      <label className="font-bold text-slate-700 dark:text-slate-300">Max Websites Allowed</label>
                      <input
                        type="number"
                        min="1"
                        max="50"
                        value={limitWebsites}
                        onChange={e => setLimitWebsites(Number(e.target.value))}
                        className="w-full px-3.5 py-2 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white text-xs font-medium"
                      />
                    </div>
                  </div>

                  <div className="pt-2 border-t border-slate-200/60 dark:border-slate-700/60">
                    <label className="flex items-start gap-3 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={limitOpenBaseDir}
                        onChange={e => setLimitOpenBaseDir(e.target.checked)}
                        className="mt-0.5 rounded text-purple-600 focus:ring-purple-500 w-4 h-4 cursor-pointer"
                      />
                      <div>
                        <div className="font-bold text-slate-800 dark:text-slate-200 text-xs">
                          Enforce Strict PHP <code className="text-purple-600 dark:text-purple-400">open_basedir</code> Directory Isolation
                        </div>
                        <div className="text-[11px] text-slate-500 dark:text-slate-400">
                          Restricts all PHP file operations strictly inside <code className="text-slate-700 dark:text-slate-300">/home/{selectedAccount.username}/public_html/</code> and <code className="text-slate-700 dark:text-slate-300">/tmp/</code>. Prevents cross-account directory traversal.
                        </div>
                      </div>
                    </label>
                  </div>
                </div>
              </div>

              {/* Footer */}
              <div className="flex items-center justify-between p-6 pt-4 border-t border-slate-100 dark:border-slate-800 flex-shrink-0 bg-slate-50/50 dark:bg-slate-800/20">
                <div className="text-[11px] text-slate-400 flex items-center gap-1.5">
                  <CheckCircle2 className="w-3.5 h-3.5 text-emerald-500" />
                  <span>Real-time cgroups v2 & kernel slice reconfiguration</span>
                </div>

                <div className="flex items-center gap-3">
                  <button
                    type="button"
                    onClick={() => setLimitsModalOpen(false)}
                    className="px-5 py-2 rounded-xl text-xs font-bold text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    onClick={handleSaveLimits}
                    disabled={actionLoading === `limits-${selectedAccount.id}`}
                    className="px-6 py-2.5 rounded-xl bg-purple-600 hover:bg-purple-700 active:bg-purple-800 text-white text-xs font-bold shadow-md shadow-purple-500/20 flex items-center gap-2 transition-all disabled:opacity-50"
                  >
                    {actionLoading === `limits-${selectedAccount.id}` && (
                      <RefreshCw className="w-4 h-4 animate-spin" />
                    )}
                    <span>Save & Apply Isolation</span>
                  </button>
                </div>
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
                      Client Session Generated
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
                  <span className="text-slate-500">Username:</span>
                  <span className="font-mono font-bold text-slate-900 dark:text-white">{loginData.username}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-500">Impersonation Token:</span>
                  <span className="font-mono text-purple-600 dark:text-purple-400">{loginData.token.slice(0, 16)}...</span>
                </div>
              </div>

              <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-100 dark:border-slate-800">
                <button
                  type="button"
                  onClick={() => setLoginModalOpen(false)}
                  className="px-5 py-2 rounded-xl text-xs font-bold text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800"
                >
                  Close
                </button>
                <a
                  href={loginData.login_url}
                  className="px-6 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-bold shadow-md shadow-emerald-500/20 flex items-center gap-1.5"
                >
                  <span>Access Client Panel</span>
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
