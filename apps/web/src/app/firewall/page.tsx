'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  ShieldAlert,
  ShieldCheck,
  Plus,
  Trash2,
  Lock,
  RefreshCw,
  AlertTriangle,
  X,
  Search,
  Power,
  UserX,
  Radio,
  CheckCircle2,
  AlertCircle,
  Unlock,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  apiFetch,
  FirewallStatus,
  FirewallRule,
  Fail2banJail,
  BannedIPItem,
} from '@/lib/api';

export default function FirewallPage() {
  const [activeTab, setActiveTab] = useState<'rules' | 'fail2ban'>('rules');

  // Status & Data
  const [status, setStatus] = useState<FirewallStatus | null>(null);
  const [rules, setRules] = useState<FirewallRule[]>([]);
  const [jails, setJails] = useState<Fail2banJail[]>([]);
  const [bannedIPs, setBannedIPs] = useState<BannedIPItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [search, setSearch] = useState('');

  // Alerts & Messages
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  // Add Rule Modal
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [port, setPort] = useState('');
  const [protocol, setProtocol] = useState<'tcp' | 'udp' | 'both'>('tcp');
  const [fromIP, setFromIP] = useState('');
  const [action, setAction] = useState<'allow' | 'deny'>('allow');
  const [comment, setComment] = useState('');
  const [submittingRule, setSubmittingRule] = useState(false);

  // Ban IP Modal
  const [banModalOpen, setBanModalOpen] = useState(false);
  const [banTargetIP, setBanTargetIP] = useState('');
  const [banSelectedJail, setBanSelectedJail] = useState('sshd');
  const [submittingBan, setSubmittingBan] = useState(false);

  // Action Loading states
  const [togglingFirewall, setTogglingFirewall] = useState(false);
  const [actionLoadingId, setActionLoadingId] = useState<string | null>(null);

  const showNotification = (msg: string, isError = false) => {
    if (isError) {
      setErrorMsg(msg);
      setTimeout(() => setErrorMsg(null), 7000);
    } else {
      setSuccessMsg(msg);
      setTimeout(() => setSuccessMsg(null), 5000);
    }
  };

  const loadData = useCallback(async (showIndicator = true) => {
    if (showIndicator) setRefreshing(true);
    setErrorMsg(null);

    try {
      // 1. Fetch Firewall Status
      const statusRes = await apiFetch<FirewallStatus>('/api/v1/firewall/status');
      if (statusRes.success && statusRes.data) {
        setStatus(statusRes.data);
      }

      // 2. Fetch Rules
      const rulesRes = await apiFetch<{ rules: FirewallRule[]; count: number }>('/api/v1/firewall/rules');
      if (rulesRes.success && rulesRes.data) {
        setRules(rulesRes.data.rules || []);
      }

      // 3. Fetch Fail2ban Jails
      const jailsRes = await apiFetch<{ installed: boolean; jails: Fail2banJail[] }>('/api/v1/firewall/fail2ban/jails');
      if (jailsRes.success && jailsRes.data && jailsRes.data.jails) {
        setJails(jailsRes.data.jails);
      }

      // 4. Fetch Banned IPs
      const bannedRes = await apiFetch<{ banned_ips: BannedIPItem[]; count: number }>('/api/v1/firewall/fail2ban/banned');
      if (bannedRes.success && bannedRes.data && bannedRes.data.banned_ips) {
        setBannedIPs(bannedRes.data.banned_ips);
      }
    } catch (err: any) {
      showNotification(err.message || 'Failed to fetch firewall status from server', true);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    loadData(false);
  }, [loadData]);

  // Toggle Firewall (Enable/Disable)
  const handleToggleFirewall = async () => {
    if (!status) return;
    const enabling = !status.is_active;

    if (!enabling) {
      const confirmDisable = window.confirm(
        'WARNING: Disabling the firewall removes packet filtering and leaves all ports exposed. Continue?'
      );
      if (!confirmDisable) return;
    }

    setTogglingFirewall(true);
    const endpoint = enabling ? '/api/v1/firewall/enable' : '/api/v1/firewall/disable';
    const res = await apiFetch<any>(endpoint, { method: 'POST' });
    setTogglingFirewall(false);

    if (res.success) {
      showNotification(enabling ? 'Firewall enabled with SSH safety rules' : 'Firewall disabled successfully');
      loadData(false);
    } else {
      showNotification(res.error?.message || 'Failed to update firewall status', true);
    }
  };

  // Add Firewall Rule
  const handleAddRule = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!port && !fromIP) {
      showNotification('Please provide either a port or a source IP/CIDR', true);
      return;
    }

    // Safety lockout check
    if (port.trim() === '22' && action === 'deny') {
      showNotification('SSH Lockout Protection: Denying port 22 is blocked to prevent losing administrative access.', true);
      return;
    }

    setSubmittingRule(true);
    const res = await apiFetch<any>('/api/v1/firewall/rules', {
      method: 'POST',
      body: JSON.stringify({
        port: port.trim(),
        protocol,
        from_ip: fromIP.trim(),
        action,
        comment: comment.trim(),
      }),
    });
    setSubmittingRule(false);

    if (res.success) {
      showNotification('Firewall rule applied successfully');
      setAddModalOpen(false);
      setPort('');
      setFromIP('');
      setComment('');
      loadData(false);
    } else {
      showNotification(res.error?.message || 'Failed to add firewall rule', true);
    }
  };

  // Delete Rule
  const handleDeleteRule = async (rule: FirewallRule) => {
    const isPort22 = rule.to.startsWith('22/') || rule.to === '22';
    if (isPort22 && rule.action.toLowerCase().includes('allow')) {
      const proceed = window.confirm(
        'CAUTION: You are attempting to remove an SSH allow rule. Make sure you have alternative console access. Proceed?'
      );
      if (!proceed) return;
    }

    if (!window.confirm(`Are you sure you want to delete rule #${rule.number} (${rule.to})?`)) {
      return;
    }

    setActionLoadingId(rule.id);
    const res = await apiFetch<any>(`/api/v1/firewall/rules/${rule.id}`, {
      method: 'DELETE',
    });
    setActionLoadingId(null);

    if (res.success) {
      showNotification(`Rule #${rule.number} deleted successfully`);
      loadData(false);
    } else {
      showNotification(res.error?.message || 'Failed to delete rule', true);
    }
  };

  // Manual Ban IP
  const handleBanIP = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!banTargetIP.trim()) return;

    setSubmittingBan(true);
    const res = await apiFetch<any>('/api/v1/firewall/fail2ban/ban', {
      method: 'POST',
      body: JSON.stringify({
        ip: banTargetIP.trim(),
        jail: banSelectedJail,
      }),
    });
    setSubmittingBan(false);

    if (res.success) {
      showNotification(`IP ${banTargetIP} has been banned in jail [${banSelectedJail}]`);
      setBanModalOpen(false);
      setBanTargetIP('');
      loadData(false);
    } else {
      showNotification(res.error?.message || 'Failed to ban IP address', true);
    }
  };

  // Unban IP
  const handleUnbanIP = async (ip: string, jail: string) => {
    if (!window.confirm(`Unban IP ${ip} immediately?`)) return;

    setActionLoadingId(`unban-${ip}`);
    const res = await apiFetch<any>('/api/v1/firewall/fail2ban/unban', {
      method: 'POST',
      body: JSON.stringify({ ip, jail }),
    });
    setActionLoadingId(null);

    if (res.success) {
      showNotification(`IP ${ip} unbanned successfully`);
      loadData(false);
    } else {
      showNotification(res.error?.message || 'Failed to unban IP', true);
    }
  };

  const handlePresetSelect = (presetPort: string, presetComment: string, proto: 'tcp' | 'udp' | 'both' = 'tcp') => {
    setPort(presetPort);
    setComment(presetComment);
    setProtocol(proto);
    setAction('allow');
  };

  // Filtering
  const filteredRules = rules.filter(
    (r) =>
      r.to.toLowerCase().includes(search.toLowerCase()) ||
      r.from.toLowerCase().includes(search.toLowerCase()) ||
      r.comment.toLowerCase().includes(search.toLowerCase()) ||
      r.action.toLowerCase().includes(search.toLowerCase())
  );

  const filteredBanned = bannedIPs.filter(
    (b) =>
      b.ip.toLowerCase().includes(search.toLowerCase()) ||
      b.jail.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <DashboardShell>
      <div className="space-y-6 animate-fadeIn max-w-7xl mx-auto pb-16">
        {/* Alerts */}
        {errorMsg && (
          <div className="p-4 rounded-xl bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900/50 flex items-start gap-3 text-rose-800 dark:text-rose-200 text-sm shadow-xs">
            <AlertCircle className="w-5 h-5 flex-shrink-0 text-rose-600 dark:text-rose-400 mt-0.5" />
            <div className="flex-1 font-medium">{errorMsg}</div>
            <button onClick={() => setErrorMsg(null)} className="text-rose-600 dark:text-rose-400 hover:opacity-75">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {successMsg && (
          <div className="p-4 rounded-xl bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200 dark:border-emerald-900/50 flex items-start gap-3 text-emerald-800 dark:text-emerald-200 text-sm shadow-xs">
            <CheckCircle2 className="w-5 h-5 flex-shrink-0 text-emerald-600 dark:text-emerald-400 mt-0.5" />
            <div className="flex-1 font-medium">{successMsg}</div>
            <button onClick={() => setSuccessMsg(null)} className="text-emerald-600 dark:text-emerald-400 hover:opacity-75">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold tracking-tight text-slate-950 dark:text-white">
                Firewall & Intrusion Defense
              </h1>
              <span className="px-2.5 py-0.5 rounded-full text-xs font-mono font-semibold uppercase bg-indigo-50 dark:bg-indigo-950/60 text-indigo-700 dark:text-indigo-400 border border-indigo-200 dark:border-indigo-800">
                {status?.backend || 'UFW'}
              </span>
            </div>
            <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
              Production packet filtering, active port rules, and real-time Fail2ban jail enforcement.
            </p>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => loadData(true)}
              disabled={refreshing}
              className="p-2.5 rounded-xl border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-900 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-surface-800 transition-colors shadow-xs"
              title="Refresh status"
            >
              <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin text-indigo-500' : ''}`} />
            </button>

            {activeTab === 'rules' ? (
              <button
                onClick={() => setAddModalOpen(true)}
                className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold shadow-md shadow-indigo-600/20 transition-all"
              >
                <Plus className="w-4 h-4" />
                Add Firewall Rule
              </button>
            ) : (
              <button
                onClick={() => setBanModalOpen(true)}
                className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-rose-600 hover:bg-rose-500 text-white text-sm font-semibold shadow-md shadow-rose-600/20 transition-all"
              >
                <UserX className="w-4 h-4" />
                Manual Ban IP
              </button>
            )}
          </div>
        </div>

        {/* Status Dashboard Banner */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {/* Card 1: Firewall Status */}
          <div className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 shadow-xs flex items-center justify-between">
            <div className="flex items-center gap-3.5">
              <div
                className={`w-11 h-11 rounded-xl flex items-center justify-center ${
                  status?.is_active
                    ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20'
                    : 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20'
                }`}
              >
                {status?.is_active ? <ShieldCheck className="w-6 h-6" /> : <AlertTriangle className="w-6 h-6" />}
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <span className="font-bold text-slate-950 dark:text-white text-sm">
                    {status?.is_active ? 'Firewall Active' : 'Firewall Inactive'}
                  </span>
                  <span
                    className={`w-2 h-2 rounded-full ${
                      status?.is_active ? 'bg-emerald-500 animate-pulse' : 'bg-amber-500'
                    }`}
                  />
                </div>
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 font-medium">
                  {status?.is_active
                    ? 'Default Ingress: DENY • Egress: ALLOW'
                    : 'Packet filtering disabled'}
                </p>
              </div>
            </div>

            <button
              onClick={handleToggleFirewall}
              disabled={togglingFirewall}
              className={`px-4 py-2 rounded-xl transition-all text-xs font-bold flex items-center gap-2 shadow-xs cursor-pointer ${
                status?.is_active
                  ? 'border border-rose-300 hover:border-rose-400 dark:border-rose-900/60 bg-rose-50 hover:bg-rose-100 dark:bg-rose-950/40 text-rose-700 dark:text-rose-400'
                  : 'bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white border border-emerald-600 shadow-sm'
              }`}
            >
              <Power className={`w-4 h-4 ${status?.is_active ? 'text-rose-600 dark:text-rose-400' : 'text-white'}`} />
              <span>{togglingFirewall ? 'Updating...' : status?.is_active ? 'Disable Firewall' : 'Enable Firewall'}</span>
            </button>
          </div>

          {/* Card 2: SSH Lockout Guard */}
          <div className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 shadow-xs flex items-center gap-3.5">
            <div className="w-11 h-11 rounded-xl bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border border-indigo-500/20 flex items-center justify-center flex-shrink-0">
              <Lock className="w-6 h-6" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold text-slate-950 dark:text-white text-sm">SSH Lockout Guard</span>
                <span className="px-2 py-0.5 rounded-full text-[10px] font-bold uppercase bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                  Enforced
                </span>
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 font-medium">
                Port 22 deny actions blocked to preserve admin connectivity
              </p>
            </div>
          </div>

          {/* Card 3: Fail2ban Summary */}
          <div className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 shadow-xs flex items-center gap-3.5">
            <div className="w-11 h-11 rounded-xl bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20 flex items-center justify-center flex-shrink-0">
              <Radio className="w-6 h-6" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold text-slate-950 dark:text-white text-sm">Fail2ban Intrusion</span>
                <span
                  className={`px-2 py-0.5 rounded-full text-[10px] font-bold uppercase ${
                    status?.fail2ban_active
                      ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20'
                      : 'bg-slate-100 dark:bg-surface-800 text-slate-500'
                  }`}
                >
                  {status?.fail2ban_active ? 'Running' : 'Offline'}
                </span>
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 font-medium">
                {jails.length} Jails Active • {bannedIPs.length} Banned IPs
              </p>
            </div>
          </div>
        </div>

        {/* Tab Navigation */}
        <div className="flex items-center gap-1.5 p-1 bg-slate-100/80 dark:bg-slate-800/60 rounded-xl overflow-x-auto text-xs font-semibold">
          <button
            role="tab"
            data-tab="true"
            onClick={() => setActiveTab('rules')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all cursor-pointer font-medium whitespace-nowrap ${
              activeTab === 'rules'
                ? 'bg-white dark:bg-slate-900 text-indigo-600 dark:text-indigo-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-white/40 dark:hover:bg-slate-700/40'
            }`}
          >
            <ShieldCheck className="w-4 h-4" />
            <span>Firewall Rules ({rules.length})</span>
          </button>
          <button
            role="tab"
            data-tab="true"
            onClick={() => setActiveTab('fail2ban')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all cursor-pointer font-medium whitespace-nowrap ${
              activeTab === 'fail2ban'
                ? 'bg-white dark:bg-slate-900 text-indigo-600 dark:text-indigo-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-white/40 dark:hover:bg-slate-700/40'
            }`}
          >
            <UserX className="w-4 h-4" />
            <span>Fail2ban Jails & Banned IPs ({bannedIPs.length})</span>
          </button>
        </div>

        {/* Search Bar */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4 bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl p-3 shadow-xs">
          <div className="flex items-center gap-3 w-full sm:w-80 bg-slate-50 dark:bg-[#121824] border border-slate-300 dark:border-surface-700 rounded-xl px-3.5 py-2 shadow-xs focus-within:border-indigo-500 focus-within:ring-2 focus-within:ring-indigo-500/20 transition-all">
            <Search className="w-4 h-4 text-slate-400 flex-shrink-0" />
            <input
              type="text"
              placeholder={activeTab === 'rules' ? 'Search port, source IP, comments...' : 'Search banned IP or jail...'}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full bg-transparent text-xs font-medium text-slate-950 dark:text-white placeholder:text-slate-400 focus:outline-none"
            />
            {search && (
              <button onClick={() => setSearch('')} className="text-slate-400 hover:text-slate-600 dark:hover:text-white p-0.5">
                <X className="w-3.5 h-3.5" />
              </button>
            )}
          </div>
          <div className="text-xs font-semibold text-slate-600 dark:text-slate-400 px-2 self-end sm:self-center">
            {activeTab === 'rules'
              ? `Showing ${filteredRules.length} of ${rules.length} rules`
              : `Showing ${filteredBanned.length} of ${bannedIPs.length} banned IPs`}
          </div>
        </div>

        {/* TAB 1: RULES TABLE */}
        {activeTab === 'rules' && (
          <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm border-collapse">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#121824] text-slate-700 dark:text-slate-300 text-[11px] font-bold uppercase tracking-wider">
                    <th className="px-6 py-3.5">#</th>
                    <th className="px-6 py-3.5">Destination / Port</th>
                    <th className="px-6 py-3.5">Protocol</th>
                    <th className="px-6 py-3.5">Traffic Action</th>
                    <th className="px-6 py-3.5">Source (From)</th>
                    <th className="px-6 py-3.5">Rule Purpose / Comment</th>
                    <th className="px-6 py-3.5 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200 dark:divide-surface-800/80">
                  {loading ? (
                    <tr>
                      <td colSpan={7} className="px-6 py-12 text-center text-slate-500 font-medium text-sm">
                        <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 text-indigo-500" />
                        Querying kernel packet filtering rules...
                      </td>
                    </tr>
                  ) : filteredRules.length === 0 ? (
                    <tr>
                      <td colSpan={7} className="px-6 py-12 text-center text-slate-500 font-medium text-sm">
                        No firewall rules matched your search filter.
                      </td>
                    </tr>
                  ) : (
                    filteredRules.map((r) => {
                      const isSSH = r.to.startsWith('22/') || r.to === '22';
                      return (
                        <tr key={r.id} className="hover:bg-slate-50/80 dark:hover:bg-[#151d2d] transition-colors">
                          <td className="px-6 py-4 font-mono text-xs font-bold text-slate-400">
                            #{r.number}
                          </td>
                          <td className="px-6 py-4 font-mono font-bold text-slate-950 dark:text-white flex items-center gap-2">
                            {isSSH && (
                              <span title="Administrative SSH Port Protected by Guard">
                                <Lock className="w-3.5 h-3.5 text-emerald-500 flex-shrink-0" />
                              </span>
                            )}
                            <span>{r.to}</span>
                          </td>
                          <td className="px-6 py-4 uppercase font-mono text-xs font-semibold text-slate-700 dark:text-slate-300">
                            {r.protocol}
                          </td>
                          <td className="px-6 py-4">
                            <span
                              className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-[11px] font-bold uppercase ${
                                r.action.toLowerCase().includes('allow')
                                  ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20'
                                  : 'bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/20'
                              }`}
                            >
                              {r.action}
                            </span>
                          </td>
                          <td className="px-6 py-4 font-mono text-xs text-slate-700 dark:text-slate-300">
                            {r.from || 'Anywhere'}
                          </td>
                          <td className="px-6 py-4 text-xs font-medium text-slate-800 dark:text-slate-300">
                            {r.comment || '—'}
                          </td>
                          <td className="px-6 py-4 text-right">
                            <button
                              onClick={() => handleDeleteRule(r)}
                              disabled={actionLoadingId === r.id}
                              className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400 hover:text-rose-600 dark:hover:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/30 transition-colors"
                              title="Delete Rule"
                            >
                              {actionLoadingId === r.id ? (
                                <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                              ) : (
                                <Trash2 className="w-3.5 h-3.5" />
                              )}
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

        {/* TAB 2: FAIL2BAN JAILS & BANNED IPS */}
        {activeTab === 'fail2ban' && (
          <div className="space-y-6">
            {/* Jails Cards Grid */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              {jails.length === 0 ? (
                <div className="col-span-3 p-6 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 text-center text-slate-500 text-sm">
                  {status?.fail2ban_installed
                    ? 'No Fail2ban jails currently configured or active.'
                    : 'Fail2ban is not installed on this host. Install fail2ban to activate brute-force protection.'}
                </div>
              ) : (
                jails.map((j) => (
                  <div
                    key={j.name}
                    className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 shadow-xs"
                  >
                    <div className="flex items-center justify-between mb-3">
                      <div className="flex items-center gap-2">
                        <span className="font-bold text-slate-950 dark:text-white font-mono text-sm">
                          [{j.name}]
                        </span>
                        <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                          Active
                        </span>
                      </div>
                      <span className="text-xs font-mono font-bold text-rose-600 dark:text-rose-400">
                        {j.currently_banned} Banned
                      </span>
                    </div>

                    <div className="grid grid-cols-2 gap-2 text-xs font-medium text-slate-600 dark:text-slate-400">
                      <div className="p-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800">
                        <div className="text-[10px] uppercase text-slate-400">Current Failed</div>
                        <div className="font-bold text-slate-900 dark:text-white text-base mt-0.5">
                          {j.currently_failed}
                        </div>
                      </div>
                      <div className="p-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800">
                        <div className="text-[10px] uppercase text-slate-400">Total Banned</div>
                        <div className="font-bold text-slate-900 dark:text-white text-base mt-0.5">
                          {j.total_banned}
                        </div>
                      </div>
                    </div>
                  </div>
                ))
              )}
            </div>

            {/* Banned IPs Table */}
            <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs">
              <div className="p-4 border-b border-slate-200 dark:border-surface-800 flex items-center justify-between">
                <div className="flex items-center gap-2 font-bold text-slate-950 dark:text-white text-sm">
                  <UserX className="w-4 h-4 text-rose-500" />
                  Currently Banned IP Addresses
                </div>
                <span className="text-xs font-mono text-slate-500">
                  {bannedIPs.length} Total Blocked
                </span>
              </div>

              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm border-collapse">
                  <thead>
                    <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#121824] text-slate-700 dark:text-slate-300 text-[11px] font-bold uppercase tracking-wider">
                      <th className="px-6 py-3.5">IP Address</th>
                      <th className="px-6 py-3.5">Detected Jail</th>
                      <th className="px-6 py-3.5">Defense Status</th>
                      <th className="px-6 py-3.5 text-right">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-200 dark:divide-surface-800/80">
                    {filteredBanned.length === 0 ? (
                      <tr>
                        <td colSpan={4} className="px-6 py-12 text-center text-slate-500 font-medium text-sm">
                          No IP addresses are currently banned by Fail2ban.
                        </td>
                      </tr>
                    ) : (
                      filteredBanned.map((b) => (
                        <tr key={`${b.jail}-${b.ip}`} className="hover:bg-slate-50/80 dark:hover:bg-[#151d2d] transition-colors">
                          <td className="px-6 py-4 font-mono font-bold text-rose-600 dark:text-rose-400">
                            {b.ip}
                          </td>
                          <td className="px-6 py-4 font-mono text-xs font-semibold text-slate-700 dark:text-slate-300">
                            {b.jail}
                          </td>
                          <td className="px-6 py-4">
                            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-[11px] font-bold uppercase bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/20">
                              Dropped
                            </span>
                          </td>
                          <td className="px-6 py-4 text-right">
                            <button
                              onClick={() => handleUnbanIP(b.ip, b.jail)}
                              disabled={actionLoadingId === `unban-${b.ip}`}
                              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-xs font-semibold text-slate-700 dark:text-slate-300 hover:bg-emerald-50 hover:text-emerald-600 hover:border-emerald-300 dark:hover:bg-emerald-950/30 transition-all"
                            >
                              <Unlock className="w-3 h-3 text-emerald-500" />
                              {actionLoadingId === `unban-${b.ip}` ? 'Unbanning...' : 'Unban IP'}
                            </button>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {/* Modal: Add Firewall Rule */}
        {addModalOpen && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="w-full max-w-lg bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setAddModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-600 dark:text-indigo-400">
                  <ShieldAlert className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-950 dark:text-white">Add Ingress Firewall Rule</h2>
                  <p className="text-xs text-slate-500 dark:text-slate-400">Configure Linux packet filtering policy</p>
                </div>
              </div>

              {/* Presets */}
              <div className="mb-4">
                <label className="block text-[11px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400 mb-1.5">
                  Common Presets
                </label>
                <div className="flex flex-wrap gap-1.5">
                  <button
                    type="button"
                    onClick={() => handlePresetSelect('80', 'HTTP Web Server')}
                    className="px-2.5 py-1 rounded-lg bg-slate-50 dark:bg-surface-900 hover:bg-slate-100 dark:hover:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs text-slate-700 dark:text-slate-300 font-medium"
                  >
                    HTTP (80)
                  </button>
                  <button
                    type="button"
                    onClick={() => handlePresetSelect('443', 'HTTPS Secure Web')}
                    className="px-2.5 py-1 rounded-lg bg-slate-50 dark:bg-surface-900 hover:bg-slate-100 dark:hover:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs text-slate-700 dark:text-slate-300 font-medium"
                  >
                    HTTPS (443)
                  </button>
                  <button
                    type="button"
                    onClick={() => handlePresetSelect('3306', 'MySQL Database')}
                    className="px-2.5 py-1 rounded-lg bg-slate-50 dark:bg-surface-900 hover:bg-slate-100 dark:hover:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs text-slate-700 dark:text-slate-300 font-medium"
                  >
                    MySQL (3306)
                  </button>
                  <button
                    type="button"
                    onClick={() => handlePresetSelect('5432', 'PostgreSQL Database')}
                    className="px-2.5 py-1 rounded-lg bg-slate-50 dark:bg-surface-900 hover:bg-slate-100 dark:hover:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs text-slate-700 dark:text-slate-300 font-medium"
                  >
                    Postgres (5432)
                  </button>
                  <button
                    type="button"
                    onClick={() => handlePresetSelect('6379', 'Redis Server')}
                    className="px-2.5 py-1 rounded-lg bg-slate-50 dark:bg-surface-900 hover:bg-slate-100 dark:hover:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs text-slate-700 dark:text-slate-300 font-medium"
                  >
                    Redis (6379)
                  </button>
                </div>
              </div>

              <form onSubmit={handleAddRule} className="space-y-4">
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                      Port or Range
                    </label>
                    <input
                      type="text"
                      placeholder="e.g. 8080 or 3000:4000"
                      value={port}
                      onChange={(e) => setPort(e.target.value)}
                      className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-sm focus:outline-none focus:border-indigo-500"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                      Protocol
                    </label>
                    <select
                      value={protocol}
                      onChange={(e) => setProtocol(e.target.value as any)}
                      className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 text-sm focus:outline-none focus:border-indigo-500 uppercase font-semibold"
                    >
                      <option value="tcp">TCP</option>
                      <option value="udp">UDP</option>
                      <option value="both">BOTH (TCP & UDP)</option>
                    </select>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                      Traffic Action
                    </label>
                    <select
                      value={action}
                      onChange={(e) => setAction(e.target.value as any)}
                      className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 text-sm focus:outline-none focus:border-indigo-500 uppercase font-semibold"
                    >
                      <option value="allow">ALLOW</option>
                      <option value="deny">DENY</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                      Source IP / CIDR (Optional)
                    </label>
                    <input
                      type="text"
                      placeholder="Anywhere (or e.g. 1.2.3.4)"
                      value={fromIP}
                      onChange={(e) => setFromIP(e.target.value)}
                      className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-sm focus:outline-none focus:border-indigo-500"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                    Purpose / Comment
                  </label>
                  <input
                    type="text"
                    placeholder="e.g. Node.js backend port"
                    value={comment}
                    onChange={(e) => setComment(e.target.value)}
                    className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 text-sm focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div className="flex justify-end gap-2 pt-3">
                  <button
                    type="button"
                    onClick={() => setAddModalOpen(false)}
                    className="px-4 py-2.5 rounded-xl text-xs font-medium text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={submittingRule}
                    className="px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md shadow-indigo-600/20 transition-all flex items-center gap-1.5"
                  >
                    {submittingRule ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : null}
                    Apply Rule
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* Modal: Manual Ban IP */}
        {banModalOpen && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="w-full max-w-md bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setBanModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-rose-500/10 border border-rose-500/20 flex items-center justify-center text-rose-600 dark:text-rose-400">
                  <UserX className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-950 dark:text-white">Manual IP Ban</h2>
                  <p className="text-xs text-slate-500 dark:text-slate-400">Ban a malicious IP address across Fail2ban</p>
                </div>
              </div>

              <form onSubmit={handleBanIP} className="space-y-4">
                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                    Target Jail
                  </label>
                  <select
                    value={banSelectedJail}
                    onChange={(e) => setBanSelectedJail(e.target.value)}
                    className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 text-sm focus:outline-none focus:border-rose-500 font-mono"
                  >
                    {jails.length > 0 ? (
                      jails.map((j) => (
                        <option key={j.name} value={j.name}>
                          {j.name}
                        </option>
                      ))
                    ) : (
                      <option value="sshd">sshd</option>
                    )}
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                    IP Address to Ban
                  </label>
                  <input
                    type="text"
                    required
                    placeholder="e.g. 198.51.100.25"
                    value={banTargetIP}
                    onChange={(e) => setBanTargetIP(e.target.value)}
                    className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-sm focus:outline-none focus:border-rose-500"
                  />
                </div>

                <div className="flex justify-end gap-2 pt-3">
                  <button
                    type="button"
                    onClick={() => setBanModalOpen(false)}
                    className="px-4 py-2.5 rounded-xl text-xs font-medium text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={submittingBan}
                    className="px-5 py-2.5 rounded-xl bg-rose-600 hover:bg-rose-500 text-white text-xs font-semibold shadow-md shadow-rose-600/20 transition-all flex items-center gap-1.5"
                  >
                    {submittingBan ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : null}
                    Ban IP Immediately
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
