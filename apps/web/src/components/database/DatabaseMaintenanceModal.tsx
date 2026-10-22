'use client';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import {
  X,
  RefreshCw,
  Wrench,
  CheckCircle2,
  AlertTriangle,
  Activity,
  HardDrive,
  Layers,
  Sparkles,
  Search,
  Check,
  Clock,
  Database as DatabaseIcon,
  Zap,
  ArrowRight,
  TrendingDown,
  ShieldCheck,
  AlertCircle,
  Play,
  RotateCcw,
} from 'lucide-react';
import {
  Database,
  DatabaseHealthReport,
  TableHealthInfo,
  DatabaseMaintenanceResult,
  DatabaseMaintenanceLog,
  getDatabaseHealth,
  optimizeDatabase,
  checkDatabaseIntegrity,
  analyzeDatabase,
  listDatabaseMaintenanceLogs,
} from '@/lib/api';

interface DatabaseMaintenanceModalProps {
  database: Database | null;
  onClose: () => void;
  showToast: (msg: string, isError?: boolean) => void;
}

function formatBytes(bytes: number, decimals = 2): string {
  if (bytes === 0 || isNaN(bytes)) return '0 B';
  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i] || 'B'}`;
}

function formatNumber(num: number): string {
  if (isNaN(num)) return '0';
  return new Intl.NumberFormat().format(num);
}

export function DatabaseMaintenanceModal({
  database,
  onClose,
  showToast,
}: DatabaseMaintenanceModalProps) {
  const [activeTab, setActiveTab] = useState<'overview' | 'tables' | 'history'>('overview');
  const [health, setHealth] = useState<DatabaseHealthReport | null>(null);
  const [logs, setLogs] = useState<DatabaseMaintenanceLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [logsLoading, setLogsLoading] = useState(false);

  // Table selection & search
  const [searchTable, setSearchTable] = useState('');
  const [selectedTables, setSelectedTables] = useState<string[]>([]);
  const [onlyFragmented, setOnlyFragmented] = useState(false);

  // In-flight operations
  const [runningOp, setRunningOp] = useState<'optimize' | 'check' | 'analyze' | null>(null);
  const [lastResult, setLastResult] = useState<DatabaseMaintenanceResult | null>(null);

  // Load Health Data
  const fetchHealth = useCallback(async () => {
    if (!database) return;
    setLoading(true);
    try {
      const res = await getDatabaseHealth(database.id);
      if (res.success && res.data) {
        setHealth(res.data);
      } else {
        showToast(res.error?.message || 'Failed to inspect database health', true);
      }
    } catch {
      showToast('Failed to load database health metrics', true);
    } finally {
      setLoading(false);
    }
  }, [database, showToast]);

  // Load Maintenance Logs
  const fetchLogs = useCallback(async () => {
    if (!database) return;
    setLogsLoading(true);
    try {
      const res = await listDatabaseMaintenanceLogs(database.id, 50);
      if (res.success && res.data) {
        setLogs(res.data);
      }
    } catch {
      // Non-critical, ignore
    } finally {
      setLogsLoading(false);
    }
  }, [database]);

  useEffect(() => {
    if (database) {
      fetchHealth();
      fetchLogs();
      setSelectedTables([]);
      setLastResult(null);
    }
  }, [database, fetchHealth, fetchLogs]);

  // Filtered Tables
  const filteredTables = useMemo(() => {
    if (!health || !health.tables) return [];
    return health.tables.filter((t) => {
      const matchesSearch = t.table_name.toLowerCase().includes(searchTable.toLowerCase());
      const matchesFrag = onlyFragmented ? t.fragmentation_percent > 5 : true;
      return matchesSearch && matchesFrag;
    });
  }, [health, searchTable, onlyFragmented]);

  const toggleSelectTable = (name: string) => {
    setSelectedTables((prev) =>
      prev.includes(name) ? prev.filter((n) => n !== name) : [...prev, name]
    );
  };

  const handleSelectAllVisible = () => {
    const visibleNames = filteredTables.map((t) => t.table_name);
    const allSelected = visibleNames.every((n) => selectedTables.includes(n));
    if (allSelected) {
      setSelectedTables((prev) => prev.filter((n) => !visibleNames.includes(n)));
    } else {
      setSelectedTables((prev) => Array.from(new Set([...prev, ...visibleNames])));
    }
  };

  // Run 1-Click Optimize
  const handleOptimize = async (targetTables?: string[]) => {
    if (!database) return;
    const tablesToRun = targetTables ?? (selectedTables.length > 0 ? selectedTables : undefined);
    setRunningOp('optimize');
    try {
      const res = await optimizeDatabase(database.id, tablesToRun);
      if (res.success && res.data) {
        setLastResult(res.data);
        const reclaimedStr = formatBytes(res.data.bytes_reclaimed);
        showToast(
          `Optimization complete! Reclaimed ${reclaimedStr} across ${res.data.tables_processed} table(s) in ${res.data.duration_ms}ms.`
        );
        fetchHealth();
        fetchLogs();
      } else {
        showToast(res.error?.message || 'Database optimization encountered an issue', true);
      }
    } catch (err: any) {
      showToast(err?.message || 'Optimization request failed', true);
    } finally {
      setRunningOp(null);
    }
  };

  // Run Integrity Check
  const handleCheck = async (targetTables?: string[]) => {
    if (!database) return;
    const tablesToRun = targetTables ?? (selectedTables.length > 0 ? selectedTables : undefined);
    setRunningOp('check');
    try {
      const res = await checkDatabaseIntegrity(database.id, tablesToRun);
      if (res.success && res.data) {
        setLastResult(res.data);
        const warningCount =
          res.data.table_results?.filter((r) => r.status !== 'OK').length || 0;
        if (warningCount > 0) {
          showToast(
            `Integrity check complete: ${warningCount} table(s) reported warnings/errors.`,
            true
          );
        } else {
          showToast(
            `Integrity check passed! All ${res.data.tables_processed} table(s) verified healthy in ${res.data.duration_ms}ms.`
          );
        }
        fetchLogs();
      } else {
        showToast(res.error?.message || 'Integrity check encountered an error', true);
      }
    } catch (err: any) {
      showToast(err?.message || 'Integrity check request failed', true);
    } finally {
      setRunningOp(null);
    }
  };

  // Run Index Statistics Analysis
  const handleAnalyze = async (targetTables?: string[]) => {
    if (!database) return;
    const tablesToRun = targetTables ?? (selectedTables.length > 0 ? selectedTables : undefined);
    setRunningOp('analyze');
    try {
      const res = await analyzeDatabase(database.id, tablesToRun);
      if (res.success && res.data) {
        setLastResult(res.data);
        showToast(
          `Index statistics updated for ${res.data.tables_processed} table(s) in ${res.data.duration_ms}ms.`
        );
        fetchLogs();
      } else {
        showToast(res.error?.message || 'Index analysis encountered an issue', true);
      }
    } catch (err: any) {
      showToast(err?.message || 'Analyze request failed', true);
    } finally {
      setRunningOp(null);
    }
  };

  if (!database) return null;

  const fragPct = health?.overall_fragmentation_percent || 0;
  const isHealthy = fragPct < 10;
  const isModerate = fragPct >= 10 && fragPct < 25;
  const isHighFrag = fragPct >= 25;

  return (
    <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-3 sm:p-5 animate-fadeIn">
      <div className="w-full max-w-4xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl overflow-hidden flex flex-col max-h-[92vh]">
        {/* Header */}
        <div className="px-6 py-4 border-b border-slate-200 dark:border-surface-800 bg-slate-50/50 dark:bg-surface-850 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="p-2.5 rounded-xl bg-emerald-50 dark:bg-emerald-950/50 text-emerald-600 dark:text-emerald-400 border border-emerald-200/60 dark:border-emerald-800/60 shadow-2xs">
              <Activity className="w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-base font-bold text-slate-900 dark:text-white">
                  Database Health & Maintenance
                </h2>
                <span className="px-2 py-0.5 rounded-full bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-300 font-mono text-[11px] font-semibold border border-emerald-200 dark:border-emerald-800">
                  {database.name}
                </span>
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                Inspect storage utilization, optimize fragmented tables, and check index integrity
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => {
                fetchHealth();
                fetchLogs();
              }}
              disabled={loading}
              title="Refresh Health Telemetry"
              className="p-2 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-surface-800 transition cursor-pointer"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin text-emerald-500' : ''}`} />
            </button>
            <button
              onClick={onClose}
              className="p-2 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-surface-800 transition cursor-pointer"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Navigation Tabs */}
        <div className="px-6 border-b border-slate-200 dark:border-surface-800 bg-white dark:bg-surface-900 flex items-center gap-1 text-xs font-semibold">
          <button
            onClick={() => setActiveTab('overview')}
            className={`py-3 px-3.5 border-b-2 transition flex items-center gap-1.5 cursor-pointer ${
              activeTab === 'overview'
                ? 'border-emerald-500 text-emerald-600 dark:text-emerald-400'
                : 'border-transparent text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
            }`}
          >
            <Activity className="w-4 h-4" />
            <span>Health & Overview</span>
          </button>
          <button
            onClick={() => setActiveTab('tables')}
            className={`py-3 px-3.5 border-b-2 transition flex items-center gap-1.5 cursor-pointer ${
              activeTab === 'tables'
                ? 'border-emerald-500 text-emerald-600 dark:text-emerald-400'
                : 'border-transparent text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
            }`}
          >
            <Layers className="w-4 h-4" />
            <span>Tables & Storage</span>
            {health?.table_count !== undefined && (
              <span className="ml-1 px-1.5 py-0.2 rounded-full bg-slate-100 dark:bg-surface-800 text-[10px] text-slate-600 dark:text-slate-300">
                {health.table_count}
              </span>
            )}
          </button>
          <button
            onClick={() => setActiveTab('history')}
            className={`py-3 px-3.5 border-b-2 transition flex items-center gap-1.5 cursor-pointer ${
              activeTab === 'history'
                ? 'border-emerald-500 text-emerald-600 dark:text-emerald-400'
                : 'border-transparent text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
            }`}
          >
            <Clock className="w-4 h-4" />
            <span>Maintenance History</span>
            {logs.length > 0 && (
              <span className="ml-1 px-1.5 py-0.2 rounded-full bg-slate-100 dark:bg-surface-800 text-[10px] text-slate-600 dark:text-slate-300">
                {logs.length}
              </span>
            )}
          </button>
        </div>

        {/* Modal Body */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6 text-xs">
          {/* Recent Action Banner (if just completed) */}
          {lastResult && (
            <div
              className={`p-4 rounded-xl border flex items-start justify-between gap-3 animate-fadeIn ${
                lastResult.status === 'success'
                  ? 'bg-emerald-50/70 dark:bg-emerald-950/30 border-emerald-200 dark:border-emerald-800 text-emerald-900 dark:text-emerald-200'
                  : 'bg-amber-50/70 dark:bg-amber-950/30 border-amber-200 dark:border-amber-800 text-amber-900 dark:text-amber-200'
              }`}
            >
              <div className="flex items-start gap-3">
                {lastResult.status === 'success' ? (
                  <CheckCircle2 className="w-5 h-5 text-emerald-600 dark:text-emerald-400 flex-shrink-0 mt-0.5" />
                ) : (
                  <AlertTriangle className="w-5 h-5 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5" />
                )}
                <div className="space-y-1">
                  <div className="font-bold text-sm capitalize">
                    {lastResult.operation} Operation Completed
                  </div>
                  <div className="text-xs opacity-90">{lastResult.details}</div>
                  <div className="flex flex-wrap items-center gap-4 text-[11px] font-mono mt-1 opacity-80">
                    <span>Processed: {lastResult.tables_processed} table(s)</span>
                    {lastResult.operation === 'optimize' && (
                      <span className="font-semibold text-emerald-700 dark:text-emerald-300">
                        Space Reclaimed: {formatBytes(lastResult.bytes_reclaimed)}
                      </span>
                    )}
                    <span>Duration: {lastResult.duration_ms}ms</span>
                  </div>
                </div>
              </div>
              <button
                onClick={() => setLastResult(null)}
                className="text-slate-400 hover:text-slate-600 dark:hover:text-white"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
          )}

          {/* TAB 1: OVERVIEW */}
          {activeTab === 'overview' && (
            <div className="space-y-6">
              {/* Storage & Fragmentation Key Metrics */}
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                {/* Total Size */}
                <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-850 border border-slate-200/80 dark:border-surface-800 shadow-2xs">
                  <div className="flex items-center justify-between text-slate-500 dark:text-slate-400 mb-1">
                    <span className="font-medium text-[11px] uppercase tracking-wider">
                      Total Size
                    </span>
                    <HardDrive className="w-4 h-4 text-emerald-500" />
                  </div>
                  <div className="text-xl font-bold text-slate-900 dark:text-white font-mono">
                    {formatBytes(health?.total_size_bytes || 0)}
                  </div>
                  <div className="text-[11px] text-slate-500 mt-1">
                    {health?.table_count || 0} total tables
                  </div>
                </div>

                {/* Data Size */}
                <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-850 border border-slate-200/80 dark:border-surface-800 shadow-2xs">
                  <div className="flex items-center justify-between text-slate-500 dark:text-slate-400 mb-1">
                    <span className="font-medium text-[11px] uppercase tracking-wider">
                      Data Size
                    </span>
                    <DatabaseIcon className="w-4 h-4 text-sky-500" />
                  </div>
                  <div className="text-xl font-bold text-slate-900 dark:text-white font-mono">
                    {formatBytes(health?.total_data_bytes || 0)}
                  </div>
                  <div className="text-[11px] text-slate-500 mt-1">Row data storage</div>
                </div>

                {/* Index Size */}
                <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-850 border border-slate-200/80 dark:border-surface-800 shadow-2xs">
                  <div className="flex items-center justify-between text-slate-500 dark:text-slate-400 mb-1">
                    <span className="font-medium text-[11px] uppercase tracking-wider">
                      Index Size
                    </span>
                    <Layers className="w-4 h-4 text-indigo-500" />
                  </div>
                  <div className="text-xl font-bold text-slate-900 dark:text-white font-mono">
                    {formatBytes(health?.total_index_bytes || 0)}
                  </div>
                  <div className="text-[11px] text-slate-500 mt-1">B-Tree & keys</div>
                </div>

                {/* Free Fragmented Space */}
                <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-850 border border-slate-200/80 dark:border-surface-800 shadow-2xs">
                  <div className="flex items-center justify-between text-slate-500 dark:text-slate-400 mb-1">
                    <span className="font-medium text-[11px] uppercase tracking-wider">
                      Fragmented Space
                    </span>
                    <TrendingDown className="w-4 h-4 text-amber-500" />
                  </div>
                  <div className="text-xl font-bold text-amber-600 dark:text-amber-400 font-mono">
                    {formatBytes(health?.total_data_free_bytes || 0)}
                  </div>
                  <div className="text-[11px] text-slate-500 mt-1">
                    {health?.fragmented_tables_count || 0} fragmented table(s)
                  </div>
                </div>
              </div>

              {/* Fragmentation Gauge Card */}
              <div className="p-5 rounded-2xl bg-gradient-to-br from-slate-50 to-slate-100/60 dark:from-surface-850 dark:to-surface-800 border border-slate-200 dark:border-surface-750 shadow-2xs">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-4">
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="font-bold text-sm text-slate-900 dark:text-white">
                        Database Fragmentation Ratio
                      </span>
                      <span
                        className={`px-2 py-0.5 rounded-full text-[11px] font-bold ${
                          isHealthy
                            ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950/60 dark:text-emerald-400 border border-emerald-300 dark:border-emerald-700'
                            : isModerate
                            ? 'bg-amber-100 text-amber-800 dark:bg-amber-950/60 dark:text-amber-400 border border-amber-300 dark:border-amber-700'
                            : 'bg-rose-100 text-rose-800 dark:bg-rose-950/60 dark:text-rose-400 border border-rose-300 dark:border-rose-700'
                        }`}
                      >
                        {fragPct.toFixed(1)}% Fragmented
                      </span>
                    </div>
                    <p className="text-slate-500 dark:text-slate-400 mt-1 leading-relaxed">
                      {isHealthy
                        ? 'Optimal health: Database storage is compact with minimal unused disk space.'
                        : isModerate
                        ? 'Moderate fragmentation: Running 1-click optimization will reclaim unused disk space.'
                        : 'High fragmentation: Significant disk capacity is trapped in deleted rows and uncompacted indexes.'}
                    </p>
                  </div>

                  {/* 1-Click Optimize Button */}
                  <button
                    onClick={() => handleOptimize()}
                    disabled={runningOp !== null}
                    className="px-4 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold transition shadow-sm flex items-center justify-center gap-2 cursor-pointer flex-shrink-0 disabled:opacity-50"
                  >
                    <Sparkles className={`w-4 h-4 ${runningOp === 'optimize' ? 'animate-spin' : ''}`} />
                    <span>
                      {runningOp === 'optimize' ? 'Optimizing Tables...' : '1-Click Optimize Database'}
                    </span>
                  </button>
                </div>

                {/* Progress Bar */}
                <div className="w-full bg-slate-200 dark:bg-surface-900 rounded-full h-3 overflow-hidden p-0.5 border border-slate-300 dark:border-surface-700">
                  <div
                    className={`h-full rounded-full transition-all duration-500 ${
                      isHealthy
                        ? 'bg-emerald-500'
                        : isModerate
                        ? 'bg-amber-500'
                        : 'bg-rose-500'
                    }`}
                    style={{ width: `${Math.min(Math.max(fragPct, 2), 100)}%` }}
                  />
                </div>
                <div className="flex justify-between items-center text-[10px] text-slate-400 mt-1.5 font-mono">
                  <span>0% (Optimal)</span>
                  <span>25% (Warning)</span>
                  <span>50%+ (Severe)</span>
                </div>
              </div>

              {/* Maintenance Actions Grid */}
              <div className="space-y-3">
                <h3 className="font-bold text-slate-900 dark:text-white text-xs uppercase tracking-wider">
                  Engine Maintenance Routines
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                  {/* Table Optimization Card */}
                  <div className="p-4 rounded-xl bg-white dark:bg-surface-850 border border-slate-200 dark:border-surface-800 flex flex-col justify-between hover:border-emerald-500/50 transition">
                    <div>
                      <div className="flex items-center gap-2 mb-2 text-emerald-600 dark:text-emerald-400 font-bold">
                        <Sparkles className="w-4 h-4" />
                        <span>OPTIMIZE TABLE</span>
                      </div>
                      <p className="text-slate-500 dark:text-slate-400 text-[11px] leading-relaxed mb-4">
                        Reorganizes the physical storage of table data and associated index data to
                        reduce storage space and improve I/O efficiency.
                      </p>
                    </div>
                    <button
                      onClick={() => handleOptimize()}
                      disabled={runningOp !== null}
                      className="w-full py-2 rounded-lg bg-emerald-50 dark:bg-emerald-950/40 hover:bg-emerald-100 dark:hover:bg-emerald-900/40 text-emerald-700 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-800 font-bold transition flex items-center justify-center gap-1.5 cursor-pointer disabled:opacity-50"
                    >
                      <Sparkles className={`w-3.5 h-3.5 ${runningOp === 'optimize' ? 'animate-spin' : ''}`} />
                      <span>{runningOp === 'optimize' ? 'Running...' : 'Optimize All Tables'}</span>
                    </button>
                  </div>

                  {/* Integrity Check Card */}
                  <div className="p-4 rounded-xl bg-white dark:bg-surface-850 border border-slate-200 dark:border-surface-800 flex flex-col justify-between hover:border-sky-500/50 transition">
                    <div>
                      <div className="flex items-center gap-2 mb-2 text-sky-600 dark:text-sky-400 font-bold">
                        <ShieldCheck className="w-4 h-4" />
                        <span>CHECK TABLE</span>
                      </div>
                      <p className="text-slate-500 dark:text-slate-400 text-[11px] leading-relaxed mb-4">
                        Verifies table structure and data integrity. Discovers corrupted row
                        pointers or damaged B-Trees without downtime.
                      </p>
                    </div>
                    <button
                      onClick={() => handleCheck()}
                      disabled={runningOp !== null}
                      className="w-full py-2 rounded-lg bg-sky-50 dark:bg-sky-950/40 hover:bg-sky-100 dark:hover:bg-sky-900/40 text-sky-700 dark:text-sky-300 border border-sky-200 dark:border-sky-800 font-bold transition flex items-center justify-center gap-1.5 cursor-pointer disabled:opacity-50"
                    >
                      <ShieldCheck className={`w-3.5 h-3.5 ${runningOp === 'check' ? 'animate-spin' : ''}`} />
                      <span>{runningOp === 'check' ? 'Checking...' : 'Check Integrity'}</span>
                    </button>
                  </div>

                  {/* Analyze Index Statistics Card */}
                  <div className="p-4 rounded-xl bg-white dark:bg-surface-850 border border-slate-200 dark:border-surface-800 flex flex-col justify-between hover:border-indigo-500/50 transition">
                    <div>
                      <div className="flex items-center gap-2 mb-2 text-indigo-600 dark:text-indigo-400 font-bold">
                        <Zap className="w-4 h-4" />
                        <span>ANALYZE TABLE</span>
                      </div>
                      <p className="text-slate-500 dark:text-slate-400 text-[11px] leading-relaxed mb-4">
                        Refreshes key distribution and index cardinality statistics so MySQL query
                        optimizers choose faster execution plans.
                      </p>
                    </div>
                    <button
                      onClick={() => handleAnalyze()}
                      disabled={runningOp !== null}
                      className="w-full py-2 rounded-lg bg-indigo-50 dark:bg-indigo-950/40 hover:bg-indigo-100 dark:hover:bg-indigo-900/40 text-indigo-700 dark:text-indigo-300 border border-indigo-200 dark:border-indigo-800 font-bold transition flex items-center justify-center gap-1.5 cursor-pointer disabled:opacity-50"
                    >
                      <Zap className={`w-3.5 h-3.5 ${runningOp === 'analyze' ? 'animate-spin' : ''}`} />
                      <span>{runningOp === 'analyze' ? 'Analyzing...' : 'Update Index Stats'}</span>
                    </button>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* TAB 2: TABLE-BY-TABLE BREAKDOWN */}
          {activeTab === 'tables' && (
            <div className="space-y-4">
              {/* Table Controls */}
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                <div className="flex items-center gap-2">
                  <div className="relative w-64">
                    <input
                      type="text"
                      placeholder="Search table name..."
                      value={searchTable}
                      onChange={(e) => setSearchTable(e.target.value)}
                      className="w-full pl-8 pr-3 py-1.5 rounded-lg bg-slate-50 dark:bg-surface-850 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-slate-100 text-xs focus:outline-none focus:border-emerald-500"
                    />
                    <Search className="w-3.5 h-3.5 text-slate-400 absolute left-2.5 top-2.5 pointer-events-none" />
                  </div>
                  <label className="flex items-center gap-1.5 text-slate-600 dark:text-slate-400 cursor-pointer select-none">
                    <input
                      type="checkbox"
                      checked={onlyFragmented}
                      onChange={(e) => setOnlyFragmented(e.target.checked)}
                      className="rounded border-slate-300 text-emerald-600 focus:ring-0"
                    />
                    <span>Only fragmented (&gt; 5%)</span>
                  </label>
                </div>

                {/* Bulk Actions on Selected Tables */}
                <div className="flex items-center gap-2">
                  {selectedTables.length > 0 && (
                    <span className="text-slate-500 text-[11px] font-mono">
                      {selectedTables.length} selected
                    </span>
                  )}
                  <button
                    onClick={() => handleOptimize()}
                    disabled={selectedTables.length === 0 || runningOp !== null}
                    className="px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold transition shadow-xs flex items-center gap-1.5 cursor-pointer disabled:opacity-40"
                  >
                    <Sparkles className="w-3.5 h-3.5" />
                    <span>Optimize Selected</span>
                  </button>
                  <button
                    onClick={() => handleCheck()}
                    disabled={selectedTables.length === 0 || runningOp !== null}
                    className="px-3 py-1.5 rounded-lg bg-sky-600 hover:bg-sky-700 text-white font-bold transition shadow-xs flex items-center gap-1.5 cursor-pointer disabled:opacity-40"
                  >
                    <ShieldCheck className="w-3.5 h-3.5" />
                    <span>Check Selected</span>
                  </button>
                </div>
              </div>

              {/* Table List */}
              <div className="border border-slate-200 dark:border-surface-800 rounded-xl overflow-hidden shadow-2xs">
                <div className="overflow-x-auto max-h-[420px]">
                  <table className="w-full text-left text-xs border-collapse">
                    <thead className="sticky top-0 z-10">
                      <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-850 text-slate-700 dark:text-slate-300 font-bold uppercase text-[10px] tracking-wider select-none">
                        <th className="px-3 py-2.5 w-8 text-center">
                          <input
                            type="checkbox"
                            checked={
                              filteredTables.length > 0 &&
                              filteredTables.every((t) => selectedTables.includes(t.table_name))
                            }
                            onChange={handleSelectAllVisible}
                            className="rounded border-slate-300 text-emerald-600 focus:ring-0 cursor-pointer"
                          />
                        </th>
                        <th className="px-3 py-2.5">Table Name</th>
                        <th className="px-3 py-2.5">Engine</th>
                        <th className="px-3 py-2.5 text-right">Rows</th>
                        <th className="px-3 py-2.5 text-right">Data Size</th>
                        <th className="px-3 py-2.5 text-right">Index Size</th>
                        <th className="px-3 py-2.5 text-right">Free Space</th>
                        <th className="px-3 py-2.5 min-w-[120px]">Fragmentation</th>
                        <th className="px-3 py-2.5 text-center">Action</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100 dark:divide-surface-800 font-mono text-[11px]">
                      {filteredTables.length === 0 ? (
                        <tr>
                          <td colSpan={9} className="px-4 py-8 text-center text-slate-400">
                            No tables found matching criteria.
                          </td>
                        </tr>
                      ) : (
                        filteredTables.map((tbl) => {
                          const isSelected = selectedTables.includes(tbl.table_name);
                          const isFrag = tbl.fragmentation_percent > 15;
                          return (
                            <tr
                              key={tbl.table_name}
                              className={`hover:bg-slate-50/70 dark:hover:bg-surface-850/60 transition ${
                                isSelected ? 'bg-emerald-50/30 dark:bg-emerald-950/20' : ''
                              }`}
                            >
                              <td className="px-3 py-2 text-center">
                                <input
                                  type="checkbox"
                                  checked={isSelected}
                                  onChange={() => toggleSelectTable(tbl.table_name)}
                                  className="rounded border-slate-300 text-emerald-600 focus:ring-0 cursor-pointer"
                                />
                              </td>
                              <td className="px-3 py-2 font-semibold text-slate-800 dark:text-slate-200">
                                {tbl.table_name}
                              </td>
                              <td className="px-3 py-2 text-slate-500">{tbl.engine}</td>
                              <td className="px-3 py-2 text-right text-slate-700 dark:text-slate-300">
                                {formatNumber(tbl.rows)}
                              </td>
                              <td className="px-3 py-2 text-right text-slate-700 dark:text-slate-300">
                                {formatBytes(tbl.data_size_bytes)}
                              </td>
                              <td className="px-3 py-2 text-right text-slate-500">
                                {formatBytes(tbl.index_size_bytes)}
                              </td>
                              <td className="px-3 py-2 text-right font-semibold text-amber-600 dark:text-amber-400">
                                {tbl.data_free_bytes > 0 ? formatBytes(tbl.data_free_bytes) : '-'}
                              </td>
                              <td className="px-3 py-2">
                                <div className="flex items-center gap-2">
                                  <div className="w-16 bg-slate-200 dark:bg-surface-800 rounded-full h-1.5 overflow-hidden">
                                    <div
                                      className={`h-full ${
                                        tbl.fragmentation_percent > 20
                                          ? 'bg-rose-500'
                                          : tbl.fragmentation_percent > 5
                                          ? 'bg-amber-500'
                                          : 'bg-emerald-500'
                                      }`}
                                      style={{
                                        width: `${Math.min(
                                          Math.max(tbl.fragmentation_percent, 0),
                                          100
                                        )}%`,
                                      }}
                                    />
                                  </div>
                                  <span
                                    className={`text-[10px] ${
                                      isFrag
                                        ? 'text-rose-600 dark:text-rose-400 font-bold'
                                        : 'text-slate-500'
                                    }`}
                                  >
                                    {tbl.fragmentation_percent.toFixed(1)}%
                                  </span>
                                </div>
                              </td>
                              <td className="px-3 py-2 text-center">
                                <button
                                  onClick={() => handleOptimize([tbl.table_name])}
                                  disabled={runningOp !== null}
                                  className="text-emerald-600 hover:text-emerald-700 dark:text-emerald-400 hover:underline font-semibold cursor-pointer disabled:opacity-40"
                                >
                                  Optimize
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
            </div>
          )}

          {/* TAB 3: OPERATION HISTORY */}
          {activeTab === 'history' && (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="font-bold text-slate-900 dark:text-white text-xs uppercase tracking-wider">
                    Past Maintenance Executions
                  </h3>
                  <p className="text-slate-500 dark:text-slate-400 text-[11px] mt-0.5">
                    Immutable audit log of all database check, optimize, and analyze operations
                  </p>
                </div>
                <button
                  onClick={fetchLogs}
                  disabled={logsLoading}
                  className="px-3 py-1 rounded-lg bg-slate-100 dark:bg-surface-800 hover:bg-slate-200 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-300 font-semibold transition flex items-center gap-1.5 cursor-pointer"
                >
                  <RefreshCw className={`w-3.5 h-3.5 ${logsLoading ? 'animate-spin' : ''}`} />
                  <span>Refresh Log</span>
                </button>
              </div>

              <div className="border border-slate-200 dark:border-surface-800 rounded-xl overflow-hidden shadow-2xs">
                <div className="overflow-x-auto max-h-[420px]">
                  <table className="w-full text-left text-xs border-collapse">
                    <thead className="sticky top-0 z-10">
                      <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-850 text-slate-700 dark:text-slate-300 font-bold uppercase text-[10px] tracking-wider select-none">
                        <th className="px-3 py-2.5">Timestamp</th>
                        <th className="px-3 py-2.5">Operation</th>
                        <th className="px-3 py-2.5">Status</th>
                        <th className="px-3 py-2.5 text-right">Tables</th>
                        <th className="px-3 py-2.5 text-right">Reclaimed</th>
                        <th className="px-3 py-2.5 text-right">Duration</th>
                        <th className="px-3 py-2.5">Initiated By</th>
                        <th className="px-3 py-2.5">Details</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100 dark:divide-surface-800 font-mono text-[11px]">
                      {logs.length === 0 ? (
                        <tr>
                          <td colSpan={8} className="px-4 py-8 text-center text-slate-400">
                            No maintenance logs recorded yet. Run an optimization or check to create history.
                          </td>
                        </tr>
                      ) : (
                        logs.map((log) => {
                          const isSuccess = log.status === 'success';
                          const isWarning = log.status === 'warning';
                          return (
                            <tr
                              key={log.id}
                              className="hover:bg-slate-50/70 dark:hover:bg-surface-850/60 transition"
                            >
                              <td className="px-3 py-2 text-slate-500 whitespace-nowrap">
                                {new Date(log.created_at).toLocaleString()}
                              </td>
                              <td className="px-3 py-2 font-bold text-slate-800 dark:text-slate-200 uppercase">
                                <span
                                  className={`px-2 py-0.5 rounded text-[10px] font-semibold ${
                                    log.operation === 'optimize'
                                      ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950/60 dark:text-emerald-400'
                                      : log.operation === 'check'
                                      ? 'bg-sky-100 text-sky-800 dark:bg-sky-950/60 dark:text-sky-400'
                                      : 'bg-indigo-100 text-indigo-800 dark:bg-indigo-950/60 dark:text-indigo-400'
                                  }`}
                                >
                                  {log.operation}
                                </span>
                              </td>
                              <td className="px-3 py-2">
                                <span
                                  className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${
                                    isSuccess
                                      ? 'bg-emerald-100 dark:bg-emerald-950/50 text-emerald-700 dark:text-emerald-400'
                                      : isWarning
                                      ? 'bg-amber-100 dark:bg-amber-950/50 text-amber-700 dark:text-amber-400'
                                      : 'bg-rose-100 dark:bg-rose-950/50 text-rose-700 dark:text-rose-400'
                                  }`}
                                >
                                  {log.status}
                                </span>
                              </td>
                              <td className="px-3 py-2 text-right text-slate-700 dark:text-slate-300">
                                {log.tables_processed}
                              </td>
                              <td className="px-3 py-2 text-right font-semibold text-emerald-600 dark:text-emerald-400">
                                {log.bytes_reclaimed > 0 ? formatBytes(log.bytes_reclaimed) : '-'}
                              </td>
                              <td className="px-3 py-2 text-right text-slate-500">
                                {log.duration_ms}ms
                              </td>
                              <td className="px-3 py-2 text-slate-500 truncate max-w-[120px]">
                                {log.initiated_by || 'system'}
                              </td>
                              <td className="px-3 py-2 text-slate-600 dark:text-slate-300 truncate max-w-[200px]" title={log.details}>
                                {log.details}
                              </td>
                            </tr>
                          );
                        })
                      )}
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Modal Footer */}
        <div className="px-6 py-3.5 border-t border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-850 flex items-center justify-between text-xs">
          <div className="flex items-center gap-2 text-slate-500">
            <Activity className="w-3.5 h-3.5 text-emerald-500" />
            <span>Hostvra Autonomous Database Maintenance Engine</span>
          </div>
          <button
            onClick={onClose}
            className="px-4 py-2 rounded-xl bg-slate-200 dark:bg-surface-700 hover:bg-slate-300 dark:hover:bg-surface-600 text-slate-800 dark:text-slate-200 font-bold transition cursor-pointer"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
