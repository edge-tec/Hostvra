'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  ShieldAlert,
  ShieldCheck,
  Shield,
  Activity,
  AlertTriangle,
  RefreshCw,
  Search,
  CheckCircle2,
  Sliders,
  Settings2,
  Globe,
  Lock,
  Flame,
  Bug,
  Terminal,
  FileCode,
  Zap,
  Eye,
  X,
  Check,
  Play,
  RotateCcw,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  apiFetch,
  WAFStatus,
  WAFRuleCategory,
  WebsiteWAFConfig,
  WAFAttackEvent,
  Website,
} from '@/lib/api';

export default function WAFPage() {
  const [activeTab, setActiveTab] = useState<'events' | 'rules' | 'websites' | 'settings'>('events');

  // Core Data
  const [status, setStatus] = useState<WAFStatus | null>(null);
  const [rules, setRules] = useState<WAFRuleCategory[]>([]);
  const [events, setEvents] = useState<WAFAttackEvent[]>([]);
  const [websites, setWebsites] = useState<Website[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  // Filters
  const [eventCategoryFilter, setEventCategoryFilter] = useState<string>('all');
  const [eventSearch, setEventSearch] = useState('');

  // Notifications
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  // Modals
  const [inspectEvent, setInspectEvent] = useState<WAFAttackEvent | null>(null);
  const [editSiteWAF, setEditSiteWAF] = useState<WebsiteWAFConfig | null>(null);
  const [savingSiteWAF, setSavingSiteWAF] = useState(false);

  // Settings State
  const [engineMode, setEngineMode] = useState<'On' | 'DetectionOnly' | 'Off'>('On');
  const [paranoiaLevel, setParanoiaLevel] = useState(1);
  const [anomalyThreshold, setAnomalyThreshold] = useState(5);
  const [savingSettings, setSavingSettings] = useState(false);

  // Probe Tester State
  const [probeType, setProbeType] = useState('sqli');
  const [probeDomain, setProbeDomain] = useState('hostvra.com');
  const [testingProbe, setTestingProbe] = useState(false);
  const [lastProbeResult, setLastProbeResult] = useState<WAFAttackEvent | null>(null);

  // Auto-dismiss notifications
  useEffect(() => {
    if (successMsg) {
      const t = setTimeout(() => setSuccessMsg(null), 4000);
      return () => clearTimeout(t);
    }
  }, [successMsg]);

  useEffect(() => {
    if (errorMsg) {
      const t = setTimeout(() => setErrorMsg(null), 6000);
      return () => clearTimeout(t);
    }
  }, [errorMsg]);

  // Load all initial data from backend
  const loadData = useCallback(async (isRefresh = false) => {
    if (isRefresh) setRefreshing(true);
    else setLoading(true);

    try {
      const [statusRes, rulesRes, eventsRes, sitesRes] = await Promise.all([
        apiFetch<WAFStatus>('/api/v1/waf/status'),
        apiFetch<WAFRuleCategory[]>('/api/v1/waf/rules'),
        apiFetch<WAFAttackEvent[]>('/api/v1/waf/events?limit=50'),
        apiFetch<Website[]>('/api/v1/websites'),
      ]);

      if (statusRes.success && statusRes.data) {
        setStatus(statusRes.data);
        setEngineMode(statusRes.data.mode);
        setParanoiaLevel(statusRes.data.paranoia_level);
        setAnomalyThreshold(statusRes.data.anomaly_threshold);
      }
      if (rulesRes.success && rulesRes.data) setRules(rulesRes.data);
      if (eventsRes.success && eventsRes.data) setEvents(eventsRes.data);
      if (sitesRes.success && sitesRes.data) setWebsites(sitesRes.data);
    } catch (err: any) {
      setErrorMsg(err.message || 'Failed to connect to WAF engine');
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Toggle Rule Category
  const handleToggleRule = async (catId: string, currentEnabled: boolean) => {
    try {
      const res = await apiFetch<WAFRuleCategory[]>('/api/v1/waf/rules/toggle', {
        method: 'POST',
        body: JSON.stringify({ category: catId, enabled: !currentEnabled }),
      });

      if (res.success && res.data) {
        setRules(res.data);
        setSuccessMsg(`Rule category updated successfully.`);
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Failed to update rule category');
    }
  };

  // Save Global Settings
  const handleSaveSettings = async (e: React.FormEvent) => {
    e.preventDefault();
    setSavingSettings(true);
    setErrorMsg(null);

    try {
      const res = await apiFetch<WAFStatus>('/api/v1/waf/config', {
        method: 'POST',
        body: JSON.stringify({
          mode: engineMode,
          paranoia_level: paranoiaLevel,
          anomaly_threshold: anomalyThreshold,
        }),
      });

      if (res.success && res.data) {
        setStatus(res.data);
        setSuccessMsg('WAF engine configuration updated and reloaded.');
      } else {
        setErrorMsg(res.error?.message || 'Failed to save WAF settings');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Error updating WAF configuration');
    } finally {
      setSavingSettings(false);
    }
  };

  // Run Simulated Attack Probe
  const handleFireProbe = async () => {
    setTestingProbe(true);
    setErrorMsg(null);

    try {
      const res = await apiFetch<WAFAttackEvent>('/api/v1/waf/probe', {
        method: 'POST',
        body: JSON.stringify({
          attack_type: probeType,
          domain: probeDomain || 'example.com',
        }),
      });

      if (res.success && res.data) {
        const newEv = res.data;
        setLastProbeResult(newEv);
        setEvents((prev) => [newEv, ...prev]);
        setSuccessMsg(`Attack probe simulated! Action taken: ${newEv.action} (Rule ${newEv.rule_id})`);
        loadData(true);
      } else {
        setErrorMsg(res.error?.message || 'Probe test failed');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Error running simulated attack');
    } finally {
      setTestingProbe(false);
    }
  };

  // Open Edit Website WAF
  const handleOpenEditSite = async (domain: string) => {
    try {
      const res = await apiFetch<WebsiteWAFConfig>(`/api/v1/waf/websites/${domain}`);
      if (res.success && res.data) {
        setEditSiteWAF(res.data);
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Failed to load website WAF policy');
    }
  };

  // Save Website WAF Config
  const handleSaveSiteWAF = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editSiteWAF) return;

    setSavingSiteWAF(true);
    try {
      const res = await apiFetch<WebsiteWAFConfig>(`/api/v1/waf/websites/${editSiteWAF.domain}`, {
        method: 'POST',
        body: JSON.stringify(editSiteWAF),
      });

      if (res.success) {
        setSuccessMsg(`WAF policy for ${editSiteWAF.domain} updated.`);
        setEditSiteWAF(null);
      } else {
        setErrorMsg(res.error?.message || 'Failed to update website WAF');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Error saving website WAF configuration');
    } finally {
      setSavingSiteWAF(false);
    }
  };

  // Filtered Events
  const filteredEvents = events.filter((ev) => {
    if (eventCategoryFilter !== 'all' && !ev.rule_category.toLowerCase().includes(eventCategoryFilter.toLowerCase())) {
      return false;
    }
    if (eventSearch) {
      const q = eventSearch.toLowerCase();
      return (
        ev.client_ip.toLowerCase().includes(q) ||
        ev.domain.toLowerCase().includes(q) ||
        ev.uri.toLowerCase().includes(q) ||
        String(ev.rule_id).includes(q) ||
        ev.message.toLowerCase().includes(q)
      );
    }
    return true;
  });

  return (
    <DashboardShell>
      <div className="space-y-6">
        {/* Banner Alerts */}
        {errorMsg && (
          <div className="p-4 rounded-xl bg-red-500/10 border border-red-500/20 text-red-400 text-sm flex items-center justify-between animate-fadeIn">
            <div className="flex items-center gap-2">
              <AlertTriangle className="w-5 h-5 flex-shrink-0" />
              <span>{errorMsg}</span>
            </div>
            <button onClick={() => setErrorMsg(null)} className="hover:text-white">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {successMsg && (
          <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-sm flex items-center justify-between animate-fadeIn">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-5 h-5 flex-shrink-0" />
              <span>{successMsg}</span>
            </div>
            <button onClick={() => setSuccessMsg(null)} className="hover:text-white">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {/* Page Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold text-white tracking-tight flex items-center gap-2.5">
              <ShieldAlert className="w-7 h-7 text-purple-400" />
              Web Application Firewall (WAF)
            </h1>
            <p className="text-sm text-slate-400 mt-1">
              ModSecurity v3 engine with OWASP Core Rule Set (CRS v4.0) protecting against SQLi, XSS, RCE, LFI, and bot scanners.
            </p>
          </div>

          <div className="flex items-center gap-2.5">
            <button
              onClick={() => loadData(true)}
              disabled={refreshing}
              className="p-2.5 rounded-lg bg-surface-900 hover:bg-surface-800 text-slate-400 hover:text-white border border-surface-800 transition-colors disabled:opacity-50"
              title="Refresh Telemetry"
            >
              <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin text-brand-400' : ''}`} />
            </button>

            <button
              onClick={() => setActiveTab('settings')}
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-lg bg-purple-600 hover:bg-purple-500 text-white text-sm font-semibold transition-colors shadow-lg shadow-purple-600/25"
            >
              <Sliders className="w-4 h-4" />
              Tune WAF Engine
            </button>
          </div>
        </div>

        {/* Top Stat Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="bg-surface-900 border border-surface-800 rounded-xl p-5 flex items-start gap-4">
            <div className="p-3 rounded-lg bg-purple-500/10 text-purple-400 border border-purple-500/20">
              <Shield className="w-5 h-5" />
            </div>
            <div>
              <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Engine State</div>
              <div className="text-lg font-bold text-white mt-1 flex items-center gap-2">
                {status?.mode === 'On' ? (
                  <span className="text-emerald-400 flex items-center gap-1">
                    <CheckCircle2 className="w-4 h-4" /> Blocking Mode
                  </span>
                ) : status?.mode === 'DetectionOnly' ? (
                  <span className="text-amber-400 flex items-center gap-1">
                    <Eye className="w-4 h-4" /> Detection Only
                  </span>
                ) : (
                  <span className="text-slate-500">Disabled</span>
                )}
              </div>
              <div className="text-xs text-slate-400 mt-1">OWASP CRS v4.0 Active</div>
            </div>
          </div>

          <div className="bg-surface-900 border border-surface-800 rounded-xl p-5 flex items-start gap-4">
            <div className="p-3 rounded-lg bg-red-500/10 text-red-400 border border-red-500/20">
              <Flame className="w-5 h-5" />
            </div>
            <div>
              <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Attacks Intercepted</div>
              <div className="text-xl font-bold text-white mt-1 font-mono">{status?.total_attacks_blocked || 0}</div>
              <div className="text-xs text-slate-400 mt-1">
                {status?.total_attacks_detected || 0} detected / logged
              </div>
            </div>
          </div>

          <div className="bg-surface-900 border border-surface-800 rounded-xl p-5 flex items-start gap-4">
            <div className="p-3 rounded-lg bg-blue-500/10 text-blue-400 border border-blue-500/20">
              <FileCode className="w-5 h-5" />
            </div>
            <div>
              <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Active Rules</div>
              <div className="text-xl font-bold text-white mt-1">{status?.rules_count || 0} Rules</div>
              <div className="text-xs text-slate-400 mt-1">
                {status?.active_categories_count || 0} categories enabled
              </div>
            </div>
          </div>

          <div className="bg-surface-900 border border-surface-800 rounded-xl p-5 flex items-start gap-4">
            <div className="p-3 rounded-lg bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              <Lock className="w-5 h-5" />
            </div>
            <div>
              <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Paranoia Level</div>
              <div className="text-xl font-bold text-white mt-1">Level {status?.paranoia_level || 1}</div>
              <div className="text-xs text-slate-400 mt-1 font-mono">
                Threshold: {status?.anomaly_threshold || 5} pts
              </div>
            </div>
          </div>
        </div>

        {/* Navigation Tabs */}
        <div className="flex items-center gap-1.5 p-1 bg-slate-100/80 dark:bg-slate-800/60 rounded-xl overflow-x-auto text-xs font-semibold">
          <button
            role="tab"
            data-tab="true"
            onClick={() => setActiveTab('events')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all cursor-pointer font-medium whitespace-nowrap ${
              activeTab === 'events'
                ? 'bg-white dark:bg-slate-900 text-purple-600 dark:text-purple-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-white/40 dark:hover:bg-slate-700/40'
            }`}
          >
            <Activity className="w-4 h-4" />
            <span>Live Attack Events ({events.length})</span>
          </button>

          <button
            role="tab"
            data-tab="true"
            onClick={() => setActiveTab('rules')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all cursor-pointer font-medium whitespace-nowrap ${
              activeTab === 'rules'
                ? 'bg-white dark:bg-slate-900 text-purple-600 dark:text-purple-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-white/40 dark:hover:bg-slate-700/40'
            }`}
          >
            <Shield className="w-4 h-4" />
            <span>OWASP Core Rule Set ({rules.length})</span>
          </button>

          <button
            role="tab"
            data-tab="true"
            onClick={() => setActiveTab('websites')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all cursor-pointer font-medium whitespace-nowrap ${
              activeTab === 'websites'
                ? 'bg-white dark:bg-slate-900 text-purple-600 dark:text-purple-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-white/40 dark:hover:bg-slate-700/40'
            }`}
          >
            <Globe className="w-4 h-4" />
            <span>Per-Website Policies ({websites.length})</span>
          </button>

          <button
            role="tab"
            data-tab="true"
            onClick={() => setActiveTab('settings')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all cursor-pointer font-medium whitespace-nowrap ${
              activeTab === 'settings'
                ? 'bg-white dark:bg-slate-900 text-purple-600 dark:text-purple-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-white/40 dark:hover:bg-slate-700/40'
            }`}
          >
            <Settings2 className="w-4 h-4" />
            <span>Engine Settings & Probes</span>
          </button>
        </div>

        {/* TAB 1: LIVE ATTACK EVENTS */}
        {activeTab === 'events' && (
          <div className="space-y-4">
            {/* Filter Bar */}
            <div className="flex flex-col sm:flex-row items-center justify-between gap-3">
              <div className="flex items-center gap-1.5 p-1 bg-surface-900 border border-surface-800 rounded-lg overflow-x-auto w-full sm:w-auto">
                {['all', 'sqli', 'xss', 'rce', 'lfi', 'scanner'].map((cat) => (
                  <button
                    key={cat}
                    onClick={() => setEventCategoryFilter(cat)}
                    className={`px-3 py-1.5 rounded-md text-xs font-semibold uppercase transition-all ${
                      eventCategoryFilter === cat
                        ? 'bg-purple-600 text-white shadow-sm'
                        : 'text-slate-400 hover:text-white'
                    }`}
                  >
                    {cat}
                  </button>
                ))}
              </div>

              <div className="relative w-full sm:w-72">
                <Search className="w-4 h-4 text-slate-500 absolute left-3 top-1/2 -translate-y-1/2" />
                <input
                  type="text"
                  placeholder="Filter by IP, domain, rule ID..."
                  value={eventSearch}
                  onChange={(e) => setEventSearch(e.target.value)}
                  className="w-full pl-9 pr-3 py-2 rounded-lg bg-surface-900 border border-surface-800 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-purple-500"
                />
              </div>
            </div>

            {/* Events Table */}
            <div className="bg-surface-900 border border-surface-800 rounded-xl overflow-hidden shadow-sm">
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm text-slate-300">
                  <thead className="bg-surface-950/60 text-xs uppercase font-semibold text-slate-400 border-b border-surface-800">
                    <tr>
                      <th className="px-6 py-3.5">Timestamp</th>
                      <th className="px-6 py-3.5">Attacker IP</th>
                      <th className="px-6 py-3.5">Target Domain</th>
                      <th className="px-6 py-3.5">Request URI</th>
                      <th className="px-6 py-3.5">Category & Rule</th>
                      <th className="px-6 py-3.5">Action</th>
                      <th className="px-6 py-3.5 text-right">Details</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-surface-800/60 text-xs">
                    {loading ? (
                      <tr>
                        <td colSpan={7} className="px-6 py-12 text-center text-slate-400">
                          <RefreshCw className="w-6 h-6 animate-spin mx-auto text-purple-400 mb-2" />
                          Reading ModSecurity audit events...
                        </td>
                      </tr>
                    ) : filteredEvents.length === 0 ? (
                      <tr>
                        <td colSpan={7} className="px-6 py-12 text-center text-slate-400">
                          <ShieldCheck className="w-8 h-8 mx-auto text-emerald-400 mb-2" />
                          No attack events recorded matching the criteria.
                          <div className="mt-2 text-xs text-slate-500">
                            Your applications are actively protected by OWASP CRS v4.0.
                          </div>
                        </td>
                      </tr>
                    ) : (
                      filteredEvents.map((ev) => (
                        <tr key={ev.id} className="hover:bg-surface-800/30 transition-colors">
                          <td className="px-6 py-4 text-slate-400 font-mono text-[11px]">
                            {new Date(ev.timestamp).toLocaleTimeString()}
                          </td>
                          <td className="px-6 py-4 font-mono font-bold text-white">{ev.client_ip}</td>
                          <td className="px-6 py-4 text-slate-200">{ev.domain}</td>
                          <td className="px-6 py-4 font-mono text-slate-300 max-w-xs truncate">
                            <span className="text-purple-400 font-semibold mr-1.5">{ev.method}</span>
                            {ev.uri}
                          </td>
                          <td className="px-6 py-4">
                            <div className="flex items-center gap-1.5">
                              <span className="px-2 py-0.5 rounded bg-red-500/10 text-red-400 border border-red-500/20 font-bold uppercase text-[10px]">
                                {ev.rule_category}
                              </span>
                              <span className="font-mono text-slate-400">#{ev.rule_id}</span>
                            </div>
                          </td>
                          <td className="px-6 py-4">
                            <span
                              className={`inline-flex items-center gap-1 px-2 py-0.5 rounded font-bold uppercase text-[10px] ${
                                ev.action === 'BLOCKED'
                                  ? 'bg-red-500/20 text-red-400 border border-red-500/30'
                                  : 'bg-amber-500/20 text-amber-400 border border-amber-500/30'
                              }`}
                            >
                              {ev.action === 'BLOCKED' ? '403 Blocked' : 'Detected'}
                            </span>
                          </td>
                          <td className="px-6 py-4 text-right">
                            <button
                              onClick={() => setInspectEvent(ev)}
                              className="px-2.5 py-1 rounded bg-surface-800 hover:bg-surface-700 text-slate-300 hover:text-white border border-surface-700 font-semibold text-xs transition-colors"
                            >
                              Inspect
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

        {/* TAB 2: OWASP CORE RULE SET (CRS) */}
        {activeTab === 'rules' && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-base font-bold text-white">OWASP Core Rule Set Collections</h3>
                <p className="text-xs text-slate-400 mt-0.5">
                  Pre-configured virtual patches and attack detection signatures applied in real time to incoming HTTP requests.
                </p>
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {rules.map((rule) => (
                <div key={rule.id} className="bg-surface-900 border border-surface-800 rounded-xl p-5 space-y-3">
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3">
                      <div
                        className={`p-2.5 rounded-lg border ${
                          rule.is_enabled
                            ? 'bg-purple-500/10 text-purple-400 border-purple-500/20'
                            : 'bg-surface-800 text-slate-500 border-surface-700'
                        }`}
                      >
                        <Shield className="w-5 h-5" />
                      </div>
                      <div>
                        <h4 className="font-bold text-white text-sm">{rule.name}</h4>
                        <div className="text-[11px] font-mono text-slate-400 mt-0.5">
                          CRS Range: {rule.crs_range} • {rule.rules_count} Rules
                        </div>
                      </div>
                    </div>

                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        checked={rule.is_enabled}
                        onChange={() => handleToggleRule(rule.id, rule.is_enabled)}
                        className="sr-only peer"
                      />
                      <div className="w-9 h-5 bg-surface-800 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-purple-600"></div>
                    </label>
                  </div>

                  <p className="text-xs text-slate-400 leading-relaxed">{rule.description}</p>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* TAB 3: PER-WEBSITE POLICIES */}
        {activeTab === 'websites' && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-base font-bold text-white">Per-Website WAF Customization</h3>
                <p className="text-xs text-slate-400 mt-0.5">
                  Tune paranoia levels and configure CMS preset rule exclusions (e.g. WordPress false-positive bypass).
                </p>
              </div>
            </div>

            <div className="bg-surface-900 border border-surface-800 rounded-xl overflow-hidden shadow-sm">
              <table className="w-full text-left text-sm text-slate-300">
                <thead className="bg-surface-950/60 text-xs uppercase font-semibold text-slate-400 border-b border-surface-800">
                  <tr>
                    <th className="px-6 py-3.5">Website Domain</th>
                    <th className="px-6 py-3.5">WAF Status</th>
                    <th className="px-6 py-3.5">CMS Whitelist Preset</th>
                    <th className="px-6 py-3.5">Paranoia Override</th>
                    <th className="px-6 py-3.5 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-surface-800/60 text-xs">
                  {websites.map((site) => (
                    <tr key={site.id} className="hover:bg-surface-800/30 transition-colors">
                      <td className="px-6 py-4 font-bold text-white flex items-center gap-2">
                        <Globe className="w-4 h-4 text-purple-400" />
                        <span>{site.primary_domain}</span>
                      </td>
                      <td className="px-6 py-4">
                        <span className="inline-flex items-center gap-1 text-emerald-400 font-semibold">
                          <CheckCircle2 className="w-3.5 h-3.5" /> Protected (Active)
                        </span>
                      </td>
                      <td className="px-6 py-4">
                        <span className="px-2 py-0.5 rounded bg-surface-800 text-slate-300 border border-surface-700 uppercase font-mono text-[10px]">
                          Auto
                        </span>
                      </td>
                      <td className="px-6 py-4 text-slate-400">Inherit Global (Level 1)</td>
                      <td className="px-6 py-4 text-right">
                        <button
                          onClick={() => handleOpenEditSite(site.primary_domain)}
                          className="px-3 py-1.5 rounded-lg bg-surface-800 hover:bg-surface-700 text-white font-semibold transition-colors border border-surface-700 inline-flex items-center gap-1.5"
                        >
                          <Sliders className="w-3.5 h-3.5 text-purple-400" />
                          Configure WAF
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* TAB 4: ENGINE SETTINGS & PROBES */}
        {activeTab === 'settings' && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Global Engine Directives */}
            <div className="bg-surface-900 border border-surface-800 rounded-xl p-6 space-y-5">
              <div>
                <h3 className="text-base font-bold text-white flex items-center gap-2">
                  <Sliders className="w-5 h-5 text-purple-400" />
                  ModSecurity v3 Engine Directives
                </h3>
                <p className="text-xs text-slate-400 mt-1">
                  Adjust global blocking behavior and anomaly scoring thresholds across all virtual hosts.
                </p>
              </div>

              <form onSubmit={handleSaveSettings} className="space-y-4 text-xs">
                <div>
                  <label className="block font-semibold text-slate-300 mb-1.5">Operational Engine Mode</label>
                  <div className="grid grid-cols-3 gap-2">
                    {[
                      { id: 'On', label: 'Blocking (On)', desc: 'Blocks attacks with 403' },
                      { id: 'DetectionOnly', label: 'Detection Only', desc: 'Logs attacks only' },
                      { id: 'Off', label: 'Disabled (Off)', desc: 'WAF inactive' },
                    ].map((m) => (
                      <button
                        type="button"
                        key={m.id}
                        onClick={() => setEngineMode(m.id as any)}
                        className={`p-3 rounded-lg text-left border transition-all ${
                          engineMode === m.id
                            ? 'bg-purple-600/20 border-purple-500 text-purple-400'
                            : 'bg-surface-800 border-surface-700 text-slate-400 hover:text-white'
                        }`}
                      >
                        <div className="font-bold">{m.label}</div>
                        <div className="text-[10px] text-slate-500 mt-0.5">{m.desc}</div>
                      </button>
                    ))}
                  </div>
                </div>

                <div>
                  <div className="flex justify-between mb-1">
                    <label className="font-semibold text-slate-300">Paranoia Level</label>
                    <span className="font-mono text-purple-400 font-bold">Level {paranoiaLevel}</span>
                  </div>
                  <div className="grid grid-cols-4 gap-2 mb-2">
                    {[1, 2, 3, 4].map((lvl) => (
                      <button
                        type="button"
                        key={lvl}
                        onClick={() => setParanoiaLevel(lvl)}
                        className={`py-1.5 rounded border font-semibold text-xs ${
                          paranoiaLevel === lvl
                            ? 'bg-purple-600 border-purple-500 text-white'
                            : 'bg-surface-800 border-surface-700 text-slate-400'
                        }`}
                      >
                        Level {lvl}
                      </button>
                    ))}
                  </div>
                  <div className="text-[11px] text-slate-400">
                    {paranoiaLevel === 1 && '• Baseline security: virtually zero false positives for standard web apps.'}
                    {paranoiaLevel === 2 && '• Enhanced security: strict HTTP compliance and SQLi boundary checks.'}
                    {paranoiaLevel === 3 && '• High security: advanced regexes; recommended for sensitive banking or commerce.'}
                    {paranoiaLevel === 4 && '• Maximum paranoia: extreme scrutiny; requires manual application tuning.'}
                  </div>
                </div>

                <div>
                  <div className="flex justify-between mb-1">
                    <label className="font-semibold text-slate-300">Inbound Anomaly Score Threshold</label>
                    <span className="font-mono text-emerald-400 font-bold">{anomalyThreshold} Points</span>
                  </div>
                  <input
                    type="range"
                    min="2"
                    max="20"
                    value={anomalyThreshold}
                    onChange={(e) => setAnomalyThreshold(Number(e.target.value))}
                    className="w-full accent-purple-500"
                  />
                  <div className="text-[10px] text-slate-500 mt-0.5">
                    Cumulative score required to trigger an automatic HTTP 403 block. Recommended default: 5.
                  </div>
                </div>

                <div className="pt-3 border-t border-surface-800 flex justify-end">
                  <button
                    type="submit"
                    disabled={savingSettings}
                    className="px-5 py-2 rounded-lg bg-purple-600 hover:bg-purple-500 text-white font-semibold transition-colors disabled:opacity-50 inline-flex items-center gap-2"
                  >
                    {savingSettings ? (
                      <>
                        <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                        Reloading Nginx & ModSec...
                      </>
                    ) : (
                      'Save & Apply Directives'
                    )}
                  </button>
                </div>
              </form>
            </div>

            {/* Interactive Attack Simulator Console */}
            <div className="bg-surface-900 border border-surface-800 rounded-xl p-6 space-y-5">
              <div>
                <h3 className="text-base font-bold text-white flex items-center gap-2">
                  <Play className="w-5 h-5 text-emerald-400" />
                  Live Security Attack Probe
                </h3>
                <p className="text-xs text-slate-400 mt-1">
                  Safely execute simulated attack payloads against the engine to verify active rule enforcement.
                </p>
              </div>

              <div className="space-y-4 text-xs">
                <div>
                  <label className="block font-semibold text-slate-300 mb-1.5">Select Attack Signature</label>
                  <div className="grid grid-cols-2 gap-2">
                    {[
                      { id: 'sqli', label: 'SQL Injection (SQLi)', payload: "' OR '1'='1' --" },
                      { id: 'xss', label: 'Cross-Site Scripting (XSS)', payload: '<script>alert(1)</script>' },
                      { id: 'rce', label: 'Command Execution (RCE)', payload: '; cat /etc/passwd' },
                      { id: 'lfi', label: 'Path Traversal (LFI)', payload: '../../../../etc/shadow' },
                    ].map((p) => (
                      <button
                        type="button"
                        key={p.id}
                        onClick={() => setProbeType(p.id)}
                        className={`p-2.5 rounded-lg text-left border transition-all ${
                          probeType === p.id
                            ? 'bg-emerald-500/20 border-emerald-500 text-emerald-400'
                            : 'bg-surface-800 border-surface-700 text-slate-400 hover:text-white'
                        }`}
                      >
                        <div className="font-bold">{p.label}</div>
                        <div className="font-mono text-[10px] text-slate-500 truncate mt-0.5">{p.payload}</div>
                      </button>
                    ))}
                  </div>
                </div>

                <div>
                  <label className="block font-semibold text-slate-300 mb-1">Target Virtual Host</label>
                  <input
                    type="text"
                    value={probeDomain}
                    onChange={(e) => setProbeDomain(e.target.value)}
                    placeholder="e.g. hostvra.com"
                    className="w-full px-3.5 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white font-mono text-xs focus:outline-none focus:border-purple-500"
                  />
                </div>

                <div className="p-3.5 rounded-lg bg-surface-950 border border-surface-800 text-[11px] text-slate-400 space-y-1">
                  <div className="font-semibold text-slate-300">Expected Result:</div>
                  <div>• ModSecurity inspects URL parameter anomaly score</div>
                  <div>• Triggers HTTP 403 Forbidden with ModSecurity audit entry</div>
                </div>

                <button
                  type="button"
                  disabled={testingProbe}
                  onClick={handleFireProbe}
                  className="w-full py-2.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs transition-colors disabled:opacity-50 inline-flex items-center justify-center gap-2"
                >
                  {testingProbe ? (
                    <>
                      <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                      Evaluating against OWASP CRS...
                    </>
                  ) : (
                    'Fire Simulated Attack Payload'
                  )}
                </button>

                {lastProbeResult && (
                  <div className="p-4 rounded-lg bg-surface-950 border border-emerald-500/30 space-y-2">
                    <div className="flex items-center justify-between">
                      <span className="font-bold text-emerald-400 flex items-center gap-1.5">
                        <CheckCircle2 className="w-4 h-4" /> Intercepted & Verified!
                      </span>
                      <span className="px-2 py-0.5 rounded bg-red-500/20 text-red-400 font-bold uppercase text-[10px]">
                        {lastProbeResult.action}
                      </span>
                    </div>
                    <div className="text-[11px] font-mono text-slate-300">
                      Rule Triggered: <span className="text-purple-400">#{lastProbeResult.rule_id}</span> ({lastProbeResult.message})
                    </div>
                    <div className="text-[10px] font-mono text-slate-500 truncate">
                      Matched: {lastProbeResult.matched_data}
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* MODAL: INSPECT EVENT */}
        {inspectEvent && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4 animate-fadeIn">
            <div className="bg-surface-900 border border-surface-800 rounded-xl max-w-lg w-full p-6 shadow-2xl space-y-4 text-xs">
              <div className="flex items-start justify-between">
                <div>
                  <h3 className="text-base font-bold text-white flex items-center gap-2">
                    <ShieldAlert className="w-5 h-5 text-red-400" />
                    Security Event Details
                  </h3>
                  <p className="text-slate-400 mt-0.5">Event ID: {inspectEvent.id}</p>
                </div>
                <button onClick={() => setInspectEvent(null)} className="text-slate-400 hover:text-white">
                  <X className="w-4 h-4" />
                </button>
              </div>

              <div className="space-y-3 bg-surface-950 p-4 rounded-lg border border-surface-800 font-mono">
                <div>
                  <span className="text-slate-500 block text-[10px] uppercase font-sans">Attacker IP & Target</span>
                  <span className="text-white font-bold">{inspectEvent.client_ip}</span> → <span className="text-purple-400">{inspectEvent.domain}</span>
                </div>
                <div>
                  <span className="text-slate-500 block text-[10px] uppercase font-sans">Request</span>
                  <span className="text-amber-400 font-bold">{inspectEvent.method}</span> {inspectEvent.uri}
                </div>
                <div>
                  <span className="text-slate-500 block text-[10px] uppercase font-sans">Rule Triggered</span>
                  <span className="text-red-400">#{inspectEvent.rule_id}</span> ({inspectEvent.rule_category})
                </div>
                <div>
                  <span className="text-slate-500 block text-[10px] uppercase font-sans">Signature Message</span>
                  <span className="text-slate-200">{inspectEvent.message}</span>
                </div>
                <div>
                  <span className="text-slate-500 block text-[10px] uppercase font-sans">Matched Payload Data</span>
                  <div className="p-2 rounded bg-surface-900 text-red-300 text-[11px] break-all border border-surface-800 mt-1">
                    {inspectEvent.matched_data}
                  </div>
                </div>
              </div>

              <div className="flex justify-end pt-2">
                <button
                  type="button"
                  onClick={() => setInspectEvent(null)}
                  className="px-4 py-2 rounded-lg bg-surface-800 hover:bg-surface-700 text-white font-semibold transition-colors"
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        )}

        {/* MODAL: EDIT WEBSITE WAF */}
        {editSiteWAF && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4 animate-fadeIn">
            <div className="bg-surface-900 border border-surface-800 rounded-xl max-w-md w-full p-6 shadow-2xl space-y-4 text-xs">
              <div className="flex items-start justify-between">
                <div>
                  <h3 className="text-base font-bold text-white flex items-center gap-2">
                    <Globe className="w-5 h-5 text-purple-400" />
                    Configure WAF for {editSiteWAF.domain}
                  </h3>
                  <p className="text-slate-400 mt-0.5">Per-vhost rules and CMS whitelist exclusions</p>
                </div>
                <button onClick={() => setEditSiteWAF(null)} className="text-slate-400 hover:text-white">
                  <X className="w-4 h-4" />
                </button>
              </div>

              <form onSubmit={handleSaveSiteWAF} className="space-y-4">
                <div>
                  <label className="block font-semibold text-slate-300 mb-1">Protection Status</label>
                  <div className="grid grid-cols-2 gap-2">
                    <button
                      type="button"
                      onClick={() => setEditSiteWAF({ ...editSiteWAF, enabled: true })}
                      className={`py-2 rounded-lg font-bold border transition-all ${
                        editSiteWAF.enabled
                          ? 'bg-purple-600/20 border-purple-500 text-purple-400'
                          : 'bg-surface-800 border-surface-700 text-slate-400'
                      }`}
                    >
                      WAF Enabled
                    </button>
                    <button
                      type="button"
                      onClick={() => setEditSiteWAF({ ...editSiteWAF, enabled: false })}
                      className={`py-2 rounded-lg font-bold border transition-all ${
                        !editSiteWAF.enabled
                          ? 'bg-red-500/20 border-red-500 text-red-400'
                          : 'bg-surface-800 border-surface-700 text-slate-400'
                      }`}
                    >
                      WAF Disabled
                    </button>
                  </div>
                </div>

                <div>
                  <label className="block font-semibold text-slate-300 mb-1">CMS False-Positive Whitelist Preset</label>
                  <select
                    value={editSiteWAF.cms_preset}
                    onChange={(e) => setEditSiteWAF({ ...editSiteWAF, cms_preset: e.target.value as any })}
                    className="w-full px-3 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white focus:outline-none focus:border-purple-500"
                  >
                    <option value="none">None (Standard OWASP Enforcement)</option>
                    <option value="wordpress">WordPress (Excludes wp-admin & Elementor false positives)</option>
                    <option value="drupal">Drupal</option>
                    <option value="nextjs">Next.js / Node App</option>
                  </select>
                </div>

                <div className="flex justify-end gap-2 pt-2 border-t border-surface-800">
                  <button
                    type="button"
                    onClick={() => setEditSiteWAF(null)}
                    className="px-4 py-2 rounded-lg font-semibold text-slate-400 hover:text-white"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={savingSiteWAF}
                    className="px-4 py-2 rounded-lg bg-purple-600 hover:bg-purple-500 text-white font-semibold transition-colors disabled:opacity-50"
                  >
                    {savingSiteWAF ? 'Saving...' : 'Save Site WAF'}
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
