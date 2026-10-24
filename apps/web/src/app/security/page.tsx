'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  ShieldAlert,
  ShieldCheck,
  Search,
  RefreshCw,
  Play,
  Trash2,
  Undo2,
  AlertTriangle,
  FolderSearch,
  CheckCircle2,
  XCircle,
  Clock,
  FileWarning,
  Eye,
  SlidersHorizontal,
  X,
  Lock,
  HardDrive,
  Info,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  apiFetch,
  SecurityScan,
  QuarantineItem,
  SecurityEngineStatus,
  ThreatItem,
} from '@/lib/api';

export default function SecurityPage() {
  const [activeTab, setActiveTab] = useState<'scans' | 'quarantine'>('scans');

  // Engine status
  const [engineStatus, setEngineStatus] = useState<SecurityEngineStatus | null>(null);

  // Scans and Quarantine lists
  const [scans, setScans] = useState<SecurityScan[]>([]);
  const [quarantineItems, setQuarantineItems] = useState<QuarantineItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [search, setSearch] = useState('');

  // Alerts
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  // New Scan Modal
  const [scanModalOpen, setScanModalOpen] = useState(false);
  const [targetPath, setTargetPath] = useState('/var/www/vhosts');
  const [selectedEngine, setSelectedEngine] = useState<'clamav' | 'heuristic' | 'combined'>('combined');
  const [exclusions, setExclusions] = useState('.git,node_modules,cache,*.log');
  const [autoQuarantine, setAutoQuarantine] = useState(false);
  const [submittingScan, setSubmittingScan] = useState(false);

  // Findings Modal
  const [findingsModalScan, setFindingsModalScan] = useState<SecurityScan | null>(null);

  // Action Loading
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
      // 1. Engine Status
      const statusRes = await apiFetch<SecurityEngineStatus>('/api/v1/security/engine/status');
      if (statusRes.success && statusRes.data) {
        setEngineStatus(statusRes.data);
      }

      // 2. Scan History
      const scansRes = await apiFetch<any>('/api/v1/security/scans');
      if (scansRes.success && scansRes.data) {
        const raw = scansRes.data;
        const list: SecurityScan[] = Array.isArray(raw)
          ? raw
          : Array.isArray(raw.scans)
          ? raw.scans
          : [];
        setScans(list);
      } else {
        setScans([]);
      }

      // 3. Quarantine Vault
      const quaranRes = await apiFetch<any>('/api/v1/security/quarantine');
      if (quaranRes.success && quaranRes.data) {
        const raw = quaranRes.data;
        const list: QuarantineItem[] = Array.isArray(raw)
          ? raw
          : Array.isArray(raw.items)
          ? raw.items
          : [];
        setQuarantineItems(list);
      } else {
        setQuarantineItems([]);
      }
    } catch (err: any) {
      showNotification(err.message || 'Failed to load security telemetry', true);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    loadData(false);
  }, [loadData]);

  // Real-time periodic polling for active running scans and live telemetry updates
  useEffect(() => {
    const safeScansList = Array.isArray(scans) ? scans : [];
    const hasRunningScan = safeScansList.some((s) => s.status === 'running' || s.status === 'pending');
    const intervalMs = hasRunningScan ? 3000 : 15000;

    const interval = setInterval(() => {
      loadData(false);
    }, intervalMs);
    return () => clearInterval(interval);
  }, [scans, loadData]);

  // Start new scan
  const handleStartScan = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!targetPath.trim()) return;

    setSubmittingScan(true);
    try {
      const exclList = exclusions
        .split(',')
        .map((s) => s.trim())
        .filter((s) => s.length > 0);

      const res = await apiFetch<SecurityScan>('/api/v1/security/scans', {
        method: 'POST',
        body: JSON.stringify({
          target_path: targetPath.trim(),
          scan_type: selectedEngine,
          exclusions: exclList,
          auto_quarantine: autoQuarantine,
        }),
      });

      if (res.success && res.data) {
        showNotification('Malware scan dispatched and running in background');
        setScanModalOpen(false);
        loadData(false);
      } else {
        showNotification(res.error?.message || 'Failed to dispatch scan', true);
      }
    } catch (err: any) {
      showNotification(err.message || 'Scan dispatch request failed', true);
    } finally {
      setSubmittingScan(false);
    }
  };

  // Restore quarantined file
  const handleRestoreQuarantine = async (id: string, originalPath: string) => {
    if (
      !window.confirm(
        `Are you sure you want to restore this file back to its original path?\n\nTarget: ${originalPath}\n\nNote: If this file is a genuine webshell, restoring it may compromise your system.`
      )
    ) {
      return;
    }

    setActionLoadingId(`restore-${id}`);
    try {
      const res = await apiFetch<{ success: boolean }>(`/api/v1/security/quarantine/${id}/restore`, {
        method: 'POST',
      });
      if (res.success) {
        showNotification(`File successfully restored to: ${originalPath}`);
        loadData(false);
      } else {
        showNotification(res.error?.message || 'Failed to restore quarantined file', true);
      }
    } catch (err: any) {
      showNotification(err.message || 'Restore request failed', true);
    } finally {
      setActionLoadingId(null);
    }
  };

  // Purge quarantined file permanently
  const handleDeleteQuarantine = async (id: string, threatName: string) => {
    if (!window.confirm(`Permanently delete quarantined threat [${threatName}] from disk? This cannot be undone.`)) {
      return;
    }

    setActionLoadingId(`delete-${id}`);
    try {
      const res = await apiFetch<{ success: boolean }>(`/api/v1/security/quarantine/${id}`, {
        method: 'DELETE',
      });
      if (res.success) {
        showNotification('Quarantined item permanently erased from quarantine vault');
        loadData(false);
      } else {
        showNotification(res.error?.message || 'Failed to delete quarantined item', true);
      }
    } catch (err: any) {
      showNotification(err.message || 'Delete request failed', true);
    } finally {
      setActionLoadingId(null);
    }
  };

  // Manual Quarantine of a detected threat from findings modal
  const handleManualQuarantine = async (filePath: string, threatName: string) => {
    if (!window.confirm(`Quarantine file immediately?\n\n${filePath}`)) return;

    setActionLoadingId(`quar-${filePath}`);
    try {
      const res = await apiFetch<any>('/api/v1/security/quarantine', {
        method: 'POST',
        body: JSON.stringify({
          scan_id: findingsModalScan?.id || '',
          file_path: filePath,
          threat_name: threatName,
        }),
      });
      if (res.success) {
        showNotification(`File isolated and moved to quarantine vault`);
        loadData(false);
        // Update local modal state
        if (findingsModalScan && findingsModalScan.findings) {
          const updatedThreats = findingsModalScan.findings.map((t) =>
            t.file_path === filePath ? { ...t, quarantined: true } : t
          );
          setFindingsModalScan({ ...findingsModalScan, findings: updatedThreats });
        }
      } else {
        showNotification(res.error?.message || 'Quarantine operation failed', true);
      }
    } catch (err: any) {
      showNotification(err.message || 'Quarantine request failed', true);
    } finally {
      setActionLoadingId(null);
    }
  };

  const calculateDuration = (scan: SecurityScan) => {
    if (!scan.started_at || !scan.completed_at) return '—';
    const diff = new Date(scan.completed_at).getTime() - new Date(scan.started_at).getTime();
    if (diff <= 0) return '< 1s';
    return `${(diff / 1000).toFixed(1)}s`;
  };

  // Safe array guards
  const safeScans = Array.isArray(scans) ? scans : [];
  const safeQuarantine = Array.isArray(quarantineItems) ? quarantineItems : [];

  // Filtering
  const filteredScans = safeScans.filter(
    (s) =>
      (s.target_path || '').toLowerCase().includes(search.toLowerCase()) ||
      (s.id || '').toLowerCase().includes(search.toLowerCase()) ||
      (s.status || '').toLowerCase().includes(search.toLowerCase())
  );

  const filteredQuarantine = safeQuarantine.filter(
    (q) =>
      (q.original_path || '').toLowerCase().includes(search.toLowerCase()) ||
      (q.threat_name || '').toLowerCase().includes(search.toLowerCase())
  );

  return (
    <DashboardShell>
      <div className="space-y-6 max-w-7xl mx-auto pb-12">
        {/* Alerts */}
        {errorMsg && (
          <div className="p-4 rounded-xl bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900 text-rose-800 dark:text-rose-300 text-sm flex items-center justify-between shadow-xs animate-fadeIn">
            <div className="flex items-center gap-2">
              <AlertTriangle className="w-4 h-4 shrink-0 text-rose-600" />
              <span>{errorMsg}</span>
            </div>
            <button onClick={() => setErrorMsg(null)} className="text-rose-500 hover:text-rose-700">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {successMsg && (
          <div className="p-4 rounded-xl bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200 dark:border-emerald-900 text-emerald-800 dark:text-emerald-300 text-sm flex items-center justify-between shadow-xs animate-fadeIn">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 shrink-0 text-emerald-600" />
              <span>{successMsg}</span>
            </div>
            <button onClick={() => setSuccessMsg(null)} className="text-emerald-500 hover:text-emerald-700">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {/* Top Header */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold tracking-tight text-slate-950 dark:text-white">
                Malware Scanning &amp; Quarantine
              </h1>
              <span className="px-2.5 py-0.5 rounded-full text-xs font-mono font-semibold uppercase bg-emerald-50 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800">
                {engineStatus?.clamav_installed ? 'ClamAV Protected' : 'Heuristic Active'}
              </span>
            </div>
            <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
              On-demand and scheduled file scanning, webshell signature detection, and zero-privilege quarantine vault.
            </p>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => loadData(true)}
              disabled={refreshing}
              className="p-2.5 rounded-xl border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-900 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-surface-800 transition-colors shadow-xs"
              title="Refresh status"
            >
              <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin text-emerald-600' : ''}`} />
            </button>

            <button
              onClick={() => setScanModalOpen(true)}
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-[#16A34A] hover:bg-[#15803D] active:bg-[#166534] text-white text-sm font-semibold shadow-sm transition-all cursor-pointer"
            >
              <Play className="w-4 h-4 fill-white" />
              <span>Run Malware Scan</span>
            </button>
          </div>
        </div>

        {/* Overview Telemetry Cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {/* Card 1: Engine Health */}
          <div className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 shadow-xs flex items-center justify-between">
            <div className="flex items-center gap-3.5">
              <div
                className={`w-11 h-11 rounded-xl flex items-center justify-center ${
                  engineStatus?.clamav_installed
                    ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20'
                    : 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20'
                }`}
              >
                <ShieldCheck className="w-6 h-6" />
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <span className="font-bold text-slate-950 dark:text-white text-sm">
                    {engineStatus?.clamav_installed ? 'ClamAV Daemon Ready' : 'Heuristic Rules Engine'}
                  </span>
                  <span
                    className={`w-2 h-2 rounded-full ${
                      engineStatus?.clamav_installed ? 'bg-emerald-500' : 'bg-amber-500'
                    }`}
                  />
                </div>
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 font-medium">
                  {engineStatus?.heuristic_rules ? `${engineStatus.heuristic_rules} heuristic rules active` : 'Standard signature database'}
                </p>
              </div>
            </div>
            <div className="text-right">
              <span className="text-[11px] font-mono text-slate-400 block">
                {engineStatus?.clamav_daemon ? 'Daemon' : 'Binary'}
              </span>
            </div>
          </div>

          {/* Card 2: Scans Run */}
          <div className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 shadow-xs flex items-center gap-3.5">
            <div className="w-11 h-11 rounded-xl bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border border-indigo-500/20 flex items-center justify-center flex-shrink-0">
              <FolderSearch className="w-6 h-6" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold text-slate-950 dark:text-white text-sm">Total Scans Run</span>
                <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-indigo-50 dark:bg-indigo-950/60 text-indigo-700 dark:text-indigo-400 border border-indigo-200 dark:border-indigo-800">
                  {safeScans.length}
                </span>
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 font-medium">
                {safeScans.filter((s) => s.status === 'running').length > 0
                  ? 'A scan is currently running'
                  : 'All background scans completed'}
              </p>
            </div>
          </div>

          {/* Card 3: Quarantine Vault */}
          <div className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 shadow-xs flex items-center gap-3.5">
            <div className="w-11 h-11 rounded-xl bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/20 flex items-center justify-center flex-shrink-0">
              <FileWarning className="w-6 h-6" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold text-slate-950 dark:text-white text-sm">Quarantine Vault</span>
                <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-rose-50 dark:bg-rose-950/60 text-rose-700 dark:text-rose-400 border border-rose-200 dark:border-rose-800">
                  {safeQuarantine.length} threats isolated
                </span>
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 font-medium">
                Vault: {engineStatus?.quarantine_dir || '/var/lib/hostvra/quarantine'} (0000 mode)
              </p>
            </div>
          </div>
        </div>

        {/* Tab Selection */}
        <div className="flex items-center gap-1.5 p-1 bg-slate-100 dark:bg-surface-800/80 rounded-xl w-fit border border-slate-200 dark:border-surface-700 text-xs font-semibold">
          <button
            onClick={() => setActiveTab('scans')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all cursor-pointer font-medium whitespace-nowrap ${
              activeTab === 'scans'
                ? 'bg-white dark:bg-surface-900 text-emerald-600 dark:text-emerald-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <FolderSearch className="w-4 h-4" />
            <span>Scan Jobs &amp; History ({safeScans.length})</span>
          </button>
          <button
            onClick={() => setActiveTab('quarantine')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all cursor-pointer font-medium whitespace-nowrap ${
              activeTab === 'quarantine'
                ? 'bg-white dark:bg-surface-900 text-emerald-600 dark:text-emerald-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <Lock className="w-4 h-4" />
            <span>Quarantine Vault ({safeQuarantine.length})</span>
          </button>
        </div>

        {/* Search Bar */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-3 shadow-xs">
          <div className="flex items-center gap-3 w-full sm:w-80 bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 rounded-xl px-3.5 py-2 shadow-xs focus-within:border-emerald-500 focus-within:ring-2 focus-within:ring-emerald-500/20 transition-all">
            <Search className="w-4 h-4 text-slate-400 flex-shrink-0" />
            <input
              type="text"
              placeholder={activeTab === 'scans' ? 'Search target path or scan ID...' : 'Search original path or threat name...'}
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
            Showing {activeTab === 'scans' ? filteredScans.length : filteredQuarantine.length} records
          </div>
        </div>

        {/* Tab 1: Scans History */}
        {activeTab === 'scans' && (
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl shadow-xs overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse text-xs min-w-[700px]">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50/50 dark:bg-surface-800/40 text-slate-500 dark:text-slate-400 uppercase font-bold tracking-wider text-[10px]">
                    <th className="py-3.5 px-4">Status</th>
                    <th className="py-3.5 px-4">Target Path</th>
                    <th className="py-3.5 px-4">Engine</th>
                    <th className="py-3.5 px-4">Scanned Files</th>
                    <th className="py-3.5 px-4">Threats Found</th>
                    <th className="py-3.5 px-4">Duration</th>
                    <th className="py-3.5 px-4">Started At</th>
                    <th className="py-3.5 px-4 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 dark:divide-surface-800/60 font-medium">
                  {loading && safeScans.length === 0 ? (
                    <tr>
                      <td colSpan={8} className="py-12 text-center text-slate-400">
                        <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 text-emerald-600" />
                        Loading scan records...
                      </td>
                    </tr>
                  ) : filteredScans.length === 0 ? (
                    <tr>
                      <td colSpan={8} className="py-12 text-center text-slate-400">
                        <FolderSearch className="w-8 h-8 mx-auto mb-2 opacity-40" />
                        No scan records found. Run your first malware scan using the button above.
                      </td>
                    </tr>
                  ) : (
                    filteredScans.map((scan) => (
                      <tr key={scan.id} className="hover:bg-slate-50/80 dark:hover:bg-surface-800/30 transition-colors">
                        <td className="py-3.5 px-4">
                          <span
                            className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full font-semibold uppercase text-[10px] ${
                              scan.status === 'completed'
                                ? scan.infected_files > 0
                                  ? 'bg-rose-50 text-rose-700 dark:bg-rose-950/40 dark:text-rose-400 border border-rose-200 dark:border-rose-900'
                                  : 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800'
                                : scan.status === 'running'
                                ? 'bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-400 border border-amber-200 dark:border-amber-800 animate-pulse'
                                : 'bg-slate-100 text-slate-700 dark:bg-surface-800 dark:text-slate-400'
                            }`}
                          >
                            <span
                              className={`w-1.5 h-1.5 rounded-full ${
                                scan.status === 'completed'
                                  ? scan.infected_files > 0
                                    ? 'bg-rose-500'
                                    : 'bg-emerald-500'
                                  : scan.status === 'running'
                                  ? 'bg-amber-500'
                                  : 'bg-slate-400'
                              }`}
                            />
                            {scan.status}
                          </span>
                        </td>
                        <td className="py-3.5 px-4 font-mono text-slate-900 dark:text-white font-semibold">
                          {scan.target_path}
                        </td>
                        <td className="py-3.5 px-4 text-slate-500 uppercase text-[11px]">{scan.scan_type}</td>
                        <td className="py-3.5 px-4 font-mono text-slate-700 dark:text-slate-300">
                          {scan.total_files.toLocaleString()} files
                        </td>
                        <td className="py-3.5 px-4">
                          {scan.infected_files > 0 ? (
                            <span className="font-bold text-rose-600 dark:text-rose-400 flex items-center gap-1">
                              <AlertTriangle className="w-3.5 h-3.5" />
                              {scan.infected_files} threats
                            </span>
                          ) : (
                            <span className="text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
                              <CheckCircle2 className="w-3.5 h-3.5" />
                              Clean
                            </span>
                          )}
                        </td>
                        <td className="py-3.5 px-4 text-slate-500 font-mono">
                          {calculateDuration(scan)}
                        </td>
                        <td className="py-3.5 px-4 text-slate-500 text-[11px]">
                          {scan.started_at ? new Date(scan.started_at).toLocaleString() : new Date(scan.created_at).toLocaleString()}
                        </td>
                        <td className="py-3.5 px-4 text-right">
                          <button
                            onClick={() => setFindingsModalScan(scan)}
                            className="px-2.5 py-1 rounded-lg bg-slate-100 dark:bg-surface-800 hover:bg-slate-200 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 font-semibold text-xs inline-flex items-center gap-1.5 transition cursor-pointer"
                          >
                            <Eye className="w-3.5 h-3.5" />
                            Findings ({scan.findings?.length || scan.infected_files || 0})
                          </button>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Tab 2: Quarantine Vault */}
        {activeTab === 'quarantine' && (
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl shadow-xs overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse text-xs min-w-[680px]">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50/50 dark:bg-surface-800/40 text-slate-500 dark:text-slate-400 uppercase font-bold tracking-wider text-[10px]">
                    <th className="py-3.5 px-4">Threat Name</th>
                    <th className="py-3.5 px-4">Original File Path</th>
                    <th className="py-3.5 px-4">Quarantine Identifier</th>
                    <th className="py-3.5 px-4">File Size</th>
                    <th className="py-3.5 px-4">Quarantined At</th>
                    <th className="py-3.5 px-4 text-right">Remediation</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 dark:divide-surface-800/60 font-medium">
                  {loading && safeQuarantine.length === 0 ? (
                    <tr>
                      <td colSpan={6} className="py-12 text-center text-slate-400">
                        <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 text-emerald-600" />
                        Loading quarantine vault...
                      </td>
                    </tr>
                  ) : filteredQuarantine.length === 0 ? (
                    <tr>
                      <td colSpan={6} className="py-12 text-center text-slate-400">
                        <CheckCircle2 className="w-8 h-8 mx-auto mb-2 text-emerald-500 opacity-60" />
                        Quarantine vault is clean. No active threats isolated.
                      </td>
                    </tr>
                  ) : (
                    filteredQuarantine.map((item) => (
                      <tr key={item.id} className="hover:bg-slate-50/80 dark:hover:bg-surface-800/30 transition-colors">
                        <td className="py-3.5 px-4 font-bold text-rose-600 dark:text-rose-400 flex items-center gap-1.5">
                          <AlertTriangle className="w-4 h-4 shrink-0" />
                          <span>{item.threat_name}</span>
                        </td>
                        <td className="py-3.5 px-4 font-mono text-slate-900 dark:text-white max-w-xs truncate" title={item.original_path}>
                          {item.original_path}
                        </td>
                        <td className="py-3.5 px-4 font-mono text-slate-500 text-[11px] max-w-xs truncate" title={item.quarantine_filename}>
                          {item.quarantine_filename}
                        </td>
                        <td className="py-3.5 px-4 font-mono text-slate-700 dark:text-slate-300">
                          {(item.file_size / 1024).toFixed(1)} KB
                        </td>
                        <td className="py-3.5 px-4 text-slate-500 text-[11px]">
                          {new Date(item.quarantined_at).toLocaleString()}
                        </td>
                        <td className="py-3.5 px-4 text-right space-x-2">
                          <button
                            onClick={() => handleRestoreQuarantine(item.id, item.original_path)}
                            disabled={actionLoadingId === `restore-${item.id}`}
                            className="px-2.5 py-1 rounded-lg bg-emerald-50 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800 hover:bg-emerald-100 font-semibold text-xs inline-flex items-center gap-1 transition cursor-pointer"
                          >
                            <Undo2 className="w-3.5 h-3.5" />
                            Restore
                          </button>
                          <button
                            onClick={() => handleDeleteQuarantine(item.id, item.threat_name)}
                            disabled={actionLoadingId === `delete-${item.id}`}
                            className="px-2.5 py-1 rounded-lg bg-rose-50 dark:bg-rose-950/40 text-rose-700 dark:text-rose-400 border border-rose-200 dark:border-rose-900 hover:bg-rose-100 font-semibold text-xs inline-flex items-center gap-1 transition cursor-pointer"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                            Purge
                          </button>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Modal: Run Malware Scan */}
        {scanModalOpen && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                type="button"
                onClick={() => setScanModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-2 mb-4">
                <div className="w-8 h-8 rounded-lg bg-emerald-100 dark:bg-emerald-950/60 flex items-center justify-center text-emerald-600">
                  <Play className="w-5 h-5 fill-emerald-600" />
                </div>
                <div>
                  <h3 className="font-bold text-base text-slate-900 dark:text-white">Run On-Demand Malware Scan</h3>
                  <p className="text-xs text-slate-500">Scan websites, uploaded files, or account roots for malware.</p>
                </div>
              </div>

              <form onSubmit={handleStartScan} className="space-y-4 text-xs font-semibold">
                <div>
                  <label className="block text-slate-700 dark:text-slate-300 mb-1">
                    Target Scan Directory <span className="text-rose-500">*</span>
                  </label>
                  <input
                    type="text"
                    required
                    value={targetPath}
                    onChange={(e) => setTargetPath(e.target.value)}
                    placeholder="/var/www/vhosts or /var/www/vhosts/example.com"
                    className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 font-mono text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  />
                  <p className="text-[11px] text-slate-400 font-normal mt-1">
                    Path traversal protection is enforced. Directory must reside within authorized hosting roots.
                  </p>
                  <div className="flex items-center gap-1.5 mt-2 flex-wrap">
                    <span className="text-[10px] text-slate-400 font-medium">Quick Presets:</span>
                    <button
                      type="button"
                      onClick={() => setTargetPath('/var/www/vhosts')}
                      className="text-[10px] px-2 py-0.5 rounded bg-slate-100 dark:bg-surface-800 hover:bg-emerald-50 dark:hover:bg-emerald-950/40 text-slate-700 dark:text-slate-300 hover:text-emerald-600 transition cursor-pointer font-medium"
                    >
                      All Websites (/var/www/vhosts)
                    </button>
                    <button
                      type="button"
                      onClick={() => setTargetPath('/var/www')}
                      className="text-[10px] px-2 py-0.5 rounded bg-slate-100 dark:bg-surface-800 hover:bg-emerald-50 dark:hover:bg-emerald-950/40 text-slate-700 dark:text-slate-300 hover:text-emerald-600 transition cursor-pointer font-medium"
                    >
                      Webroot (/var/www)
                    </button>
                    <button
                      type="button"
                      onClick={() => setTargetPath('/tmp')}
                      className="text-[10px] px-2 py-0.5 rounded bg-slate-100 dark:bg-surface-800 hover:bg-emerald-50 dark:hover:bg-emerald-950/40 text-slate-700 dark:text-slate-300 hover:text-emerald-600 transition cursor-pointer font-medium"
                    >
                      Temp Uploads (/tmp)
                    </button>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 mb-1">Scanning Engine</label>
                    <select
                      value={selectedEngine}
                      onChange={(e) => setSelectedEngine(e.target.value as any)}
                      className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white focus:outline-none"
                    >
                      <option value="combined">Combined (ClamAV + Heuristic)</option>
                      <option value="clamav">ClamAV Daemon / Clamscan</option>
                      <option value="heuristic">Heuristic Webshell Scanner</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 mb-1">Threat Remediation</label>
                    <div className="pt-1.5 flex items-center gap-2">
                      <input
                        type="checkbox"
                        id="autoQuar"
                        checked={autoQuarantine}
                        onChange={(e) => setAutoQuarantine(e.target.checked)}
                        className="w-4 h-4 text-emerald-600 rounded"
                      />
                      <label htmlFor="autoQuar" className="text-slate-700 dark:text-slate-300 cursor-pointer font-medium">
                        Auto-quarantine threats
                      </label>
                    </div>
                  </div>
                </div>

                <div>
                  <label className="block text-slate-700 dark:text-slate-300 mb-1">Path / Pattern Exclusions</label>
                  <input
                    type="text"
                    value={exclusions}
                    onChange={(e) => setExclusions(e.target.value)}
                    placeholder=".git, node_modules, cache, *.log"
                    className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 font-mono text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  />
                  <p className="text-[11px] text-slate-400 font-normal mt-1">
                    Comma-separated list of filenames, extensions, or directory patterns to exclude.
                  </p>
                </div>

                <div className="flex justify-end gap-3 pt-3 border-t border-slate-200 dark:border-surface-700">
                  <button
                    type="button"
                    onClick={() => setScanModalOpen(false)}
                    className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 font-bold"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={submittingScan}
                    className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 text-white font-bold transition flex items-center gap-2 cursor-pointer shadow-md"
                  >
                    {submittingScan ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Play className="w-3.5 h-3.5 fill-white" />}
                    <span>{submittingScan ? 'Dispatching...' : 'Start Scan Now'}</span>
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* Modal: Findings Detail */}
        {findingsModalScan && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative max-h-[85vh] flex flex-col">
              <button
                type="button"
                onClick={() => setFindingsModalScan(null)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-2 mb-4">
                <div className="w-8 h-8 rounded-lg bg-indigo-100 dark:bg-indigo-950/60 flex items-center justify-center text-indigo-600">
                  <FileWarning className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="font-bold text-base text-slate-900 dark:text-white">Scan Findings Report</h3>
                  <p className="text-xs text-slate-500 font-mono">
                    Target: {findingsModalScan.target_path} • Status: {findingsModalScan.status}
                  </p>
                </div>
              </div>

              <div className="overflow-y-auto flex-1 pr-1 space-y-3">
                {(!findingsModalScan.findings || findingsModalScan.findings.length === 0) ? (
                  <div className="p-8 text-center bg-slate-50 dark:bg-surface-800 rounded-xl border border-slate-200 dark:border-surface-700">
                    <CheckCircle2 className="w-10 h-10 text-emerald-500 mx-auto mb-2" />
                    <p className="font-bold text-slate-800 dark:text-slate-200 text-sm">No Malware Detected</p>
                    <p className="text-xs text-slate-400 mt-1">
                      Scanned {findingsModalScan.total_files} files across {findingsModalScan.target_path}. All files passed signature verification.
                    </p>
                  </div>
                ) : (
                  findingsModalScan.findings.map((threat, idx) => (
                    <div
                      key={idx}
                      className="p-3.5 bg-slate-50 dark:bg-surface-800/90 border border-slate-200 dark:border-surface-700 rounded-xl text-xs space-y-2"
                    >
                      <div className="flex items-center justify-between">
                        <span className="font-bold text-rose-600 dark:text-rose-400 flex items-center gap-1.5">
                          <AlertTriangle className="w-4 h-4 shrink-0" />
                          {threat.threat_name}
                        </span>
                        {threat.quarantined ? (
                          <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-slate-200 dark:bg-surface-700 text-slate-700 dark:text-slate-300">
                            Quarantined
                          </span>
                        ) : (
                          <button
                            onClick={() => handleManualQuarantine(threat.file_path, threat.threat_name)}
                            disabled={actionLoadingId === `quar-${threat.file_path}`}
                            className="px-2.5 py-1 rounded-lg bg-rose-600 hover:bg-rose-700 text-white font-bold text-[11px] shadow-xs cursor-pointer"
                          >
                            Quarantine Now
                          </button>
                        )}
                      </div>
                      <div className="font-mono text-slate-800 dark:text-slate-200 break-all bg-white dark:bg-surface-900 p-2 rounded-lg border border-slate-200 dark:border-surface-700 text-[11px]">
                        {threat.file_path}
                      </div>
                    </div>
                  ))
                )}
              </div>

              <div className="flex justify-end pt-3 border-t border-slate-200 dark:border-surface-700 mt-3">
                <button
                  type="button"
                  onClick={() => setFindingsModalScan(null)}
                  className="px-5 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 text-xs font-bold"
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
