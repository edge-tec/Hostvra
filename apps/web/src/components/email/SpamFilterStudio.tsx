'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  ShieldAlert,
  ShieldCheck,
  Activity,
  RefreshCw,
  Sliders,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  Inbox,
  Sparkles,
  Trash2,
  Check,
  Send,
  Radio,
  Cpu,
  Layers,
  Search,
  Filter,
  Info,
} from 'lucide-react';
import {
  EmailDomain,
  EmailSpamSettings,
  RspamdTelemetry,
  SpamQuarantineItem,
  getEmailSpamSettings,
  updateEmailSpamSettings,
  getEmailSpamTelemetry,
  listSpamQuarantine,
  releaseQuarantineItem,
  learnHamQuarantineItem,
  deleteQuarantineItem,
} from '@/lib/api';

interface SpamFilterStudioProps {
  domains: EmailDomain[];
  showToast?: (msg: string) => void;
}

export function SpamFilterStudio({ domains, showToast }: SpamFilterStudioProps) {
  const [selectedDomain, setSelectedDomain] = useState<string>('all');
  const [loading, setLoading] = useState(true);
  const [savingSettings, setSavingSettings] = useState(false);
  const [refreshingTelemetry, setRefreshingTelemetry] = useState(false);

  // Telemetry
  const [telemetry, setTelemetry] = useState<RspamdTelemetry | null>(null);

  // Settings State
  const [settings, setSettings] = useState<EmailSpamSettings>({
    reject_score: 15.0,
    add_header_score: 6.0,
    greylist_score: 4.0,
    greylist_enabled: true,
    bayes_enabled: true,
    fuzzy_enabled: true,
    dmarc_reporting_enabled: true,
    custom_rules: '',
  });

  // Quarantine State
  const [quarantineItems, setQuarantineItems] = useState<SpamQuarantineItem[]>([]);
  const [quarantineLoading, setQuarantineLoading] = useState(false);
  const [searchFilter, setSearchFilter] = useState('');
  const [actionLoadingId, setActionLoadingId] = useState<string | null>(null);

  const toast = useCallback((msg: string) => {
    if (showToast) {
      showToast(msg);
    } else {
      alert(msg);
    }
  }, [showToast]);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const domainId = selectedDomain !== 'all' ? selectedDomain : undefined;

      const [telemetryRes, settingsRes, quarantineRes] = await Promise.all([
        getEmailSpamTelemetry(),
        getEmailSpamSettings(domainId),
        listSpamQuarantine(domainId),
      ]);

      if (telemetryRes.data) {
        setTelemetry(telemetryRes.data);
      }
      if (settingsRes.data) {
        setSettings(settingsRes.data);
      }
      if (quarantineRes.data) {
        setQuarantineItems(quarantineRes.data);
      }
    } catch (err: any) {
      console.error('Failed to load anti-spam data', err);
      toast('Failed to load spam filter telemetry');
    } finally {
      setLoading(false);
    }
  }, [selectedDomain, toast]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleRefreshTelemetry = async () => {
    setRefreshingTelemetry(true);
    try {
      const res = await getEmailSpamTelemetry();
      if (res.data) {
        setTelemetry(res.data);
        toast('Telemetry refreshed');
      }
    } catch (err: any) {
      toast('Failed to update telemetry');
    } finally {
      setRefreshingTelemetry(false);
    }
  };

  const handleSaveSettings = async (e: React.FormEvent) => {
    e.preventDefault();
    setSavingSettings(true);
    try {
      const payload: Partial<EmailSpamSettings> = {
        ...settings,
        domain_id: selectedDomain !== 'all' ? selectedDomain : undefined,
      };
      const res = await updateEmailSpamSettings(payload);
      if (res.data) {
        setSettings(res.data);
        toast('Spam filter thresholds and policies applied to Rspamd');
      }
    } catch (err: any) {
      toast(err.message || 'Failed to save spam settings');
    } finally {
      setSavingSettings(false);
    }
  };

  const handleRelease = async (item: SpamQuarantineItem) => {
    setActionLoadingId(item.id);
    try {
      const res = await releaseQuarantineItem(item.id);
      if (res.data?.success) {
        toast(`Quarantined message released to ${item.recipient}`);
        setQuarantineItems((prev) =>
          prev.map((it) => (it.id === item.id ? { ...it, status: 'released' } : it))
        );
      } else {
        toast(res.data?.error || 'Failed to release message');
      }
    } catch (err: any) {
      toast(err.message || 'Release failed');
    } finally {
      setActionLoadingId(null);
    }
  };

  const handleLearnHam = async (item: SpamQuarantineItem) => {
    setActionLoadingId(item.id);
    try {
      const res = await learnHamQuarantineItem(item.id);
      if (res.data?.success) {
        toast(`Bayes model updated: message trained as Ham & released`);
        setQuarantineItems((prev) =>
          prev.map((it) => (it.id === item.id ? { ...it, status: 'learned_ham' } : it))
        );
        // Refresh telemetry to show incremented ham count
        handleRefreshTelemetry();
      }
    } catch (err: any) {
      toast(err.message || 'Failed to train Bayes classifier');
    } finally {
      setActionLoadingId(null);
    }
  };

  const handleDeleteItem = async (item: SpamQuarantineItem) => {
    if (!confirm('Are you sure you want to permanently delete this quarantined message?')) return;
    try {
      await deleteQuarantineItem(item.id);
      toast('Quarantined item deleted');
      setQuarantineItems((prev) => prev.filter((it) => it.id !== item.id));
    } catch (err: any) {
      toast(err.message || 'Failed to delete item');
    }
  };

  const formatUptime = (seconds?: number): string => {
    if (!seconds || seconds <= 0) return '0h';
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    if (days > 0) return `${days}d ${hours}h`;
    return `${hours}h ${mins}m`;
  };

  const filteredQuarantine = quarantineItems.filter((it) => {
    if (!searchFilter) return true;
    const term = searchFilter.toLowerCase();
    return (
      it.sender.toLowerCase().includes(term) ||
      it.recipient.toLowerCase().includes(term) ||
      it.subject.toLowerCase().includes(term) ||
      it.symbols.some((s) => s.toLowerCase().includes(term))
    );
  });

  return (
    <div className="space-y-6">
      {/* Header Banner */}
      <div className="p-6 rounded-2xl bg-gradient-to-r from-emerald-900/40 via-teal-900/30 to-indigo-900/30 border border-emerald-500/20 backdrop-blur-md shadow-xs">
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div className="space-y-1.5">
            <div className="inline-flex items-center gap-2 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">
              <ShieldCheck className="w-3.5 h-3.5 text-emerald-400" />
              <span>Next-Gen Content Inspection</span>
            </div>
            <h2 className="text-xl font-bold text-white tracking-tight">Rspamd Anti-Spam &amp; Quarantine Studio</h2>
            <p className="text-xs text-slate-300 max-w-2xl">
              High-throughput statistical Bayesian filtering, dynamic greylisting, neural network fuzzy hashing, and 1-click quarantine triage with real-time feedback training.
            </p>
          </div>

          <div className="flex items-center gap-3">
            <button
              onClick={handleRefreshTelemetry}
              disabled={refreshingTelemetry}
              className="p-2.5 rounded-xl bg-white/10 hover:bg-white/15 text-white border border-white/10 transition shadow-xs disabled:opacity-50"
              title="Refresh Rspamd Telemetry"
            >
              <RefreshCw className={`w-4 h-4 ${refreshingTelemetry ? 'animate-spin' : ''}`} />
            </button>
            <div className="flex items-center gap-2">
              <span className="text-xs font-semibold text-slate-300">Domain Scope:</span>
              <select
                value={selectedDomain}
                onChange={(e) => setSelectedDomain(e.target.value)}
                className="px-3 py-1.5 bg-slate-900/80 border border-slate-700 rounded-xl text-xs text-white focus:outline-none focus:border-emerald-400"
              >
                <option value="all">Server-wide Global Policies</option>
                {domains.map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.domain}
                  </option>
                ))}
              </select>
            </div>
          </div>
        </div>

        {/* Live Telemetry Row */}
        <div className="mt-6 pt-4 border-t border-white/10 grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-7 gap-3">
          <div className="p-3 rounded-xl bg-white/5 border border-white/10">
            <span className="text-[10px] font-semibold text-slate-300 uppercase tracking-wider block">Messages Scanned</span>
            <span className="text-lg font-bold text-white font-mono">{telemetry?.scanned?.toLocaleString() || 0}</span>
          </div>

          <div className="p-3 rounded-xl bg-emerald-500/10 border border-emerald-500/20">
            <span className="text-[10px] font-semibold text-emerald-300 uppercase tracking-wider block">Clean Deliveries</span>
            <span className="text-lg font-bold text-emerald-400 font-mono">{telemetry?.clean?.toLocaleString() || 0}</span>
          </div>

          <div className="p-3 rounded-xl bg-amber-500/10 border border-amber-500/20">
            <span className="text-[10px] font-semibold text-amber-300 uppercase tracking-wider block">Greylisted</span>
            <span className="text-lg font-bold text-amber-400 font-mono">{telemetry?.greylisted?.toLocaleString() || 0}</span>
          </div>

          <div className="p-3 rounded-xl bg-rose-500/10 border border-rose-500/20">
            <span className="text-[10px] font-semibold text-rose-300 uppercase tracking-wider block">Spam Flagged</span>
            <span className="text-lg font-bold text-rose-400 font-mono">{telemetry?.spam?.toLocaleString() || 0}</span>
          </div>

          <div className="p-3 rounded-xl bg-purple-500/10 border border-purple-500/20">
            <span className="text-[10px] font-semibold text-purple-300 uppercase tracking-wider block">SMTP Rejections</span>
            <span className="text-lg font-bold text-purple-400 font-mono">{telemetry?.rejected?.toLocaleString() || 0}</span>
          </div>

          <div className="p-3 rounded-xl bg-indigo-500/10 border border-indigo-500/20">
            <span className="text-[10px] font-semibold text-indigo-300 uppercase tracking-wider block">Bayes Trained (Ham)</span>
            <span className="text-lg font-bold text-indigo-400 font-mono">{telemetry?.learned_ham?.toLocaleString() || 0}</span>
          </div>

          <div className="p-3 rounded-xl bg-white/5 border border-white/10">
            <span className="text-[10px] font-semibold text-slate-300 uppercase tracking-wider block">Rspamd Uptime</span>
            <span className="text-lg font-bold text-white font-mono">{formatUptime(telemetry?.uptime_seconds)}</span>
          </div>
        </div>
      </div>

      {/* Main Grid: Policy Configuration & Quarantine Table */}
      <div className="grid grid-cols-1 xl:grid-cols-3 gap-6">
        {/* Left Column: Policy Thresholds Form */}
        <div className="xl:col-span-1 space-y-4">
          <form
            onSubmit={handleSaveSettings}
            className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 space-y-5 shadow-xs"
          >
            <div className="flex items-center justify-between pb-3 border-b border-slate-100 dark:border-surface-800">
              <h3 className="text-sm font-bold text-slate-900 dark:text-white flex items-center gap-2">
                <Sliders className="w-4 h-4 text-emerald-500" />
                <span>Spam Score Action Thresholds</span>
              </h3>
              <span className="text-[11px] font-mono font-semibold text-slate-500">local.d/actions.conf</span>
            </div>

            {/* Threshold Sliders */}
            <div className="space-y-4">
              <div className="space-y-1.5">
                <div className="flex justify-between items-center text-xs">
                  <label className="font-semibold text-slate-700 dark:text-slate-300">Reject Score (Drop Message)</label>
                  <span className="font-mono font-bold text-rose-500 bg-rose-500/10 px-2 py-0.5 rounded-md">
                    {settings.reject_score.toFixed(1)}
                  </span>
                </div>
                <input
                  type="range"
                  min="8.0"
                  max="30.0"
                  step="0.5"
                  value={settings.reject_score}
                  onChange={(e) => setSettings({ ...settings, reject_score: parseFloat(e.target.value) })}
                  className="w-full accent-rose-500"
                />
                <p className="text-[10px] text-slate-400">
                  Messages scoring equal to or above this score will be immediately rejected with SMTP 554 5.7.1.
                </p>
              </div>

              <div className="space-y-1.5">
                <div className="flex justify-between items-center text-xs">
                  <label className="font-semibold text-slate-700 dark:text-slate-300">Add Header Score (Spam Box)</label>
                  <span className="font-mono font-bold text-amber-500 bg-amber-500/10 px-2 py-0.5 rounded-md">
                    {settings.add_header_score.toFixed(1)}
                  </span>
                </div>
                <input
                  type="range"
                  min="3.0"
                  max="14.0"
                  step="0.5"
                  value={settings.add_header_score}
                  onChange={(e) => setSettings({ ...settings, add_header_score: parseFloat(e.target.value) })}
                  className="w-full accent-amber-500"
                />
                <p className="text-[10px] text-slate-400">
                  Marks email with <code className="font-mono text-indigo-500">X-Spam: Yes</code> header and routes to recipient Junk folder.
                </p>
              </div>

              <div className="space-y-1.5">
                <div className="flex justify-between items-center text-xs">
                  <label className="font-semibold text-slate-700 dark:text-slate-300">Greylist Score (Delay New Send)</label>
                  <span className="font-mono font-bold text-blue-500 bg-blue-500/10 px-2 py-0.5 rounded-md">
                    {settings.greylist_score.toFixed(1)}
                  </span>
                </div>
                <input
                  type="range"
                  min="1.0"
                  max="8.0"
                  step="0.5"
                  value={settings.greylist_score}
                  onChange={(e) => setSettings({ ...settings, greylist_score: parseFloat(e.target.value) })}
                  className="w-full accent-blue-500"
                />
                <p className="text-[10px] text-slate-400">
                  Suspicious inbound servers receive a temporary deferral to verify legitimate SMTP retry capability.
                </p>
              </div>
            </div>

            {/* Filter Module Toggles */}
            <div className="space-y-2.5 pt-3 border-t border-slate-100 dark:border-surface-800">
              <span className="text-[11px] font-bold text-slate-700 dark:text-slate-300 uppercase tracking-wider block">
                Active Inspection Engines
              </span>

              <label className="flex items-center justify-between p-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 cursor-pointer">
                <div>
                  <span className="text-xs font-semibold text-slate-900 dark:text-white block">Adaptive Greylisting</span>
                  <span className="text-[10px] text-slate-400">Temporarily defers unknown senders</span>
                </div>
                <input
                  type="checkbox"
                  checked={settings.greylist_enabled}
                  onChange={(e) => setSettings({ ...settings, greylist_enabled: e.target.checked })}
                  className="rounded text-emerald-600 focus:ring-0"
                />
              </label>

              <label className="flex items-center justify-between p-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 cursor-pointer">
                <div>
                  <span className="text-xs font-semibold text-slate-900 dark:text-white block">Bayesian Classifier</span>
                  <span className="text-[10px] text-slate-400">Statistical learning from ham and spam</span>
                </div>
                <input
                  type="checkbox"
                  checked={settings.bayes_enabled}
                  onChange={(e) => setSettings({ ...settings, bayes_enabled: e.target.checked })}
                  className="rounded text-emerald-600 focus:ring-0"
                />
              </label>

              <label className="flex items-center justify-between p-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 cursor-pointer">
                <div>
                  <span className="text-xs font-semibold text-slate-900 dark:text-white block">Fuzzy Hash Recognition</span>
                  <span className="text-[10px] text-slate-400">Matches known global spam pattern hashes</span>
                </div>
                <input
                  type="checkbox"
                  checked={settings.fuzzy_enabled}
                  onChange={(e) => setSettings({ ...settings, fuzzy_enabled: e.target.checked })}
                  className="rounded text-emerald-600 focus:ring-0"
                />
              </label>

              <label className="flex items-center justify-between p-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 cursor-pointer">
                <div>
                  <span className="text-xs font-semibold text-slate-900 dark:text-white block">DMARC Reporting &amp; Alignment</span>
                  <span className="text-[10px] text-slate-400">Enforce strict DKIM/SPF alignment</span>
                </div>
                <input
                  type="checkbox"
                  checked={settings.dmarc_reporting_enabled}
                  onChange={(e) => setSettings({ ...settings, dmarc_reporting_enabled: e.target.checked })}
                  className="rounded text-emerald-600 focus:ring-0"
                />
              </label>
            </div>

            <button
              type="submit"
              disabled={savingSettings}
              className="w-full inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs transition shadow-sm disabled:opacity-50"
            >
              {savingSettings && <RefreshCw className="w-3.5 h-3.5 animate-spin" />}
              <span>{savingSettings ? 'Writing to Rspamd...' : 'Apply Anti-Spam Policy'}</span>
            </button>
          </form>
        </div>

        {/* Right Columns: Quarantine Inspection & Triage Queue */}
        <div className="xl:col-span-2 space-y-4">
          <div className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 space-y-4 shadow-xs">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div>
                <h3 className="text-base font-bold text-slate-900 dark:text-white flex items-center gap-2">
                  <Inbox className="w-4 h-4 text-emerald-500" />
                  <span>Quarantine Triage Queue</span>
                </h3>
                <p className="text-xs text-slate-500 mt-0.5">
                  Suspicious inbound emails held for administrative review or false-positive remediation.
                </p>
              </div>

              <div className="flex items-center gap-2">
                <div className="relative">
                  <Search className="w-3.5 h-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
                  <input
                    type="text"
                    placeholder="Search sender, recipient, symbols..."
                    value={searchFilter}
                    onChange={(e) => setSearchFilter(e.target.value)}
                    className="pl-8 pr-3 py-1.5 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-emerald-500 w-56"
                  />
                </div>
              </div>
            </div>

            {/* Quarantine Table */}
            {loading ? (
              <div className="p-10 text-center">
                <RefreshCw className="w-6 h-6 text-emerald-500 animate-spin mx-auto mb-2" />
                <span className="text-xs text-slate-500">Scanning quarantine spool...</span>
              </div>
            ) : filteredQuarantine.length === 0 ? (
              <div className="p-12 text-center rounded-xl bg-slate-50/50 dark:bg-surface-950/50 border border-dashed border-slate-200 dark:border-surface-800 space-y-2">
                <CheckCircle2 className="w-8 h-8 text-emerald-500 mx-auto" />
                <h4 className="text-sm font-bold text-slate-800 dark:text-slate-200">Quarantine Spool is Clean</h4>
                <p className="text-xs text-slate-400 max-w-sm mx-auto">
                  No quarantined messages are awaiting review. Outbound and inbound deliveries are processing according to policy thresholds.
                </p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead>
                    <tr className="border-b border-slate-200 dark:border-surface-800 text-slate-400 font-semibold uppercase text-[10px]">
                      <th className="pb-2.5">Date / Time</th>
                      <th className="pb-2.5">Sender &amp; Recipient</th>
                      <th className="pb-2.5">Subject &amp; Symbols</th>
                      <th className="pb-2.5">Score</th>
                      <th className="pb-2.5">Status</th>
                      <th className="pb-2.5 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-surface-800">
                    {filteredQuarantine.map((item) => {
                      const isHighSpam = item.score >= 15;
                      const isMediumSpam = item.score >= 6 && item.score < 15;
                      const isActionBusy = actionLoadingId === item.id;

                      return (
                        <tr key={item.id} className="hover:bg-slate-50/50 dark:hover:bg-surface-800/40 transition group">
                          <td className="py-3 text-slate-500 font-mono text-[11px] whitespace-nowrap">
                            {new Date(item.created_at).toLocaleString([], {
                              month: 'short',
                              day: 'numeric',
                              hour: '2-digit',
                              minute: '2-digit',
                            })}
                          </td>

                          <td className="py-3 space-y-0.5">
                            <span className="font-semibold text-slate-800 dark:text-slate-200 truncate max-w-[200px] block" title={item.sender}>
                              {item.sender}
                            </span>
                            <span className="text-[11px] text-slate-400 truncate max-w-[200px] block" title={item.recipient}>
                              &rarr; {item.recipient}
                            </span>
                          </td>

                          <td className="py-3 space-y-1">
                            <span className="font-medium text-slate-900 dark:text-white truncate max-w-[240px] block" title={item.subject}>
                              {item.subject || '(No Subject)'}
                            </span>
                            <div className="flex flex-wrap gap-1 max-w-[280px]">
                              {item.symbols &&
                                item.symbols.slice(0, 3).map((sym, sIdx) => (
                                  <span
                                    key={sIdx}
                                    className="px-1.5 py-0.5 rounded text-[9px] font-mono bg-slate-100 dark:bg-surface-800 text-slate-600 dark:text-slate-300 border border-slate-200 dark:border-surface-700"
                                  >
                                    {sym}
                                  </span>
                                ))}
                              {item.symbols && item.symbols.length > 3 && (
                                <span className="text-[9px] text-slate-400 font-mono">
                                  +{item.symbols.length - 3} more
                                </span>
                              )}
                            </div>
                          </td>

                          <td className="py-3">
                            <span
                              className={`px-2 py-0.5 rounded-full font-mono font-bold text-[11px] ${
                                isHighSpam
                                  ? 'bg-rose-500/10 text-rose-500'
                                  : isMediumSpam
                                  ? 'bg-amber-500/10 text-amber-500'
                                  : 'bg-emerald-500/10 text-emerald-500'
                              }`}
                            >
                              {item.score.toFixed(1)}
                            </span>
                          </td>

                          <td className="py-3 whitespace-nowrap">
                            {item.status === 'released' ? (
                              <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-emerald-600 dark:text-emerald-400">
                                <CheckCircle2 className="w-3.5 h-3.5" />
                                <span>Released</span>
                              </span>
                            ) : item.status === 'learned_ham' ? (
                              <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-indigo-500">
                                <Sparkles className="w-3.5 h-3.5" />
                                <span>Trained Ham</span>
                              </span>
                            ) : (
                              <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-amber-500">
                                <ShieldAlert className="w-3.5 h-3.5" />
                                <span>Quarantined</span>
                              </span>
                            )}
                          </td>

                          <td className="py-3 text-right whitespace-nowrap">
                            <div className="flex items-center justify-end gap-1.5">
                              {item.status === 'quarantined' && (
                                <>
                                  <button
                                    onClick={() => handleRelease(item)}
                                    disabled={isActionBusy}
                                    className="p-1.5 rounded-lg bg-emerald-50 dark:bg-emerald-950/40 hover:bg-emerald-100 dark:hover:bg-emerald-900/40 text-emerald-600 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800 transition disabled:opacity-50"
                                    title="1-Click Release to Recipient Inbox via LMTP"
                                  >
                                    <Send className="w-3.5 h-3.5" />
                                  </button>

                                  <button
                                    onClick={() => handleLearnHam(item)}
                                    disabled={isActionBusy}
                                    className="p-1.5 rounded-lg bg-indigo-50 dark:bg-indigo-950/40 hover:bg-indigo-100 dark:hover:bg-indigo-900/40 text-indigo-600 dark:text-indigo-400 border border-indigo-200 dark:border-indigo-800 transition disabled:opacity-50"
                                    title="Train Bayes Classifier as Ham & Release"
                                  >
                                    <Sparkles className="w-3.5 h-3.5" />
                                  </button>
                                </>
                              )}

                              <button
                                onClick={() => handleDeleteItem(item)}
                                disabled={isActionBusy}
                                className="p-1.5 rounded-lg text-slate-400 hover:text-rose-500 hover:bg-rose-50 dark:hover:bg-rose-950/30 transition disabled:opacity-50"
                                title="Delete from Quarantine"
                              >
                                <Trash2 className="w-3.5 h-3.5" />
                              </button>
                            </div>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
